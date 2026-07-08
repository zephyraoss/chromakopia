package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	path string

	mu      sync.RWMutex
	db      *sql.DB
	modTime time.Time
	size    int64
}

const dsnParams = "?mode=ro&_pragma=busy_timeout(10000)"

func Open(ctx context.Context, path string) (*Store, error) {
	db, fi, err := openFile(ctx, path)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, db: db, modTime: fi.ModTime(), size: fi.Size()}, nil
}

func openFile(ctx context.Context, path string) (*sql.DB, os.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("stat metadata db: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+path+dsnParams)
	if err != nil {
		return nil, nil, fmt.Errorf("open metadata db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ping metadata db: %w", err)
	}
	return db, fi, nil
}

func (s *Store) DB() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

func (s *Store) Path() string { return s.path }

func (s *Store) Reopen(ctx context.Context) error {
	db, fi, err := openFile(ctx, s.path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	old := s.db
	s.db = db
	s.modTime = fi.ModTime()
	s.size = fi.Size()
	s.mu.Unlock()
	if old != nil {
		go old.Close()
	}
	return nil
}

func (s *Store) Watch(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		fi, err := os.Stat(s.path)
		if err != nil {
			log.Printf("metadata db watch: %v", err)
			continue
		}
		s.mu.RLock()
		changed := !fi.ModTime().Equal(s.modTime) || fi.Size() != s.size
		s.mu.RUnlock()
		if !changed {
			continue
		}
		log.Printf("metadata db changed on disk, reopening %s", s.path)
		reopenCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = s.Reopen(reopenCtx)
		cancel()
		if err != nil {
			log.Printf("metadata db reopen failed (keeping previous handle): %v", err)
		}
	}
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
