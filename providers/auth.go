//go:generate mockgen -destination=./mocks/auth.go -package=mocks . Auth

package providers

import (
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/spf13/viper"
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
