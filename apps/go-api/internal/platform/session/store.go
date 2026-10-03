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

	// corruptSessionTTL : un fichier de session .json au JSON illisible n'est purgé
	// que s'il est plus vieux que ce délai. Sœur d'orphanTmpTTL, même raison : un
	// fichier corrompu FRAIS peut être un write en vol d'un AUTRE process (doublon
	// `air` — os.ReadFile peut observer un état transitoire sous Windows), pas une
	// corruption durable. On le conserve pour ne jamais déconnecter une session
	// vivante sur une corruption illusoire ; seule une corruption qui persiste
	// au-delà du délai est réellement supprimée.
	corruptSessionTTL = time.Hour

	// touchPersistInterval : Touch ne réécrit une session INCHANGÉE que si le last_seen_at
	// que porte son fichier a plus de cet âge (plan perf 2026-09-23, D3.5). Avant : une
	// écriture de fichier sous le verrou du store à CHAQUE requête, pollings compris (état
	// des lieux perf, C8). Le TTL glissant (basé sur last_seen_at) reste exact à cet
	// intervalle près.
	touchPersistInterval = 5 * time.Minute
)

// Store gère la persistance des sessions dans des fichiers JSON.
type Store struct {
	dir    string
	ttl    time.Duration
	secret []byte
	// mu sérialise les accès disque INTRA-process : Save prend le Lock exclusif,
	// Load prend le RLock partagé, et PurgeExpired prend le Lock exclusif PAR FICHIER
	// (critical section courte : lecture + décision + suppression d'un fichier de
	// session, cf. purgeSessionFileLocked) afin de ne pas bloquer les requêtes
	// pendant tout le scan du répertoire. Deux raisons :
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
	// now : horloge du store (time.Now ; WithClock la remplace dans les tests).
	now func() time.Time
	// marksMu protège marks. Verrou DISTINCT de mu : Delete, qui oublie une marque, est
	// appelé par Load sous mu.RLock. Quand les deux sont tenus : mu d'abord, marksMu ensuite.
	marksMu sync.Mutex
	// marks : pour chaque session, ce que porte son fichier d'après la DERNIÈRE écriture de
	// ce process — en mémoire seulement, jamais dans le JSON. Marque absente (session jamais
	// écrite par ce process, redémarrage) : Touch écrit.
	marks map[string]persistMark
}

// persistMark : ce que porte le fichier d'une session d'après la dernière écriture de ce
// process (plan perf 2026-09-23, D3.5).
type persistMark struct {
	// LastPersistedAt : le last_seen_at que porte cette écriture — et non l'heure de
	// l'écriture. Un Save de handler qui réécrit le last_seen_at ancien d'une session
	// chargée ne doit pas dispenser Touch de rafraîchir la présence sur disque.
	LastPersistedAt time.Time
	// content : empreinte du contenu hors last_seen_at (contentDigest).
	content [sha256.Size]byte
}

// StoreOption règle un Store à sa construction.
type StoreOption func(*Store)

// WithClock remplace l'horloge du Store (défaut : time.Now). Sert aux tests du throttle
// de Touch, qui avancent le temps sans dormir.
func WithClock(now func() time.Time) StoreOption {
	return func(s *Store) { s.now = now }
}

