package log15adapter

import (
	"os"

	"github.com/grafana/devtools/pkg/streams/log"
	"github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
)

func New(logger log15.Logger) *log.LogHandler {
	return &log.LogHandler{
		DebugHandler: func(msg string, ctx ...interface{}) {
			logger.Debug(msg, ctx...)
		},
		InfoHandler: func(msg string, ctx ...interface{}) {
			logger.Info(msg, ctx...)
		},
		WarnHandler: func(msg string, ctx ...interface{}) {
			logger.Warn(msg, ctx...)
		},
		ErrorHandler: func(msg string, ctx ...interface{}) {
			logger.Error(msg, ctx...)
		},
		FatalHandler: func(msg string, ctx ...interface{}) {
			logger.Crit(msg, ctx...)
		},
	}
}

func GetConsoleFormat() log15.Format {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return log15.TerminalFormat()
	}

	return log15.LogfmtFormat()
}
