package providers

import (
	"context"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiter is a function that returns true if the request is allowed
type RateLimiter interface {
	// Allow returns true if the request is allowed
	Allow(ctx context.Context) bool
}

// RateLimiterFunc is a function that returns true if the request is allowed
type RateLimiterFunc func(ctx context.Context) bool

// Allow returns true if the request is allowed
func (r RateLimiterFunc) Allow(ctx context.Context) bool {
	return r(ctx)
}

// RateLimiterProvider is a function that returns a RateLimiter
type RateLimiterProvider func(ctx context.Context, cfg *viper.Viper) (RateLimiter, error)

// UnaryRateLimitInterceptor returns a grpc unary server interceptor that rate limits requests
func UnaryRateLimitInterceptor(rateLimiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if rateLimiter.Allow(ctx) {
			return handler(ctx, req)
		}
		return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}
}

// StreamRateLimitInterceptor returns a grpc stream server interceptor that rate limits requests
func StreamRateLimitInterceptor(rateLimiter RateLimiter) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if rateLimiter.Allow(ss.Context()) {
			return handler(srv, ss)
		}
		return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}
}
