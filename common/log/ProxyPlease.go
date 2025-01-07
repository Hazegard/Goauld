package log

func ProxyPleaseLog() func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) {
		Trace().Str("From", "ProxyPlease").Msgf(format, a...)
	}
}
