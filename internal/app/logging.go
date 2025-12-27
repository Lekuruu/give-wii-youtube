package app

import (
	"fmt"
	"time"
)

type Logger struct {
	Name string
}

func NewLogger(config interface{}) *Logger {
	name := "wii-youtube"
	if cfg, ok := config.(*Config); ok {
		_ = cfg // Could use config for log level, etc.
	} else if n, ok := config.(string); ok {
		name = n
	}
	return &Logger{Name: name}
}

func (l *Logger) WriteLevel(bs []byte, level string) (int, error) {
	now := time.Now().UTC().Format(time.DateTime)
	logMessage := fmt.Sprintf("[%s] - <%s> %s: %s", now, l.Name, level, bs)
	return fmt.Print(logMessage)
}

func (l *Logger) Write(bs []byte) (int, error) {
	return l.WriteLevel(bs, "INFO")
}

func (l *Logger) Log(args ...interface{}) {
	logMessage := fmt.Sprint(args...)
	l.WriteLevel([]byte(logMessage+"\n"), "INFO")
}

func (l *Logger) Logf(format string, args ...interface{}) {
	logMessage := fmt.Sprintf(format, args...)
	l.WriteLevel([]byte(logMessage+"\n"), "INFO")
}

func (l *Logger) Error(args ...interface{}) {
	logMessage := fmt.Sprint(args...)
	l.WriteLevel([]byte("[ERROR] "+logMessage+"\n"), "ERROR")
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	logMessage := fmt.Sprintf(format, args...)
	l.WriteLevel([]byte("[ERROR] "+logMessage+"\n"), "ERROR")
}

func (l *Logger) Debug(args ...interface{}) {
	logMessage := fmt.Sprint(args...)
	l.WriteLevel([]byte("[DEBUG] "+logMessage+"\n"), "DEBUG")
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	logMessage := fmt.Sprintf(format, args...)
	l.WriteLevel([]byte("[DEBUG] "+logMessage+"\n"), "DEBUG")
}
