# tools.go.monitoring
vmstat and beyond, in go

## How to...

### Build

= compile all and create exe for local architecture in current dir

```sh
$ go build
```
or `;b` in vim

### Install

= cross-compile all and create exe for other architecture in go bin dir

```sh
$ GOOS=linux GOARCH=amd64 go install
```

### Test

= run on a host without proc fs

```sh
$ FS_ROOT=.samples go run cmd/cpustat/main.go (options)
```
