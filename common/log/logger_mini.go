//go:build mini
// +build mini

package log

import (
	"fmt"
	"sync"
	"time"
)

var (
	once        sync.Once
	printLogger Logger
	mutex       sync.Mutex
)

func initLogger() {
	printLogger = Logger{
		data: make(map[string]interface{}),
	}
	mutex = sync.Mutex{}
}

type Logger struct {
	tag  string
	err  error
	data map[string]interface{}
}

func (l *Logger) print(str string) {
	mutex.Lock()
	defer mutex.Unlock()
	data := ""
	for k, v := range l.data {
		data += fmt.Sprintf("%v: %v, ", k, v)
	}
	if data != "" {
		fmt.Println(l.tag + ": " + str + " (" + data + ")")
	} else {
		fmt.Println(l.tag + ": " + str)
	}

}

func (l *Logger) Str(name string, value string) *Logger {
	l.data[name] = value
	return l
}

func (l *Logger) Printf(format string, v ...any) {
	l.print(fmt.Sprintf(format, v...))
}

func (l *Logger) Logger() Logger {
	return *l
}

func (l Logger) With() *Logger {
	return &l
}

func (l *Logger) Bool(name string, value bool) *Logger {
	l.data[name] = value
	return l
}

func (l *Logger) Time(name string, value time.Time) *Logger {
	l.data[name] = value
	return l
}

func (l *Logger) Int(name string, value int) *Logger {
	l.data[name] = value
	return l
}

func (l *Logger) Err(err error) *Logger {
	l.err = err
	return l
}

func (l *Logger) Msgf(format string, v ...interface{}) {
	l.print(fmt.Sprintf(format, v...))
}

func (l *Logger) Msg(str string) {
	l.print(str)
}

// Get return the global logger.
func Get() Logger {
	once.Do(initLogger)

	return printLogger
}

// Trace zerolog event.
func Trace() *Logger {
	return &Logger{tag: "TRC", data: make(map[string]interface{})}
}

// Debug zerolog event.
func Debug() *Logger {
	return &Logger{tag: "DBG", data: make(map[string]interface{})}
}

// Info zerolog event.
func Info() *Logger {
	return &Logger{tag: "INF", data: make(map[string]interface{})}
}

// Run zerolog event.
func Run() *Logger {
	return &Logger{tag: "RUN", data: make(map[string]interface{})}
}

// Kill zerolog event.
func Kill() *Logger {
	return &Logger{tag: "KIL", data: make(map[string]interface{})}
}

// Reset zerolog event.
func Reset() *Logger {
	return &Logger{tag: "RST", data: make(map[string]interface{})}
}

// Warn zerolog event.
func Warn() *Logger {
	return &Logger{tag: "WRN", data: make(map[string]interface{})}
}

// Error zerolog event.
func Error() *Logger {
	return &Logger{tag: "ERR", data: make(map[string]interface{})}
}

func SetLogLevel(level int) {}

// ProxyPleaseLog logs the ProxyPlease infos.
func ProxyPleaseLog() func(format string, a ...any) {
	return func(format string, a ...any) {
		Trace().Str("From", "ProxyPlease").Msgf(format, a...)
	}
}
