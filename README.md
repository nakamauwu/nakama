# Nakama [![join slack](https://img.shields.io/badge/slack-join-none.svg?style=social&logo=slack)](https://join.slack.com/t/nakama-social/shared_invite/zt-143j6bzie-spuCdq79xIZJQa4DaPb0uQ)

The next social network for anime fans.

Follow the development on [YouTube](https://www.youtube.com/playlist?list=PLOzDrFftjC09iI8rgmj9JZr4bwImeUpkf).

## Dependencies

- [Go Programming Language](https://go.dev)
- [CockroachDB](https://cockroachlabs.com)
- [Minio](https://min.io)

```bash
cockroach start-single-node --insecure --listen-addr 127.0.0.1
```

```bash
minio server ./minio-data
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
  -s3-access-key string
        S3 access key (default "minioadmin")
  -s3-endpoint string
        S3 endpoint (default "localhost:9000")
  -s3-secret-key string
        S3 secret key (default "minioadmin")
  -s3-secure
        Enable S3 SSL
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

## Development

When adding a new feature, you start by modifying the `schema.sql` file,
and then writing the necessary queries in a `_store.go` file.
All methods are part of the `*Service` type and prefixed with `sql`.

Now create a new exported method as part of the `nakama.Service` struct.

When creating a new method, if the input data has many fields,
then create a struct with the same name and the output in past-tense.

Example:

```go
package nakama

type CreatePost struct {}
type CreatedPost struct {}

func (*Service) CreatePost(ctx context.Context, in CreatePost) (CreatedPost, error) {}
```

To register a new route, go to the `web` package
and register it inside the `web/handler.go` file
at the `init()` method of `web.Handler`.

Add a new unexported method to the `web.Handler` and you should be able to
access `nakama.Service` from within your handler.

Routes that render a page follow the naming of `showPage`.

Example:

```go
package web

func (*Handler) showLogin(w http.ResponseWriter, r *http.Request) {}
```

HTML templates are defined in the `web/template` directory.
To render a template, parse them first on the `web` package by using the
`parseTmpl` function. And save them as an unexported package level variable.

The directory `web/template/include` contains templates that are not pages;
these are like "partials" and are the pieces that other templates include.

Any new dependency must be injected explicitly on `cmd/nakama/main.go`.
Here is were configuration is also read. No other place is allowed to read
configuration but here. That means, any call to `os.Getenv()` for example
is discouraged but on `main.go`.

## Troubleshoot

If you run into migrations issues, that is because the SQL schema
is not applied progressively. For now, you will have to cleanup the database
and run the server again.

You could stop cockroach. Remove the `cockroach-data` directory:

```bash
rm -rf ./cockroach-data
```

And start cockroach again.
