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

## Development

Nakama uses [sqlc](https://sqlc.dev) to quickstart all queries to the database.

When adding a new feature, you start by modifying the `schema.sql` file,
and then writing the necessary queries in the `queries.sql` file.

Then you run:

```bash
sqlc generate
````

That will update `nakama.Queries` struct and add the new queries as a method.
It will generate the necessary types too.

Now create a new method as part of the `nakama.Service` struct.
You should have access to the queries from within service.

When creating a new method, if input data has many fields,
then create a struct following the naming of `MethodInput`
and output `MethodOutput`.

Example:

```go
package nakama

type LoginInput struct {}
type LoginOutput struct {}

func (*Service) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {}
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
