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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	// tmpSuffix est l'extension des fichiers temporaires d'écriture atomique.
	tmpSuffix = ".tmp"

	// orphanTmpTTL : un fichier .tmp plus vieux que ce délai est forcément un
	// orphelin (crash entre write et rename) — un rename normal dure des ms. On
	// garde une marge large pour ne jamais supprimer un .tmp d'un Save en vol.
	orphanTmpTTL = time.Hour
)

// Store gère la persistance des sessions dans des fichiers JSON.
type Store struct {
	dir    string
	ttl    time.Duration
	secret []byte
	// mu sérialise les accès disque INTRA-process : Save prend le Lock exclusif,
	// Load prend le RLock partagé. Deux raisons :
	//  1. Anti torn-read : un Load ne peut pas lire pendant un Save (indispensable
	//     sous Windows, où os.Rename échoue et os.ReadFile prend une sharing
	//     violation si un handle concurrent tient le fichier — le rename atomique
	//     seul ne protège pas la lecture concurrente intra-process sur Windows).
	//  2. Anti lost-update entre deux Save concurrents sur la même session
	//     (login/OAuth vs Touch de fin de requête).
	// La protection CROSS-process (doublon `air`) reste assurée par le rename
	// atomique de Save (rename(2)/MoveFileEx REPLACE_EXISTING) : un lecteur d'un
	// autre process voit toujours un fichier complet (l'ancien ou le nouveau).
	mu sync.RWMutex
}

// NewStore crée un Store. Le répertoire sera créé s'il n'existe pas.
func NewStore(dir string, ttl time.Duration, secret string) *Store {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		// Non fatal — les writes échoueront proprement — mais on trace : sans
		// répertoire, toute session devient non persistable (login cassé). Pas de
		// ctx au montage ; module auto-détecté = "session" → logs/session.log.
		slog.Error("session: création du répertoire de sessions échouée", "dir", dir, "err", err)
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

// Load charge une session depuis le fichier JSON. Retourne nil si absente,
// illisible ou expirée. Le ctx sert au traçage corrélé (event_id → logs/session.log) :
// un retour nil ANORMAL (IO/JSON, par opposition à un fichier simplement absent) est
// logué — c'était le point aveugle de la boucle /login (nil silencieux → session
// anonyme transitoire → éjection). Un fichier absent reste silencieux (cas nominal :
// session neuve ou expirée-supprimée).
func (s *Store) Load(ctx context.Context, sessionID string) *domain.SessionData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	path := s.path(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "session: lecture du fichier de session échouée", "err", err)
		}
		return nil
	}
	var sess domain.SessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		slog.WarnContext(ctx, "session: fichier de session illisible (JSON corrompu ?)", "err", err, "bytes", len(data))
		return nil
	}
	if s.isExpired(&sess) {
		_ = s.Delete(sessionID)
		return nil
	}
	return &sess
}

// Save persiste la session dans son fichier JSON de façon ATOMIQUE : écriture
// dans un fichier temporaire du même répertoire, puis os.Rename vers la cible.
// os.Rename est un remplacement atomique cross-plateforme (rename(2) sous Linux,
// MoveFileEx(MOVEFILE_REPLACE_EXISTING) sous Windows) → un Load concurrent voit
// TOUJOURS un fichier complet (l'ancien ou le nouveau), jamais un fichier tronqué.
// C'était la cause racine de la « boucle /login » : os.WriteFile (truncate+write
// non atomique) exposait un fichier vide/partiel aux Load concurrents déclenchés
// par la rafale refetchOnWindowFocus → session lue nil → anonyme transitoire.
func (s *Store) Save(sess *domain.SessionData) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.path(sess.SessionID)
	tmp, err := os.CreateTemp(s.dir, sanitizeID(sess.SessionID)+"-*"+tmpSuffix)
	if err != nil {
		return fmt.Errorf("session tmp create: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("session tmp write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("session tmp close: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("session rename: %w", err)
	}
	return nil
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
		// Répertoire illisible : le purge ne peut pas tourner (sessions expirées
		// non nettoyées → fuite disque). Pas de ctx (appelé depuis un ticker) ;
		// module auto = "session" → logs/session.log.
		slog.Error("session: PurgeExpired — lecture du répertoire échouée", "dir", s.dir, "err", err)
		return 0
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// Nettoyer les fichiers temporaires orphelins (crash entre write et
		// rename dans Save). On respecte orphanTmpTTL pour ne jamais supprimer
		// le .tmp d'un Save encore en vol. Non compté dans `removed` (ce ne sont
		// pas des sessions).
		if strings.HasSuffix(e.Name(), tmpSuffix) {
			if info, ierr := e.Info(); ierr == nil && time.Since(info.ModTime()) > orphanTmpTTL {
				_ = os.Remove(filepath.Join(s.dir, e.Name()))
			}
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
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
