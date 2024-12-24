package log

import (
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	logger *zerolog.Logger
	once   sync.Once
)

func Get() *zerolog.Logger {

	once.Do(func() {
		l := zerolog.New(
			zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
		).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
		logger = &l
	})

	return logger
}

var (
	Trace   = Get().Trace
	Debug   = Get().Debug
	Info    = Get().Info
	Warn    = Get().Warn
	Error   = Get().Error
	Print   = Get().Print
	Println = Get().Println
	Printf  = Get().Printf
	Log     = Get().Log
)
