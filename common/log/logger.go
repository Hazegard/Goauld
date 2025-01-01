package log

import (
	"gorm.io/gorm/logger"
	"log"
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
			return &GormLoggerEvent{Event: zerologger.Trace().Str("Src", "Gorm")}
		}).
		WithError(func() Event {
			return &GormLoggerEvent{Event: zerologger.Error().Str("Src", "Gorm")}
		}).
		WithWarn(func() Event {
			return &GormLoggerEvent{Event: zerologger.Warn().Str("Src", "Gorm")}
		}).LogMode(gormLogLevel)

	log.SetOutput(zerologger)
	log.SetFlags(0)
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

func Trace() *zerolog.Event {
	return Get().Trace()
}

func Debug() *zerolog.Event {
	return Get().Debug()
}

func Info() *zerolog.Event {
	return Get().Info()
}

func Warn() *zerolog.Event {
	return Get().Warn()
}

func Error() *zerolog.Event {
	return Get().Error()
}

func Print(v ...interface{}) {
	Get().Print(v...)
}

func Println(v ...interface{}) {
	Get().Println(v...)
}

func Printf(format string, v ...interface{}) {
	Get().Printf(format, v...)
}

func Log() *zerolog.Event {
	return Get().Log()
}
