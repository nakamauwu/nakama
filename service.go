package nakama

import (
	"context"
	"log"

	"github.com/rs/xid"
)

type Service struct {
	Queries     *Queries
	Logger      *log.Logger
	BaseContext func() context.Context
}

func genID() string {
	return xid.New().String()
}

func isID(s string) bool {
	_, err := xid.FromString(s)
	return err == nil
}
