package common

import "log"

type LabelLogger struct {
	label string
}

func NewLabelLogger(label string) *LabelLogger {
	return &LabelLogger{
		label: label,
	}
}

func (l *LabelLogger) Debug(text string, args ...interface{}) {
	log.Printf(l.label+": "+text, args...)
}
