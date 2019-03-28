package log

import (
	"os"
)

type Logger interface {
	New(ctx ...interface{}) Logger
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Fatal(msg string, ctx ...interface{})
}

type LogHandlerFunc func(msg string, ctx ...interface{})
type CatchAllLogHandlerFunc func(logLevel LogLevel, msg string, ctx ...interface{})

type LogHandler struct {
	DebugHandler    LogHandlerFunc
	InfoHandler     LogHandlerFunc
	WarnHandler     LogHandlerFunc
	ErrorHandler    LogHandlerFunc
	FatalHandler    LogHandlerFunc
	CatchAllHandler CatchAllLogHandlerFunc
}

type LogLevel int

const (
	LogLevelDebug LogLevel = 1 << iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

func (ll LogLevel) String() string {
	names := map[int]string{
		int(LogLevelDebug):   "debug",
		int(LogLevelInfo):    "info",
		int(LogLevelWarning): "warn",
		int(LogLevelError):   "error",
		int(LogLevelFatal):   "fatal",
	}
	return names[int(ll)]
}

type RootLogger struct {
	Logger
	context          []interface{}
	debugHandlers    []LogHandlerFunc
	infoHandlers     []LogHandlerFunc
	warnHandlers     []LogHandlerFunc
	errorHandlers    []LogHandlerFunc
	fatalHandlers    []LogHandlerFunc
	catchAllHandlers []CatchAllLogHandlerFunc
}

func New() *RootLogger {
	logger := NewWithContext(nil)
	return logger
}

func NewWithContext(ctx []interface{}) *RootLogger {
	return &RootLogger{
		context: ctx,
	}
}

func (l *RootLogger) AddHandler(handler *LogHandler) {
	if handler == nil {
		return
	}

	if handler.DebugHandler != nil {
		l.debugHandlers = append(l.debugHandlers, handler.DebugHandler)
	}

	if handler.InfoHandler != nil {
		l.infoHandlers = append(l.infoHandlers, handler.InfoHandler)
	}

	if handler.WarnHandler != nil {
		l.warnHandlers = append(l.warnHandlers, handler.WarnHandler)
	}

	if handler.ErrorHandler != nil {
		l.errorHandlers = append(l.errorHandlers, handler.ErrorHandler)
	}

	if handler.FatalHandler != nil {
		l.fatalHandlers = append(l.fatalHandlers, handler.FatalHandler)
	}

	if handler.CatchAllHandler != nil {
		l.catchAllHandlers = append(l.catchAllHandlers, handler.CatchAllHandler)
	}
}

func (l *RootLogger) New(ctx ...interface{}) Logger {
	return newSubLoggerWithContext(l, ctx...)
}

func (l *RootLogger) Debug(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.debugHandlers {
		handle(msg, resultCtx...)
	}
	l.handleCatchAll(LogLevelDebug, msg, ctx...)
}

func (l *RootLogger) Info(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.infoHandlers {
		handle(msg, resultCtx...)
	}
	l.handleCatchAll(LogLevelInfo, msg, ctx...)
}

func (l *RootLogger) Warn(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.warnHandlers {
		handle(msg, resultCtx...)
	}
	l.handleCatchAll(LogLevelWarning, msg, ctx...)
}

func (l *RootLogger) Error(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.errorHandlers {
		handle(msg, resultCtx...)
	}
	l.handleCatchAll(LogLevelError, msg, ctx...)
}

func (l *RootLogger) Fatal(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.fatalHandlers {
		handle(msg, resultCtx...)
	}
	l.handleCatchAll(LogLevelFatal, msg, ctx...)
	os.Exit(1)
}

func (l *RootLogger) handleCatchAll(logLevel LogLevel, msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	for _, handle := range l.catchAllHandlers {
		handle(logLevel, msg, resultCtx...)
	}
}

func (l *RootLogger) newContext(ctx []interface{}) []interface{} {
	resultCtx := []interface{}{}
	resultCtx = append(resultCtx, l.context...)
	resultCtx = append(resultCtx, ctx...)

	return resultCtx
}

type subLogger struct {
	Logger
	root    *RootLogger
	context []interface{}
}

func newSubLoggerWithContext(root *RootLogger, ctx ...interface{}) Logger {
	return &subLogger{
		root:    root,
		context: ctx,
	}
}

func (l *subLogger) New(ctx ...interface{}) Logger {
	return l.root.New(ctx...)
}

func (l *subLogger) Debug(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	l.root.Debug(msg, resultCtx...)
}

func (l *subLogger) Info(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	l.root.Info(msg, resultCtx...)
}

func (l *subLogger) Warn(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	l.root.Warn(msg, resultCtx...)
}

func (l *subLogger) Error(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	l.root.Error(msg, resultCtx...)
}

func (l *subLogger) Fatal(msg string, ctx ...interface{}) {
	resultCtx := l.newContext(ctx)
	l.root.Fatal(msg, resultCtx...)
}

func (l *subLogger) newContext(ctx []interface{}) []interface{} {
	resultCtx := []interface{}{}
	resultCtx = append(resultCtx, l.context...)
	resultCtx = append(resultCtx, ctx...)
	return resultCtx
}
