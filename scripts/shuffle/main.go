package main

import (
	"bufio"
	"bytes"
	"container/heap"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

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

func main() {
	var (
		seed = flag.String("seed", "42", "seed string (same seed => same output)")
		n    = flag.Int("n", 500, "number of lines to output")
		in   = flag.String("in", "", "input file (default stdin)")
	)
	flag.Parse()

	if *n <= 0 {
		return
	}

	var r io.Reader = os.Stdin
	if *in != "" {
		f, err := os.Open(*in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "open:", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	sc := bufio.NewScanner(r)
	// increase buffer in case of long lines
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	h := &maxHeap{}
	heap.Init(h)

	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()

		key := fmt.Sprintf("%s|%d|%s", *seed, lineNo, line)
		sum := sha256.Sum256([]byte(key))

		it := item{hash: sum, line: line}

		if h.Len() < *n {
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
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)

		return
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

	for _, it := range out {
		//nolint:forbidigo
		fmt.Println(it.line)
	}
}
