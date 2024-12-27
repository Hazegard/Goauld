package old

import (
	"net"
	"sync"
	"time"
)

type session struct {
	conn       net.Conn
	lastActive time.Time
	buffer     []byte
	mu         sync.Mutex
}

type SSHHttpServer struct {
	sessions sync.Map
}

func NewServer() *SSHHttpServer {
	s := &SSHHttpServer{
		sessions: sync.Map{},
	}
	return s
}

func (s *SSHHttpServer) cleanupSessions() {
	for {
		time.Sleep(1 * time.Minute)
		now := time.Now()
		s.sessions.Range(func(key, value interface{}) bool {
			session := value.(*session)
			session.mu.Lock()
			defer session.mu.Unlock()
			if now.Sub(session.lastActive) > 5*time.Minute {
				session.conn.Close()
				s.sessions.Delete(key)
			}
			return true
		})
	}
}
