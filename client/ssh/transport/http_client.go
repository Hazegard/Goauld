package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	minLen  = 1
	maxLen  = 15
)

type HttpClient struct {
	host        string
	port        int
	scheme      string
	sessionID   string
	client      *http.Client
	debug       bool
	maxBodySize int64
	rateLimiter *rate.Limiter
}

func NewHttpClient(host string, port int, scheme string, debug bool, maxBodySize int64) *HttpClient {

	scheme = strings.ToLower(scheme)
	if scheme == "" || (scheme != "http" && scheme != "https") {
		scheme = "https"
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionSSL30,
		},
		MaxIdleConns:       0,
		IdleConnTimeout:    0,
		DisableCompression: true,
	}

	return &HttpClient{
		host:      host,
		port:      port,
		scheme:    scheme,
		sessionID: "",
		client: &http.Client{
			Transport: transport,
			Timeout:   time.Second * 30,
		},
		debug:       debug,
		maxBodySize: maxBodySize,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 50),
	}
}

func (c *HttpClient) req(method string, url string, body io.Reader) (*http.Request, error) {
	url = strings.TrimSuffix(removeScheme(url), "/")

	var fullURL string
	if (c.scheme == "https" && c.port == 443) || (c.scheme == "http" && c.port == 80) {
		fullURL = fmt.Sprintf("%s://%s/%s", c.scheme, url, randomFilename())
	} else {
		fullURL = fmt.Sprintf("%s://%s:%d/%s", c.scheme, url, c.port, randomFilename())
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	req.Host = removeScheme(c.host)
	// Cache control
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")

	// Modern Chrome headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Ch-Ua", "\"Google Chrome\";v=\"119\", \"Chromium\";v=\"119\", \"Not?A_Brand\";v=\"24\"")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "\"Windows\"")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("DNT", "1")

	return req, nil
}

func (c *HttpClient) handleConnection(conn net.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)

	defer cancel()
	if !c.rateLimiter.Allow() {
		return
	}

	defer conn.Close()

	// lAddr := conn.LocalAddr().String()
	rAddr := conn.RemoteAddr().String()

	done := make(chan struct{})

	var closeOne sync.Once

	safeClose := func() {
		closeOne.Do(func() {
			close(done)

			req, err := c.req(http.MethodPost, c.host, nil)
			if err != nil {
				log.Printf("http client: failed to send request: %s", err)
			}
			if req != nil {
				req = req.WithContext(ctx)
				req.Header.Set("X-Ephemeral", c.sessionID)
				resp, _ := c.client.Do(req)
				if resp != nil {
					resp.Body.Close()
				}
			}

		})
	}

	defer safeClose()

	go func() {
		defer safeClose()
		buf := make([]byte, c.maxBodySize)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := conn.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading from local connection: %v", err)
					}
					return
				}

				if n > 0 {
					log.Printf("Read %d bytes from local connection", n)
					req, err := c.req(http.MethodPost, c.host, nil)
					if err != nil {
						log.Printf("Error creating POST request: %v", err)
						return
					}

					req = req.WithContext(ctx)

					// start = time.Now()
					resp, err := c.client.Do(req)
					if err != nil {
						log.Printf("Error creating POST request: %v", err)
					}
					resp.Body.Close()
				}
			}
		}
	}()

	go func() {
		defer safeClose()
		for {
			select {
			case <-ctx.Done():
				log.Printf("Context cancelled, stopping polling for %s", rAddr)
				return
			case <-done:
				log.Printf("Polling stopped for %s", rAddr)
				return
			default:
				req, err := c.req(http.MethodGet, c.host, nil)
				if err != nil {
					log.Printf("Error creating GET request: %v", err)
					return
				}

				req = req.WithContext(ctx)
				req.Header.Set("X-Ephemeral", c.sessionID)

				resp, err := c.client.Do(req)
				if err != nil {
					log.Printf("Error creating GET request: %v", err)
					time.Sleep(time.Second)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, c.maxBodySize))
					log.Printf("Server returned non-200 status. Body: %s", string(body))
					resp.Body.Close()
					time.Sleep(time.Second)
					continue
				}

				data, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBodySize))
				resp.Body.Close()
				if err != nil {
					log.Printf("Error reading response body: %v", err)
					continue
				}

				if len(data) > 0 {
					if bytes.HasPrefix(data, []byte("<")) {
						log.Printf("Received HTML response instead of hex data")
						time.Sleep(time.Second)
						continue
					}

					decoded, err := hex.DecodeString(string(data))
					if err != nil {
						log.Printf("Error decoding response body: %v", err)
						time.Sleep(time.Second)
						continue
					}

					_, err = conn.Write(decoded)

					if err != nil {
						log.Printf("Error writing response body to local connection: %v", err)
						return
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Context timeout reached for %s", rAddr)
	case <-done:
		log.Println("Connection handler completed for %s", rAddr)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func randomString(min, max int) string {
	if min < 0 || max < min {
		min, max = 1, 15
	}
	length := min + rand.Intn(max-min+1)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randomFilename() string {
	extensions := []string{
		// Common web files
		".html", ".htm", ".php", ".asp", ".jsp", ".js", ".css",
		// Images
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".bmp",
		// Documents
		".pdf", ".txt", ".doc", ".docx",
		// Media
		".mp3", ".mp4", ".wav", ".avi",
		// Archives
		".zip", ".rar", ".7z",
		// Data
		".xml", ".json", ".csv",
		// Web fonts
		".woff", ".woff2", ".ttf", ".eot",
		// Config files
		".conf", ".cfg", ".ini",
	}
	return randomString(minLen, maxLen) + extensions[rand.Intn(len(extensions))]
}

func removeScheme(url string) string {
	return strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")
}
