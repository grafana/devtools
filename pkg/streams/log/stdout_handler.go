package log

import (
	corelog "log"
)

func NewStdOutHandler(useLogLevel LogLevel) *LogHandler {
	return &LogHandler{
		CatchAllHandler: func(logLevel LogLevel, msg string, ctx ...interface{}) {
			if logLevel < useLogLevel {
				return
			}

			level := ""
			switch logLevel {
			case LogLevelDebug:
				level = "DBUG"
				break
			case LogLevelInfo:
				level = "INFO"
				break
			case LogLevelWarning:
				level = "WARN"
				break
			case LogLevelError:
				level = "EROR"
				break
			case LogLevelFatal:
				level = "FTAL"
				break
			}

			ctx = append([]interface{}{msg, "level", level}, ctx...)

			corelog.Println(ctx...)
		},
	}
}
