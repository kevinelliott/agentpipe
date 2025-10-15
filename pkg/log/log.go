// Package log provides structured logging using zerolog.
// It wraps zerolog to provide a clean API for application-wide logging.
package log

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog.Logger with application-specific methods.
type Logger struct {
	zlog zerolog.Logger
}

// Global logger instance
var global *Logger

func init() {
	// Initialize with console writer for development
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}
	global = &Logger{
		zlog: zerolog.New(output).With().Timestamp().Logger(),
	}
}

// New creates a new Logger instance with the given writer.
func New(w io.Writer) *Logger {
	return &Logger{
		zlog: zerolog.New(w).With().Timestamp().Logger(),
	}
}

// NewWithLevel creates a new Logger with a specific log level.
func NewWithLevel(w io.Writer, level zerolog.Level) *Logger {
	return &Logger{
		zlog: zerolog.New(w).With().Timestamp().Logger().Level(level),
	}
}

// SetGlobalLevel sets the global logging level.
func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(logger *Logger) {
	global = logger
}

// With creates a child logger with additional fields.
func (l *Logger) With() *Logger {
	return &Logger{
		zlog: l.zlog.With().Logger(),
	}
}

// WithField adds a field to the logger context.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		zlog: l.zlog.With().Interface(key, value).Logger(),
	}
}

// WithFields adds multiple fields to the logger context.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zlog.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{
		zlog: ctx.Logger(),
	}
}

// WithError adds an error field to the logger context.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		zlog: l.zlog.With().Err(err).Logger(),
	}
}

// Debug logs a message at debug level.
func (l *Logger) Debug(msg string) {
	l.zlog.Debug().Msg(msg)
}

// Debugf logs a formatted message at debug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zlog.Debug().Msgf(format, args...)
}

// Info logs a message at info level.
func (l *Logger) Info(msg string) {
	l.zlog.Info().Msg(msg)
}

// Infof logs a formatted message at info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.zlog.Info().Msgf(format, args...)
}

// Warn logs a message at warn level.
func (l *Logger) Warn(msg string) {
	l.zlog.Warn().Msg(msg)
}

// Warnf logs a formatted message at warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zlog.Warn().Msgf(format, args...)
}

// Error logs a message at error level.
func (l *Logger) Error(msg string) {
	l.zlog.Error().Msg(msg)
}

// Errorf logs a formatted message at error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.zlog.Error().Msgf(format, args...)
}

// Fatal logs a message at fatal level and exits.
func (l *Logger) Fatal(msg string) {
	l.zlog.Fatal().Msg(msg)
}

// Fatalf logs a formatted message at fatal level and exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zlog.Fatal().Msgf(format, args...)
}

// Global logger convenience functions

// Debug logs a message at debug level using the global logger.
func Debug(msg string) {
	global.Debug(msg)
}

// Debugf logs a formatted message at debug level using the global logger.
func Debugf(format string, args ...interface{}) {
	global.Debugf(format, args...)
}

// Info logs a message at info level using the global logger.
func Info(msg string) {
	global.Info(msg)
}

// Infof logs a formatted message at info level using the global logger.
func Infof(format string, args ...interface{}) {
	global.Infof(format, args...)
}

// Warn logs a message at warn level using the global logger.
func Warn(msg string) {
	global.Warn(msg)
}

// Warnf logs a formatted message at warn level using the global logger.
func Warnf(format string, args ...interface{}) {
	global.Warnf(format, args...)
}

// Error logs a message at error level using the global logger.
func Error(msg string) {
	global.Error(msg)
}

// Errorf logs a formatted message at error level using the global logger.
func Errorf(format string, args ...interface{}) {
	global.Errorf(format, args...)
}

// Fatal logs a message at fatal level and exits using the global logger.
func Fatal(msg string) {
	global.Fatal(msg)
}

// Fatalf logs a formatted message at fatal level and exits using the global logger.
func Fatalf(format string, args ...interface{}) {
	global.Fatalf(format, args...)
}

// WithField creates a logger with an additional field using the global logger.
func WithField(key string, value interface{}) *Logger {
	return global.WithField(key, value)
}

// WithFields creates a logger with multiple fields using the global logger.
func WithFields(fields map[string]interface{}) *Logger {
	return global.WithFields(fields)
}

// WithError creates a logger with an error field using the global logger.
func WithError(err error) *Logger {
	return global.WithError(err)
}

// GetZerolog returns the underlying zerolog.Logger for advanced usage.
func (l *Logger) GetZerolog() *zerolog.Logger {
	return &l.zlog
}

// InitLogger initializes the global logger with specific configuration.
// This should be called at application startup.
func InitLogger(w io.Writer, level zerolog.Level, pretty bool) {
	var output io.Writer = w

	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
		}
	}

	global = &Logger{
		zlog: zerolog.New(output).With().Timestamp().Logger().Level(level),
	}

	log.Logger = global.zlog
}

// ParseLevel converts a string level to zerolog.Level.
func ParseLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
