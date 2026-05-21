// Package auth — watcher_attempt_store.go : tentative Device Code Flow dédiée au watcher.
//
// Contrairement à AttemptStore (par session), WatcherAttemptStore est global :
// il n'y a qu'un seul watcher, donc une seule tentative active à la fois.
package auth

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// WatcherAttempt représente une tentative d'auth Xbox XSTS pour le watcher.
type WatcherAttempt struct {
	AttemptID       string
	UserCode        string
	VerificationURI string
	ExpiresInSec    int
	StartedAt       time.Time

	// Champs mis à jour par la goroutine
	Status       string // pending | authorized | failed | expired
	Gamertag     string
	XUID         string
	XSTSToken    string
	XSTSUserHash string
	ErrorCode    string
	ErrorDetail  string

	// Référence interne au Device Code Flow (jamais exposé)
	DevFlow DeviceFlow
}

// WatcherAttemptStore est le registre de la tentative d'auth watcher.
// Thread-safe. Une seule tentative active à la fois.
type WatcherAttemptStore struct {
	mu      sync.RWMutex
	current *WatcherAttempt
}

// NewWatcherAttemptStore crée un store vide.
func NewWatcherAttemptStore() *WatcherAttemptStore {
	return &WatcherAttemptStore{}
}

// GetOrCreate retourne la tentative en cours (si "pending"), ou crée une nouvelle.
// Retourne (attempt, isNew).
func (s *WatcherAttemptStore) GetOrCreate() (*WatcherAttempt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && s.current.Status == AttemptStatusPending {
		return s.current, false
	}

	a := &WatcherAttempt{
		AttemptID: uuid.New().String(),
		StartedAt: time.Now(),
		Status:    AttemptStatusPending,
	}
	s.current = a
	return a, true
}

// Get retourne la tentative par ID, ou nil si introuvable.
func (s *WatcherAttemptStore) Get(attemptID string) *WatcherAttempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current != nil && s.current.AttemptID == attemptID {
		// retourner une copie
		copy := *s.current
		return &copy
	}
	return nil
}

// Update applique fn sur la tentative active (thread-safe).
func (s *WatcherAttemptStore) Update(attemptID string, fn func(*WatcherAttempt)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil && s.current.AttemptID == attemptID {
		fn(s.current)
	}
}

// Snapshot retourne une copie de la tentative (ou nil).
func (s *WatcherAttemptStore) Snapshot(attemptID string) *WatcherAttempt {
	return s.Get(attemptID)
}
