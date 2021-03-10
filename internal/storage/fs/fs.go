package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/nicolasparada/nakama/internal/storage"
)

type Store struct {
	Root string
	once sync.Once
}

func (s *Store) init() {
	err := os.MkdirAll(s.Root, fs.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("could not create filesystem store root dir: %v\n", err))
	}
}

func (s *Store) Store(_ context.Context, name string, data []byte, opts ...func(*storage.StoreOpts)) error {
	s.once.Do(s.init)

	f, err := os.Create(filepath.Join(s.Root, name))
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}

	defer f.Close()

	_, err = io.Copy(f, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("could not copy data to file: %w", err)
	}

	return nil
}

func (s *Store) Open(_ context.Context, name string) (*storage.File, error) {
	s.once.Do(s.init)

	filename := filepath.Join(s.Root, name)

	stat, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("could not stat file: %w", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	// http.DetectContentType needs at most 512 bytes.
	firstBytes := make([]byte, 512)
	read, err := f.Read(firstBytes)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("could not read first 512 bytes from file: %w", err)
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("could not reset file reader after sniffing its content-type: %w", err)
	}

	// remove fill-up zero values which cause a wrong content type detection.
	firstBytes = firstBytes[:read]

	return &storage.File{
		ReadSeekCloser: f,
		Size:           stat.Size(),
		ContentType:    http.DetectContentType(firstBytes),
		ETag:           "",
		LastModified:   stat.ModTime(),
	}, nil
}

func (s *Store) Delete(_ context.Context, name string) error {
	s.once.Do(s.init)

	err := os.Remove(filepath.Join(s.Root, name))
	if err != nil {
		return fmt.Errorf("could not remove file: %w", err)
	}
	return nil
}
