package main

import (
	"encoding/json"
	"fmt"

	"Goauld/common/ssh"
)

func main() {
	bbb := []byte("{\"serverPort\":54321,\"agentPort\":-1,\"agentIP\":\"0.0.0.0\",\"tag\":\"SSHD\"}")
	ccc := ssh.RemotePortForwarding{}
	fmt.Println(string(bbb))
	err := json.Unmarshal(bbb, &ccc)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", ccc)

	bb := []byte("[{\"serverPort\":54321,\"agentPort\":-1,\"agentIP\":\"0.0.0.0\",\"tag\":\"SSHD\"},{\"serverPort\":54322,\"agentPort\":-1,\"agentIP\":\"0.0.0.0\",\"tag\":\"SOCKS\"}]")
	cc := []ssh.RemotePortForwarding{}
	fmt.Println(string(bb))
	err = json.Unmarshal(bb, &cc)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", cc)
}
