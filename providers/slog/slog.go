package slog

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/autom8ter/grpcx/providers"
)

// Logger is an slog logger that implements the logging.Logger interface
type Logger struct {
	logger *slog.Logger
}

// NewTextLogger returns a new text logger
func NewTextLogger(opts *slog.HandlerOptions) *Logger {
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	return &Logger{logger: logger}
}

// NewJSONLogger returns a new json logger
func NewJSONLogger(opts *slog.HandlerOptions) *Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	return &Logger{logger: logger}
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, tags ...map[string]any) {
	var args []any
	t, ok := providers.GetTags(ctx)
	if ok {
		tags = append(tags, t.LogTags())
	}
	for _, tag := range tags {
		for k, v := range tag {
			args = append(args, k, v)
		}
	}
	l.logger.InfoContext(ctx, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string, tags ...map[string]any) {
	var args []any
	t, ok := providers.GetTags(ctx)
	if ok {
		tags = append(tags, t.LogTags())
	}
	for _, tag := range tags {
		for k, v := range tag {
			args = append(args, k, v)
		}
	}
	l.logger.ErrorContext(ctx, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, tags ...map[string]any) {
	var args []any
	t, ok := providers.GetTags(ctx)
	if ok {
		tags = append(tags, t.LogTags())
	}
	for _, tag := range tags {
		for k, v := range tag {
			args = append(args, k, v)
		}
	}
	l.logger.WarnContext(ctx, msg, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, tags ...map[string]any) {
	var args []any
	t, ok := providers.GetTags(ctx)
	if ok {
		tags = append(tags, t.LogTags())
	}
	for _, tag := range tags {
		for k, v := range tag {
			args = append(args, k, v)
		}
	}
	l.logger.DebugContext(ctx, msg, args...)
}

// Provider returns a slog logger(logging.level)
func Provider(ctx context.Context, cfg *viper.Viper) (providers.Logger, error) {
	lcfg := &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	}
	switch strings.ToLower(cfg.GetString("logging.level")) {
	case "info":
		lcfg.Level = slog.LevelInfo
	case "warn":
		lcfg.Level = slog.LevelWarn
	case "error":
		lcfg.Level = slog.LevelError
	default:
		lcfg.Level = slog.LevelDebug
	}
	return NewJSONLogger(lcfg), nil
}
