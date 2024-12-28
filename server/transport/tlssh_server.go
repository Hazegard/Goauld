package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/store"
	"context"
	"crypto/tls"
	"github.com/caddyserver/certmagic"
	"io"
	"net"
)

type TLSSHServer struct {
	store *store.AgentStore
}

func Test() {

	cfg := certmagic.NewDefault()
	certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	_, err := cfg.CacheUnmanagedCertificatePEMFile(context.Background(), "./tls_cert.pem", "./key.pem", []string{"a.hazegard.fr", "b.hazegard.fr"})
	if err != nil {
		log.Error().Err(err).Msg("Failed to load TLS certificate")

	}
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certificate certmagic.Certificate) (*certmagic.Config, error) {
			return cfg, nil
		},
		OCSPCheckInterval:  0,
		RenewCheckInterval: 0,
		Capacity:           0,
		Logger:             nil,
	})
	cmg := certmagic.New(cache, *cfg)
	tlsConfig := cmg.TLSConfig()
	certmagic.Default.TLSConfig()
	ln, err := tls.Listen("tcp", ":443", tlsConfig)
	//ln, err := certmagic.Listen([]string{"b.hazegard.fr"})
	log.Trace().Msgf("listen %s", ln.Addr())
	if err != nil {
		log.Error().Err(err).Msg("error listening TLS")
	}
	for {

		conn, err := ln.Accept()
		if err != nil {
			log.Error().Err(err).Msg("error accepting connection")
			continue
		}
		go func() {

			sshConn, err := net.Dial("tcp", config.Get().LocalSShServer())
			if err != nil {
				log.Error().Err(err).Msg("error connecting to server")
				return
			}
			go io.Copy(conn, sshConn)
			io.Copy(sshConn, conn)

		}()
	}

}
