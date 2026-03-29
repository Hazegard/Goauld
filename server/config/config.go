// Package config holds the server configuration
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/net"

	"github.com/alecthomas/kong"
)

var srvCfg *ServerConfig

// All the default values used in the configuration
// They can be overridden via:
//
// From the most priority to the least
// 1. Command line argument (--config=1)
// 2. Environment variable (CONFIG=1)
// 3. Configuration file (./config.yaml)
// 4. Compile defined variable (-ldflag)
// 5. Hardcoded value (defined below).
var (
	_age_privKey = "" //nolint:revive

	_http_domain    = "www.example.com" //nolint:revive
	_tls_domain     = "app.example.com" //nolint:revive
	_dns_domain     = "t.example.com"   //nolint:revive
	_dns_domain_alt = "s.example.com"   //nolint:revive

	_http_listen_addr  = ":80"   //nolint:revive
	_https_listen_addr = ":443"  //nolint:revive
	_sshd_listen_addr  = ":2222" //nolint:revive
	_dns_listen_addr   = ":53"   //nolint:revive
	_quic_listen_addr  = ":443"  //nolint:revive

	_verbosity = "0"
	_quiet     = "false"

	_tls               = "true"
	_tls_cert          = "" //nolint:revive
	_tls_key           = "" //nolint:revive
	_letsencrypt_email = "mail@example.com"
	_quic              = "true" //nolint:revive

	_dns = "true"

	_no_db   = "false"                  //nolint:revive
	_db_name = common.AppName() + ".db" //nolint:revive

	_allowed_ips  = "127.0.0.1,0.0.0.0/32" //nolint:revive
	_access_token = "TODO_TOKEN"           //nolint:revive
	_admin_token  = "TODO_TOKEN"           //nolint:revive

	_binaries_basicauth = "username:password" //nolint:revive
	_binaries_path      = "./binaries"        //nolint:revive

	_version         = "false"
	_generate_config = "false" //nolint:revive
	_config_file     = ""      //nolint:revive

	defaultValues = kong.Vars{
		"_age_privKey": _age_privKey,

		"_http_domain":    _http_domain,
		"_tls_domain":     _tls_domain,
		"_dns_domain":     _dns_domain,
		"_dns_domain_alt": _dns_domain_alt,

		"_http_listen_addr":  _http_listen_addr,
		"_https_listen_addr": _https_listen_addr,
		"_sshd_listen_addr":  _sshd_listen_addr,
		"_dns_listen_addr":   _dns_listen_addr,
		"_quic_listen_addr":  _quic_listen_addr,

		"_verbosity": _verbosity,
		"_quiet":     _quiet,

		"_tls":               _tls,
		"_tls_cert":          _tls_cert,
		"_tls_key":           _tls_key,
		"_letsencrypt_email": _letsencrypt_email,
		"_quic":              _quic,

		"_dns": _dns,

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

// ServerConfig represents the global server configuration.
type ServerConfig struct {
	PrivKey string `default:"${_age_privKey}" group:"Server configuration:" name:"age-privkey" yaml:"age-privkey" json:"age-privkey" required:"" help:"Age private key used by the server."`

	HTTPDomain   []string `default:"${_http_domain}" group:"Domain configuration:" name:"http-domain" yaml:"http-domain" json:"http-domain" optional:"" help:"Domains used to serve HTTP and WebSocket traffic."`
	TLSDomain    []string `default:"${_tls_domain}" group:"Domain configuration:" name:"tls-domain" yaml:"tls-domain" json:"tls-domain" optional:"" help:"Domains used to serve raw TLS traffic (SSH over TLS)."`
	DNSDomain    string   `default:"${_dns_domain}" group:"Domain configuration:" name:"dns-domain" yaml:"dns-domain" json:"dns-domain" optional:"" help:"Domain used to serve DNS-based traffic (SSH over DNS)."`
	DNSDomainAlt string   `default:"${_dns_domain_alt}" group:"Domain configuration:" name:"dns-domain-alt" yaml:"dns-domain-alt" json:"dns-domain-alt" optional:"" help:"Domain used to serve DNS-based traffic (SSH over DNS-ALT)."`

	HTTPAddr  string `default:"${_http_listen_addr}" group:"Local addresses binding configuration:" name:"http-listen-addr" yaml:"http-listen-addr" json:"http-listen-addr" optional:"" help:"Address and port to bind for HTTP connections (port 0 = random)."`
	HTTPSAddr string `default:"${_https_listen_addr}" group:"Local addresses binding configuration:" name:"https-listen-addr" yaml:"https-listen-addr" json:"https-listen-addr" optional:"" help:"Address and port to bind for HTTPS connections (port 0 = random)."`
	SSHDAddr  string `default:"${_sshd_listen_addr}" group:"Local addresses binding configuration:" name:"ssh-listen-addr" yaml:"ssh-listen-addr" json:"ssh-listen-addr" optional:"" help:"Address and port to bind for SSH connections (port 0 = random)."`
	DNSAddr   string `default:"${_dns_listen_addr}" group:"Local addresses binding configuration:" name:"dns-listen-addr" yaml:"dns-listen-addr" json:"dns-listen-addr" optional:"" help:"Address and port to bind for DNS connections (port 0 = random)."`
	QuicAddr  string `default:"${_quic_listen_addr}" group:"Local addresses binding configuration:" name:"quic-listen-addr" yaml:"quic-listen-addr" json:"quic-listen-addr" optional:"" help:"Address and port to bind for QUIC connections (port 0 = random)."`

	TLS             bool   `default:"${_tls}" group:"Listeners configuration:" name:"tls" yaml:"tls" json:"tls" negatable:"" help:"Enable TLS support."`
	TLSKey          string `default:"${_tls_key}" group:"Listeners configuration:" name:"tls-key" yaml:"tls-key" json:"tls-key" help:"Path to the TLS private key file."`
	TLSCert         string `default:"${_tls_cert}" group:"Listeners configuration:" name:"tls-cert" yaml:"tls-cert" json:"tls-cert" help:"Path to the TLS certificate file."`
	LetsEncryptMail string `default:"${_letsencrypt_email}" group:"Listeners configuration:" name:"letsencrypt-mail" yaml:"letsencrypt-mail" json:"letsencrypt-mail" help:"Email used when generating Let's Encrypt certificates."`
	Quic            bool   `default:"${_quic}" group:"Listeners configuration:" name:"quic" yaml:"quic" json:"quic" negatable:"" help:"Enable QUIC protocol support."`
	DNS             bool   `default:"${_dns}" group:"Listeners configuration:" name:"dns" yaml:"dns" json:"dns" negatable:"" help:"Enable DNS server for SSH-over-DNS connections."`

	NoDB       bool   `default:"${_no_db}" group:"Database configuration" name:"db" yaml:"db" json:"db" negatable:"" help:"Disable database usage."`
	DbFileName string `default:"${_db_name}" group:"Database configuration" name:"db-file-name" yaml:"db-file-name" json:"db-file-name" help:"Path or filename of the database to use."`

	AllowedIPs  []string `default:"${_allowed_ips}" group:"Access control configuration:" name:"allowed-ips" yaml:"allowed-ips" json:"allowed-ips" help:"List of IP addresses allowed to access the /manage/ endpoint."`
	AccessToken []string `default:"${_access_token}" group:"Access control configuration:" name:"access-token" yaml:"access-token" json:"access-token" help:"Access token required for the /manage/ API endpoint."` //nolint:gosec
	AdminToken  []string `default:"${_admin_token}" group:"Access control configuration:" name:"admin-token" yaml:"admin-token" json:"admin-token" help:"Admin token required for the /admin/ API endpoint."`

	BinariesBasicAuth    string `default:"${_binaries_basicauth}" group:"Agent binaries configuration:" name:"binaries-basic-auth" yaml:"binaries-basic-auth" json:"binaries-basic-auth" help:"HTTP Basic Auth credentials required to access the binaries endpoint."`
	BinariesPathLocation string `default:"${_binaries_path}" group:"Agent binaries configuration:" name:"binaries-path-location" yaml:"binaries-path-location" json:"binaries-path-location" help:"Filesystem path where agent binaries are stored."`

	Verbose        int    `default:"${_verbosity}" group:"Server configuration:" short:"v" name:"verbose" yaml:"verbose" json:"verbose" type:"counter" help:"Increase log verbosity. Repeat for more detail."`
	Quiet          bool   `default:"${_quiet}" group:"Server configuration:" short:"q" name:"quiet" yaml:"quiet" json:"quiet" help:"Suppress all log output."`
	Version        bool   `default:"${_version}" group:"Server configuration:" short:"V" name:"version" yaml:"version" json:"version" help:"Show version information and exit."`
	GenerateConfig bool   `default:"${_generate_config}" group:"Server configuration:" name:"generate-config" yaml:"generate-config" json:"generate-config" help:"Generate a configuration file from the current settings."`
	ConfigFile     string `default:"${_config_file}" group:"Server configuration:" short:"c" name:"config-file" yaml:"config-file" json:"config-file" optional:"" help:"Path to the configuration file to use."`

	Hooks       []func(id string, name string) `hidden:"" json:"-"`
	SingleAgent bool                           `hidden:"" json:"-"`
}

// InitServer initialize the application configuration.
func InitServer() (*kong.Context, *ServerConfig, error) {
	cfgTmp := &ServerConfig{}
	dir, err := os.Getwd()
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
		kong.Description(common.Title(common.Appname+" server") + "\n" + description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(common.AppName())),
		defaultValues,
	}
	_ = kong.Parse(cfgTmp, kongOptions...)
	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar([]string{}), cfgTmp.ConfigFile))
	}
	cfg := &ServerConfig{}
	app := kong.Parse(cfg, kongOptions...)
	srvCfg = cfg

	if cfg.Quiet {
		log.SetLogLevel(-1)
	} else {
		log.SetLogLevel(cfg.Verbose)
	}

	return app, cfg, nil
}

