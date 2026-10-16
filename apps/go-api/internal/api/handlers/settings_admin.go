// Package handlers — settings_admin.go : le rôle admin du demandeur d'un réglage
// réservé (verrou d'instance), relu dans le magasin d'utilisateurs.
package handlers

import "levelup/go-api/internal/authz"

// WithUserLookup câble le magasin d'utilisateurs : le rôle admin se lit dans le
// compte, jamais dans la session (un admin rétrogradé perd la main aussitôt).
func (h *SettingsHandler) WithUserLookup(users authz.UserLookup) *SettingsHandler {
	h.users = users
	return h
}
