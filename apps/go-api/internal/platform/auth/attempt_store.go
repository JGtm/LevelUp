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

// defaultAttemptTTL borne la durée de vie d'une tentative dans le store.
// Au-delà, la tentative est balayée : le device code Microsoft expire de toute
// façon vers ~15 min, donc une tentative plus vieille est inexploitable. Sans ce
// TTL le store fuyait (aucune purge) — cf. fix onboarding 2026-06-08. La
// récupération côté frontend (relance sur attempt_not_found) couvre le cas d'une
// tentative balayée pendant que l'utilisateur saisit encore son code.
const defaultAttemptTTL = 30 * time.Minute

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
	SpartanToken     string
	ClearanceToken   string
	SpartanExpiresAt time.Time // expiry réel du Spartan (ExpiresUtc), 0 = inconnu

	// --- PR 2.5a — SSO Xbox : capture pour persistance par-XUID ---
	// Ces champs ne sont JAMAIS exposés via la réponse HTTP (deviceFlowStatusResponse).
	// Ils sont consommés par XboxSSOLinkStrategy.OnAuthSuccess pour persister
	// les tokens RTA dans MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json).
	MicrosoftAccessToken string    // access_token MS brut (durée vie ~1h)
	OAuthRefreshToken    string    // refresh_token Microsoft brut (flow SISU natif)
	XSTSRTAToken         string    // XSTS audience http://xboxlive.com (RTA)
	XSTSRTAUserHash      string    // userhash associé au XSTS RTA
	XSTSRTAExpiresAt     time.Time // expiration du XSTS RTA (typiquement ~55min)
}

// AttemptStore est le registre en mémoire des tentatives Device Code Flow.
// Toutes les méthodes sont thread-safe.
type AttemptStore struct {
	mu        sync.RWMutex
	attempts  map[string]*Attempt // clé : attempt_id
	bySession map[string]string   // session_id → attempt_id (pour single-flight)
	ttl       time.Duration       // durée de vie max d'une tentative (purge lazy)
}

// NewAttemptStore crée un AttemptStore vide avec le TTL par défaut.
func NewAttemptStore() *AttemptStore {
	return NewAttemptStoreWithTTL(defaultAttemptTTL)
}

// NewAttemptStoreWithTTL crée un AttemptStore avec un TTL explicite (tuning/tests).
// Un ttl <= 0 retombe sur le défaut.
func NewAttemptStoreWithTTL(ttl time.Duration) *AttemptStore {
	if ttl <= 0 {
		ttl = defaultAttemptTTL
	}
	return &AttemptStore{
		attempts:  make(map[string]*Attempt),
		bySession: make(map[string]string),
		ttl:       ttl,
	}
}

// PurgeExpired supprime les tentatives plus vieilles que le TTL. Retourne le
// nombre supprimé. Appelée en lazy depuis GetOrCreate (pas de goroutine janitor à
// gérer), exposée pour les tests et un éventuel appel périodique.
func (s *AttemptStore) PurgeExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.purgeExpiredLocked()
}

// purgeExpiredLocked supprime les tentatives expirées. L'appelant DOIT tenir s.mu.
func (s *AttemptStore) purgeExpiredLocked() int {
	cutoff := time.Now().Add(-s.ttl)
	removed := 0
	for id, a := range s.attempts {
		if a.StartedAt.Before(cutoff) {
			delete(s.attempts, id)
			if s.bySession[a.SessionID] == id {
				delete(s.bySession, a.SessionID)
			}
			removed++
		}
	}
	return removed
}

// GetOrCreate retourne la tentative existante pour une session (si pending),
// ou crée une nouvelle entrée placeholder. Retourne (attempt, isNew).
// L'appelant doit remplir DevFlow si isNew est true.
func (s *AttemptStore) GetOrCreate(sessionID string) (*Attempt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Purge lazy : borne la mémoire et évite de renvoyer une tentative périmée.
	s.purgeExpiredLocked()

	if id, ok := s.bySession[sessionID]; ok {
		if a := s.attempts[id]; a != nil && a.Status == AttemptStatusPending {
			return a, false
		}
	}

	a := &Attempt{
		AttemptID: uuid.New().String(),
		SessionID: sessionID,
		StartedAt: time.Now(),
		Status:    AttemptStatusPending,
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
