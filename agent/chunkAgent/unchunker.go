//go:build mini

package chunkAgent

import (
	"Goauld/common/log"
	"sync"

	socketio "Goauld/common/socket.io"
)

type ChunkedReconstructor struct {
	mu    sync.Mutex
	m     map[int][]byte
	total int
}

func (cr *ChunkedReconstructor) AddChunk(agent *socketio.ChunkedAgent) bool {
	log.Info().Int("Chunk", agent.Chunk).Int("Total", agent.LastChunk).Int("Length", len(agent.Data)).Msg("Chunk received")
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.m == nil {
		cr.m = make(map[int][]byte)
		cr.total = agent.LastChunk
	}

	cr.m[agent.Chunk] = agent.Data
	if len(cr.m) == agent.LastChunk {
		return true
	}

	return false
}

func (cr *ChunkedReconstructor) Rebuild() []byte {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	data := []byte{}
	for i := 1; i <= cr.total; i++ {
		if cr.m[i] != nil {
			data = append(data, cr.m[i]...)
		}
	}

	return data
}
