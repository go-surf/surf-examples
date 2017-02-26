package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-surf/surf/db"
	"github.com/go-surf/surf/db/sqlite3"
)

type EntryStore interface {
	ByID(context.Context, string) (*Entry, error)
	Latest(context.Context, int) ([]Entry, error)
	Create(context.Context, string, string) (*Entry, error)
	Delete(context.Context, string) error
}

type Entry struct {
	ID      string
	Title   string
	Content string
	Created time.Time
}

func (e *Entry) ContentSummary() string {
	if len(e.Content) < 200 {
		return e.Content
	}
	return e.Content[:200] + "[..]"
}

type MemoryEntryStore struct {
	mu      sync.Mutex
	entries []Entry
}

var _ EntryStore = (*MemoryEntryStore)(nil)

func (es *MemoryEntryStore) ByID(ctx context.Context, entryID string) (*Entry, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	for _, entry := range es.entries {
		if entry.ID == entryID {
			return &entry, nil
		}
	}
	return nil, ErrNotFound
}

func (es *MemoryEntryStore) Latest(ctx context.Context, limit int) ([]Entry, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if limit > len(es.entries) {
		limit = len(es.entries)
	}
	if limit == 0 {
		return nil, nil
	}
	return es.entries[:limit], nil
}

func (es *MemoryEntryStore) Create(ctx context.Context, title, content string) (*Entry, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("cannot generate id: %s", err)
	}

	entry := Entry{
		ID:      hex.EncodeToString(id),
		Title:   title,
		Content: content,
		Created: time.Now(),
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	es.entries = append(es.entries, entry)
	return &entry, nil
}

func (es *MemoryEntryStore) Delete(ctx context.Context, entryID string) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	for i, entry := range es.entries {
		if entry.ID == entryID {
			es.entries = append(es.entries[:i], es.entries[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

var ErrNotFound = errors.New("not found")

type SqliteEntryStore struct {
	db db.Database
}

var _ EntryStore = (*SqliteEntryStore)(nil)

func OpenSqliteEntryStore(dbpath string) (*SqliteEntryStore, error) {
	db, err := sqlite3.Connect(dbpath)
	if err != nil {
		return nil, err
	}
	return &SqliteEntryStore{db: db}, nil
}

func (es *SqliteEntryStore) Migrate(ctx context.Context) error {
	_, err := es.db.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS entries (
		id TEXT NOT NULL PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created DATETIME NOT NULL
	)
	`)
	return err
}

func (es *SqliteEntryStore) ByID(ctx context.Context, entryID string) (*Entry, error) {
	var entry Entry
	err := es.db.Get(ctx, &entry, `
		SELECT * FROM entries
		WHERE id = ?
	`, entryID)
	switch err {
	case nil:
		return &entry, nil
	case db.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (es *SqliteEntryStore) Latest(ctx context.Context, limit int) ([]Entry, error) {
	var entries []Entry
	err := es.db.Select(ctx, &entries, `
		SELECT * FROM entries
		ORDER BY created DESC LIMIT ?
	`, limit)
	return entries, err
}

func (es *SqliteEntryStore) Create(ctx context.Context, title, content string) (*Entry, error) {
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("cannot generate id: %s", err)
	}

	entry := Entry{
		ID:      hex.EncodeToString(id),
		Title:   title,
		Content: content,
		Created: time.Now(),
	}
	_, err := es.db.Exec(ctx, `
		INSERT INTO entries (id, title, content, created)
		VALUES (?, ?, ?, ?)
	`, entry.ID, entry.Title, entry.Content, entry.Created)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (es *SqliteEntryStore) Delete(ctx context.Context, entryID string) error {
	res, err := es.db.Exec(ctx, `
		DELETE FROM entries WHERE id = ?
	`, entryID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return ErrNotFound
	}
	return nil
}
