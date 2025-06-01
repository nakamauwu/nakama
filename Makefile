.PHONY: launch-db
launch-db:
	cockroach start-single-node --insecure --listen-addr localhost:36257 --sql-addr localhost:26257

.PHONY: migrate-db
migrate-db:
	cat cockroach/migrations/0000_init.sql | cockroach sql --insecure
