package log

import (
	"gorm.io/gorm/logger"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	zerologger   *zerolog.Logger
	once         sync.Once
	gormlogger   logger.Interface
	logLevel     = zerolog.InfoLevel
	gormLogLevel = logger.Warn
)

func initLoggers() {
	l := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(logLevel).With().Timestamp().Caller().Logger()
	zerologger = &l

	gormlogger = NewGormLogger().
		WithInfo(func() Event {
			return &GormLoggerEvent{Event: zerologger.Info()}
		}).
		WithError(func() Event {
			return &GormLoggerEvent{Event: zerologger.Error()}
		}).
		WithWarn(func() Event {
			return &GormLoggerEvent{Event: zerologger.Warn()}
		}).LogMode(gormLogLevel)

}

func SetLogLevel(verbosity int) {
	logLevel = verbosityToLogLevel(verbosity)
	gormLogLevel = verbosityToGormLogLevel(verbosity)
}

func verbosityToGormLogLevel(verbosity int) logger.LogLevel {
	switch verbosity {
	case 0:
		return logger.Error
	case 1:
		return logger.Warn
	case 2:
		return logger.Info
	default:
		return logger.Info
	}
}

func verbosityToLogLevel(verbosity int) zerolog.Level {
	switch verbosity {
	case 0:
		return zerolog.InfoLevel
	case 1:
		return zerolog.DebugLevel
	case 2:
		return zerolog.TraceLevel
	default:
		return zerolog.TraceLevel
	}
}

func Get() *zerolog.Logger {

	once.Do(initLoggers)

	return zerologger
}

func GetGormLogger() logger.Interface {
	once.Do(initLoggers)
	return gormlogger
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
