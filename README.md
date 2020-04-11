![banner](assets/banner.svg)

# Nakama

Source code of the next social network for anime fans. Still on development.

## Building

Besides having [Go](https://golang.org) installed, the server needs [CockroachDB](https://www.cockroachlabs.com) and [NATS](https://nats.io).

First, you need a cockroach node running.
```bash
cockroach start-single-node --insecure --host 127.0.0.1
```

Then, you need to create the database and tables.
```bash
cat schema.sql | cockroach sql --insecure
```

Then you need to start NATS server.
```bash
nats-server
```

Now, you can build and run the server.

```bash
go build
./nakama
```

Front-end doesn't need any tools or building because it's standard vanilla JavaScript 🙂

## Dependencies

These are the Go libraries used in the source code. Thank you very much.

 - [github.com/cockroachdb/cockroach-go/crdb](https://github.com/cockroachdb/cockroach-go)
 - [github.com/disintegration/imaging](https://github.com/disintegration/imaging)
 - [github.com/hako/branca](https://github.com/hako/branca)
 - [github.com/joho/godotenv](https://github.com/joho/godotenv)
 - [github.com/lib/pq](https://github.com/lib/pq)
 - [github.com/matoous/go-nanoid](https://github.com/matoous/go-nanoid)
 - [github.com/matryer/way](https://github.com/matryer/way)
 - [github.com/nats-io/nats.go](https://github.com/nats-io/nats.go)

[Eva Icons](https://github.com/akveo/eva-icons) are being used in the front-end. Thank you as well.
