package logger

import (
	"fmt"
	"log"
)

type Logger struct {
	l *log.Logger
}

func New(l *log.Logger) *Logger {
	return &Logger{l: l}
}

func (l *Logger) LogErrorf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.l.Printf("[Error]: %s\n", msg)
}

func (l *Logger) LogInfo(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.l.Printf("[Info]: %s\n", msg)
}
