package logger

import (
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const DEBUG_VERBOSITY = 1
const TRACE_VERBOSITY = 2

type Logger struct {
	Name string
	Log  logr.Logger
}

func New(name string) Logger {
	logger := ctrl.Log.WithName(fmt.Sprintf("[%s]", name))

	return Logger{
		Name: name,
		Log:  logger,
	}
}

func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.Log.Info(msg, keysAndValues)
}

func (l *Logger) Error(err error, msg string, keysAndValues ...any) {
	l.Log.Error(err, msg, keysAndValues)
}

func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.Log.V(DEBUG_VERBOSITY).Info(msg, keysAndValues)
}

func (l *Logger) Trace(msg string, keysAndValues ...any) {
	l.Log.V(TRACE_VERBOSITY).Info(msg, keysAndValues)
}
