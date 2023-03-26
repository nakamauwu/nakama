package nakama

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/minio/minio-go/v7"
	"github.com/rs/xid"
)

type Service struct {
	Store       *Store
	S3          *minio.Client
	Logger      *log.Logger
	S3Prefix    string
	BaseContext func() context.Context
	wg          sync.WaitGroup
}

// background runs a function in a goroutine,
// recovering from panics and logging errors.
func (svc *Service) background(fn func(ctx context.Context) error) {
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				svc.Logger.Error("recover", "err", fmt.Errorf("%v", err))
			}
		}()

		if err := fn(svc.BaseContext()); err != nil {
			svc.Logger.Error("background", "err", err)
		}
	}()
}

func (svc *Service) Wait() {
	svc.wg.Wait()
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
