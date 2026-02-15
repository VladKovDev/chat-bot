package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error

	String(key, value string) Field
	Int(key string, value int) Field
	Int32(key string, value int32) Field
	Int64(key string, value int64) Field
	Bool(key string, value bool) Field
	Err(err error) Field
	Any(key string, value any) Field
}

type Field struct {
	key   string
	value any
}

type logger struct {
	zap *zap.Logger
}

func New(cfg Config) (Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if isEnableColors(cfg) {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	encoder, err := createEncoder(cfg.Format, encoderConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid log format: %w", err)
	}

	writeSyncer, err := createWriteSyncer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create write syncer: %w", err)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &logger{zap: zapLogger}, nil
}

func Noop() Logger {
	return &logger{zap: zap.NewNop()}
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
	case "fatal":
		return zap.FatalLevel, nil
	default:
		return 0, fmt.Errorf("unknown level: %s", level)
	}
}

func createEncoder(format string, encoderConfig zapcore.EncoderConfig) (zapcore.Encoder, error) {
	switch format {
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case "console":
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

func createWriteSyncer(cfg Config) (zapcore.WriteSyncer, error) {
	switch cfg.Output {
	case "stdout":
		return zapcore.AddSync(os.Stdout), nil
	case "stderr":
		return zapcore.AddSync(os.Stderr), nil
	case "file":
		if cfg.FilePath == "" {
			return nil, fmt.Errorf("file_path is required when output is 'file'")
		}

		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
			LocalTime:  true,
		}
		return zapcore.AddSync(lumberjackLogger), nil
	default:
		return nil, fmt.Errorf("unknown output: %s", cfg.Output)
	}
}

func isEnableColors(cfg Config) bool {
	if !cfg.EnableColors {
		return false
	}
	if cfg.Output == "file" {
		return false
	}

	if cfg.Format == "json" {
		return false
	}

	var fd uintptr
	if cfg.Output == "stderr" {
		fd = os.Stderr.Fd()
	} else {
		fd = os.Stdout.Fd()
	}

	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// constructors
func (l *logger) String(key, value string) Field {
	return Field{key: key, value: value}
}

func (l *logger) Int(key string, value int) Field {
	return Field{key: key, value: value}
}

func (l *logger) Int32(key string, value int32) Field {
	return Field{key: key, value: value}
}

func (l *logger) Int64(key string, value int64) Field {
	return Field{key: key, value: value}
}

func (l *logger) Bool(key string, value bool) Field {
	return Field{key: key, value: value}
}

func (l *logger) Err(err error) Field {
	if err == nil {
		return Field{}
	}
	return Field{
		key:   "error",
		value: err,
	}
}

func (l *logger) Any(key string, value any) Field {
	return Field{key: key, value: value}
}

// zap.Field conversion
func convertFields(fields []Field) []zapcore.Field {
	zapFields := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		switch v := f.value.(type) {
		case string:
			zapFields[i] = zap.String(f.key, v)
		case int:
			zapFields[i] = zap.Int(f.key, v)
		case int32:
			zapFields[i] = zap.Int32(f.key, v)
		case int64:
			zapFields[i] = zap.Int64(f.key, v)
		case bool:
			zapFields[i] = zap.Bool(f.key, v)
		case error:
			zapFields[i] = zap.Error(v)
		default:
			zapFields[i] = zap.Any(f.key, v)
		}
	}
	return zapFields
}

// Logger methods
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

func (l *logger) Fatal(msg string, fields ...Field) {
	l.zap.Fatal(msg, convertFields(fields)...)
}

func (l *logger) With(fields ...Field) Logger {
	return &logger{zap: l.zap.With(convertFields(fields)...)}
}

func (l *logger) Sync() error {
	return l.zap.Sync()
}
