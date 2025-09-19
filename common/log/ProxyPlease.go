package log

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var initPPLog sync.Once
var pplogger *zerolog.Logger

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
