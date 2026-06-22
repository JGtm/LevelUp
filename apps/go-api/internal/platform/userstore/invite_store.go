// Package userstore — invite_store.go : gestion des codes d'invitation.
//
// Fichier : data/auth/invites.json
// Format : { "version": "1.0", "invites": { "<code>": { ... } } }
package userstore

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	inviteCodeLen     = 8
	inviteFileVersion = "1.0"
	// Alphabet sans caractères ambigus (0/O, 1/I/l).
	inviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // pragma: allowlist secret
)

// Erreurs de l'invite store.
var (
	ErrInviteNotFound = errors.New("code d'invitation introuvable")
	ErrInviteExpired  = errors.New("code d'invitation expiré")
	ErrInviteUsed     = errors.New("code d'invitation déjà utilisé")
)

// invitesFile représente le format JSON du fichier invites.json.
type invitesFile struct {
	Version string                       `json:"version"`
	Invites map[string]domain.InviteCode `json:"invites"`
}

// InviteStore gère la persistance des codes d'invitation.
type InviteStore struct {
	mu   sync.RWMutex
	path string
}

// NewInviteStore crée un InviteStore pointant vers le fichier JSON donné.
func NewInviteStore(path string) *InviteStore {
	return &InviteStore{path: path}
}

// load lit le fichier JSON.
func (s *InviteStore) load() (*invitesFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &invitesFile{Version: inviteFileVersion, Invites: make(map[string]domain.InviteCode)}, nil
		}
		return nil, fmt.Errorf("lecture invites.json : %w", err)
	}
	var f invitesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing invites.json : %w", err)
	}
	if f.Invites == nil {
		f.Invites = make(map[string]domain.InviteCode)
	}
	return &f, nil
}

// save écrit le fichier JSON de manière atomique.
func (s *InviteStore) save(f *invitesFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("création répertoire : %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation invites.json : %w", err)
	}
	tmp := s.path + ".tmp." + randomHex(4)
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("écriture tmp : %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename atomique : %w", err)
	}
	return nil
}

// Generate crée un nouveau code d'invitation. groupID (optionnel) désigne le
// groupe que l'invité rejoint via le flow "rejoindre un groupe" (login Xbox SSO) ;
// vide = invitation d'inscription legacy (mode password, sans groupe).
func (s *InviteStore) Generate(createdBy string, expiresInDays int, groupID string) (*domain.InviteCode, error) {
	if expiresInDays <= 0 {
		expiresInDays = 7
	}

	code, err := generateCode()
	if err != nil {
		return nil, fmt.Errorf("génération code : %w", err)
	}

	now := time.Now().UTC()
	invite := domain.InviteCode{
		Code:      code,
		CreatedBy: createdBy,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(time.Duration(expiresInDays) * 24 * time.Hour).Format(time.RFC3339),
		GroupID:   groupID,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	f.Invites[code] = invite
	if err := s.save(f); err != nil {
		return nil, err
	}
	return &invite, nil
}

// Validate vérifie qu'un code est valide (existe, non utilisé, non expiré).
func (s *InviteStore) Validate(code string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	invite, exists := f.Invites[code]
	if !exists {
		return ErrInviteNotFound
	}
	if invite.IsUsed() {
		return ErrInviteUsed
	}
	if invite.IsExpired() {
		return ErrInviteExpired
	}
	return nil
}

// Get retourne le code d'invitation (sans vérifier sa validité). Utile pour lire
// le GroupID associé. ErrInviteNotFound si le code n'existe pas.
func (s *InviteStore) Get(code string) (*domain.InviteCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	invite, exists := f.Invites[code]
	if !exists {
		return nil, ErrInviteNotFound
	}
	return &invite, nil
}

// Consume marque un code comme utilisé.
func (s *InviteStore) Consume(code, usedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	invite, exists := f.Invites[code]
	if !exists {
		return ErrInviteNotFound
	}
	if invite.IsUsed() {
		return ErrInviteUsed
	}
	if invite.IsExpired() {
		return ErrInviteExpired
	}

	now := time.Now().UTC().Format(time.RFC3339)
	invite.UsedBy = &usedBy
	invite.UsedAt = &now
	f.Invites[code] = invite

	return s.save(f)
}

// List retourne toutes les invitations.
func (s *InviteStore) List() ([]domain.AdminInviteSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	result := make([]domain.AdminInviteSummary, 0, len(f.Invites))
	for _, inv := range f.Invites {
		result = append(result, domain.AdminInviteSummary{
			Code:      inv.Code,
			CreatedBy: inv.CreatedBy,
			CreatedAt: inv.CreatedAt,
			ExpiresAt: inv.ExpiresAt,
			UsedBy:    inv.UsedBy,
			UsedAt:    inv.UsedAt,
			Valid:     inv.IsValid(),
		})
	}
	return result, nil
}

// Revoke supprime un code d'invitation.
func (s *InviteStore) Revoke(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	if _, exists := f.Invites[code]; !exists {
		return ErrInviteNotFound
	}

	delete(f.Invites, code)
	return s.save(f)
}

// generateCode crée un code aléatoire de inviteCodeLen caractères.
func generateCode() (string, error) {
	b := make([]byte, inviteCodeLen)
	alphabetLen := big.NewInt(int64(len(inviteAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		b[i] = inviteAlphabet[n.Int64()]
	}
	return string(b), nil
}
