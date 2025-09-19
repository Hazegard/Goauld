package sshd

import (
	"io"

	"Goauld/common/log"

	"github.com/charmbracelet/ssh"

	"github.com/pkg/sftp"
)

// SftpHandler handle sftp connections
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
		err := server.Close()
		if err != nil {
			log.Debug().Err(err).Msg("sftp server error")
		}
		log.Debug().Msg("sftp agent exited session.")
	} else if err != nil {
		log.Debug().Err(err).Msg("sftp server error")
	}
}
