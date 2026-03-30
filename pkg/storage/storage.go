// Package storage provides SQLite-based fingerprint storage for element tracking.
// It stores element fingerprints keyed by URL + selector identifier, enabling
// adaptive element relocation when website structures change.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// ElementDict represents a serialized element fingerprint.
type ElementDict struct {
	Tag        string            `json:"tag"`
	Text       string            `json:"text"`
	Attributes map[string]string `json:"attributes"`
	Path       []string          `json:"path"`
	Parent     *ParentDict       `json:"parent,omitempty"`
	Siblings   []string          `json:"siblings"`
	Children   []string          `json:"children"`
}

// ParentDict represents parent element metadata for fingerprinting.
type ParentDict struct {
	Tag        string            `json:"tag"`
	Attributes map[string]string `json:"attributes"`
	Text       string            `json:"text"`
}

// Store is a thread-safe SQLite storage for element fingerprints.
type Store struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// New creates a new Store with the given database file path.
// Pass ":memory:" for an in-memory database.
func New(dbPath string) (*Store, error) {
	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = dbPath + "?_journal_mode=WAL&_busy_timeout=5000"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}

	// Enable WAL mode for better concurrency
	if dbPath != ":memory:" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("set WAL mode: %w", err)
		}
	}

	// Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS storage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		identifier TEXT NOT NULL,
		element_data TEXT NOT NULL,
		UNIQUE(url, identifier)
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &Store{db: db, path: dbPath}, nil
}

// Save stores an element fingerprint for the given URL and identifier.
func (s *Store) Save(rawURL, identifier string, elem *ElementDict) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := normalizeURL(rawURL)
	data, err := json.Marshal(elem)
	if err != nil {
		return fmt.Errorf("marshal element: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO storage (url, identifier, element_data) VALUES (?, ?, ?)`,
		normalized, identifier, string(data),
	)
	if err != nil {
		return fmt.Errorf("save element: %w", err)
	}
	return nil
}

// Load retrieves an element fingerprint for the given URL and identifier.
// Returns nil if not found.
func (s *Store) Load(rawURL, identifier string) (*ElementDict, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalized := normalizeURL(rawURL)
	var data string
	err := s.db.QueryRow(
		`SELECT element_data FROM storage WHERE url = ? AND identifier = ?`,
		normalized, identifier,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load element: %w", err)
	}

	var elem ElementDict
	if err := json.Unmarshal([]byte(data), &elem); err != nil {
		return nil, fmt.Errorf("unmarshal element: %w", err)
	}
	return &elem, nil
}

// Delete removes a stored fingerprint.
func (s *Store) Delete(rawURL, identifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := normalizeURL(rawURL)
	_, err := s.db.Exec(
		`DELETE FROM storage WHERE url = ? AND identifier = ?`,
		normalized, identifier,
	)
	if err != nil {
		return fmt.Errorf("delete element: %w", err)
	}
	return nil
}

// List returns all identifiers stored for a given URL.
func (s *Store) List(rawURL string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalized := normalizeURL(rawURL)
	rows, err := s.db.Query(
		`SELECT identifier FROM storage WHERE url = ?`,
		normalized,
	)
	if err != nil {
		return nil, fmt.Errorf("list elements: %w", err)
	}
	defer rows.Close()

	var identifiers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan identifier: %w", err)
		}
		identifiers = append(identifiers, id)
	}
	return identifiers, rows.Err()
}

// Count returns the total number of stored fingerprints.
func (s *Store) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM storage`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count elements: %w", err)
	}
	return count, nil
}

// Clear removes all stored fingerprints.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM storage`)
	if err != nil {
		return fmt.Errorf("clear storage: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// normalizeURL extracts the base domain from a URL for consistent keying.
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := u.Hostname()
	// Remove www. prefix
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return rawURL
	}
	return host + u.Path
}
