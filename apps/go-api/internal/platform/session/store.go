// Package session — store.go : gestion des sessions web avec fichiers JSON + cookies HMAC-SHA256.
//
// Architecture :
//   - SessionData est stocké dans data/sessions/<session_id>.json (côté serveur).
//   - Le cookie navigateur contient uniquement l'identifiant opaque signé HMAC-SHA256.
//   - Format du cookie : "<session_id>.<hex(HMAC-SHA256(secret, session_id))>"
//
// Sécurité :
//   - Cookie httpOnly, Secure en production, SameSite=Lax.
//   - Le session_id est un UUID v4 opaque — jamais devinable.
//   - Signature HMAC empêche toute falsification côté navigateur.
//   - TTL configurable (défaut : 7 jours, basé sur last_seen_at).
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"levelup/go-api/internal/domain"

	"github.com/google/uuid"
)

const (
	// CookieName est le nom du cookie de session.
	CookieName = "levelup_session"

	// DefaultTTL est la durée de vie par défaut d'une session (7 jours).
	DefaultTTL = 7 * 24 * time.Hour
)

// Store gère la persistance des sessions dans des fichiers JSON.
type Store struct {
	dir    string
	ttl    time.Duration
	secret []byte
}

// NewStore crée un Store. Le répertoire sera créé s'il n'existe pas.
func NewStore(dir string, ttl time.Duration, secret string) *Store {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		// Non fatal — les writes échoueront proprement.
		_ = err
	}
	return &Store{
		dir:    dir,
		ttl:    ttl,
		secret: []byte(secret),
	}
}

// New crée une nouvelle session avec un ID UUID v4.
func (s *Store) New() *domain.SessionData {
	now := time.Now().Unix()
	locale := "fr"
	return &domain.SessionData{
		SessionID:    uuid.New().String(),
		CreatedAt:    now,
		LastSeenAt:   now,
		Locale:       locale,
		HintsVisible: true,
		AuthReady:    false,
	}
}

// Load charge une session depuis le fichier JSON. Retourne nil si absente ou expirée.
func (s *Store) Load(sessionID string) *domain.SessionData {
	path := s.path(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var sess domain.SessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil
	}
	if s.isExpired(&sess) {
		_ = s.Delete(sessionID)
		return nil
	}
	return &sess
}

// Save persiste la session dans son fichier JSON.
func (s *Store) Save(sess *domain.SessionData) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}
	return os.WriteFile(s.path(sess.SessionID), data, 0o600)
}

// Touch met à jour last_seen_at et sauvegarde la session.
func (s *Store) Touch(sess *domain.SessionData) error {
	sess.LastSeenAt = time.Now().Unix()
	return s.Save(sess)
}

// Delete supprime le fichier de session.
func (s *Store) Delete(sessionID string) error {
	err := os.Remove(s.path(sessionID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PurgeExpired supprime les sessions expirées. Retourne le nombre supprimé.
func (s *Store) PurgeExpired() int {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			_ = os.Remove(path)
			removed++
			continue
		}
		var sess domain.SessionData
		if err := json.Unmarshal(data, &sess); err != nil || s.isExpired(&sess) {
			_ = os.Remove(path)
			removed++
		}
	}
	return removed
}

// =============================================================================
// Cookie signing / unsigning
// =============================================================================

// SignCookie retourne la valeur du cookie signée : "<sessionID>.<hex_sig>".
func (s *Store) SignCookie(sessionID string) string {
	sig := s.sign(sessionID)
	return sessionID + "." + sig
}

// UnsignCookie vérifie la signature et retourne le sessionID. Retourne "" si invalide.
func (s *Store) UnsignCookie(cookieValue string) string {
	dot := strings.LastIndex(cookieValue, ".")
	if dot < 0 {
		return ""
	}
	sessionID := cookieValue[:dot]
	gotSig := cookieValue[dot+1:]
	expectedSig := s.sign(sessionID)
	// Comparaison en temps constant pour éviter les timing attacks.
	if !hmac.Equal([]byte(gotSig), []byte(expectedSig)) {
		return ""
	}
	return sessionID
}

// =============================================================================
// Helpers internes
// =============================================================================

// sign calcule HMAC-SHA256(secret, sessionID) encodé en hex.
func (s *Store) sign(sessionID string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

// path retourne le chemin du fichier de session. Nettoie l'ID pour éviter le path traversal.
func (s *Store) path(sessionID string) string {
	safe := sanitizeID(sessionID)
	return filepath.Join(s.dir, safe+".json")
}

// isExpired retourne true si la session a dépassé le TTL (basé sur last_seen_at).
func (s *Store) isExpired(sess *domain.SessionData) bool {
	lastSeen := time.Unix(sess.LastSeenAt, 0)
	return time.Since(lastSeen) > s.ttl
}

// sanitizeID conserve uniquement les caractères alphanumériques et tirets.
func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
