# Nakama

Source code of the next social network for anime fans. Still on development.

```bash
$ nakama -h
Usage of nakama:
  -port int
        Port ($PORT) (default 3000)
  -origin string
        Origin URL ($ORIGIN) (default "http://localhost:3000")
  -db string
        Database URN ($DB_URN) (default "postgresql://root@127.0.0.1:26257/nakama?sslmode=disable")
  -key string
        32 bytes long secret key to sign tokens ($SECRET_KEY) (default "supersecretkeyyoushouldnotcommit")
  -smtp.host string
        SMTP host ($SMTP_HOST) (default "smtp.mailtrap.io")
  -smtp.port int
        SMTP port ($SMTP_PORT) (default 25)
  -smtp.username string
        SMTP username ($SMTP_USERNAME)
  -smtp.password string
        SMTP password ($SMTP_PASSWORD)
```

## Getting the code

Besides having [Go](https://golang.org/) installed, the server needs two external services. A SQL database; I'm using [CockroachDB](https://www.cockroachlabs.com/), but Postgres should work too. Also, an SMTP server; I recommend [mailtrap.io](https://mailtrap.io/) for development.

These are the Go libraries used in the source code. Thank you very much.
 - [github.com/disintegration/imaging](https://github.com/disintegration/imaging)
 - [github.com/hako/branca](https://github.com/hako/branca)
 - [github.com/joho/godotenv](https://github.com/joho/godotenv)
 - [github.com/lib/pq](https://github.com/lib/pq)
 - [github.com/matoous/go-nanoid](https://github.com/matoous/go-nanoid)
 - [github.com/matryer/way](https://github.com/matryer/way)
