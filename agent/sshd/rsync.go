package sshd

import (
	"Goauld/common/log"
	"Goauld/common/utils"
	stdlog "log"
	"runtime"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/gokrazy/rsync/rsyncd"
)

func HandleRsync(s ssh.Session) {
	log.Debug().Str("Username", s.User()).Str("RemoteAddr", s.RemoteAddr().String()).Str("Command", s.RawCommand()).Msgf("Handling Rsync command")
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

	var newArgs []string

	if runtime.GOOS == "windows" {
		var paths []string
		isPath := false
		for _, arg := range s.Command()[1:] {
			if isPath {
				paths = append(paths, arg)

				continue
			}
			if arg == "." {
				isPath = true

				continue
			}
			newArgs = append(newArgs, arg)
		}

		var drives []string
		var relPaths []string
		for _, p := range paths {
			split := strings.Split(p, ":")
			if len(split) != 2 {
				log.Warn().Str("path", p).Str("command", "rsync").Msg("Invalid path")
				_, _ = s.Write([]byte("Invalid path: " + p))

				continue
			}
			drives = append(drives, split[0])
			relPath := strings.TrimPrefix(split[1], "/")
			relPath = strings.TrimPrefix(relPath, "\\")
			relPaths = append(relPaths, relPath)
		}
		drives = utils.Unique(drives)

		if len(drives) != 1 {
			log.Error().Str("Drives", strings.Join(drives, "/")).Str("path", strings.Join(paths, ":")).Msg("Rsync should only contain on drive")
			_, _ = s.Write([]byte("Rsync should only contain on drive: " + strings.Join(paths, ":")))

			return
		}

		newArgs = append(newArgs, relPaths...)
		mods.Path = drives[0] + ":"
	} else {
		isDotFound := false
		for _, arg := range s.Command()[1:] {
			if arg == "." && !isDotFound {
				isDotFound = true

				continue
			}
			newArgs = append(newArgs, arg)
		}
	}

	conn := rsyncd.NewConnection(s, s, s.User())
	err = srv.HandleConnArgs(s.Context(), conn, &mods, newArgs)
	if err != nil {
		log.Error().Err(err).Str("Command", s.RawCommand()).Msg("sshd.HandleConn")
	}
}
