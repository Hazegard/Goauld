//nolint:revive
package context

import "context"

type CancelStatus int

const (
	Restart CancelStatus = iota
	Exit
	Delete
)

// CancelReason allows to pass the cancellation information.
type CancelReason struct {
	Status CancelStatus
	Msg    string
}

// GlobalCanceler is a cancelFunc that holds the information whether the cancellation requires a restart or an exit.
type GlobalCanceler struct {
	Cancel       context.CancelFunc
	CancelReason chan<- CancelReason
}

// Exit triggers the cancellation and requires to exit.
func (cg *GlobalCanceler) Exit(msg string) {
	cg.Cancel()
	cg.CancelReason <- CancelReason{Exit, msg}
}

// Restart triggers the cancellation and requires to restart.
func (cg *GlobalCanceler) Restart(msg string) {
	cg.Cancel()
	cg.CancelReason <- CancelReason{Restart, msg}
}

// Delete triggers the cancellation and requires to exit and delete the agent.
func (cg *GlobalCanceler) Delete(msg string) {
	cg.Cancel()
	cg.CancelReason <- CancelReason{Delete, msg}
}
