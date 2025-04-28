package log

import (
	Sources "Goauld"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"

	"github.com/rs/zerolog"
)

const CustomLevelType zerolog.Level = zerolog.Level(10) // Pick an unused int > 6
const Custom = "OK "

var (
	zerologger    *zerolog.Logger
	once          sync.Once
	gormlogger    logger.Interface
	negronilogger *NegroniLogger
	// logLevel     = zerolog.InfoLevel
	gormLogLevel = logger.Warn
)

func initLoggers() {
	root := Sources.GetRoot()
	writer := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string {
		if l == CustomLevelType {
			return Custom
		}
		return l.String()
	}
	// fileWriter := zerolog.Writer
	// writers := zerolog.MultiLevelWriter(writer)
	writer.FormatLevel = func(i interface{}) string {
		if level, ok := i.(string); ok {
			if level == Custom {
				return fmt.Sprintf("\x1b[35m%s\x1b[0m", Custom) // Purple (or whatever ANSI color you want)
			}
			// Optionally color built-in levels too
			switch strings.ToUpper(level) {
			case "TRACE":
				return "\x1b[1;34mTRC\x1b[0m"
			case "DEBUG":
				return "\x1b[1;37mDBG\x1b[0m"
			case "INFO":
				return "\x1b[1;32mINF\x1b[0m"
			case "WARN":
				return "\x1b[1;33mWRN\x1b[0m"
			case "ERROR":
				return "\x1b[1;31mERR\x1b[0m"
			case "FATAL":
				return "\x1b[31;1mFAT\x1b[0m"
			case "PANIC":
				return "\x1b[1;41mPNC\x1b[0m"
			default:
				return level // unstyled
			}
		}
		return "\x1b[1;37m???\x1b[0m"
	}
	writer.FormatMessage = func(i interface{}) string {

		return fmt.Sprintf("%v", i)
	}

	ml := zerolog.MultiLevelWriter(writer)
	l := zerolog.New(ml).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return fmt.Sprintf("%s:%d", strings.TrimPrefix(file, root+"/"), line)
	}
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

	nl := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Str("Src", "Negroni").Logger()
	negronilogger = &NegroniLogger{
		logger: nl,
	}
	log.SetOutput(zerologger)
	log.SetFlags(0)
}

func UpdateLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
	//if level.String() > 3 {
	//	gormLogLevel = verbosityToGormLogLevel(verbosity)
	//} else {
	//	gormLogLevel = -1
	//}
	//newLogger := zerologger.Level(level)
	//zerologger = &newLogger
	//gormlogger = gormlogger.LogMode(zerologLevelToGormLogLevel(level))
}

func SetLogLevel(verbosity int) {
	// logLevel = VerbosityToLogLevel(verbosity)
	zerolog.SetGlobalLevel(VerbosityToLogLevel(verbosity))

	if verbosity > 3 {
		gormLogLevel = verbosityToGormLogLevel(verbosity)
	} else {
		gormLogLevel = -1
	}
}

func zerologLevelToGormLogLevel(level zerolog.Level) logger.LogLevel {
	switch level {
	case zerolog.ErrorLevel:
		return logger.Error
	case zerolog.WarnLevel:
		return logger.Warn
	case zerolog.InfoLevel:
		return logger.Info
	default:
		return logger.Info
	}
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

func VerbosityToLogLevel(verbosity int) zerolog.Level {
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

type NegroniLogger struct {
	logger zerolog.Logger
}

func GetNegroniLogger() *NegroniLogger {
	once.Do(initLoggers)
	return negronilogger
}

func (n *NegroniLogger) Println(v ...interface{}) {
	n.logger.Trace().Msg(fmt.Sprint(v...))
}

func (n *NegroniLogger) Printf(format string, v ...interface{}) {
	n.logger.Trace().Msg(fmt.Sprint(format, v))
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

func OK() *zerolog.Event {
	// Log using the custom level
	return Get().WithLevel(CustomLevelType)
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
