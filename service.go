package nakama

import (
	"context"
	"log"

	"github.com/nakamauwu/nakama/db"
	"github.com/rs/xid"
)

type Service struct {
	Logger      *log.Logger
	DB          *db.DB
	BaseContext func() context.Context
}

func genID() string {
	return xid.New().String()
}

func isID(s string) bool {
	_, err := xid.FromString(s)
	return err == nil
}
