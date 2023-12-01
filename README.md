# gRPCX

![GoDoc](https://godoc.org/github.com/autom8ter/grpcx?status.svg)

gRPCX is a framework for building enterprise-grade Saas applications with gRPC.
It is meant to be a batteries-included framework for building enterprise-grade gRPC services.

    go get github.com/autom8ter/grpcx

## Features

- [x] [gRPC](https://grpc.io/)
- [x] [gRPC-gateway](https://grpc.io/docs/languages/go/basics/#grpc-gateway)
- [x] Serve Gateway/gRPC same port(cmux)
- [x] Flexible Provider/Interface based architecture: use the same code with different providers depending on tech stack
- [x] Metrics(prometheus)
- [x] Structured Logging(slog)
- [x] Database Migrations(golang-migrate)
- [x] Database dependency injection(sqlite,mysql,postgres)
- [x] Cache dependency injection(redis)
- [x] Stream dependency injection(redis,nats)
- [x] Email dependency injection(smtp)
- [x] Payment Processing dependency injection(stripe)
- [x] Context Tagging Middleware/Interceptor
- [x] Authentication with [protoc-gen-authenticate](https://github.com/autom8ter/protoc-gen-authenticate)
- [x] Authorization with [protoc-gen-authorize](https://github.com/autom8ter/protoc-gen-authorize)
- [x] Rate-limiting with [protoc-gen-ratelimit](https://github.com/autom8ter/protoc-gen-ratelimit)
- [x] Request Validation with [protoc-gen-validate](https://github.com/bufbuild/protoc-gen-validate)
- [x] Compatible with protoc-gen-validate
- [x] Flexible configuration(viper)
- [x] Graceful Shutdown
- [x] Panic Recovery middleware/interceptor
- [x] CORS middleware(for gRPC-gateway)
- [x] Can be run completely in-memory for testing

## Other gRPC Projects

- [protoc-gen-authorize](https://github.com/autom8ter/protoc-gen-authenticate)
- [protoc-gen-authorize](https://github.com/autom8ter/protoc-gen-authorize)
- [protoc-gen-ratelimit](https://github.com/autom8ter/protoc-gen-ratelimit)

## TODO

- [ ] Add more tests
- [ ] Add more examples
- [ ] Add more documentation
- [ ] Add kafka stream provider
- [ ] Add rabbitmq stream provider
- [ ] Add sendgrid email provider
- [ ] Add mailgun email provider
- [ ] Add tracing support