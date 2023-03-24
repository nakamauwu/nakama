package nakama

import (
	"context"
	"fmt"
	"sync"
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
	wg            sync.WaitGroup
}

// background runs a function in a goroutine,
// recovering from panics and logging errors.
func (svc *Service) background(fn func(ctx context.Context) error) {
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				svc.Logger.Error("recover", fmt.Errorf("%v", err))
			}
		}()

		if err := fn(svc.BaseContext()); err != nil {
			svc.Logger.Error("background", err)
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
