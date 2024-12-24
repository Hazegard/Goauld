package sshd

import (
	"Goauld/common/log"
	"github.com/gliderlabs/ssh"

	"github.com/pkg/sftp"
	"io"
)

func SftpHandler(sess ssh.Session) {
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		sess,
		serverOptions...,
	)
	if err != nil {
		log.Debug().Err(err).Msg("sftp server error")
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
		log.Debug().Msg("sftp agent exited session.")
	} else if err != nil {
		log.Debug().Err(err).Msg("sftp server error")
	}
}
