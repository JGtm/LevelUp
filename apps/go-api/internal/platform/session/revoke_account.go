// Package session — revoke_account.go : fin de toutes les sessions d'un compte, sauf une
// (changement de mot de passe).
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
)

// RevokeAccount révoque (cf. Revoke) toutes les sessions dont le compte est `username`
// (casse ignorée), sauf `keepID`. Rend le nombre de sessions révoquées. Un nom vide ne
// désigne aucun compte : rien n'est révoqué.
//
// Le magasin n'a pas d'index par compte : le répertoire est parcouru, chaque fichier lu
// sous le verrou de lecture. Un fichier illisible n'est pas attribué au compte (laissé en
// place) ; une révocation qui échoue est rendue dans l'erreur jointe, les autres sont
// faites quand même.
func (s *Store) RevokeAccount(username, keepID string) (int, error) {
	if strings.TrimSpace(username) == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, fmt.Errorf("session revoke account: %w", err)
	}
	keep := sanitizeID(keepID)
	revoked := 0
	var errs []error
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if e.IsDir() || !ok || id == keep || !s.ownedBy(id, username) {
			continue
		}
		if err := s.revoke(id, ""); err != nil {
			errs = append(errs, err)
			continue
		}
		revoked++
	}
	return revoked, errors.Join(errs...)
}

// ownedBy dit si le fichier de session `id` porte le compte `username` (casse ignorée).
func (s *Store) ownedBy(id, username string) bool {
	s.mu.RLock()
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	s.mu.RUnlock()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("session: révocation du compte — fichier de session illisible, laissé en place",
				"file", id+".json", "err", err)
		}
		return false
	}
	var sess domain.SessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		slog.Warn("session: révocation du compte — fichier de session au JSON illisible, laissé en place",
			"file", id+".json", "err", err)
		return false
	}
	return sess.Username != nil && strings.EqualFold(*sess.Username, username)
}
