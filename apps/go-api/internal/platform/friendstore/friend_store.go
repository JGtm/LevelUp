// Package friendstore gère la persistance des listes d'amis PAR PROFIL JOUEUR.
//
// Fichier : data/global/player_friends.json (PathResolver.PlayerFriendsPath).
// Format : { "version": "1.0", "friends": { "<xuid>": { "gamertags": [...], "updated_at": "..." } } }
//
// Lisible sans session HTTP : les flux de fond (sync, recompute is_with_friends,
// CLI) résolvent la liste du joueur qu'ils traitent.
//
// Thread-safe via sync.RWMutex. Lecture/écriture atomique (write-to-temp + rename).
// Calqué sur platform/groupstore.GroupStore.
package friendstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
)

const fileVersion = "1.0"

// ErrMissingXUID : aucune liste d'amis n'existe sans profil joueur cible.
var ErrMissingXUID = errors.New("xuid requis")

// friendsEntry : la liste d'un joueur telle que persistée.
type friendsEntry struct {
	Gamertags []string `json:"gamertags"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// friendsFile représente le format JSON du fichier player_friends.json.
type friendsFile struct {
	Version string                  `json:"version"`
	Friends map[string]friendsEntry `json:"friends"`
}

// FriendStore gère la persistance des listes d'amis par joueur.
type FriendStore struct {
	mu   sync.RWMutex
	path string
}

// NewFriendStore crée un FriendStore pointant vers le fichier JSON donné.
func NewFriendStore(path string) *FriendStore {
	return &FriendStore{path: path}
}

// Path retourne le chemin du fichier géré (diagnostic, migration).
func (s *FriendStore) Path() string { return s.path }

// load lit le fichier JSON. Retourne un fichier vide si absent.
func (s *FriendStore) load() (*friendsFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &friendsFile{Version: fileVersion, Friends: make(map[string]friendsEntry)}, nil
		}
		return nil, fmt.Errorf("lecture player_friends.json : %w", err)
	}
	var f friendsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing player_friends.json : %w", err)
	}
	if f.Friends == nil {
		f.Friends = make(map[string]friendsEntry)
	}
	// Invariant de contrat : `gamertags` est non nullable dans l'OpenAPI généré.
	// Un fichier legacy (ou un `"gamertags": null`) donnerait une slice nil,
	// sérialisée `null` par encoding/json. Normalisation au SEUL point de lecture.
	for xuid, e := range f.Friends {
		if e.Gamertags == nil {
			e.Gamertags = []string{}
			f.Friends[xuid] = e
		}
	}
	return &f, nil
}

// save écrit le fichier JSON de manière atomique (write-to-temp + rename).
func (s *FriendStore) save(f *friendsFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("création répertoire : %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation player_friends.json : %w", err)
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

// Get retourne la liste d'amis du joueur. Joueur absent du fichier (ou xuid
// vide) → slice vide, JAMAIS une erreur : ne pas avoir d'amis est un état
// normal, pas une panne.
func (s *FriendStore) Get(xuid string) ([]string, error) {
	if xuid == "" {
		return []string{}, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	e, exists := f.Friends[xuid]
	if !exists || e.Gamertags == nil {
		return []string{}, nil
	}
	out := make([]string, len(e.Gamertags))
	copy(out, e.Gamertags)
	return out, nil
}

// UpdatedAt retourne l'horodatage de dernière écriture de la liste du joueur
// (vide si le joueur n'a pas d'entrée).
func (s *FriendStore) UpdatedAt(xuid string) (string, error) {
	if xuid == "" {
		return "", nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return "", err
	}
	return f.Friends[xuid].UpdatedAt, nil
}

// Set remplace la liste d'amis du joueur. `ownGamertag` (optionnel) est exclu de
// la liste : un joueur n'est pas son propre ami. Normalisation domain
// (trim, dédoublonnage insensible à la casse) appliquée ici pour que TOUS les
// écrivains (API, migration, CLI) partagent les mêmes règles.
func (s *FriendStore) Set(xuid, ownGamertag string, gamertags []string) ([]string, error) {
	if xuid == "" {
		return nil, ErrMissingXUID
	}
	normalized := domain.NormalizeFriendGamertags(gamertags, ownGamertag)

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	f.Version = fileVersion
	f.Friends[xuid] = friendsEntry{
		Gamertags: normalized,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.save(f); err != nil {
		return nil, err
	}
	return normalized, nil
}

// All retourne la table complète xuid → liste d'amis (copie défensive).
func (s *FriendStore) All() (map[string][]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(f.Friends))
	for xuid, e := range f.Friends {
		list := make([]string, len(e.Gamertags))
		copy(list, e.Gamertags)
		out[xuid] = list
	}
	return out, nil
}

// randomHex : suffixe du fichier temporaire (write-to-temp + rename).
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
