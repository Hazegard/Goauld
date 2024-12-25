package log

import (
	"gorm.io/gorm/logger"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	zerologger *zerolog.Logger
	once       sync.Once
	gormlogger logger.Interface
)

func initLoggers() {
	l := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
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
		}).LogMode(logger.Warn)

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
