# P2P Chat Signaling service

service to manage chatting rooms & user access to it

## Development

to develop this project you should have these tools

- `go` with go module enabled
- `make` utility
- `gcc` to compile some deps. that required `CGO` functionality

compile binary using this command

```bash
make all
```

to just run this project, make sure you install deps before run

```bash
make init # (optional) for first build only
make run
```

## Test

this service are test using `ginkgo` BDD test kit, make sure you already init the project before make testing

```bash
make test
```

## GRPC server

to consume grpc service to web we need to run GRPC web proxy

```bash
grpcwebproxy --allow_all_origins --run_tls_server=false --use_websockets --backend_tls=false --backend_addr=localhost:8053 --server_http_debug_port=9012
```
