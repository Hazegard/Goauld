//go:build windows

package proxy

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/alexbrainman/sspi"
	"github.com/alexbrainman/sspi/negotiate"
	"github.com/alexbrainman/sspi/ntlm"
)

// SSPITransport is an http.RoundTripper that performs NTLM or
// Negotiate (Kerberos) authentication using Windows SSPI
// when the server responds with WWW-Authenticate challenges.
//
// It is safe for concurrent use by multiple goroutines, but each
// request gets its own SSPI context.
type SSPITransport struct {
	// Base is the underlying transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// Optional explicit credentials. If Username is empty, the
	// current logged-on user's credentials are used.
	Domain   string
	Username string
	Password string

	// MaxAuthSteps limits the number of Negotiate token exchanges.
	// If zero or negative, a default of 5 is used.
	MaxAuthSteps int

	// MaxReplayBodySize, if > 0, limits the number of bytes that will be
	// buffered from the request body for potential replay during the
	// authentication handshake. If the request body exceeds this limit,
	// RoundTrip returns an error.
	// If 0, the body is read fully with no explicit limit.
	MaxReplayBodySize int64

	// RespectExistingAuth, if true, causes RoundTrip to skip SSPI-based
	// authentication if the request already contains an Authorization header.
	RespectExistingAuth bool

	mu       sync.Mutex
	negCred  *sspi.Credentials
	ntlmCred *sspi.Credentials
}

// ErrBodyTooLarge is returned when the request body exceeds MaxReplayBodySize.
var ErrBodyTooLarge = errors.New("sspi: request body too large to buffer for replay")

// RoundTrip implements http.RoundTripper.
func (t *SSPITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// If user already set Authorization and we must respect it, bypass auth.
	if t.RespectExistingAuth && req.Header.Get("Authorization") != "" {
		return base.RoundTrip(req)
	}

	// Buffer request body for retry.
	bodyBuf, err := t.bufferRequestBody(req)
	if err != nil {
		return nil, err
	}

	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Only handle server auth (401).
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Preserve original 401 for fallback.
	origBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sspi: read original 401 body: %w", err)
	}
	resp.Body.Close()

	// helper: return original response at any failure
	returnOriginal := func() (*http.Response, error) {
		resp.Body = io.NopCloser(bytes.NewReader(origBody))
		resp.ContentLength = int64(len(origBody))
		resp.Header.Del("Content-Encoding")
		return resp, nil
	}

	challenges := resp.Header.Values("Www-Authenticate")
	scheme := pickAuthScheme(challenges)
	if scheme == "" {
		return returnOriginal()
	}

	switch scheme {
	case "Negotiate":
		return t.roundTripNegotiate(req, bodyBuf, base, resp, origBody)
	case "NTLM":
		return t.roundTripNTLM(req, bodyBuf, base, resp, origBody)
	default:
		return returnOriginal()
	}
}

// Close releases handles.
func (t *SSPITransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var firstErr error
	if t.negCred != nil {
		if err := t.negCred.Release(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("sspi: release negotiate credentials: %w", err)
		}
		t.negCred = nil
	}
	if t.ntlmCred != nil {
		if err := t.ntlmCred.Release(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("sspi: release ntlm credentials: %w", err)
		}
		t.ntlmCred = nil
	}
	return firstErr
}

// -------------------- Credentials --------------------

func (t *SSPITransport) negotiateCred() (*sspi.Credentials, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.negCred != nil {
		return t.negCred, nil
	}

	var cred *sspi.Credentials
	var err error
	if t.Username != "" {
		cred, err = negotiate.AcquireUserCredentials(t.Domain, t.Username, t.Password)
	} else {
		cred, err = negotiate.AcquireCurrentUserCredentials()
	}
	if err != nil {
		return nil, fmt.Errorf("sspi: acquire negotiate credentials: %w", err)
	}
	t.negCred = cred
	return cred, nil
}