// NewStore crée un Store. Le répertoire sera créé s'il n'existe pas.
func NewStore(dir string, ttl time.Duration, secret string, opts ...StoreOption) *Store {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		// Non fatal — les writes échoueront proprement — mais on trace : sans
		// répertoire, toute session devient non persistable (login cassé). Pas de
		// ctx au montage ; module auto-détecté = "session" → logs/session.log.
		slog.Error("session: création du répertoire de sessions échouée", "dir", dir, "err", err)
	}
	s := &Store{
		dir:    dir,
		ttl:    ttl,
		secret: []byte(secret),
		now:    time.Now,
		marks:  make(map[string]persistMark),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// New crée une nouvelle session avec un ID UUID v4.
func (s *Store) New() *domain.SessionData {
	now := s.now().Unix()
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
// Save écrit toujours ; seul Touch s'abstient quand l'écriture serait inutile.
func (s *Store) Save(sess *domain.SessionData) error {
	content, err := contentDigest(sess)
	if err != nil {
		return err
	}
	return s.save(sess, content)
}

// save écrit le fichier (cf. Save) puis retient la marque de cette écriture.
func (s *Store) save(sess *domain.SessionData, content [sha256.Size]byte) error {
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
	s.remember(sess.SessionID, persistMark{LastPersistedAt: time.Unix(sess.LastSeenAt, 0), content: content})
	return nil
}

// Touch rafraîchit last_seen_at et persiste la session — sauf si l'écriture est inutile
// (plan perf 2026-09-23, D3.5) : même contenu (hors last_seen_at) que la dernière écriture
// de ce process, dont le last_seen_at a moins de touchPersistInterval. Une session MODIFIÉE
// (login, préférence, flux OAuth, joueur courant...) est donc toujours écrite ; une session
// inchangée l'est au plus une fois par intervalle, ce qui borne le retard du TTL glissant.
func (s *Store) Touch(sess *domain.SessionData) error {
	content, err := contentDigest(sess)
	if err != nil {
		return err
	}
	now := s.now()
	sess.LastSeenAt = now.Unix()
	if s.persistedRecently(sess.SessionID, content, now) {
		return nil
	}
	return s.save(sess, content)
}

// Delete supprime le fichier de session (et oublie la marque de sa dernière écriture).
func (s *Store) Delete(sessionID string) error {
	s.forget(sessionID)
	err := os.Remove(s.path(sessionID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PurgeExpired supprime les sessions expirées. Retourne le nombre supprimé.
//
// Tourne au boot puis toutes les 6h (startSessionPurgeLoop). Le scan du répertoire
// (os.ReadDir) est hors verrou, mais chaque fichier de session .json est traité sous
// le Lock exclusif s.mu, pris et relâché PAR FICHIER (critical section courte, cf.
// purgeSessionFileLocked) : sans ce verrou, sous Windows, l'os.ReadFile de la purge
// concurrent d'un Save pouvait faire échouer le os.Rename du Save (sharing violation)
// — exactement la course que s.mu ferme (Save=Lock, Load=RLock). Le verrouillage par
// fichier évite de bloquer les requêtes pendant tout le scan.
// Oublie enfin les marques d'écriture périmées (pruneMarks, plan perf 2026-09-23).
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
		if s.purgeSessionFileLocked(e.Name()) {
			removed++
		}
	}
	s.pruneMarks()
	return removed
}

// purgeSessionFileLocked traite UN fichier de session .json sous le Lock exclusif
// s.mu (le même que Save/Load), pris et relâché ici pour garder la critical section
// courte. Retourne true si le fichier a été supprimé. Invariants anti-déconnexion
// d'une session VIVANTE :
//   - Erreur de lecture hors ErrNotExist : WARN + skip, JAMAIS de suppression — une
//     sharing violation transitoire d'un Save concurrent (doublon `air`) ne doit pas
//     déconnecter en permanence une session potentiellement vivante.
//   - JSON illisible : suppression UNIQUEMENT si le fichier est durablement corrompu
//     (mtime plus vieux que corruptSessionTTL) ; un corrompu FRAIS peut être un write
//     en vol d'un autre process → skip + WARN.
//   - JSON valide : suppression si et seulement si la session a dépassé son TTL.
func (s *Store) purgeSessionFileLocked(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false // disparu entre le scan et ici (Load sur expiry / Delete) — RAS
		}
		slog.Warn("session: PurgeExpired — lecture d'un fichier de session échouée, fichier conservé",
			"file", name, "err", err)
		return false
	}
	var sess domain.SessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		info, ierr := os.Stat(path)
		if ierr != nil || time.Since(info.ModTime()) <= corruptSessionTTL {
			slog.Warn("session: PurgeExpired — fichier au JSON illisible mais récent, conservé (write en vol ?)",
				"file", name, "err", err)
			return false
		}
		slog.Warn("session: PurgeExpired — fichier au JSON durablement corrompu, supprimé",
			"file", name, "err", err)
		return s.removeSessionFile(path, name)
	}
	if !s.isExpired(&sess) {
		return false
	}
	return s.removeSessionFile(path, name)
}

// removeSessionFile supprime un fichier de session et retourne true si la
// suppression a effectivement eu lieu. Un ErrNotExist (déjà supprimé par un autre
// chemin) n'est pas une erreur mais ne compte pas comme suppression de CE purge ;
// toute autre erreur est tracée (anti swallowed-error) sans faire échouer la boucle.
// Appelé sous s.mu.Lock() (via purgeSessionFileLocked).
func (s *Store) removeSessionFile(path, name string) bool {
	if err := os.Remove(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("session: PurgeExpired — suppression d'un fichier de session échouée",
				"file", name, "err", err)
		}
		return false
	}
	return true
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
	return s.now().Sub(lastSeen) > s.ttl
}

// contentDigest : empreinte SHA-256 du JSON de la session, last_seen_at exclu — seul champ
// que Touch change à chaque requête. Copie superficielle : la session de l'appelant n'est
// pas modifiée.
func contentDigest(sess *domain.SessionData) ([sha256.Size]byte, error) {
	c := *sess
	c.LastSeenAt = 0
	data, err := json.Marshal(&c)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("session digest: %w", err)
	}
	return sha256.Sum256(data), nil
}

// persistedRecently : le fichier de la session porte déjà ce contenu, avec un last_seen_at
// de moins de touchPersistInterval (d'après la dernière écriture de ce process). Une
// horloge revenue en arrière (âge négatif) force l'écriture.
func (s *Store) persistedRecently(sessionID string, content [sha256.Size]byte, now time.Time) bool {
	s.marksMu.Lock()
	mark, ok := s.marks[sessionID]
	s.marksMu.Unlock()
	if !ok || mark.content != content {
		return false
	}
	age := now.Sub(mark.LastPersistedAt)
	return age >= 0 && age < touchPersistInterval
}

// remember retient la marque de la dernière écriture d'une session.
func (s *Store) remember(sessionID string, mark persistMark) {
	s.marksMu.Lock()
	s.marks[sessionID] = mark
	s.marksMu.Unlock()
}

// forget oublie la marque d'une session supprimée.
func (s *Store) forget(sessionID string) {
	s.marksMu.Lock()
	delete(s.marks, sessionID)
	s.marksMu.Unlock()
}

// pruneMarks oublie les marques dont le last_seen_at a dépassé le TTL : leur session est
// expirée (ou supprimée hors de ce process) ; les garder ferait croître la table sans fin.
func (s *Store) pruneMarks() {
	limit := s.now().Add(-s.ttl)
	s.marksMu.Lock()
	defer s.marksMu.Unlock()
	for id, mark := range s.marks {
		if mark.LastPersistedAt.Before(limit) {
			delete(s.marks, id)
		}
	}
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
