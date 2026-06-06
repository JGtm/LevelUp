// Package authz porte la logique pure d'autorisation d'accès aux joueurs
// (multi-utilisateur strict, ADR 0024). Aucune I/O, aucune dépendance DuckDB :
// la résolution de l'utilisateur courant passe par l'interface UserLookup,
// fournie par le caller (userstore).
//
// Règle de propriété : un utilisateur possède le profil joueur dont le `xuid`
// est égal à son xuid lié. Les admins accèdent à tout. En mode demo ou
// auth_mode ∉ {password, xbox}, l'enforcement est désactivé (accès ouvert).
package authz

import (
	"strings"

	"levelup/go-api/internal/domain"
)

// Modes d'authentification reconnus pour l'enforcement.
const (
	AuthModeNone     = "none"
	AuthModePassword = "password"
	AuthModeXbox     = "xbox"
)

// UserLookup résout un utilisateur enregistré. Satisfait par *userstore.Store.
type UserLookup interface {
	Get(username string) (*domain.User, error)
	GetByXUID(xuid string) (*domain.User, error)
}

// Enforced indique si le contrôle de propriété des joueurs s'applique.
// Faux en mode demo ou quand l'auth n'est pas activée (none / vide) : l'instance
// est alors mono-utilisateur ou publique, et tous les profils restent accessibles.
func Enforced(demoMode bool, authMode string) bool {
	if demoMode {
		return false
	}
	return authMode == AuthModePassword || authMode == AuthModeXbox
}

// CanAccessPlayer décide si `user` peut accéder au profil identifié par
// `profileXUID`. Quand `enforced` est faux, l'accès est toujours accordé. Un
// `user` nil (non authentifié ou non lié à une identité Halo) ne possède rien.
//
// Règles (de la plus permissive à la plus stricte) :
//   - admin → accès à tout ;
//   - propriétaire → son propre xuid ;
//   - membre de la famille → accès aux autres profils de la famille (switch BDD
//     entre amis/famille). `familyXUIDs` est l'ensemble des xuids du groupe
//     famille (FriendGamertags résolus). L'accès famille exige que l'utilisateur
//     ET le profil demandé en fassent partie — un étranger (xuid hors famille)
//     ne peut donc pas consulter le parc.
//
// familyXUIDs nil/vide → comportement strict d'origine (propriétaire only).
func CanAccessPlayer(enforced bool, user *domain.User, profileXUID string, familyXUIDs map[string]bool) bool {
	if !enforced {
		return true
	}
	if user == nil {
		return false
	}
	if user.Role == domain.RoleAdmin {
		return true
	}
	if user.XUID == "" {
		return false
	}
	if user.XUID == profileXUID {
		return true
	}
	// Accès famille : les deux parties appartiennent au même groupe.
	return familyXUIDs[user.XUID] && familyXUIDs[profileXUID]
}

// ResolveFamilyXUIDs construit l'ensemble des xuids du groupe famille à partir
// des gamertags amis (FriendGamertags, settings) et de la liste des profils
// connus (db_profiles.json). La correspondance gamertag→xuid est insensible à
// la casse. Retourne nil si aucun ami n'est configuré → CanAccessPlayer retombe
// alors sur le comportement strict (propriétaire only).
func ResolveFamilyXUIDs(friendGamertags []string, players []domain.PlayerSummary) map[string]bool {
	if len(friendGamertags) == 0 {
		return nil
	}
	wanted := make(map[string]bool, len(friendGamertags))
	for _, gt := range friendGamertags {
		if gt != "" {
			wanted[strings.ToLower(gt)] = true
		}
	}
	out := make(map[string]bool, len(wanted))
	for _, p := range players {
		if p.XUID != "" && wanted[strings.ToLower(p.Gamertag)] {
			out[p.XUID] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CurrentUser résout l'utilisateur authentifié derrière une session, ou nil si
// aucun. Priorité au login local (sess.Username → lookup.Get), puis à l'identité
// Halo liée (sess.LinkedHaloIdentity.XUID → lookup.GetByXUID). Si l'identité Halo
// est liée en session mais sans compte persisté, un utilisateur minimal est
// synthétisé (rôle standard, propriétaire de son propre xuid).
func CurrentUser(sess *domain.SessionData, lookup UserLookup) *domain.User {
	if sess == nil || lookup == nil {
		return nil
	}
	if sess.Username != nil && *sess.Username != "" {
		if u, err := lookup.Get(*sess.Username); err == nil && u != nil {
			return u
		}
	}
	if sess.LinkedHaloIdentity != nil && sess.LinkedHaloIdentity.XUID != "" {
		if u, err := lookup.GetByXUID(sess.LinkedHaloIdentity.XUID); err == nil && u != nil {
			return u
		}
		return &domain.User{
			XUID:     sess.LinkedHaloIdentity.XUID,
			Gamertag: sess.LinkedHaloIdentity.Gamertag,
			Role:     domain.RoleUser,
		}
	}
	return nil
}
