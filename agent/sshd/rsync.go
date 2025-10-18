package sshd

import (
	"Goauld/common/log"
	stdlog "log"

	"github.com/charmbracelet/ssh"
	"github.com/gokrazy/rsync/rsyncd"
)

func HandleRsync(s ssh.Session) {
	log.Trace().Str("Username", s.User()).Str("RemoteAddr", s.RemoteAddr().String()).Str("Command", s.RawCommand()).Msgf("Handling Rsync command")
	l := log.Get().With().Str("Module", "rsync").Logger()
	ll := stdlog.New(log.CustomSlog{L: &l}, "", 0)

	srv, err := rsyncd.NewServer([]rsyncd.Module{}, rsyncd.WithLogger(ll))
	if err != nil {
		_, _ = s.Stderr().Write([]byte(err.Error()))
		_ = s.Exit(1)
		log.Warn().Err(err).Str("Username", s.User()).Str("RemoteAddr", s.RemoteAddr().String()).Str("Command", s.RawCommand()).Msg("Failed to create a rsync server")

		return
	}

	mods := rsyncd.Module{
		Name:     "root",
		Path:     "/",
		Writable: true,
	}
	conn := rsyncd.NewConnection(s, s, s.User())
	err = srv.HandleConnArgs(s.Context(), conn, &mods, s.Command()[1:])
	if err != nil {
		log.Error().Err(err).Str("Command", s.RawCommand()).Msg("sshd.HandleConn")
	}
}
