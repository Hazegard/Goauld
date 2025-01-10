package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/net"
	"Goauld/common/utils"

	"github.com/alecthomas/kong"
)

var srvCfg *ServerConfig

// All the default values used in the configuration
// They can be overridden via:
//
// From the most priority to the least
// 1. Command line argument (--config=1 )
// 2. Environment variable (CONFIG=1)
// 3. Configuration file (./config.yaml)
// 4. Compile defined variable (-ldflag)
// 5. Hardcoded value (defined below)
var (
	_age_privKey = ""

	_http_domain = "a.hazegard.fr"
	_tls_domain  = "b.hazegard.fr"

	// TODO: voir pour listen sur ine IP spécifique
	_http_port  = "80"
	_https_port = "443"
	_sshd_port  = "2222"

	_verbosity = "0"
	_tls       = "true"

	_no_db   = "false"
	_db_name = common.APP_NAME + ".db"

	_allowed_ips  = "127.0.0.1,0.0.0.0/32"
	_access_token = "TODO_TOKEN"
	_admin_token  = "TODO_TOKEN"

	defaultValues = kong.Vars{
		"_age_privKey": _age_privKey,

		"_http_domain": _http_domain,
		"_tls_domain":  _tls_domain,

		"_http_port":  _http_port,
		"_https_port": _https_port,
		"_sshd_port":  _sshd_port,

		"_verbosity": _verbosity,
		"_tls":       _tls,

		"_no_db":   _no_db,
		"_db_name": _db_name,

		"_allowed_ips":  _allowed_ips,
		"_access_token": _access_token,
		"_admin_token":  _admin_token,
	}
)

type ServerConfig struct {
	PrivKey string `default:"${_age_privKey}"  name:"age-key" optional:"" help:"Age private key to use."`

	HttpDomain string `default:"${_http_domain}"  name:"http-domain" optional:"" help:"Domain used to serve HTTP content (HTTP/Websockets)."`
	TlsDomain  string `default:"${_tls_domain}"  name:"tls-domain" optional:"" help:"Domain used to serve raw TLS content (SSH over TLS)."`

	HttpPort  int `default:"${_http_port}"  name:"http-port" optional:"" help:"HTTP port to bind to, 0 => Random."`
	HttpsPort int `default:"${_https_port}"  name:"https-port" optional:"" help:"HTTPS port to bind to, 0 => Random."`
	SshdPort  int `default:"${_sshd_port}"  name:"ssh-port" optional:"" help:"Remote port to bind to, 0 => Random."`

	Verbose int `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`

	Tls        bool   `default:"${_tls}" negatable:"" help:"Enable TLS."`
	TlsKey     string `help:"Path to TLS certificate key."`
	TlsCert    string `help:"Path to TLS certificate."`
	NoDB       bool   `default:"${_no_db}" negatable:"" help:"Disable database usage."`
	DbFileName string `default:"${_db_name}" help:"Database filename to use."`

	AllowedIPs  []string `default:"${_allowed_ips}" name:"allowed-ip" help:"List of IP allowed to access the /manage/ endpoint."`
	AccessToken string   `default:"${_access_token}" help:"Access token required to access the /manage/ endpoint."`
	AdminToken  string   `default:"${_access_token}" help:"Access token required to access the /manage/ endpoint."`
}

// InitServer initialize the application configuration
func InitServer() (*kong.Context, *ServerConfig, error) {
	cfg := &ServerConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfg, err
	}
	app := kong.Parse(cfg,
		kong.Name(common.APP_NAME),
		kong.Description("TODO"),
		kong.UsageOnError(),
		kong.Configuration(cli.YAML, filepath.Join(dir, strings.ToLower(common.APP_NAME)+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(common.APP_NAME)),
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

// Decrypt returns the encrypted data using the configured private key
func (s *ServerConfig) Decrypt(data []byte) (string, error) {
	return crypto.AsymDecrypt(s.PrivKey, data)
}

// LocalSShServer return the local SSH address
func (s *ServerConfig) LocalSShServer() string {
	return fmt.Sprintf("%s:%d", "127.0.0.1", s.SshdPort)
}

// LocalHttpsServer return the local HTTPS address
func (s *ServerConfig) LocalHttpsServer() string {
	return fmt.Sprintf("%s:%d", "0.0.0.0", s.HttpsPort)
}

// LocalHttpServer return the local HTTP address
func (s *ServerConfig) LocalHttpServer() string {
	return fmt.Sprintf("%s:%d", "0.0.0.0", s.HttpPort)
}

// Validate perform kong validation to ensure that fields are correct
func (s *ServerConfig) Validate() error {
	for _, ip := range s.AllowedIPs {
		if !net.IsIPorCIDR(ip) {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}
	return nil
}

func (s *ServerConfig) IsCustomTLS() bool {
	return s.TlsCert != "" && s.TlsKey != ""
}

func (s *ServerConfig) GetTlsDomains() []string {
	return []string{s.TlsDomain, s.HttpDomain}
}
