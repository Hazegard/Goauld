//go:build !mini

package log

import (
	Sources "Goauld"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"

	"github.com/rs/zerolog"
)

// CustomLevelTypeRun hgolds the RUN log level.
const CustomLevelTypeRun = zerolog.Level(10) // Pick an unused int > 6
// CustomRun prefix used when printing the log.
const CustomRun = "RUN"

// CustomLevelTypeKil holds the KILL log level.
const CustomLevelTypeKil = zerolog.Level(11) // Pick an unused int > 6
// CustomKil prefix used when printing the log.
const CustomKil = "KIL"

// CustomLevelTypeRst holds the RESET log level.
const CustomLevelTypeRst = zerolog.Level(12) // Pick an unused int > 6
// CustomRst prefix used when printing the log.
const CustomRst = "RST"

var (
	zerologger    *zerolog.Logger
	once          sync.Once
	gormlogger    logger.Interface
	negronilogger *NegroniLogger
	// logLevel     = zerolog.InfoLevel.
	gormLogLevel = logger.Warn
)

// CustomSlog holds the zerolog logger.
type CustomSlog struct {
	L *zerolog.Logger
}

// Write override the zerolog write to alter the default stacktrace.
func (cs CustomSlog) Write(p []byte) (int, error) {
	n := len(p)
	if n > 0 && p[n-1] == '\n' {
		// Trim CR added by stdlog.
		p = p[0 : n-1]
	}
	m := 10
	i := 0
	for {
		_, file, _, _ := runtime.Caller(i)
		if strings.HasPrefix(file, "../") {
			i++
		} else {
			break
		}
		if i == m {
			break
		}
	}
	cs.L.Trace().CallerSkipFrame(i).Msg(string(p))

	return n, nil
}

// colorize returns the string s wrapped in ANSI code c, unless disabled is true or c is 0.
//
//nolint:unparam
func colorize(s any, c int, disabled bool) string {
	e := os.Getenv("NO_COLOR")
	if e != "" || c == 0 {
		disabled = true
	}

	if disabled {
		return fmt.Sprintf("%s", s)
	}

	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func trimSubstr(s string, substr string) string {
	var t string
	for {
		t = strings.TrimPrefix(s, substr)
		t = strings.TrimSuffix(t, substr)
		if t == s { // exit if nothing was trimmed from s
			break
		}
		s = t // update to last result
	}

	return t
}

func initLoggers() {
	root := Sources.GetRoot()
	writer := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string {
		if l == CustomLevelTypeRun {
			return CustomRun
		}
		if l == CustomLevelTypeKil {
			return CustomKil
		}
		if l == CustomLevelTypeRst {
			return CustomRst
		}

		return l.String()
	}
	writer.FormatLevel = func(i any) string {
		if i == nil {
			return colorize("TRC", 34, false)
		}

		if level, ok := i.(string); ok {
			if level == CustomRun || level == CustomKil || level == CustomRst {
				return colorize(level, 35, false)
			}
			// Optionally color built-in levels too
			switch strings.ToUpper(level) {
			case "TRACE":
				return colorize("TRC", 34, false) // "\x1b[1;34mTRC\x1b[0m"
			case "DEBUG":
				return colorize("DBG", 37, false) // "\x1b[1;37mDBG\x1b[0m"
			case "INFO":
				return colorize("INF", 32, false) // "\x1b[1;32mINF\x1b[0m"
			case "WARN":
				return colorize("WRN", 33, false) // "\x1b[1;33mWRN\x1b[0m"
			case "ERROR":
				return colorize("ERR", 31, false) // "\x1b[1;31mERR\x1b[0m"
			case "FATAL":
				return colorize("FAT", 31, false) // "\x1b[31;1mFAT\x1b[0m"
			case "PANIC":
				return colorize("PNC", 41, false) // "\x1b[1;41mPNC\x1b[0m"
			default:
				return level // unstyled
			}
		}

		return colorize(fmt.Sprintf("%s", i), 37, false) // "\x1b[1;37m???\x1b[0m"
	}
	writer.FormatMessage = func(i any) string {
		return fmt.Sprintf("%v", i)
	}

	writer.FormatFieldName = func(i any) string {
		//nolint:gocritic
		switch v := i.(type) {
		case string:
			if strings.ToLower(v) == "reason" {
				return colorize(v+"=", 35, false)
			}
		}

		return colorize(fmt.Sprintf("%s=", i), 36, false)
	}

	ml := zerolog.MultiLevelWriter(writer)
	l := zerolog.New(ml).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		file = trimSubstr(file, "../")
		if strings.HasSuffix(file, root+"/") {
			return fmt.Sprintf("%s:%d", strings.TrimPrefix(file, root+"/"), line)
		}
		regex := regexp.MustCompile(`([a-zA-Z0-9\-_]+@[a-zA-Z0-9.]+).*`)
		m := regex.FindString(file)
		if m == "" {
			return fmt.Sprintf("%s:%d", file, line)
		}

		return fmt.Sprintf("%s:%d", m, line)
	}
	zerologger = &l

	gormlogger = NewGormLogger().
		WithInfo(func() Event {
			//nolint:zerologlint
			return &GormLoggerEvent{Event: zerologger.Trace().Str("Src", "Gorm")}
		}).
		WithError(func() Event {
			//nolint:zerologlint
			return &GormLoggerEvent{Event: zerologger.Error().Str("Src", "Gorm")}
		}).
		WithWarn(func() Event {
			//nolint:zerologlint
			return &GormLoggerEvent{Event: zerologger.Warn().Str("Src", "Gorm")}
		}).LogMode(gormLogLevel)

	nl := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Str("Src", "Negroni").Logger()
	negronilogger = &NegroniLogger{
		logger: nl,
	}
	log.SetOutput(CustomSlog{zerologger})
	log.SetFlags(0)
}

