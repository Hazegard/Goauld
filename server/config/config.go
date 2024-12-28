package config

import (
	"Goauld/common/cli"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/utils"
	"fmt"
	"github.com/alecthomas/kong"
	"path/filepath"
	"strings"
	"sync"
)

const APP_NAME = "Goa'uld"

var serverOnce sync.Once
var srvCfg *ServerConfig

var (
	_age_privKey = ""
	_http_port   = "80"
	_http_domain = "localhost"
	_sshd_port   = "2222"
	_verbosity   = "0"

	defaultValues = kong.Vars{
		"_age_privKey": _age_privKey,
		"_http_domain": _http_domain,
		"_http_port":   _http_port,
		"_sshd_port":   _sshd_port,
		"_verbosity":   _verbosity,
	}
)

type ServerConfig struct {
	PrivKey    string `default:"${_age_privKey}"  name:"age-key" optional:"" help:"Age private key to use."`
	httpDomain string `default:"${_http_domain}"  name:"http-domain" optional:"" help:"Domain to listen to."`
	HttpPort   int    `default:"${_http_port}"  name:"http-port" optional:"" help:"Remote port to bind to, 0 => Random."`
	SshdPort   int    `default:"${_sshd_port}"  name:"ssh-port" optional:"" help:"Remote port to bind to, 0 => Random."`
	Verbose    int    `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`
	Tls        bool   `default:"true" help:"Enable TLS."`
}

func InitServer() (*kong.Context, *ServerConfig, error) {
	cfg := &ServerConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfg, err
	}
	app := kong.Parse(cfg,
		kong.Name(APP_NAME),
		kong.Description("TODO"),
		kong.UsageOnError(),
		kong.Configuration(cli.YAML, filepath.Join(dir, strings.ToLower(APP_NAME)+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(APP_NAME)),
		defaultValues,
	)
	srvCfg = cfg

	log.SetLogLevel(cfg.Verbose)
	return app, srvCfg, nil
}

func Get() *ServerConfig {
	/*serverOnce.Do(func() {
		srvCfg = &ServerConfig{
			PrivKey:           privKey,
			httpDomain: ":3000",
			SshdPort:          0,
		}
	})*/
	return srvCfg
}

func (s *ServerConfig) Decrypt(data []byte) (string, error) {
	return crypto.AsymDecrypt(s.PrivKey, data)
}

func (s *ServerConfig) LocalSShServer() string {
	return fmt.Sprintf("%s:%d", "127.0.0.1", s.SshdPort)
}

func (s *ServerConfig) LocalHttpServer() string {
	return fmt.Sprintf("%s:%d", s.httpDomain, s.HttpPort)
}
