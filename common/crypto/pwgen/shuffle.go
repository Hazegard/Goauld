package pwgen

import (
	"Goauld/common/log"
	"bytes"
	"compress/gzip"
	"container/heap"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func GenWordlist(seed string, wordlistPath string, outDir string) (string, error) {
	wordlist, err := shuffle(seed, 100, wordlistPath)
	if err != nil {
		log.Error().Err(err).Msg("could not generate wordlist")

		return "", fmt.Errorf("could not generate wordlist: %w", err)
	}

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err = gz.Write([]byte(strings.Join(wordlist, "\n")))
	if err != nil {
		_ = gz.Close()
		log.Error().Err(err).Msg("could not compile wordlist")

		return "", fmt.Errorf("could not gzip wordlist: %w", err)
	}
	err = gz.Close()
	if err != nil {
		log.Error().Err(err).Msg("could not compile wordlist")

		return "", fmt.Errorf("could not gzip wordlist: %w", err)
	}

	outPath := filepath.Join(outDir, "common", "crypto", "pwgen", "wordlist.txt.gz")
	err = os.WriteFile(
		outPath,
		b.Bytes(),
		0o600,
	)

	return outPath, err
}

type item struct {
	hash [32]byte
	line string
}

// maxHeap keeps the *largest* hash on top, so we can eject it
// when we find a smaller hash and size > N.
// nolint:recvcheck
type maxHeap []item

func (h maxHeap) Len() int { return len(h) }

// Less reversed => max-heap by hash lexicographic order.
func (h maxHeap) Less(i, j int) bool {
	return bytes.Compare(h[i].hash[:], h[j].hash[:]) > 0
}
func (h maxHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// nolint:forcetypeassert
func (h *maxHeap) Push(x any) { *h = append(*h, x.(item)) }
func (h *maxHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]

	return it
}

// lexicographic compare for [32]byte.
func lessHash(a, b [32]byte) bool {
	return bytes.Compare(a[:], b[:]) < 0
}

func shuffle(seed string, n int, wordlistPath string) ([]string, error) {
	if n <= 0 {
		return []string{}, errors.New("n must be > 0")
	}

	data, err := os.ReadFile(wordlistPath)
	if err != nil {
		return nil, fmt.Errorf("error reading wordlist file %s: %w", wordlistPath, err)
	}

	h := &maxHeap{}
	heap.Init(h)

	lineNo := 0
	for _, line := range strings.Split(string(data), "\n") {
		lineNo++

		key := fmt.Sprintf("%s|%d|%s", seed, lineNo, line)
		sum := sha256.Sum256([]byte(key))

		it := item{hash: sum, line: line}

		if h.Len() < n {
			heap.Push(h, it)
		} else {
			// if new hash is smaller than current largest, replace
			top := (*h)[0]
			if lessHash(it.hash, top.hash) {
				heap.Pop(h)
				heap.Push(h, it)
			}
		}
	}

	// heap now holds N smallest hashes, but unordered
	out := make([]item, h.Len())
	for i := range out {
		// nolint:forcetypeassert
		out[i] = heap.Pop(h).(item)
	}

	// sort ascending so output order is stable
	sort.Slice(out, func(i, j int) bool {
		return lessHash(out[i].hash, out[j].hash)
	})

	result := []string{}
	for _, it := range out {
		//nolint:forbidigo
		result = append(result, it.line)
	}

	return result, nil
}
