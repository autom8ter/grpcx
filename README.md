# gRPCX

gRPCX is a library for easily building gRPC services in Go.
It has support for REST APIs, ORM, Database Migrations, Cache, Message Streaming,
Structured Logging, Config, Healthchecks, Prometheus Metrics, and more.

It is meant to be a batteries-included framework for building gRPC services.

    go get github.com/autom8ter/grpcx

## Example

```go


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

