package nakama

import (
	"log"

	"github.com/rs/xid"
)

type Service struct {
	Queries *Queries
	Logger  *log.Logger
}

func genID() string {
	return xid.New().String()
}

func isID(s string) bool {
	_, err := xid.FromString(s)
	return err == nil
}
