# gRPCX

gRPCX is a library for easily building gRPC services in Go.
It has support for REST APIs, ORM, Database Migrations, Cache, Message Streaming,
Structured Logging, Config, Healthchecks, Prometheus Metrics, and more.

It is meant to be a batteries-included framework for building gRPC services.

    go get github.com/autom8ter/grpcx

## Example
    
```go

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := grpcx.NewConfig()
	if err != nil {
		panic(err)
	}
	cfg.Set("auth.username", "test")
	cfg.Set("auth.password", "test")
	srv, err := grpcx.NewServer(
		ctx,
		cfg,
		// Register Auth provider
		grpcx.WithAuth(basicauth.New()),
		// Register Cache Provider
		grpcx.WithCache(redis2.InMemProvider),
		// Register Stream Provider
		grpcx.WithStream(redis2.InMemStreamProvider),
		// Register Context Tagger
		grpcx.WithContextTagger(maptags.Provider),
		// Register Logger
		grpcx.WithLogger(slog2.Provider),
		// Register Database
		grpcx.WithDatabase(sqlite.Provider),
	)
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Serve(ctx, grpcx.EchoService()); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("localhost:%v", cfg.GetInt("api.port")), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	echoClient := echov1.NewEchoServiceClient(conn)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "test:test")
	resp, err := echoClient.Echo(ctx, &echov1.EchoRequest{
		Message: "hello",
	})
	if err != nil {
		panic(err)
	}
	println(resp.Message)

```

## Features

- [x] [gRPC](https://grpc.io/)
- [x] [gRPC-gateway](https://grpc.io/docs/languages/go/basics/#grpc-gateway)
- [x] Serve Gateway/gRPC same port
- [x] Flexible Provider/Interface based architecture: use the same code with different providers depending on tech stack
- [x] Metrics(prometheus)
- [x] Structured Logging(slog)
- [x] Database Migrations(golang-migrate)
- [x] Database dependency injection(sqlite,mysql,postgres)
- [x] Cache dependency injection(redis)
- [x] Stream dependency injection(redis,nats,kafka,rabbitmq)
- [x] Context Tagging Middleware(maptags)
- [x] Auth dependency injection(basicauth)
- [x] Flexible configuration(viper)
- [x] Graceful Shutdown
- [x] Request validation middleware/interceptor
- [x] Panic Recovery middleware/interceptor
- [x] CORS middleware
- [x] Rate Limiting middleware/interceptor
- [ ] Tracing

