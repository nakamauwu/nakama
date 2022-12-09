package nakama

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/nakamauwu/nakama/db"
	"github.com/rs/xid"
)

type Service struct {
	Logger        *log.Logger
	DB            *db.DB
	S3            *minio.Client
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
