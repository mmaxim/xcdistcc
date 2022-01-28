package common

import (
	"io"
	"log"
	"os"
)

type Logger interface {
	Log(fmt string, args ...any)
}

// =============================================================================

type StdLogger struct {
	logger *log.Logger
}

func NewStdLogger() *StdLogger {
	return &StdLogger{
		logger: log.Default(),
	}
}

func NewStdLoggerWithFilepath(path string) (*StdLogger, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &StdLogger{
		logger: log.New(file, "", 0),
	}, nil
}

func NewStdLoggerWithWriter(w io.Writer) *StdLogger {
	return &StdLogger{
		logger: log.New(w, "", 0),
	}
}

func (l *StdLogger) Log(fmt string, args ...any) {
	l.logger.Printf(fmt, args...)
}

// =============================================================================

type QuietLogger struct{}

func NewQuietLogger() QuietLogger {
	return QuietLogger{}
}

func (l QuietLogger) Log(fmt string, args ...any) {}

// =============================================================================

type LabelLogger struct {
	label  string
	logger Logger
}

func NewLabelLogger(label string, logger Logger) *LabelLogger {
	return &LabelLogger{
		label:  label,
		logger: logger,
	}
}

func (l *LabelLogger) Debug(text string, args ...interface{}) {
	l.logger.Log(l.label+": "+text, args...)
}

func (l *LabelLogger) GetLogger() Logger {
	return l.logger
}
