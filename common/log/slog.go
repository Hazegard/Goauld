package log

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm/logger"
)

const (
	traceErrMsg  = "%s %s\n[%.3fms] [rows:%v] %s"
	traceWarnMsg = "%s %s\n[%.3fms] [rows:%v] %s"
	traceInfoMsg = "%s\n[%.3fms] [rows:%v] %s"
)

// Event represents a proxy object between GORM Logger and zerolog.
type Event interface {
	Str(key, value string) Event
	Msgf(format string, v ...any)
}

// GormLoggerEvent hold the logger used by gorm.
type GormLoggerEvent struct {
	*zerolog.Event
}

// Str wraps the zerolog in the gorm logger.
func (e *GormLoggerEvent) Str(key, value string) Event {
	e.Event = e.Event.Str(key, value)

	return e
}

// Msgf wraps the zerolog in the gorm logger.
func (e *GormLoggerEvent) Msgf(format string, v ...any) {
	e.Event.Msgf(format, v...)
}

func newGormLoggerEventInfo() Event {
	return &GormLoggerEvent{
		//nolint:zerologlint
		Event: log.Trace(),
	}
}

func newGormLoggerEventWarn() Event {
	return &GormLoggerEvent{
		//nolint:zerologlint
		Event: log.Warn(),
	}
}

func newGormLoggerEventError() Event {
	return &GormLoggerEvent{
		//nolint:zerologlint
		Event: log.Error(),
	}
}

// GormLogger represents an logging object for handling GORM logs with zerolog.
type GormLogger struct {
	logLevel                logger.LogLevel
	ignoreRecordNotFoundErr bool
	slowThreshold           time.Duration
	loggers                 map[logger.LogLevel]func() Event

	AdditionalData map[string]string
}

// NewGormLogger creates a new GORM zerolog zerologger.
func NewGormLogger() *GormLogger {
	return &GormLogger{
		logLevel:      logger.Info,
		slowThreshold: time.Millisecond * 200,
		loggers: map[logger.LogLevel]func() Event{
			logger.Info:  newGormLoggerEventInfo,
			logger.Warn:  newGormLoggerEventWarn,
			logger.Error: newGormLoggerEventError,
		},
	}
}

// WithInfo sets a zerologger builder for info level logging.
func (l *GormLogger) WithInfo(info func() Event) *GormLogger {
	l.loggers[logger.Info] = info

	return l
}

// WithWarn sets a zerologger builder for warn level logging.
func (l *GormLogger) WithWarn(warn func() Event) *GormLogger {
	l.loggers[logger.Warn] = warn

	return l
}

// WithError sets a zerologger builder for error level logging.
func (l *GormLogger) WithError(err func() Event) *GormLogger {
	l.loggers[logger.Error] = err

	return l
}

// IgnoreRecordNotFoundError sets a flag for ignoring ErrRecordNotFound error.
func (l *GormLogger) IgnoreRecordNotFoundError(b bool) {
	l.ignoreRecordNotFoundErr = b
}

// LogMode sets a log level value.
func (l *GormLogger) LogMode(logLevel logger.LogLevel) logger.Interface {
	l.logLevel = logLevel

	return l
}

// SlowThreshold sets a slow threshold level value.
func (l *GormLogger) SlowThreshold(slowThreshold time.Duration) {
	l.slowThreshold = slowThreshold
}

func (l *GormLogger) log(logLevel logger.LogLevel, msg string, data ...any) {
	if l.logLevel >= logLevel {
		if f, ok := l.loggers[logLevel]; ok {
			event := f()
			for k, v := range l.AdditionalData {
				event = event.Str(k, v)
			}

			event.Msgf(msg, data...)
		}
	}
}

// Info starts a new message with info level.
func (l *GormLogger) Info(_ context.Context, msg string, data ...any) {
	l.log(logger.Info, msg, data...)
}

// Warn starts a new message with WARN level.
func (l *GormLogger) Warn(_ context.Context, msg string, data ...any) {
	l.log(logger.Warn, msg, data...)
}

// Error starts a new message with error level.
func (l *GormLogger) Error(_ context.Context, msg string, data ...any) {
	l.log(logger.Error, msg, data...)
}

// Trace starts a new message with trace level.
func (l *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	var rowsAffected any = rows
	if rows == -1 {
		rowsAffected = "-"
	}

	switch {
	case err != nil && (!errors.Is(err, logger.ErrRecordNotFound) || !l.ignoreRecordNotFoundErr):
		l.log(logger.Error, traceErrMsg, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rowsAffected, sql)
	case elapsed > l.slowThreshold && l.slowThreshold != 0:
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.slowThreshold)
		l.log(logger.Warn, traceWarnMsg, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rowsAffected, sql)
	}

	l.log(logger.Info, traceInfoMsg, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rowsAffected, sql)
}

var gormSourceDir string

// fileWithLineNum return the file name and line number of the current file.
func fileWithLineNum() string {
	// the second caller usually from gorm internal, so set i start from 2
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!strings.HasPrefix(file, gormSourceDir) || strings.HasSuffix(file, "_test.go")) {
			return file + ":" + strconv.FormatInt(int64(line), 10)
		}
	}

	return ""
}

//nolint:gochecknoinits
func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	gormSourceDir = regexp.MustCompile(`gorm.utils.utils\.go`).ReplaceAllString(file, "")
}