func (t *SSPITransport) ntlmCredHandle() (*sspi.Credentials, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ntlmCred != nil {
		return t.ntlmCred, nil
	}

	var cred *sspi.Credentials
	var err error
	if t.Username != "" {
		cred, err = ntlm.AcquireUserCredentials(t.Domain, t.Username, t.Password)
	} else {
		cred, err = ntlm.AcquireCurrentUserCredentials()
	}
	if err != nil {
		return nil, fmt.Errorf("sspi: acquire ntlm credentials: %w", err)
	}
	t.ntlmCred = cred
	return cred, nil
}

// -------------------- Helpers --------------------

func (t *SSPITransport) bufferRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	defer req.Body.Close()

	if t.MaxReplayBodySize > 0 {
		// We read up to limit+1 bytes so we can detect overflow.
		var buf bytes.Buffer
		lr := io.LimitedReader{R: req.Body, N: t.MaxReplayBodySize + 1}
		_, err := io.Copy(&buf, &lr)
		if err != nil {
			return nil, fmt.Errorf("sspi: read request body: %w", err)
		}
		if int64(buf.Len()) > t.MaxReplayBodySize {
			return nil, ErrBodyTooLarge
		}
		req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		req.ContentLength = int64(buf.Len())
		return buf.Bytes(), nil
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("sspi: read request body: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))
	return data, nil
}

// pickAuthScheme prefers Negotiate over NTLM when both are present.
func pickAuthScheme(challenges []string) string {
	// Prefer Negotiate.
	for _, c := range challenges {
		if authSchemeMatches(c, "Negotiate") {
			return "Negotiate"
		}
	}
	for _, c := range challenges {
		if authSchemeMatches(c, "NTLM") {
			return "NTLM"
		}
	}
	return ""
}

// authSchemeMatches checks whether a WWW-Authenticate header line
// contains the given scheme as its first token (case-insensitive).
func authSchemeMatches(header, scheme string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	if idx := strings.IndexByte(header, ','); idx >= 0 {
		header = header[:idx]
	}
	f := strings.Fields(header)
	if len(f) == 0 {
		return false
	}
	return strings.EqualFold(f[0], scheme)
}

// getChallengeToken finds the Base64 token in WWW-Authenticate headers
// for a specific scheme (e.g. "Negotiate" or "NTLM").
func getChallengeToken(challenges []string, scheme string) (string, bool) {
	for _, c := range challenges {
		// Split possible combined headers first.
		for _, part := range strings.Split(c, ",") {
			part = strings.TrimSpace(part)
			fields := strings.Fields(part)
			if len(fields) >= 2 && strings.EqualFold(fields[0], scheme) {
				return fields[1], true
			}
		}
	}
	return "", false
}

func cloneRequestWithBody(r *http.Request, body []byte) *http.Request {
	clone := r.Clone(r.Context())
	if body != nil {
		clone.Body = io.NopCloser(bytes.NewReader(body))
		clone.ContentLength = int64(len(body))
	} else {
		clone.Body = http.NoBody
		clone.ContentLength = 0
	}
	return clone
}

// servicePrincipalName builds an HTTP SPN for Negotiate.
// For typical HTTP services, SPN is "HTTP/host" or "HTTP/host:port"
// for non-default ports.
func servicePrincipalName(req *http.Request) string {
	host := req.URL.Hostname()
	if host == "" {
		host = req.URL.Host
	}

	port := req.URL.Port()
	if port != "" && port != "80" && port != "443" {
		return "HTTP/" + host + ":" + port
	}
	return "HTTP/" + host
}

func (t *SSPITransport) maxAuthSteps() int {
	if t.MaxAuthSteps <= 0 {
		return 5
	}
	return t.MaxAuthSteps
}

// -------------------- Negotiate / Kerberos --------------------

func (t *SSPITransport) roundTripNegotiate(orig *http.Request, body []byte, base http.RoundTripper, resp *http.Response, origBody []byte) (*http.Response, error) {
	returnOriginal := func() (*http.Response, error) {
		resp.Body = io.NopCloser(bytes.NewReader(origBody))
		resp.ContentLength = int64(len(origBody))
		resp.Header.Del("Content-Encoding")
		return resp, nil
	}

	cred, err := t.negotiateCred()
	if err != nil {
		return returnOriginal()
	}

	spn := servicePrincipalName(orig)

	var ctx *negotiate.ClientContext
	var outToken []byte

	t.mu.Lock()
	ctx, outToken, err = negotiate.NewClientContext(cred, spn)
	t.mu.Unlock()
	if err != nil {
		return returnOriginal()
	}
	defer ctx.Release()

	authHeader := "Negotiate " + base64.StdEncoding.EncodeToString(outToken)

	// First authenticated attempt.
	req1 := cloneRequestWithBody(orig, body)
	req1.Header.Set("Authorization", authHeader)

	resp2, err := base.RoundTrip(req1)
	if err != nil {
		return returnOriginal()
	}
	if resp2.StatusCode != http.StatusUnauthorized {
		return resp2, nil
	}

	for i := 0; i < t.maxAuthSteps(); i++ {
		challenges := resp2.Header.Values("Www-Authenticate")
		tokenB64, ok := getChallengeToken(challenges, "Negotiate")
		if !ok {
			return returnOriginal()
		}
		challenge, err := base64.StdEncoding.DecodeString(tokenB64)
		if err != nil {
			return returnOriginal()
		}

		// Update the context with the server token.
		_, outToken, err = ctx.Update(challenge)
		if err != nil {
			return returnOriginal()
		}

		resp2.Body.Close()

		authHeader = "Negotiate " + base64.StdEncoding.EncodeToString(outToken)
		reqN := cloneRequestWithBody(orig, body)
		reqN.Header.Set("Authorization", authHeader)

		resp2, err = base.RoundTrip(reqN)
		if err != nil {
			return returnOriginal()
		}
		if resp2.StatusCode != http.StatusUnauthorized {
			break
		}
	}

	return returnOriginal()
}

// -------------------- NTLM --------------------

func (t *SSPITransport) roundTripNTLM(orig *http.Request, body []byte, base http.RoundTripper, resp *http.Response, origBody []byte) (*http.Response, error) {
	returnOriginal := func() (*http.Response, error) {
		resp.Body = io.NopCloser(bytes.NewReader(origBody))
		resp.ContentLength = int64(len(origBody))
		resp.Header.Del("Content-Encoding")
		return resp, nil
	}

	cred, err := t.ntlmCredHandle()
	if err != nil {
		return returnOriginal()
	}

	var ctx *ntlm.ClientContext
	var outToken []byte

	t.mu.Lock()
	ctx, outToken, err = ntlm.NewClientContext(cred)
	t.mu.Unlock()
	if err != nil {
		return returnOriginal()
	}
	defer ctx.Release()

	authHeader := "NTLM " + base64.StdEncoding.EncodeToString(outToken)

	// First authenticated attempt.
	req1 := cloneRequestWithBody(orig, body)
	req1.Header.Set("Authorization", authHeader)

	resp2, err := base.RoundTrip(req1)
	if err != nil {
		return returnOriginal()
	}
	if resp2.StatusCode != http.StatusUnauthorized {
		return resp2, nil
	}

	challenges := resp2.Header.Values("Www-Authenticate")
	tokenB64, ok := getChallengeToken(challenges, "NTLM")
	if !ok {
		return returnOriginal()
	}

	challenge, err := base64.StdEncoding.DecodeString(tokenB64)
	if err != nil {
		return returnOriginal()
	}

	// Second NTLM message (Authenticate).
	outToken, err = ctx.Update(challenge)
	if err != nil {
		return returnOriginal()
	}

	resp2.Body.Close()

	authHeader = "NTLM " + base64.StdEncoding.EncodeToString(outToken)
	req2 := cloneRequestWithBody(orig, body)
	req2.Header.Set("Authorization", authHeader)

	return base.RoundTrip(req2)
}
