# impulse

Analyze the response of a protocol request

## Build Instruction

```
$ go test -cover -v ./...
$ go build -o impulse -v cmd/impulse/main.go
```

## Examples

* TCP

```
$ ./impulse -url "tcp://google.com:443" -tls
```