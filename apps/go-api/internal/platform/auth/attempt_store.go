// Package auth — attempt_store.go : registre en mémoire des tentatives Device Code Flow.
//
// AttemptStore est thread-safe. Il garantit le single-flight par session :
// si une tentative "pending" existe déjà pour une session, elle est renvoyée.
//
// Cycle de vie d'une tentative :
//
//	pending → authorized (MS a validé) → provisioned (gamertag récupéré)
//	pending → failed (erreur MSAL ou Halo)
//	pending → expired (expirée avant autorisation)
package auth

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Statuts possibles d'une Attempt (champ Status).
const (
	AttemptStatusPending     = "pending"
	AttemptStatusAuthorized  = "authorized"
	AttemptStatusProvisioned = "provisioned"
	AttemptStatusFailed      = "failed"
	AttemptStatusExpired     = "expired"
)

// Attempt représente une tentative Device Code Flow en cours.
type Attempt struct {
	AttemptID       string
	UserCode        string
	VerificationURI string
	ExpiresInSec    int
	StartedAt       time.Time
	SessionID       string

	// Champs mis à jour par la goroutine de polling
	Status      string // pending | authorized | provisioned | failed | expired
	Gamertag    string
	XUID        string
	ErrorCode   string
	ErrorDetail string

	// Référence interne au Device Code Flow (jamais exposé)
	DevFlow DeviceFlow

	// HaloTokens contient les tokens Halo obtenus après ExchangeAccessToken.
	// Transférés en session lors du prochain GetDeviceFlowStatus.
	SpartanToken   string
	ClearanceToken string
}

// AttemptStore est le registre en mémoire des tentatives Device Code Flow.
// Toutes les méthodes sont thread-safe.
type AttemptStore struct {
	mu        sync.RWMutex
	attempts  map[string]*Attempt // clé : attempt_id
	bySession map[string]string   // session_id → attempt_id (pour single-flight)
}

// NewAttemptStore crée un AttemptStore vide.
func NewAttemptStore() *AttemptStore {
	return &AttemptStore{
		attempts:  make(map[string]*Attempt),
		bySession: make(map[string]string),
	}
}

// GetOrCreate retourne la tentative existante pour une session (si pending),
// ou crée une nouvelle entrée placeholder. Retourne (attempt, isNew).
// L'appelant doit remplir DevFlow si isNew est true.
func (s *AttemptStore) GetOrCreate(sessionID string) (*Attempt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.bySession[sessionID]; ok {
		if a := s.attempts[id]; a != nil && a.Status == "pending" {
			return a, false
		}
	}

	a := &Attempt{
		AttemptID: uuid.New().String(),
		SessionID: sessionID,
		StartedAt: time.Now(),
		Status:    "pending",
	}
	s.attempts[a.AttemptID] = a
	s.bySession[sessionID] = a.AttemptID
	return a, true
}

// Get retourne une tentative par ID si elle appartient à la session donnée.
// Retourne nil si introuvable ou si la session ne correspond pas.
func (s *AttemptStore) Get(attemptID, sessionID string) *Attempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.attempts[attemptID]
	if !ok || a.SessionID != sessionID {
		return nil
	}
	return a
}

// Update permet de modifier une tentative de façon thread-safe.
func (s *AttemptStore) Update(attemptID string, fn func(*Attempt)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.attempts[attemptID]; ok {
		fn(a)
	}
}

// Snapshot retourne une copie d'une tentative (safe pour lecture sans mutex).
func (s *AttemptStore) Snapshot(attemptID string) *Attempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.attempts[attemptID]
	if !ok {
		return nil
	}
	copy := *a
	return &copy
}
