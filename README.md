# Nakama

Start database:

```bash
cockroach start-single-node --insecure --listen-addr localhost:36257 --sql-addr localhost:26257
````

Run db migrations:

```bash
cat cockroach/migrations/0000_init.sql | cockroach sql --insecure
```

Start development:

```bash
go tool air
```