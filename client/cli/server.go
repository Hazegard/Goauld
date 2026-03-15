package cli

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/server"
	"context"
	"sync/atomic"
	"time"
)

type Server struct {
	PrivKey string `default:"${_age_privKey}" group:"Server configuration:" name:"age-privkey" yaml:"age-privkey" json:"age-privkey" optional:"" help:"Age private key used by the server."`

	HTTPAddr  string `default:"${_http_listen_addr}" group:"Local addresses binding configuration:" name:"http-listen-addr" yaml:"http-listen-addr" json:"http-listen-addr" optional:"" help:"Address and port to bind for HTTP connections (port 0 = random)."`
	HTTPSAddr string `default:"${_https_listen_addr}" group:"Local addresses binding configuration:" name:"https-listen-addr" yaml:"https-listen-addr" json:"https-listen-addr" optional:"" help:"Address and port to bind for HTTPS connections (port 0 = random)."`
	SSHDAddr  string `default:"${_sshd_listen_addr}" group:"Local addresses binding configuration:" name:"ssh-listen-addr" yaml:"ssh-listen-addr" json:"ssh-listen-addr" optional:"" help:"Address and port to bind for SSH connections (port 0 = random)."`
	DNSAddr   string `default:"${_dns_listen_addr}" group:"Local addresses binding configuration:" name:"dns-listen-addr" yaml:"dns-listen-addr" json:"dns-listen-addr" optional:"" help:"Address and port to bind for DNS connections (port 0 = random)."`
	QuicAddr  string `default:"${_quic_listen_addr}" group:"Local addresses binding configuration:" name:"quic-listen-addr" yaml:"quic-listen-addr" json:"quic-listen-addr" optional:"" help:"Address and port to bind for QUIC connections (port 0 = random)."`
}

func (s *Server) Run(clientAPI *api.API, cfg ClientConfig) error {
	ctx, cancel := context.WithCancel(context.Background())

	var alreadyConnected atomic.Bool
	hooks := []func(id string, name string){
		func(id string, name string) {
			log.Info().Str("ID", id).Str("Name", name).Msg("agent connected")

			if alreadyConnected.CompareAndSwap(false, true) {
				lvl := log.GetLogLevel()
				log.SetLogLevel(-1)

				cfg.SSH.Target = name

				err := cfg.SSH.Run(clientAPI, cfg)
				log.UpdateLogLevel(lvl)
				if err != nil {
					log.Error().Err(err).Str("ID", id).Str("Name", name).Msg("agent failed")
				}
				clientAPI.KillAgent(id, true, false, cfg.PrivatePassword)
				time.Sleep(1 * time.Second)
				cancel()
			} else {
				clientAPI.KillAgent(id, true, false, cfg.PrivatePassword)
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

	return server.Run(ctx, cancel)
}
