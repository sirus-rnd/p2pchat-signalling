# Console API

[![pipeline status](https://gitlab.com/quantum-hive/cloud-centrals/console-api/badges/develop/pipeline.svg)](https://gitlab.com/quantum-hive/cloud-centrals/console-api/-/commits/develop)
[![coverage report](https://gitlab.com/quantum-hive/cloud-centrals/console-api/badges/develop/coverage.svg)](https://gitlab.com/quantum-hive/cloud-centrals/console-api/-/commits/develop)

expose admin console functionality as a webservice

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

