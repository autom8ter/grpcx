//go:generate mockgen -destination=./mocks/auth.go -package=mocks . Auth

package providers

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Auth is an interface for authentication
type Auth interface {
	// Auth returns an auth function that authenticates/authorizes requests
	Auth(cfg *viper.Viper, all All) grpc_auth.AuthFunc
}

// AuthFunc is a function that returns an auth function that authenticates requests
type AuthFunc func(cfg *viper.Viper, all All) grpc_auth.AuthFunc

// Auth returns an auth function that authenticates requests
func (a AuthFunc) Auth(cfg *viper.Viper, all All) grpc_auth.AuthFunc {
	return a(cfg, all)
}

// ChainAuthFuncs chains auth functions together. The first auth function to return a nil error will be used.
// If all auth functions return an error, the request will be rejected.
func ChainAuthFuncs(auths ...Auth) Auth {
	return AuthFunc(func(cfg *viper.Viper, all All) grpc_auth.AuthFunc {
		return func(ctx context.Context) (context.Context, error) {
			for _, auth := range auths {
				ctx, err := auth.Auth(cfg, all)(ctx)
				if err == nil {
					return ctx, nil
				}
			}
			return ctx, status.Errorf(codes.Unauthenticated, "unauthenticated")
		}
	})
}
