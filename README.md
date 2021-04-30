![banner](assets/banner.svg)

# Nakama

Source code of the next social network for anime fans. Still on development.

## Docker build

The easies way to start the server and its dependencies is by using [Docker](https://www.docker.com/).
```
docker-compose up --build
```

## Building

Instead of Docker, you can also install and build stuff by yourself, that way you have complete control.

So, besides having [Go](https://golang.org) installed, the server needs [CockroachDB](https://www.cockroachlabs.com) and [NATS](https://nats.io).

First, you need a cockroach node running.
```bash
cockroach start-single-node --insecure --listen-addr 127.0.0.1
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
go build ./cmd/nakama
./nakama
```

Front-end doesn't need any tools or building because it's standard vanilla JavaScript 🙂

## Dependencies

These are the Go libraries used in the source code. Thank you very much.

 - [github.com/cockroachdb/cockroach-go/crdb](https://github.com/cockroachdb/cockroach-go)
 - [github.com/disintegration/imaging](https://github.com/disintegration/imaging)
 - [github.com/duo-labs/webauthn](https://github.com/duo-labs/webauthn)
 - [github.com/go-mail/mail](https://github.com/go-mail/mail)
 - [github.com/gorilla/securecookie](https://github.com/gorilla/securecookie)
 - [github.com/hako/branca](https://github.com/hako/branca)
 - [github.com/joho/godotenv](https://github.com/joho/godotenv)
 - [github.com/lib/pq](https://github.com/lib/pq)
 - [github.com/matoous/go-nanoid](https://github.com/matoous/go-nanoid)
 - [github.com/matryer/moq](https://github.com/matryer/moq)
 - [github.com/matryer/way](https://github.com/matryer/way)
 - [github.com/minio/minio-go/v7](https://github.com/minio/minio-go)
 - [github.com/nats-io/nats.go](https://github.com/nats-io/nats.go)
 - [github.com/ory/dockertest/v3](https://github.com/ory/dockertest)
 - [github.com/sendgrid/sendgrid-go](github.com/sendgrid/sendgrid-go)

[Eva Icons](https://github.com/akveo/eva-icons) are being used in the front-end. Thank you as well.
