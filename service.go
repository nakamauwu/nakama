package nakama

import (
	"context"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/rs/xid"
)

type Service struct {
	Store         *Store
	S3            *minio.Client
	Logger        *log.Logger
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
