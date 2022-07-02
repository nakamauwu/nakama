# Nakama

The next social network for anime fans.

## Dependencies

- [Go Programming Language](https://go.dev)
- [CockroachDB](https://cockroachlabs.com)

```bash
cockroach start-single-node --insecure --listen-addr 127.0.0.1
```

## Build

```bash
go mod download
go build ./cmd/nakama
```

## Usage (`-h`)

```bash
Usage of nakama:
  -addr string
        HTTP service address (default ":4000")
  -session-key string
        Session key (default "secretkeyyoushouldnotcommit")
  -sql-addr string
        SQL address (default "postgresql://root@127.0.0.1:26257/defaultdb?sslmode=disable")
```

## Testing

Make sure to have [Docker](https://www.docker.com/) running before running the tests.

```bash
go test .
```
