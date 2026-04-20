// Package userstore gère la persistance des utilisateurs (JSON + bcrypt).
//
// Fichier : data/auth/users.json
// Format : { "version": "1.0", "users": { "<slug>": { ... } } }
//
// Thread-safe via sync.RWMutex. Lecture/écriture atomique (write-to-temp + rename).
package userstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"levelup/go-api/internal/domain"
)

const (
	bcryptCost     = 12
	fileVersion    = "1.0"
	minUsernameLen = 3
	maxUsernameLen = 30
	minPasswordLen = 8
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Erreurs du user store.
var (
	ErrUserNotFound       = errors.New("utilisateur introuvable")
	ErrUserAlreadyExists  = errors.New("nom d'utilisateur déjà pris")
	ErrInvalidUsername    = errors.New("nom d'utilisateur invalide (3-30 caractères alphanumériques, _, -)")
	ErrPasswordTooShort   = errors.New("mot de passe trop court (8 caractères minimum)")
	ErrInvalidCredentials = errors.New("identifiants incorrects")
)

// usersFile représente le format JSON du fichier users.json.
type usersFile struct {
	Version string                 `json:"version"`
	Users   map[string]domain.User `json:"users"`
}

// Store gère la persistance des utilisateurs.
type Store struct {
	mu   sync.RWMutex
	path string
}

// NewStore crée un Store pointant vers le fichier JSON donné.
// Le répertoire parent est créé automatiquement si absent.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// slugify normalise un username en clé de stockage.
func slugify(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// load lit le fichier JSON. Retourne un fichier vide si le fichier n'existe pas.
func (s *Store) load() (*usersFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &usersFile{Version: fileVersion, Users: make(map[string]domain.User)}, nil
		}
		return nil, fmt.Errorf("lecture users.json : %w", err)
	}
	var f usersFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing users.json : %w", err)
	}
	if f.Users == nil {
		f.Users = make(map[string]domain.User)
	}
	return &f, nil
}

// save écrit le fichier JSON de manière atomique (write-to-temp + rename).
func (s *Store) save(f *usersFile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("création répertoire : %w", err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("sérialisation users.json : %w", err)
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

// IsEmpty retourne true si aucun utilisateur n'est enregistré.
func (s *Store) IsEmpty() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, err := s.load()
	if err != nil {
		return false, err
	}
	return len(f.Users) == 0, nil
}

// Create crée un nouvel utilisateur. Retourne ErrUserAlreadyExists si le slug existe.
func (s *Store) Create(username, password string, role domain.UserRole) (*domain.User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if len(password) < minPasswordLen {
		return nil, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash bcrypt : %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	slug := slugify(username)
	if _, exists := f.Users[slug]; exists {
		return nil, ErrUserAlreadyExists
	}

	user := domain.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	f.Users[slug] = user

	if err := s.save(f); err != nil {
		return nil, err
	}
	return &user, nil
}

// Authenticate vérifie les identifiants et retourne l'utilisateur.
func (s *Store) Authenticate(username, password string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		// Timing-safe : toujours comparer pour éviter une attaque par timing.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$000000000000000000000000000000000000000000000000000000"), []byte(password))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

// Get retourne un utilisateur par username.
func (s *Store) Get(username string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

// List retourne tous les utilisateurs (sans les hash de mot de passe).
func (s *Store) List() ([]domain.AdminUserSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, err := s.load()
	if err != nil {
		return nil, err
	}

	result := make([]domain.AdminUserSummary, 0, len(f.Users))
	for _, u := range f.Users {
		result = append(result, domain.AdminUserSummary{
			Username:    u.Username,
			Role:        u.Role,
			Gamertag:    u.Gamertag,
			CreatedAt:   u.CreatedAt,
			LastLoginAt: u.LastLoginAt,
		})
	}
	return result, nil
}

// LinkIdentity associe un gamertag et un XUID à un utilisateur.
func (s *Store) LinkIdentity(username, gamertag, xuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		return ErrUserNotFound
	}

	user.Gamertag = gamertag
	user.XUID = xuid
	f.Users[slug] = user

	return s.save(f)
}

// UpdateLastLogin met à jour le timestamp de dernière connexion.
func (s *Store) UpdateLastLogin(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		return ErrUserNotFound
	}

	user.LastLoginAt = time.Now().UTC().Format(time.RFC3339)
	f.Users[slug] = user

	return s.save(f)
}

// ResetPassword remplace le mot de passe d'un utilisateur.
func (s *Store) ResetPassword(username, newPassword string) error {
	if len(newPassword) < minPasswordLen {
		return ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash bcrypt : %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		return ErrUserNotFound
	}

	user.PasswordHash = string(hash)
	f.Users[slug] = user

	return s.save(f)
}

// SetRole change le rôle d'un utilisateur.
func (s *Store) SetRole(username string, role domain.UserRole) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	slug := slugify(username)
	user, exists := f.Users[slug]
	if !exists {
		return ErrUserNotFound
	}

	user.Role = role
	f.Users[slug] = user

	return s.save(f)
}

// Delete supprime un utilisateur.
func (s *Store) Delete(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.load()
	if err != nil {
		return err
	}

	slug := slugify(username)
	if _, exists := f.Users[slug]; !exists {
		return ErrUserNotFound
	}

	delete(f.Users, slug)
	return s.save(f)
}

// validateUsername vérifie les contraintes sur le nom d'utilisateur.
func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return ErrInvalidUsername
	}
	if !usernameRe.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

// randomHex génère n octets aléatoires encodés en hexadécimal.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
