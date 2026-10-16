package handlers

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/userstore"
)

// testAccounts : comptes des tests de garde admin, rôle par nom d'utilisateur. Le rôle
// admin se relit dans le magasin (authz.IsAdmin), jamais dans la session.
type testAccounts map[string]domain.UserRole

func (a testAccounts) Get(username string) (*domain.User, error) {
	if role, ok := a[username]; ok {
		return &domain.User{Username: username, Role: role}, nil
	}
	return nil, userstore.ErrUserNotFound
}

func (a testAccounts) GetByXUID(string) (*domain.User, error) {
	return nil, userstore.ErrUserNotFound
}

// gateAccounts : « bob » utilisateur, « boss » administrateur.
var gateAccounts = testAccounts{"bob": domain.RoleUser, "boss": domain.RoleAdmin}
