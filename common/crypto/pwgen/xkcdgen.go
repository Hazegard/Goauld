package pwgen

import (
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"strings"
)

func GetXKCDPassword() (string, error) {
	g := NewGenerator()
	wordList, err := getWordList()
	if err != nil {
		return "", fmt.Errorf("error getting word list: %v", err)
	}
	g.UseCustomWordlist(wordList)
	g.SetDelimiter("-")
	g.SetCapitalize(true)
	g.SetNumWords(3)
	return g.GeneratePasswordString(), nil
}

//go:embed wordlist.txt.gz
var sources embed.FS

func getWordList() ([]string, error) {
	content, err := sources.ReadFile("wordlist.txt.gz")
	if err != nil {
		return nil, fmt.Errorf("could not read file wordlist: %v", err)
	}
	content, err = decompressGz(content)
	if err != nil {
		return nil, fmt.Errorf("could not decompress wordlist: %v", err)
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
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
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
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read decompressed data: %v", err)
		}
	}

	return decompressedData, nil
}
