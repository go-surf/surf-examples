package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type EntryStore struct {
	mu      sync.Mutex
	entries []Entry
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

func (es *EntryStore) ByID(ctx context.Context, entryID string) (*Entry, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	for _, entry := range es.entries {
		if entry.ID == entryID {
			return &entry, nil
		}
	}
	return nil, ErrNotFound
}

func (es *EntryStore) Latest(ctx context.Context, limit int) ([]Entry, error) {
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

func (es *EntryStore) Create(ctx context.Context, title, content string) (*Entry, error) {
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

func (es *EntryStore) Delete(ctx context.Context, entryID string) error {
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
