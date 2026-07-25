// Package groupstore gère la persistance des groupes/familles (accès mutuel aux données).
//
// Fichier : data/auth/groups.json
// Format : { "version": "1.0", "groups": { "<id>": { ... } } }
//
// Thread-safe via sync.RWMutex. Lecture/écriture atomique (write-to-temp + rename).
// Calqué sur userstore.Store / userstore.InviteStore.
package groupstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	fileVersion = "1.0"
	maxNameLen  = 60
	idPrefix    = "grp_"
	idRandomLen = 8 // octets → 16 hex chars
)

// Erreurs du group store.
var (
	ErrGroupNotFound     = errors.New("groupe introuvable")
	ErrInvalidName       = errors.New("nom de groupe invalide (1-60 caractères)")
	ErrCannotRemoveOwner = errors.New("le propriétaire ne peut pas être retiré (supprimer le groupe à la place)")
	ErrMissingOwnerXUID  = errors.New("xuid propriétaire requis")
)

// groupsFile représente le format JSON du fichier groups.json.
type groupsFile struct {
	Version string                  `json:"version"`
	Groups  map[string]domain.Group `json:"groups"`
}

// GroupStore gère la persistance des groupes.
type GroupStore struct {
	mu   sync.RWMutex
	path string
}

// NewGroupStore crée un GroupStore pointant vers le fichier JSON donné.
func NewGroupStore(path string) *GroupStore {
	return &GroupStore{path: path}
}

// load lit le fichier JSON. Retourne un fichier vide si absent.
func (s *GroupStore) load() (*groupsFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &groupsFile{Version: fileVersion, Groups: make(map[string]domain.Group)}, nil
		}
		return nil, fmt.Errorf("lecture groups.json : %w", err)
	}
	var f groupsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing groups.json : %w", err)
	}
	if f.Groups == nil {
		f.Groups = make(map[string]domain.Group)
	}
	// Invariant de contrat (V72-01 / H7) : `Group.members` est non nullable dans
	// l'OpenAPI généré. Un fichier legacy (ou un `"members": null`) donnerait une slice
	// nil, sérialisée `null` par encoding/json. Normalisation au SEUL point de lecture.
	for id, g := range f.Groups {
		if g.Members == nil {
			g.Members = []domain.GroupMember{}
			f.Groups[id] = g
		}
	}
	return &f, nil
}

// save écrit le fichier JSON de manière atomique (write-to-temp + rename).
func (s *GroupStore) save(f *groupsFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("création répertoire : %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation groups.json : %w", err)
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

// Create crée un groupe avec le propriétaire comme premier membre (rôle owner).
func (s *GroupStore) Create(name, ownerXUID, ownerGamertag string) (*domain.Group, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLen {
		return nil, ErrInvalidName
	}
	if ownerXUID == "" {
		return nil, ErrMissingOwnerXUID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	g := domain.Group{
		ID:        idPrefix + randomHex(idRandomLen),
		Name:      name,
		OwnerXUID: ownerXUID,
		Members: []domain.GroupMember{{
			XUID:     ownerXUID,
			Gamertag: ownerGamertag,
			Role:     domain.GroupRoleOwner,
			JoinedAt: now,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}
	f.Groups[g.ID] = g

	if err := s.save(f); err != nil {
		return nil, err
	}
	return &g, nil
}

// Get retourne un groupe par id.
func (s *GroupStore) Get(id string) (*domain.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	g, exists := f.Groups[id]
	if !exists {
		return nil, ErrGroupNotFound
	}
	return &g, nil
}

// List retourne tous les groupes.
func (s *GroupStore) List() ([]domain.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]domain.Group, 0, len(f.Groups))
	for _, g := range f.Groups {
		out = append(out, g)
	}
	return out, nil
}

// ListForXUID retourne les groupes dont le xuid est membre.
func (s *GroupStore) ListForXUID(xuid string) ([]domain.Group, error) {
	if xuid == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	var out []domain.Group
	for _, g := range f.Groups {
		if g.HasMember(xuid) {
			out = append(out, g)
		}
	}
	return out, nil
}

// CoMemberXUIDs retourne l'union des xuids de tous les membres des groupes dont
// `xuid` fait partie (xuid inclus). Clé pour l'autorisation : c'est l'ensemble des
// profils auxquels l'utilisateur a accès via partage de groupe. Retourne nil si le
// user n'appartient à aucun groupe (→ authz retombe sur propriétaire-only strict).
func (s *GroupStore) CoMemberXUIDs(xuid string) (map[string]bool, error) {
	if xuid == "" {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool)
	for _, g := range f.Groups {
		if !g.HasMember(xuid) {
			continue
		}
		for _, m := range g.Members {
			if m.XUID != "" {
				out[m.XUID] = true
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// AddMember ajoute un membre au groupe. Idempotent : si le xuid est déjà membre,
// rafraîchit son gamertag et retourne sans erreur.
func (s *GroupStore) AddMember(id, xuid, gamertag string) error {
	if xuid == "" {
		return ErrMissingOwnerXUID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}
	g, exists := f.Groups[id]
	if !exists {
		return ErrGroupNotFound
	}
	for i, m := range g.Members {
		if m.XUID == xuid {
			if gamertag != "" && g.Members[i].Gamertag != gamertag {
				g.Members[i].Gamertag = gamertag
				g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				f.Groups[id] = g
				return s.save(f)
			}
			return nil // déjà membre, gamertag inchangé
		}
	}
	g.Members = append(g.Members, domain.GroupMember{
		XUID:     xuid,
		Gamertag: gamertag,
		Role:     domain.GroupRoleMember,
		JoinedAt: time.Now().UTC().Format(time.RFC3339),
	})
	g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	f.Groups[id] = g
	return s.save(f)
}

// RemoveMember retire un membre du groupe. Refuse de retirer le propriétaire
// (utiliser Delete à la place). No-op si le xuid n'est pas membre.
func (s *GroupStore) RemoveMember(id, xuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}
	g, exists := f.Groups[id]
	if !exists {
		return ErrGroupNotFound
	}
	if g.IsOwner(xuid) {
		return ErrCannotRemoveOwner
	}
	kept := g.Members[:0]
	removed := false
	for _, m := range g.Members {
		if m.XUID == xuid {
			removed = true
			continue
		}
		kept = append(kept, m)
	}
	if !removed {
		return nil
	}
	g.Members = kept
	g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	f.Groups[id] = g
	return s.save(f)
}

// Rename change le nom d'un groupe.
func (s *GroupStore) Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLen {
		return ErrInvalidName
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}
	g, exists := f.Groups[id]
	if !exists {
		return ErrGroupNotFound
	}
	g.Name = name
	g.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	f.Groups[id] = g
	return s.save(f)
}

// Delete supprime un groupe.
func (s *GroupStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := f.Groups[id]; !exists {
		return ErrGroupNotFound
	}
	delete(f.Groups, id)
	return s.save(f)
}

// randomHex génère n octets aléatoires encodés en hexadécimal.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
