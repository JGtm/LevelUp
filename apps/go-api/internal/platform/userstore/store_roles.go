// Package userstore — store_roles.go : les rôles décidés sous le verrou du magasin.
//
// Le premier compte de l'instance devient administrateur : la décision est prise
// sous le verrou d'écriture, de deux inscriptions simultanées une seule l'obtient.
// L'instance garde toujours au moins un administrateur : ni suppression ni
// rétrogradation du dernier.
package userstore

import (
	"errors"

	"levelup/go-api/internal/domain"
)

var (
	// ErrNotFirstAccount : CreateFirstAdmin alors qu'un compte existe déjà.
	ErrNotFirstAccount = errors.New("userstore: not the first account")
	// ErrLastAdmin : l'opération retirerait le dernier administrateur.
	ErrLastAdmin = errors.New("userstore: last administrator cannot be removed or demoted")
)

// CreateFirstAdmin crée le PREMIER compte de l'instance, administrateur. Refus
// (ErrNotFirstAccount) si un compte existe déjà, vérifié sous le verrou d'écriture :
// l'appelant reprend alors le chemin d'une inscription ordinaire, avec ses gardes.
func (s *Store) CreateFirstAdmin(username, password string) (*domain.User, error) {
	return s.create(username, password, domain.RoleAdmin, true)
}

// SetRole change le rôle d'un utilisateur. Refus (ErrLastAdmin) de rétrograder le
// dernier administrateur.
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
	if role != domain.RoleAdmin && isLastAdmin(f, slug) {
		return ErrLastAdmin
	}

	user.Role = role
	f.Users[slug] = user

	return s.save(f)
}

// Delete supprime un utilisateur. Refus (ErrLastAdmin) de supprimer le dernier
// administrateur.
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
	if isLastAdmin(f, slug) {
		return ErrLastAdmin
	}

	delete(f.Users, slug)
	return s.save(f)
}

// isLastAdmin dit si le compte slug est administrateur et le seul. L'appelant tient
// s.mu.
func isLastAdmin(f *usersFile, slug string) bool {
	if f.Users[slug].Role != domain.RoleAdmin {
		return false
	}
	for other, u := range f.Users {
		if other != slug && u.Role == domain.RoleAdmin {
			return false
		}
	}
	return true
}
