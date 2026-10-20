// Package session — revoke.go : fin d'une session côté serveur (déconnexion) et
// renouvellement de son identifiant à la connexion (rotation).
//
// Une session révoquée est supprimée du disque ET son identifiant est retenu en
// mémoire un temps (revokedTTL) : une requête concurrente qui l'avait chargée avant
// la révocation ne peut plus la réécrire en fin de requête (Save refuse,
// ErrSessionRevoked), et le middleware de session répond à son cookie par le cookie
// du successeur (rotation) ou par un cookie expiré (déconnexion).
package session

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"levelup/go-api/internal/domain"
)

// ErrSessionRevoked : écriture refusée d'une session révoquée (déconnexion ou
// identifiant renouvelé).
var ErrSessionRevoked = errors.New("session: revoked")

const (
	// revokedTTL : durée pendant laquelle un identifiant révoqué reste connu. Couvre
	// toute requête concurrente encore en vol au moment de la révocation.
	revokedTTL = time.Hour
	// maxRevoked borne la table des identifiants révoqués ; au-delà, les plus
	// anciens sont oubliés.
	maxRevoked = 10_000
)

// revocation : moment de la révocation et identifiant qui succède (vide pour une
// déconnexion).
type revocation struct {
	at        time.Time
	successor string
}

// Revoke met fin à la session côté serveur : fichier supprimé, identifiant retenu
// comme révoqué sans successeur. Idempotent.
func (s *Store) Revoke(sessionID string) error {
	return s.revoke(sessionID, "")
}

// Rotate donne à la session un NOUVEL identifiant en gardant tout son état
// (navigation, identité), révoque l'ancien avec ce successeur, puis écrit la session
// sous le nouveau. Un identifiant planté avant la connexion ne vaut donc plus rien
// après. sess est modifiée en place : le cookie posé en fin de requête porte le
// nouvel identifiant. Rend l'ancien identifiant.
func (s *Store) Rotate(sess *domain.SessionData) (string, error) {
	oldID := sess.SessionID
	sess.SessionID = uuid.New().String()
	sess.CreatedAt = s.now().Unix()
	if err := s.revoke(oldID, sess.SessionID); err != nil {
		return oldID, fmt.Errorf("session rotate: %w", err)
	}
	if err := s.Save(sess); err != nil {
		return oldID, fmt.Errorf("session rotate: %w", err)
	}
	return oldID, nil
}

// Revoked dit si l'identifiant a été révoqué, et rend son successeur ("" pour une
// déconnexion).
func (s *Store) Revoked(sessionID string) (successor string, revoked bool) {
	s.marksMu.Lock()
	defer s.marksMu.Unlock()
	r, ok := s.revoked[sessionID]
	if !ok || s.now().Sub(r.at) > revokedTTL {
		return "", false
	}
	return r.successor, true
}

// revoke retient l'identifiant comme révoqué puis supprime le fichier, sous le verrou
// d'écriture : un Save concurrent voit la révocation avant d'écrire.
func (s *Store) revoke(sessionID, successor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marksMu.Lock()
	s.pruneRevokedLocked()
	s.revoked[sessionID] = revocation{at: s.now(), successor: successor}
	delete(s.marks, sessionID)
	s.marksMu.Unlock()
	if err := os.Remove(s.path(sessionID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("session revoke: %w", err)
	}
	return nil
}

// isRevoked : l'identifiant ne doit plus être écrit. L'appelant tient s.mu.
func (s *Store) isRevoked(sessionID string) bool {
	_, revoked := s.Revoked(sessionID)
	return revoked
}

// pruneRevokedLocked oublie les révocations périmées puis, si la table reste pleine,
// les plus anciennes. L'appelant tient s.marksMu.
func (s *Store) pruneRevokedLocked() {
	now := s.now()
	for id, r := range s.revoked {
		if now.Sub(r.at) > revokedTTL {
			delete(s.revoked, id)
		}
	}
	for len(s.revoked) >= maxRevoked {
		oldestID, oldest := "", now
		for id, r := range s.revoked {
			if !r.at.After(oldest) {
				oldestID, oldest = id, r.at
			}
		}
		delete(s.revoked, oldestID)
	}
}
