package ssh

type ReverseDynamicPortForward struct {
	RemotePort int
}

type ReversePortForwarding struct {
	RemotePort int
	LocalPort  int
	LocalIP    string
}
