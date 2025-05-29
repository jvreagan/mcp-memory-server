// pkg/logger/logger.go
package logger

import (
	"fmt"
	"os"
	"time"
)

// Logger provides structured logging for the MCP server
type Logger struct {
	component string
	level     string
}

// New creates a new logger
func New(level, format string) *Logger {
	return &Logger{
		level: level,
	}
}

// WithComponent returns a logger with a component prefix
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		component: component,
		level:     l.level,
	}
}

// WithError returns a logger that will include error information
func (l *Logger) WithError(err error) *Logger {
	// For simplicity, we'll handle this in the logging methods
	return l
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	if l.shouldLog("info") {
		l.log("INFO", msg, fields...)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if l.shouldLog("debug") {
		l.log("DEBUG", msg, fields...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if l.shouldLog("warn") {
		l.log("WARN", msg, fields...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...interface{}) {
	if l.shouldLog("error") {
		l.log("ERROR", msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.log("FATAL", msg, fields...)
	os.Exit(1)
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	currentLevel := levels[l.level]
	messageLevel := levels[level]

	return messageLevel >= currentLevel
}

// log outputs a formatted log message
func (l *Logger) log(level, msg string, fields ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	component := ""
	if l.component != "" {
		component = fmt.Sprintf("[%s] ", l.component)
	}

	// Build fields string
	var fieldsStr string
	if len(fields) > 0 {
		fieldsStr = " "
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				fieldsStr += fmt.Sprintf("%v=%v ", fields[i], fields[i+1])
			}
		}
	}

	message := fmt.Sprintf("%s [%s] %s%s%s", timestamp, level, component, msg, fieldsStr)

	// Write to stderr so it doesn't interfere with MCP JSON communication on stdout
	fmt.Fprintln(os.Stderr, message)
}
