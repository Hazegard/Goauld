package utils

import "context"

type CancelReason int

const (
	Restart CancelReason = iota
	Exit
)

type GlobalCanceler struct {
	Cancel       context.CancelFunc
	CancelReason chan<- CancelReason
}

func (cg *GlobalCanceler) Exit() {
	cg.Cancel()
	cg.CancelReason <- Exit
}

func (cg *GlobalCanceler) Restart() {
	cg.Cancel()
	cg.CancelReason <- Restart
}