// UpdateLogLevel updates the global log level.
func UpdateLogLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// SetLogLevel set the default log level.
func SetLogLevel(verbosity int) {
	zerolog.SetGlobalLevel(VerbosityToLogLevel(verbosity))

	if verbosity > 3 {
		gormLogLevel = verbosityToGormLogLevel(verbosity)
	} else {
		gormLogLevel = -1
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

// VerbosityToLogLevel returns the zerolog.Level verbosity given an integer representing the verbosity.
func VerbosityToLogLevel(verbosity int) zerolog.Level {
	switch verbosity {
	case -1:
		return zerolog.Disabled
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

// Get return the global logger.
func Get() *zerolog.Logger {
	once.Do(initLoggers)

	return zerologger
}

// NegroniLogger holds the logger used by negroni.
type NegroniLogger struct {
	logger zerolog.Logger
}

// GetNegroniLogger return the Negroni logger.
func GetNegroniLogger() *NegroniLogger {
	once.Do(initLoggers)

	return negronilogger
}

// Println negroni logger.
func (n *NegroniLogger) Println(v ...any) {
	n.logger.Trace().Msg(fmt.Sprint(v...))
}

// Printf negroni logger.
func (n *NegroniLogger) Printf(format string, v ...any) {
	n.logger.Trace().Msg(fmt.Sprint(format, v))
}

// GetGormLogger return the GORM logger.
func GetGormLogger() logger.Interface {
	once.Do(initLoggers)

	return gormlogger
}

// Trace zerolog event.
func Trace() *zerolog.Event {
	//nolint:zerologlint
	return Get().Trace()
}

// Debug zerolog event.
func Debug() *zerolog.Event {
	//nolint:zerologlint
	return Get().Debug()
}

// Info zerolog event.
func Info() *zerolog.Event {
	//nolint:zerologlint
	return Get().Info()
}

// Run zerolog event.
func Run() *zerolog.Event {
	// Log using the custom level
	//nolint:zerologlint
	return Get().WithLevel(CustomLevelTypeRun)
}

// Kill zerolog event.
func Kill() *zerolog.Event {
	// Log using the custom level
	//nolint:zerologlint
	return Get().WithLevel(CustomLevelTypeKil)
}

// Reset zerolog event.
func Reset() *zerolog.Event {
	// Log using the custom level
	//nolint:zerologlint
	return Get().WithLevel(CustomLevelTypeRst)
}

// Warn zerolog event.
func Warn() *zerolog.Event {
	//nolint:zerologlint
	return Get().Warn()
}

// Error zerolog event.
func Error() *zerolog.Event {
	//nolint:zerologlint
	return Get().Error()
}

// Print zerolog event.
func Print(v ...any) {
	Get().Print(v...)
}

// Println zerolog event.
func Println(v ...any) {
	Get().Println(v...)
}

// Printf zerolog event.
func Printf(format string, v ...any) {
	Get().Printf(format, v...)
}

// Log zerolog event.
func Log() *zerolog.Event {
	//nolint:zerologlint
	return Get().Log()
}
