//go:build !mini

package pwgen

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func GetXKCDPassword(wlPath string) (string, error) {
	g := NewGenerator()
	wordList, err := getWordList(wlPath)
	if err != nil {
		return "", fmt.Errorf("error getting word list: %w", err)
	}
	g.UseCustomWordlist(wordList)
	g.SetDelimiter("-")
	g.SetCapitalize(true)
	g.SetNumWords(3)

	return g.GeneratePasswordString(), nil
}

func getWordList(wlPath string) ([]string, error) {
	var content []byte
	var err error
	if wlPath == "" {
		content, err = sources.ReadFile(wl_name)
	} else {
		content, err = os.ReadFile(wlPath)
	}
	if err != nil {
		return nil, fmt.Errorf("could not read file wordlist: %w", err)
	}
	content, err = decompressGz(content)
	if err != nil {
		return nil, fmt.Errorf("could not decompress wordlist: %w", err)
	}
	wordlist := []string{}
	for _, word := range strings.Split(string(content), "\n") {
		wordlist = append(wordlist, strings.TrimSpace(word))
	}

	return wordlist, nil
}

func decompressGz(data []byte) ([]byte, error) {
	// Create a gzip reader from the byte slice
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Read the decompressed data
	var decompressedData []byte
	buf := make([]byte, 1024) // Buffer to read the data in chunks
	for {
		n, err := gzipReader.Read(buf)
		if n > 0 {
			decompressedData = append(decompressedData, buf[:n]...)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read decompressed data: %w", err)
		}
	}

	return decompressedData, nil
}
