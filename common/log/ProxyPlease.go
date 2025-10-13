// Package log holds the loggers
package log

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var initPPLog sync.Once
var pplogger *zerolog.Logger

// ProxyPleaseLog logs the ProxyPlease infos.
func ProxyPleaseLog() func(format string, a ...interface{}) {
	initPPLog.Do(func() {
		ppl := Get().Sample(
			&zerolog.BurstSampler{
				Burst:       3,
				Period:      10 * time.Second,
				NextSampler: nil,
			})
		pplogger = &ppl
	})

	return func(format string, a ...interface{}) {
		pplogger.Trace().Str("From", "ProxyPlease").Msgf(format, a...)
	}
}
