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

	_http_domain = "www.example.com"
	_tls_domain  = "app.example.com"

	// TODO: voir pour listen sur ine IP spécifique
	_http_listen_addr  = ":80"
	_https_listen_addr = ":443"
	_sshd_listen_addr  = ":2222"

	_verbosity = "0"

	_tls      = "true"
	_tls_cert = ""
	_tls_key  = ""

	_no_db   = "false"
	_db_name = common.AppName() + ".db"

	_allowed_ips  = "127.0.0.1,0.0.0.0/32"
	_access_token = "TODO_TOKEN"
	_admin_token  = "TODO_TOKEN"

	_binaries_basicauth = "username:password"
	_binaries_path      = "./binaries"

	_version         = "false"
	_generate_config = "false"
	_config_file     = ""

	defaultValues = kong.Vars{
		"_age_privKey": _age_privKey,

		"_http_domain": _http_domain,
		"_tls_domain":  _tls_domain,

		"_http_listen_addr":  _http_listen_addr,
		"_https_listen_addr": _https_listen_addr,
		"_sshd_listen_addr":  _sshd_listen_addr,

		"_verbosity": _verbosity,

		"_tls":      _tls,
		"_tls_cert": _tls_cert,
		"_tls_key":  _tls_key,

		"_no_db":   _no_db,
		"_db_name": _db_name,

		"_allowed_ips":  _allowed_ips,
		"_access_token": _access_token,
		"_admin_token":  _admin_token,

		"_binaries_basicauth": _binaries_basicauth,
		"_binaries_path":      _binaries_path,

		"_version":         _version,
		"_generate_config": _generate_config,
		"_config_file":     _config_file,
	}
)

var (
	description = "Server used to listen and manage the agent connections." +
		"\nThe server will try to load configuration from " + filepath.Join("$HOME", ".config", "goauld_server.yaml") +
		"\nAs well as goauld_server.yaml on the current directory."
)

type ServerConfig struct {
	PrivKey string `default:"${_age_privKey}"  name:"age-privkey" optional:"" help:"Age private key to use."`

	HttpDomain []string `default:"${_http_domain}"  name:"http-domain" optional:"" help:"Domain used to serve HTTP content (HTTP/Websockets)."`
	TlsDomain  []string `default:"${_tls_domain}"  name:"tls-domain" optional:"" help:"Domain used to serve raw TLS content (SSH over TLS)."`

	HttpAddr  string `default:"${_http_listen_addr}"  name:"http-listen-addr" optional:"" help:"HTTP address to bind to, port 0 => Random."`
	HttpsAddr string `default:"${_https_listen_addr}"  name:"https-listen-addr" optional:"" help:"HTTPS address to bind to, port 0 => Random."`
	SshdAddr  string `default:"${_sshd_listen_addr}"  name:"ssh-listen-addr" optional:"" help:"Remote address to bind to, port 0 => Random."`

	Verbose int `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`

	Tls     bool   `default:"${_tls}" negatable:"" name:"tls" help:"Enable TLS."`
	TlsKey  string `default:"${_tls_key}" help:"Path to TLS certificate key." name:"tls-key"`
	TlsCert string `default:"${_tls_cert}" help:"Path to TLS certificate." name:"tls-cert"`

	NoDB       bool   `default:"${_no_db}" negatable:"" name:"db" help:"Disable database usage."`
	DbFileName string `default:"${_db_name}" name:"db-file-name" help:"Database filename to use."`

	AllowedIPs  []string `default:"${_allowed_ips}" name:"allowed-ips" help:"List of IP allowed to access the /manage/ endpoint."`
	AccessToken string   `default:"${_access_token}" name:"access-token" help:"Access token required to access the /manage/ endpoint."`
	AdminToken  string   `default:"${_admin_token}" name:"admin-token" help:"Access token required to access the /manage/ endpoint."`

	BinariesBasicAuth    string `default:"${_binaries_basicauth}" name:"binaries-basic-auth" help:"HTTP Basic Auth used to access the binaries endpoint."`
	BinariesPathLocation string `default:"${_binaries_path}" name:"binaries-path-location" help:"Path where are stored binaries on the filesystem."`

	Version        bool   `default:"${_version}" name:"version" short:"V" help:"Show version information"`
	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" help:"Generate configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" name:"config-file" optionnal:"" short:"c" help:"Configuration file to use."`
}

