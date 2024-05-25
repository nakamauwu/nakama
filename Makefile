.PHONY: launch-db
launch-db:
	cockroach start-single-node --insecure --listen-addr localhost:36257 --sql-addr localhost:26257
