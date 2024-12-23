package proxy

func NewTlsConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionSSL30,
		NextProtos:         []string{"http/1.1", "h2", "h2c"},
	}
}
