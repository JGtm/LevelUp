// Package session — touch.go : persistance de fin de requête (Touch, TouchAfter) et
// marques d'écriture qui évitent les réécritures inutiles.
package session

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/domain"
)

// Fingerprint : empreinte du contenu d'une session (last_seen_at exclu), prise à son
// chargement pour que TouchAfter sache si la requête l'a modifiée.
type Fingerprint struct {
	digest [sha256.Size]byte
	ok     bool
}

// FingerprintOf prend l'empreinte de sess. Une empreinte impossible à calculer rend une
// Fingerprint vide, que TouchAfter traite comme une session modifiée (écriture entière).
func FingerprintOf(sess *domain.SessionData) Fingerprint {
	digest, err := contentDigest(sess)
	return Fingerprint{digest: digest, ok: err == nil}
}

// TouchAfter termine une requête. Une session que la requête a MODIFIÉE (empreinte
// différente de loaded) est écrite entière, comme par Touch. Une session INCHANGÉE ne voit
// que son last_seen_at rafraîchi sur le fichier relu sous verrou : la copie chargée au début
// de la requête n'est jamais réécrite, sans quoi elle écraserait ce qu'une requête
// concurrente a écrit entre-temps (joueur courant choisi pendant que les requêtes de la
// page précédente sont encore en vol, titre, locale...).
func (s *Store) TouchAfter(sess *domain.SessionData, loaded Fingerprint) error {
	content, err := contentDigest(sess)
	if err != nil {
		return err
	}
	if !loaded.ok || content != loaded.digest {
		return s.Touch(sess)
	}
	now := s.now()
	sess.LastSeenAt = now.Unix()
	return s.refreshPresence(sess.SessionID, now)
}

// refreshPresence rafraîchit le last_seen_at du fichier d'une session sans toucher à son
// contenu, au plus une fois par touchPersistInterval. Un fichier disparu (déconnexion,
// purge) n'est pas recréé.
func (s *Store) refreshPresence(sessionID string, now time.Time) error {
	if s.writtenRecently(sessionID, now, nil) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path(sessionID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("session presence read: %w", err)
	}
	var cur domain.SessionData
	if err := json.Unmarshal(data, &cur); err != nil {
		return fmt.Errorf("session presence decode: %w", err)
	}
	cur.LastSeenAt = now.Unix()
	content, err := contentDigest(&cur)
	if err != nil {
		return err
	}
	return s.writeLocked(&cur, content)
}

// Touch rafraîchit last_seen_at et persiste la session — sauf si l'écriture est inutile
// (plan perf 2026-09-23, D3.5) : même contenu (hors last_seen_at) que la dernière écriture
// de ce process, dont le last_seen_at a moins de touchPersistInterval. Une session MODIFIÉE
// (login, préférence, flux OAuth, joueur courant...) est donc toujours écrite ; une session
// inchangée l'est au plus une fois par intervalle, ce qui borne le retard du TTL glissant.
// Touch écrit la copie de l'appelant : en fin de requête, TouchAfter le remplace.
func (s *Store) Touch(sess *domain.SessionData) error {
	content, err := contentDigest(sess)
	if err != nil {
		return err
	}
	now := s.now()
	sess.LastSeenAt = now.Unix()
	if s.writtenRecently(sess.SessionID, now, &content) {
		return nil
	}
	return s.save(sess, content)
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

// writtenRecently : d'après la dernière écriture de ce process, le fichier de la session
// porte un last_seen_at de moins de touchPersistInterval — et, si content est fourni, ce
// contenu-là. Une horloge revenue en arrière (âge négatif) force l'écriture.
func (s *Store) writtenRecently(sessionID string, now time.Time, content *[sha256.Size]byte) bool {
	s.marksMu.Lock()
	mark, ok := s.marks[sessionID]
	s.marksMu.Unlock()
	if !ok || (content != nil && mark.content != *content) {
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
	s.pruneRevokedLocked()
}
