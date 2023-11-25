package basicauth

import (
	"context"
	"strings"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/autom8ter/grpcx/providers"
)

// BasicAuth is a provider that ensures the request is authenticated with the configured username/password
// in the config('auth.username' and 'auth.password')
type BasicAuth struct{}

// New returns a new BasicAuth provider
func New() *BasicAuth {
	return &BasicAuth{}
}

// Auth returns an auth function that ensures the request is authenticated with the configured username/password
// in the config('auth.username' and 'auth.password')
func (b *BasicAuth) Auth(cfg *viper.Viper, all providers.All) grpc_auth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		header, err := grpc_auth.AuthFromMD(ctx, "basic")
		if err != nil {
			return nil, err
		}
		split := strings.Split(header, ":")
		if split[0] != cfg.GetString("auth.username") || split[1] != cfg.GetString("auth.password") {
			return nil, status.Error(codes.Unauthenticated, "invalid username/password")
		}
		return ctx, nil
	}
}
