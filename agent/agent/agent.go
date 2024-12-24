package agent

import (
	"Goauld/common/crypto"
	"crypto/md5"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/denisbrodbeck/machineid"
	"os"
	"os/user"
)

type Agent struct {
	Id                       string
	SShPrivateKey            string
	AgePubKey                string
	SharedSecret             string
	Cryptor                  *crypto.SymCryptor
	cfg                      *Config
	RemoteDynamicPortForward []int
	RemotePortForward        []int
}

var agent *Agent

var agePubKey = "age1fz7j9zck3qmafdkynu3ldvkjdrsstanhz8py8scx07hw7vja7aysuccrtn"

func InitAgent() (*kong.Context, error) {
	ctx, cfg, err := parse()
	if err != nil {
		return nil, fmt.Errorf("parsing arguments: %v", err)
	}
	sharedSecret, err := crypto.GeneratePassword(crypto.PasswordLength)
	if err != nil {
		return nil, fmt.Errorf("error generating ssh password: %v", err)
	}
	if cfg.LocalSshPassword == "" {
		sshPassword, err := crypto.GeneratePassword(crypto.PasswordLength)
		if err != nil {
			return nil, fmt.Errorf("error generating ssh password: %v", err)
		}
		cfg.LocalSshPassword = sshPassword
	}

	cryptor, err := crypto.NewCryptor(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("initializing cryptor: %v", err)
	}

	mid, err := machineid.ID()
	if err != nil {
		return nil, fmt.Errorf("error generating machine id: %v", err)
	}
	id := fmt.Sprintf("%x", md5.Sum([]byte(mid)))

	if cfg.Name == _name {
		userName, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error getting current user: %v", err)
		}
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %v", err)
		}
		cfg.Name = fmt.Sprintf("%s@%s", userName.Username, hostname)
	}

	agent = &Agent{
		Id:           id,
		AgePubKey:    agePubKey,
		cfg:          cfg,
		SharedSecret: sharedSecret,
		Cryptor:      cryptor,
	}
	return ctx, nil
}

func Get() *Agent {
	return agent
}

func (a *Agent) LocalSShdAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.SshdPort)
}

func (a *Agent) IsSshdRandomPort() bool {
	return a.cfg.SshdPort == 0
}

func (a *Agent) LocalSShdPassword() string {
	return a.cfg.LocalSshPassword
}

func (a *Agent) SetLocalSshdPort(p int) {
	a.cfg.SshdPort = p
}

func (a *Agent) ControlSshServer() string {
	return a.cfg.SshServer
}

func (a *Agent) IsRemoteForwardedSshdPortRandom() bool {
	return a.cfg.RsshPort == 0
}

func (a *Agent) RemoteForwardedSshdAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.RsshPort)
}

func (a *Agent) SocketIoUrl() string {
	return fmt.Sprintf("http://%s/socket.io/", a.cfg.Server)
}

func (a *Agent) parseRemotePortForward() error {

	//rpfs := strings.Split(a.Config().RemoteDynamicPortForwarding, ",")
	//for _, rpf := range rpfs {

	//}
	return nil
}

func (a *Agent) Name() string {
	return a.cfg.Name
}

// func getID() string {
// 	id, err != machin
// }
