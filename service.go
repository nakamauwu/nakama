package nakama

import (
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rs/xid"
	"golang.org/x/exp/slog"
)

type Service struct {
	Store         *Store
	S3            *minio.Client
	Logger        *slog.Logger
	AvatarsPrefix string
	BaseContext   func() context.Context
}

func genID() string {
	return xid.New().String()
}

func validID(s string) bool {
	_, err := xid.FromString(s)
	return err == nil
}

type Created struct {
	ID        string
	CreatedAt time.Time
}
