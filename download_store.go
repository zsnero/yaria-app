package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var downloadBucket = []byte("downloads")

// DownloadRecord represents a persisted download entry.
type DownloadRecord struct {
	ID        string  `json:"id"`
	URL       string  `json:"url"`
	Title     string  `json:"title"`
	Thumbnail string  `json:"thumbnail,omitempty"`
	Status    string  `json:"status"`
	Percent   float64 `json:"percent"`
	Error     string  `json:"error,omitempty"`
	StartedAt string  `json:"started_at"`
	FilePath  string  `json:"file_path,omitempty"`
}

// DownloadStore provides persistent storage for download history using BoltDB.
type DownloadStore struct {
	db *bolt.DB
}

// NewDownloadStore opens (or creates) the BoltDB database at ~/.yaria/downloads.db.
func NewDownloadStore() (*DownloadStore, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".yaria")
	os.MkdirAll(dir, 0755)
	dbPath := filepath.Join(dir, "downloads.db")

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(downloadBucket)
		return err
	})

	return &DownloadStore{db: db}, nil
}

// Save persists a download record to the store.
func (s *DownloadStore) Save(record DownloadRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(downloadBucket)
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}
		return b.Put([]byte(record.ID), data)
	})
}

// Get retrieves a single download record by ID.
func (s *DownloadStore) Get(id string) (*DownloadRecord, error) {
	var record DownloadRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(downloadBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("not found")
		}
		return json.Unmarshal(data, &record)
	})
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetAll returns all stored download records.
func (s *DownloadStore) GetAll() []DownloadRecord {
	var records []DownloadRecord
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(downloadBucket)
		b.ForEach(func(k, v []byte) error {
			var r DownloadRecord
			json.Unmarshal(v, &r)
			records = append(records, r)
			return nil
		})
		return nil
	})
	return records
}

// Delete removes a download record from the store.
func (s *DownloadStore) Delete(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(downloadBucket).Delete([]byte(id))
	})
}

// Close closes the BoltDB database.
func (s *DownloadStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