// Get return the global config.
func Get() *ServerConfig {
	return srvCfg
}
func Set(cfg *ServerConfig) {
	srvCfg = cfg
}
func (s *ServerConfig) Copy() ServerConfig {
	//nolint:gosec,g117
	js, err := json.Marshal(*s)
	if err != nil {
		return ServerConfig{}
	}
	sc := ServerConfig{}
	err = json.Unmarshal(js, &sc)
	if err != nil {
		return ServerConfig{}
	}

	return sc
}

// Decrypt returns the encrypted data using the configured private key.
func (s *ServerConfig) Decrypt(data []byte) (string, error) {
	return crypto.AsymDecrypt(s.PrivKey, data)
}

// LocalSSHAddr return the local SSH address.
func (s *ServerConfig) LocalSSHAddr() string {
	return s.SSHDAddr
}

// UpdateSSHAddr return the local SSH address.
func (s *ServerConfig) UpdateSSHAddr(port int) {
	split := strings.Split(s.SSHDAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.SSHDAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// LocalHTTPSAddr return the local HTTPS address.
func (s *ServerConfig) LocalHTTPSAddr() string {
	return s.HTTPSAddr
}

// UpdateHTTPSAddr return the local SSH address.
func (s *ServerConfig) UpdateHTTPSAddr(port int) {
	split := strings.Split(s.HTTPSAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.HTTPSAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// UpdateQUICAddr return the local SSH address.
func (s *ServerConfig) UpdateQUICAddr(port int) {
	split := strings.Split(s.QuicAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.QuicAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// UpdateDNSAddr updates the local DNS address.
func (s *ServerConfig) UpdateDNSAddr(port int) {
	split := strings.Split(s.DNSAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.DNSAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// UpdateHTTPAddr updates the local HTTP address.
func (s *ServerConfig) UpdateHTTPAddr(port int) {
	split := strings.Split(s.HTTPSAddr, ":")
	if len(split) == 2 && split[1] == "0" {
		s.HTTPSAddr = fmt.Sprintf("%s:%d", split[0], port)
	}
}

// LocalHTTPAddr return the local HTTP address.
func (s *ServerConfig) LocalHTTPAddr() string {
	return s.HTTPAddr
}

// Validate perform kong validation to ensure that fields are correct.
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

// IsCustomTLS return whether a TLS certificate is provided.
func (s *ServerConfig) IsCustomTLS() bool {
	return s.TLSCert != "" && s.TLSKey != ""
}

// GetTLSDomains return the list of domains served under TLS.
func (s *ServerConfig) GetTLSDomains() []string {
	return append(s.TLSDomain, s.HTTPDomain...)
}

// GetBinariesBasicAuth return the username and password used to restrict the hosted binaries.
func (s *ServerConfig) GetBinariesBasicAuth() (string, string) {
	split := strings.Split(s.BinariesBasicAuth, ":")
	if len(s.BinariesBasicAuth) < 2 {
		return "", ""
	}

	return split[0], strings.Join(split[1:], ":")
}

// GenerateYAMLConfig return the YAML configuration using the current configuration
// (command line argument and parsed configuration files).
func (s *ServerConfig) GenerateYAMLConfig() (string, error) {
	s.GenerateConfig = false

	return cli.GenerateYAMLWithComments(*s)
}

// CopySanitized return a copy of the server configuration while removing secrets.
func (s *ServerConfig) CopySanitized() ServerConfig {
	ss := s.Copy()
	ss.GenerateConfig = false
	ss.AdminToken = sanitizeSlice(ss.AdminToken)
	ss.AccessToken = sanitizeSlice(ss.AccessToken)
	ss.PrivKey = "[REDACTED]"
	ss.BinariesBasicAuth = "[REDACTED]"

	return ss
}

func sanitizeSlice(slice []string) []string {
	var newSlice []string
	for range slice {
		newSlice = append(newSlice, "[REDACTED]")
	}

	return newSlice
}

// GenerateSafeYAMLConfig return the YAML configuration using the current configuration
// (command line argument and parsed configuration files)
// Without sensitive information.
func (s *ServerConfig) GenerateSafeYAMLConfig() (string, error) {
	return cli.GenerateYAMLWithComments(s.CopySanitized())
}

func (s *ServerConfig) ReloadDomains() error {
	_, cfg, err := InitServer()
	if err != nil {
		return err
	}

	s.TLSDomain = cfg.TLSDomain
	s.HTTPDomain = cfg.HTTPDomain

	return nil
}
