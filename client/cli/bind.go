package cli

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/server"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

type Bind struct {
	PrivKey string `default:"${_age_privKey}" group:"Server configuration:" name:"age-privkey" yaml:"age-privkey" json:"age-privkey" optional:"" help:"Age private key used by the server."`

	HTTPAddr  string `default:"${_http_listen_addr}" group:"Local addresses binding configuration:" name:"http-listen-addr" yaml:"http-listen-addr" json:"http-listen-addr" optional:"" help:"Address and port to bind for HTTP connections (port 0 = random)."`
	HTTPSAddr string `default:"${_https_listen_addr}" group:"Local addresses binding configuration:" name:"https-listen-addr" yaml:"https-listen-addr" json:"https-listen-addr" optional:"" help:"Address and port to bind for HTTPS connections (port 0 = random)."`
	SSHDAddr  string `default:"${_sshd_listen_addr}" group:"Local addresses binding configuration:" name:"ssh-listen-addr" yaml:"ssh-listen-addr" json:"ssh-listen-addr" optional:"" help:"Address and port to bind for SSH connections (port 0 = random)."`
	DNSAddr   string `default:"${_dns_listen_addr}" group:"Local addresses binding configuration:" name:"dns-listen-addr" yaml:"dns-listen-addr" json:"dns-listen-addr" optional:"" help:"Address and port to bind for DNS connections (port 0 = random)."`
	QuicAddr  string `default:"${_quic_listen_addr}" group:"Local addresses binding configuration:" name:"quic-listen-addr" yaml:"quic-listen-addr" json:"quic-listen-addr" optional:"" help:"Address and port to bind for QUIC connections (port 0 = random)."`
	Agent     string `arg:"" default:"_{_bind_agent}" group:"Bind" name:"bind-agent" yaml:"bind-agent" json:"bind-agent" help:"The address of the agent to bind to"`
	Kill      bool   `group:"Bind" name:"kill" yaml:"kill" json:"kill" help:"Kill the agent on disconnection"`
}

func (s *Bind) Run(clientAPI *api.API, cfg ClientConfig) error {
	ctx, cancel := context.WithCancel(context.Background())

	var alreadyConnected atomic.Bool
	hooks := []func(id string, name string){
		func(id string, name string) {
			log.Info().Str("ID", id).Str("Name", name).Msg("agent connected")

			if alreadyConnected.CompareAndSwap(false, true) {
				lvl := log.GetLogLevel()
				// We disable logs to not pollute the shell session
				log.SetLogLevel(-1)

				cfg.SSH.Target = name

				err := cfg.SSH.Run(clientAPI, cfg)
				log.UpdateLogLevel(lvl)
				if err != nil {
					log.Error().Err(err).Str("ID", id).Str("Name", name).Msg("agent failed")
				}
				if cfg.Bind.Kill {
					_ = clientAPI.KillAgent(id, true, false, cfg.PrivatePassword)
					time.Sleep(1 * time.Second)
				}
				cancel()
			} else {
				_ = clientAPI.KillAgent(id, true, false, cfg.PrivatePassword)
			}
		},
	}

	realServer := config.ServerConfig{
		PrivKey:              s.PrivKey,
		HTTPDomain:           []string{"goauld.local"},
		TLSDomain:            nil,
		DNSDomain:            "",
		DNSDomainAlt:         "",
		HTTPAddr:             s.HTTPAddr,
		HTTPSAddr:            s.HTTPSAddr,
		SSHDAddr:             s.SSHDAddr,
		DNSAddr:              s.DNSAddr,
		QuicAddr:             s.QuicAddr,
		TLS:                  false,
		TLSKey:               "",
		TLSCert:              "",
		LetsEncryptMail:      "",
		Quic:                 false,
		DNS:                  false,
		NoDB:                 true,
		DbFileName:           "",
		AllowedIPs:           nil,
		AccessToken:          []string{cfg.AccessToken},
		AdminToken:           []string{cfg.AdminToken},
		BinariesBasicAuth:    "",
		BinariesPathLocation: "",
		Verbose:              0,
		Quiet:                false,
		Version:              false,
		GenerateConfig:       false,
		ConfigFile:           "",
		Hooks:                hooks,
		SingleAgent:          true,
	}

	config.Set(&realServer)

	go func() {
		err := server.Run(ctx, cancel)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start server")
			cancel()
		}
	}()
	time.Sleep(2 * time.Second)
	httpPort := strings.Split(cfg.Bind.HTTPAddr, ":")[1]

	srvSocketIO := fmt.Sprintf("http://%s:%s/live/00000000000000000000000000000000/?EIO=4&transport=websocket", "127.0.0.1", httpPort)
	agentSocketIO := fmt.Sprintf("http://%s/live/", cfg.Bind.Agent)
	srvWssh := fmt.Sprintf("http://%s:%s/wssh/00000000000000000000000000000000", "127.0.0.1", httpPort)
	agentWssh := fmt.Sprintf("http://%s/wssh/", cfg.Bind.Agent)

	go PipeWS(srvSocketIO, agentSocketIO, ctx)

	return PipeWS(srvWssh, agentWssh, ctx)
}

func newWs(address string, ctx context.Context) (*websocket.Conn, error) {
	srcCon, res, err := websocket.Dial(ctx, address, &websocket.DialOptions{
		HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		}},
	})
	if err != nil {
		log.Error().Err(err).Str("Address", address).Msg("Failed to dial the WSSH local server through websocket")

		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	return srcCon, nil
}

func PipeWS(src string, dst string, ctx context.Context) error {
	srcConn, err := newWs(src, ctx)
	if err != nil {
		return err
	}
	dstConn, err := newWs(dst, ctx)
	if err != nil {
		return err
	}

	go func() {
		err := wsPipe(ctx, srcConn, dstConn)
		if err != nil {
			log.Error().Err(err).Str("Source", src).Str("Destination", dst).Msg("Error piping")
		}
	}()
	err = wsPipe(ctx, dstConn, srcConn)
	if err != nil {
		log.Error().Err(err).Str("Source", dst).Str("Destination", src).Msg("Error piping")
	}

	return err
}

func wsPipe(ctx context.Context, src, dst *websocket.Conn) error {
	for {
		msgType, data, err := src.Read(ctx)
		if err != nil {
			return err
		}

		err = dst.Write(ctx, msgType, data)
		if err != nil {
			return err
		}
	}
}
