// Package auth — multi_user_token_store.go : persistance multi-user des tokens RTA.
//
// Layout : data/auth/watcher_tokens/{xuid}.json — 1 fichier par utilisateur.
// Décision D4 (cf. SPRINT_XBOX_SSO §0bis / thought_log 2026-05-18) :
//   - Source unique des tokens RTA d'un user (pas de duplication dans sync_meta).
//   - Write-to-temp + os.Rename atomique → zéro contention sur writes parallèles.
//   - Permissions 0600 fichiers / 0700 répertoire.
//   - Au boot : LoadAll() scanne le dossier et reconstruit le map RAM.
//
// LEGACY : le watcher daemon historique utilise TokenStore (mono-user, fichier
// data/auth/watcher_tokens.json). MultiUserTokenStore est utilisé par le flow
// SSO Xbox PR 2.5a uniquement. Migration du watcher : différée à PR 2.5b.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UserTokens regroupe les tokens persistés pour un utilisateur Xbox SSO.
//
// Le `MSALCacheJSON` est le cache MSAL sérialisé : il contient le refresh_token
// Microsoft (que le SDK n'expose pas directement) et permet à `AcquireTokenSilent`
// de rafraîchir l'access_token plus tard.
type UserTokens struct {
	XUID           string    `json:"xuid"`
	Gamertag       string    `json:"gamertag"`
	XSTSToken      string    `json:"xsts_token"`
	XSTSUserHash   string    `json:"xsts_user_hash"`
	XSTSExpiresAt  time.Time `json:"xsts_expires_at"`
	AccessToken    string    `json:"access_token,omitempty"`
	OAuthExpiresAt time.Time `json:"oauth_expires_at,omitempty"`
	MSALCacheJSON  string    `json:"msal_cache_json,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IsXSTSValid retourne true si le XSTS est encore valide (avec marge).
func (u *UserTokens) IsXSTSValid(margin time.Duration) bool {
	if u.XSTSToken == "" {
		return false
	}
	return time.Now().Add(margin).Before(u.XSTSExpiresAt)
}

// AuthHeader retourne l'header XBL3.0 pour l'authentification RTA.
// Retourne "" si les champs nécessaires sont vides.
func (u *UserTokens) AuthHeader() string {
	if u.XSTSToken == "" || u.XSTSUserHash == "" {
		return ""
	}
	return fmt.Sprintf("XBL3.0 x=%s;%s", u.XSTSUserHash, u.XSTSToken)
}

// MultiUserTokenStore persiste les UserTokens d'un ensemble d'utilisateurs.
// Thread-safe. Chaque fichier `{xuid}.json` est écrit/lu de manière atomique.
type MultiUserTokenStore struct {
	dir string
	mu  sync.RWMutex
}

// ErrUserTokensNotFound est retourné par Load quand aucun fichier n'existe pour le xuid.
var ErrUserTokensNotFound = errors.New("multi_user_token_store: user tokens not found")

// NewMultiUserTokenStore crée un MultiUserTokenStore pointant vers le dossier donné.
// Le répertoire est créé avec perms 0700 si absent.
func NewMultiUserTokenStore(dir string) *MultiUserTokenStore {
	return &MultiUserTokenStore{dir: dir}
}

// Dir retourne le chemin du répertoire de stockage.
func (s *MultiUserTokenStore) Dir() string { return s.dir }

// xuidIsSafe vérifie qu'un XUID est utilisable comme nom de fichier sans risque.
// Les XUID Xbox Live sont des entiers décimaux ; on accepte aussi `-` (rare).
// Refuse tout caractère de path traversal (`.`, `/`, `\`).
func xuidIsSafe(xuid string) bool {
	if xuid == "" {
		return false
	}
	for _, r := range xuid {
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// pathFor retourne le chemin du fichier d'un XUID donné.
// Retourne "" si le XUID n'est pas safe (validation anti path-traversal).
func (s *MultiUserTokenStore) pathFor(xuid string) string {
	if !xuidIsSafe(xuid) {
		return ""
	}
	return filepath.Join(s.dir, xuid+".json")
}

// Upsert écrit ou remplace les tokens d'un utilisateur.
// Crée le répertoire si absent. Garantit CreatedAt à la première écriture
// et touche UpdatedAt à chaque écriture.
func (s *MultiUserTokenStore) Upsert(tokens *UserTokens) error {
	if tokens == nil {
		return fmt.Errorf("multi_user_token_store: tokens nil")
	}
	if !xuidIsSafe(tokens.XUID) {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", tokens.XUID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("multi_user_token_store: mkdir %s: %w", s.dir, err)
	}

	path := s.pathFor(tokens.XUID)
	if path == "" {
		return fmt.Errorf("multi_user_token_store: path résolu vide pour xuid=%q", tokens.XUID)
	}

	// Préserver CreatedAt si le fichier existe déjà.
	if existing, err := s.loadLocked(tokens.XUID); err == nil && !existing.CreatedAt.IsZero() {
		tokens.CreatedAt = existing.CreatedAt
	} else if tokens.CreatedAt.IsZero() {
		tokens.CreatedAt = time.Now().UTC()
	}
	tokens.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("multi_user_token_store: marshal: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("multi_user_token_store: écriture tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("multi_user_token_store: rename atomique: %w", err)
	}

	slog.Debug("multi_user_token_store: tokens persistés",
		"xuid", tokens.XUID, "gamertag", tokens.Gamertag,
		"xsts_valid", tokens.IsXSTSValid(0))
	return nil
}

// Load lit les tokens d'un utilisateur par XUID.
// Retourne ErrUserTokensNotFound si aucun fichier n'existe pour ce xuid.
func (s *MultiUserTokenStore) Load(xuid string) (*UserTokens, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadLocked(xuid)
}

// loadLocked est la version interne sans verrouillage (callers doivent tenir le mutex).
func (s *MultiUserTokenStore) loadLocked(xuid string) (*UserTokens, error) {
	path := s.pathFor(xuid)
	if path == "" {
		return nil, fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrUserTokensNotFound
		}
		return nil, fmt.Errorf("multi_user_token_store: lecture %s: %w", path, err)
	}
	var t UserTokens
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("multi_user_token_store: décodage %s: %w", path, err)
	}
	return &t, nil
}

// LoadAll scanne le répertoire et retourne tous les tokens persistés, indexés par XUID.
// Ignore silencieusement les fichiers mal formés (log warning), retourne map vide si
// le répertoire n'existe pas.
func (s *MultiUserTokenStore) LoadAll() (map[string]*UserTokens, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*UserTokens{}, nil
		}
		return nil, fmt.Errorf("multi_user_token_store: scan %s: %w", s.dir, err)
	}

	result := make(map[string]*UserTokens)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		xuid := strings.TrimSuffix(name, ".json")
		if !xuidIsSafe(xuid) {
			slog.Warn("multi_user_token_store: fichier au nom invalide, ignoré", "name", name)
			continue
		}
		t, err := s.loadLocked(xuid)
		if err != nil {
			slog.Warn("multi_user_token_store: lecture échouée, fichier ignoré", "xuid", xuid, "err", err)
			continue
		}
		result[xuid] = t
	}
	return result, nil
}

// Remove supprime les tokens d'un utilisateur (révocation).
// No-op si le fichier n'existe pas.
func (s *MultiUserTokenStore) Remove(xuid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.pathFor(xuid)
	if path == "" {
		return fmt.Errorf("multi_user_token_store: xuid invalide: %q", xuid)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("multi_user_token_store: remove %s: %w", path, err)
	}
	return nil
}
