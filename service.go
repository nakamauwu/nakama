package nakama

import "github.com/rs/xid"

type Service struct {
	Queries *Queries
}

func genID() string {
	return xid.New().String()
}

func isID(s string) bool {
	_, err := xid.FromString(s)
	return err == nil
}