// InitServer initialize the application configuration
func InitServer() (*kong.Context, *ServerConfig, error) {
	cfgTmp := &ServerConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfgTmp, err
	}
	configSearchDir := []string{
		filepath.Join(dir, "goauld_server.yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", "goauld_server.yaml")
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(common.AppName()),
		kong.Description(common.Title(common.App_Name+" server") + "\n" + description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(common.AppName())),
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

// LocalSShAddr return the local SSH address
func (s *ServerConfig) LocalSShAddr() string {
	return s.SshdAddr
}

// UpdateSSHAddr return the local SSH address
func (s *ServerConfig) UpdateSSHAddr(port int) {
	split := strings.Split(s.SshdAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.SshdAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// LocalHttpsAddr return the local HTTPS address
func (s *ServerConfig) LocalHttpsAddr() string {
	return s.HttpsAddr
}

// UpdateHTTPSAddr return the local SSH address
func (s *ServerConfig) UpdateHTTPSAddr(port int) {
	split := strings.Split(s.HttpsAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.HttpsAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// UpdateHTTPAddr return the local SSH address
func (s *ServerConfig) UpdateHTTPAddr(port int) {
	split := strings.Split(s.HttpsAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.HttpsAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// LocalHttpAddr return the local HTTP address
func (s *ServerConfig) LocalHttpAddr() string {
	return s.HttpAddr
}

// Validate perform kong validation to ensure that fields are correct
func (s *ServerConfig) Validate() error {
	for _, ip := range s.AllowedIPs {
		if !net.IsIPorCIDR(ip) {
			return fmt.Errorf("invalid IP address: %s", ip)
		}
	}

	basicAuth := strings.Split(s.BinariesBasicAuth, ":")
	if len(basicAuth) < 2 {
		return fmt.Errorf("invalid basic auth: %s", s.BinariesBasicAuth)
	}
	if basicAuth[0] == "" || strings.Join(basicAuth[1:], ":") == "" {
		return fmt.Errorf("invalid basic auth: %s", s.BinariesBasicAuth)
	}

	return nil
}

// IsCustomTLS return whether a TLS certificate is provided
func (s *ServerConfig) IsCustomTLS() bool {
	return s.TlsCert != "" && s.TlsKey != ""
}

// GetTlsDomains return the list of domains served under TLS
func (s *ServerConfig) GetTlsDomains() []string {
	return append(s.TlsDomain, s.HttpDomain...)
}

// GetBinariesBasicAuth return the username and password used to restrict the hosted binaries
func (s *ServerConfig) GetBinariesBasicAuth() (string, string) {
	split := strings.Split(s.BinariesBasicAuth, ":")
	if len(s.BinariesBasicAuth) < 2 {
		return "", ""
	}

	return split[0], strings.Join(split[1:], ":")
}

// GenerateYAMLConfig return the yaml configuration using the current configuration
// (command line argument and parsed configuration files)
func (s *ServerConfig) GenerateYAMLConfig() (string, error) {
	s.GenerateConfig = false
	return cli.GenerateYAMLWithComments(*s)
}

// GenerateYAMLConfig return the yaml configuration using the current configuration
// (command line argument and parsed configuration files)
func (s *ServerConfig) GenerateSafeYAMLConfig() (string, error) {
	ss := *s
	ss.GenerateConfig = false
	ss.AdminToken = ""
	ss.AccessToken = ""
	ss.PrivKey = ""
	return cli.GenerateYAMLWithComments(ss)
}
