package config

import (
	"fmt"
	"os"
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

	_binaries_basicauth = "M6FWvoAMszJV@5Zj5R9JugbpsieCE9qumDIv6UWLZbxjKKz2j"
	_binaries_path      = "./binaries"

	_generate_config = "false"
	defaultValues    = kong.Vars{
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

		"_binaries_basicauth": _binaries_basicauth,
		"_binaries_path":      _binaries_path,

		"_generate_config": _generate_config,
	}
)

type ServerConfig struct {
	PrivKey string `default:"${_age_privKey}"  name:"age-privkey" optional:"" help:"Age private key to use."`

	HttpDomain string `default:"${_http_domain}"  name:"http-domain" optional:"" help:"Domain used to serve HTTP content (HTTP/Websockets)."`
	TlsDomain  string `default:"${_tls_domain}"  name:"tls-domain" optional:"" help:"Domain used to serve raw TLS content (SSH over TLS)."`

	HttpPort  int `default:"${_http_port}"  name:"http-port" optional:"" help:"HTTP port to bind to, 0 => Random."`
	HttpsPort int `default:"${_https_port}"  name:"https-port" optional:"" help:"HTTPS port to bind to, 0 => Random."`
	SshdPort  int `default:"${_sshd_port}"  name:"ssh-port" optional:"" help:"Remote port to bind to, 0 => Random."`

	Verbose int `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`

	Tls        bool   `default:"${_tls}" negatable:"" name:"tls" help:"Enable TLS."`
	TlsKey     string `help:"Path to TLS certificate key." name:"tls-key"`
	TlsCert    string `help:"Path to TLS certificate." name:"tls-cert"`
	NoDB       bool   `default:"${_no_db}" negatable:"" name:"db" help:"Disable database usage."`
	DbFileName string `default:"${_db_name}" name:"db-file-name" help:"Database filename to use."`

	AllowedIPs  []string `default:"${_allowed_ips}" name:"allowed-ips" help:"List of IP allowed to access the /manage/ endpoint."`
	AccessToken string   `default:"${_access_token}" name:"access-token" help:"Access token required to access the /manage/ endpoint."`
	AdminToken  string   `default:"${_access_token}" name:"admin-token" help:"Access token required to access the /manage/ endpoint."`

	BinariesBasicAuth    string `default:"${_binaries_basicauth}" name:"binaries-basic-auth" help:"HTTP Basic Auth used to access the binaries endpoint."`
	BinariesPathLocation string `default:"${_binaries_path}" name:"binaries-path-location" help:"Path where are stored binaries on the filesystem."`

	GenerateConfig bool   `default:"${_generate_config}" help:"Generate configuration file based on the current options."`
	ConfigFile     string `name:"config-file" type:"existingfile" optionnal:"" short:"c" help:"Configuration file to use."`
}

// InitServer initialize the application configuration
func InitServer() (*kong.Context, *ServerConfig, error) {
	cfgTmp := &ServerConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfgTmp, err
	}
	configSearchDir := []string{
		filepath.Join(dir, "agent_config.yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		configSearchDir = append(configSearchDir, home)
	}
	var kongOptions = []kong.Option{
		kong.Name(common.APP_NAME),
		kong.Description(common.Title("Server")),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(common.APP_NAME)),
		defaultValues,
	}
	_ = kong.Parse(cfgTmp, kongOptions...)
	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar, cfgTmp.ConfigFile))
	}
	cfg := &ServerConfig{}
	app := kong.Parse(cfg, kongOptions...)
	srvCfg = cfg

	log.SetLogLevel(cfg.Verbose)
	return app, cfg, nil
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

	basicAuth := strings.Split(s.BinariesBasicAuth, "@")
	if len(basicAuth) != 2 {
		return fmt.Errorf("invalid basic auth: %s", s.BinariesBasicAuth)
	}
	if basicAuth[0] == "" || basicAuth[1] == "" {
		return fmt.Errorf("invalid basic auth: %s", s.BinariesBasicAuth)
	}

	return nil
}

func (s *ServerConfig) IsCustomTLS() bool {
	return s.TlsCert != "" && s.TlsKey != ""
}

func (s *ServerConfig) GetTlsDomains() []string {
	return []string{s.TlsDomain, s.HttpDomain}
}

func (s *ServerConfig) GetBinariesBasicAuth() (string, string) {
	split := strings.Split(s.BinariesBasicAuth, "@")
	if len(s.BinariesBasicAuth) != 2 {
		return "", ""
	}

	return split[0], split[1]
}

func (s *ServerConfig) GenerateYAMLConfig() (string, error) {
	return cli.GenerateYAMLWithComments(*s)
}
