package grpcxtest

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/autom8ter/grpcx"
)

func randInt(min int, max int) int {
	return min + rand.Intn(max-min+1)
}

// TestFunc is a function that runs a test against a grpc client connection pointed at the test server
type TestFunc func(t *testing.T, ctx context.Context, client grpc.ClientConnInterface)

// Fixture is a test fixture that runs a test against a grpc client connection pointed at the test server
type Fixture struct {
	Config     *viper.Viper
	Timeout    time.Duration
	ServerOpts []grpcx.ServerOption
	Services   []grpcx.Service
	Name       string
	Test       TestFunc
	ClientMeta map[string]string
}

// RunTest runs the test against a grpc client connection pointed at the test server
func (f *Fixture) RunTest(t *testing.T) {
	if f.Timeout == 0 {
		f.Timeout = time.Second * 10
	}
	if f.ClientMeta == nil {
		f.ClientMeta = map[string]string{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), f.Timeout)
	defer cancel()
	if f.Config == nil {
		cfg, err := grpcx.NewConfig()
		require.NoError(t, err)
		f.Config = cfg
	}
	f.Config.Set("api.port", randInt(49152, 65535))
	grpcPort := f.Config.GetInt("api.port")
	go func() {
		t.Run("server", func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			srv, err := grpcx.NewServer(
				ctx,
				f.Config,
				f.ServerOpts...,
			)
			if err != nil {
				panic(err)
			}
			if err := srv.Serve(ctx, f.Services...); err != nil {
				srv.Providers().Logger.Error(ctx, "server failure", map[string]any{
					"error": err.Error(),
				})
			}
		})

	}()
	time.Sleep(time.Second * 2)
	{
		for k, v := range f.ClientMeta {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
		grpcClient, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%v", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		t.Run(f.Name, func(t *testing.T) {
			f.Test(t, ctx, grpcClient)
		})
	}
}
