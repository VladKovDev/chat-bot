package logger

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface for logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

type logger struct {
	zap *zap.Logger
}

// Config holds logger configuration
type Config struct {
	Level  string
	Format string // json or text
}

// New creates a new logger
func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	// Enable colors for console output
	if cfg.Format == "text" && isatty.IsTerminal(os.Stdout.Fd()) {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	encoder, err := createEncoder(cfg.Format, encoderConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid log format: %w", err)
	}

	writeSyncer := zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(encoder, writeSyncer, level)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &logger{zap: zapLogger}, nil
}

func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	default:
		return zap.InfoLevel, nil
	}
}

func createEncoder(format string, encoderConfig zapcore.EncoderConfig) (zapcore.Encoder, error) {
	switch format {
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case "text":
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

func convertFields(fields []Field) []zapcore.Field {
	zapFields := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		switch v := f.Value.(type) {
		case string:
			zapFields[i] = zap.String(f.Key, v)
		case int:
			zapFields[i] = zap.Int(f.Key, v)
		case int64:
			zapFields[i] = zap.Int64(f.Key, v)
		case error:
			if v != nil {
				zapFields[i] = zap.Error(v)
			} else {
				zapFields[i] = zap.Skip()
			}
		default:
			zapFields[i] = zap.Any(f.Key, v)
		}
	}
	return zapFields
}

func (l *logger) Debug(msg string, fields ...Field) {
	l.zap.Debug(msg, convertFields(fields)...)
}

func (l *logger) Info(msg string, fields ...Field) {
	l.zap.Info(msg, convertFields(fields)...)
}

func (l *logger) Warn(msg string, fields ...Field) {
	l.zap.Warn(msg, convertFields(fields)...)
}

func (l *logger) Error(msg string, fields ...Field) {
	l.zap.Error(msg, convertFields(fields)...)
}