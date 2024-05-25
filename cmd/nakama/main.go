package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/nakamauwu/nakama/cockroach"
	"github.com/nakamauwu/nakama/oauth"
	"github.com/nakamauwu/nakama/service"
	"github.com/nakamauwu/nakama/web"
)

type configuration struct {
	CockroachSQLAddr string `env:"COCKROACH_SQL_ADDR,notEmpty" envDefault:"postgresql://root@localhost:26257/defaultdb?sslmode=disable"`
	Github           struct {
		ClientID     string `env:"OAUTH2_CLIENT_ID_GITHUB,notEmpty"`
		ClientSecret string `env:"OAUTH2_CLIENT_SECRET_GITHUB,notEmpty,unset"`
	}
}

func main() {
	_ = godotenv.Load()

	errLogger := slog.New(tint.NewHandler(os.Stderr, nil))

	if err := run(errLogger); err != nil {
		errLogger.Error("run", "err", err)
		os.Exit(1)
	}
}

func run(errLogger *slog.Logger) error {
	var cfg configuration
	if err := env.Parse(&cfg); err != nil {
		return err
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.CockroachSQLAddr)
	if err != nil {
		return err
	}

	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		return err
	}

	svc := &service.Service{
		Cockroach: cockroach.New(db),
	}

	handler := &web.Handler{
		Logger:       errLogger.With("component", "web"),
		SessionStore: pgxstore.New(db),
		Service:      svc,
		Providers: []oauth.Provider{
			oauth.NewGitHubProvider(cfg.Github.ClientID, cfg.Github.ClientSecret, "http://localhost:4000/oauth2/github/callback"),
		},
	}

	srv := &http.Server{
		Handler:  handler,
		Addr:     ":4000",
		ErrorLog: slog.NewLogLogger(errLogger.Handler(), slog.LevelError),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	infoLogger := slog.New(tint.NewHandler(os.Stdout, nil))
	infoLogger.Info("starting server", "addr", srv.Addr)
	return srv.ListenAndServe()
}
