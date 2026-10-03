// Package auth — token_store.go : état du watcher RTA mono-user sur disque.
//
// Fichier : data/auth/watcher_tokens.json
//
// Structure :
//
//	{
//	  "access_token": "...",
//	  "xsts_token": "...",
//	  "xsts_user_hash": "...",
//	  "xsts_gamertag": "...",
//	  "xsts_xuid": "...",
//	  "xsts_expires_at": "2026-04-20T15:30:00Z",
//	  "oauth_expires_at": "2026-04-20T14:00:00Z"
//	}
//
// ADR 0023 Phase 5 (2026-08-25) : ce store ne porte PLUS de refresh_token. Il
// n'est plus une source de credentials — le refresh_token du tracker vit dans
// le MultiUserTokenStore (data/auth/watcher_tokens/{xuid}.json), source unique.
// La clé `refresh_token` des fichiers écrits avant cette date est ignorée au
// décodage et disparaît à la première réécriture.
package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StoredTokens représente les tokens persistés sur disque.
type StoredTokens struct {
	AccessToken    string    `json:"access_token"`
	XSTSToken      string    `json:"xsts_token"`
	XSTSUserHash   string    `json:"xsts_user_hash"`
	XSTSGamertag   string    `json:"xsts_gamertag"`
	XSTSXUID       string    `json:"xsts_xuid"`
	XSTSExpiresAt  time.Time `json:"xsts_expires_at"`
	OAuthExpiresAt time.Time `json:"oauth_expires_at"`
}

// IsXSTSValid retourne true si le token XSTS est encore valide (avec marge de sécurité).
func (t *StoredTokens) IsXSTSValid(margin time.Duration) bool {
	if t.XSTSToken == "" {
		return false
	}
	return time.Now().Add(margin).Before(t.XSTSExpiresAt)
}

// IsOAuthValid retourne true si l'access_token OAuth est encore valide (avec marge).
func (t *StoredTokens) IsOAuthValid(margin time.Duration) bool {
	if t.AccessToken == "" {
		return false
	}
	return time.Now().Add(margin).Before(t.OAuthExpiresAt)
}

// TokenStore persiste et lit les tokens sur disque (thread-safe).
type TokenStore struct {
	path string
	mu   sync.RWMutex
}

// NewTokenStore crée un token store au chemin donné.
// Le répertoire parent est créé si nécessaire.
func NewTokenStore(path string) *TokenStore {
	return &TokenStore{path: path}
}

// Path retourne le chemin du fichier token.
func (s *TokenStore) Path() string { return s.path }

// Load lit les tokens depuis le disque.
// Retourne un StoredTokens vide (pas d'erreur) si le fichier n'existe pas.
func (s *TokenStore) Load() (*StoredTokens, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		slog.Debug("token_store: fichier absent, tokens vides", "path", s.path)
		return &StoredTokens{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("token_store: lecture %s: %w", s.path, err)
	}

	var tokens StoredTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("token_store: décodage JSON %s: %w", s.path, err)
	}
	slog.Debug("token_store: tokens chargés",
		"path", s.path,
		"has_access", tokens.AccessToken != "",
		"xsts_valid", tokens.IsXSTSValid(0),
	)
	return &tokens, nil
}

// Save persiste les tokens sur disque (atomic write).
func (s *TokenStore) Save(tokens *StoredTokens) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("token_store: mkdir %s: %w", filepath.Dir(s.path), err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("token_store: encodage JSON: %w", err)
	}

	// Écriture atomique : tmp + rename
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("token_store: écriture tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("token_store: rename: %w", err)
	}

	slog.Debug("token_store: tokens sauvegardés", "path", s.path)
	return nil
}

// UpdateXSTS met à jour uniquement les champs XSTS dans le store.
// Utilise result.NotAfter comme date d'expiration réelle si disponible,
// sinon fallback sur time.Now().Add(fallbackTTL).
func (s *TokenStore) UpdateXSTS(result *XSTSResult, fallbackTTL time.Duration) error {
	tokens, err := s.Load()
	if err != nil {
		return err
	}
	tokens.XSTSToken = result.Token
	tokens.XSTSUserHash = result.UserHash
	tokens.XSTSGamertag = result.Gamertag
	tokens.XSTSXUID = result.XUID
	if !result.NotAfter.IsZero() {
		tokens.XSTSExpiresAt = result.NotAfter
	} else {
		tokens.XSTSExpiresAt = time.Now().Add(fallbackTTL)
	}
	return s.Save(tokens)
}

// UpdateOAuth met à jour l'access_token du watcher et sa date d'expiration.
// Ne persiste JAMAIS de refresh_token : celui-ci vit dans le MultiUserTokenStore
// (source unique ADR 0023).
func (s *TokenStore) UpdateOAuth(accessToken string, expiresIn time.Duration) error {
	tokens, err := s.Load()
	if err != nil {
		return err
	}
	tokens.AccessToken = accessToken
	tokens.OAuthExpiresAt = time.Now().Add(expiresIn)
	return s.Save(tokens)
}
