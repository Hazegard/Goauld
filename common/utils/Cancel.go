package utils

import "context"

type CancelReason int

const (
	Restart CancelReason = iota
	Exit
)

// GlobalCanceler is a cancelFunc that holds the information whether the cancellation requires a restart or an exit
type GlobalCanceler struct {
	Cancel       context.CancelFunc
	CancelReason chan<- CancelReason
}

// Exit triggers the cancellation and requires to exit
func (cg *GlobalCanceler) Exit() {
	cg.Cancel()
	cg.CancelReason <- Exit
}

// Restart triggers the cancellation and requires to restart
func (cg *GlobalCanceler) Restart() {
	cg.Cancel()
	cg.CancelReason <- Restart
}
