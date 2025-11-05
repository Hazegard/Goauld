//go:build mini
// +build mini

package config

// getIPs returns the IP on the hosts, excluding local network addresses.
func getIPs() ([]string, []error) {

	return nil, nil
}

// PrivateSshdPassword return the static password.
func (a *Agent) PrivateSshdPassword() string {
	return ""
}

var agent *Agent

// Get returns the Agent global object.
func Get() *Agent {
	return agent
}

// WorkingDay holds the working day configuration.
type WorkingDay struct {
	Start string
	End   string
	TZ    string
}
