//go:generate mockgen -destination=./mocks/logging.go -package=mocks . Logger

package providers

import (
	"context"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logger is an interface for logging
type Logger interface {
	// Info logs an info message
	Info(ctx context.Context, msg string, tags ...map[string]any)
	// Error logs an error message
	Error(ctx context.Context, msg string, tags ...map[string]any)
	// Warn logs a warning message
	Warn(ctx context.Context, msg string, tags ...map[string]any)
	// Debug logs a debug message
	Debug(ctx context.Context, msg string, tags ...map[string]any)
}

// LoggingProvider is a function that returns a Logger
type LoggingProvider func(ctx context.Context, cfg *viper.Viper) (Logger, error)

// UnaryLoggingInterceptor adds a unary logging interceptor to the server
func UnaryLoggingInterceptor(requestBody bool, logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		tags, ok := GetTags(ctx)
		if !ok {
			return nil, status.Error(codes.Internal, "failed to get tags from context")
		}
		logger.Info(ctx, "request received")
		resp, err := handler(ctx, req)
		if err != nil {
			tags = tags.WithError(err)
			logger.Error(WithTags(ctx, tags), "request failed")
		}
		return resp, err
	}
}

// StreamLoggingInterceptor adds a stream logging interceptor to the server
func StreamLoggingInterceptor(requestBody bool, logger Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		tags, ok := GetTags(ss.Context())
		if !ok {
			return status.Error(codes.Internal, "failed to get tags from context")
		}
		logger.Info(ss.Context(), "request received")
		err := handler(srv, ss)
		if err != nil {
			tags = tags.WithError(err)
			logger.Error(WithTags(ss.Context(), tags), "request failed")
		}
		return err
	}
}
