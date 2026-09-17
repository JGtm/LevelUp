// Package authz porte la logique pure d'autorisation d'accès aux joueurs
// (multi-utilisateur strict, ADR 0029). Aucune I/O, aucune dépendance DuckDB :
// la résolution de l'utilisateur courant passe par l'interface UserLookup,
// fournie par le caller (userstore).
//
// Règle de propriété : un utilisateur possède le profil joueur dont le `xuid`
// est égal à son xuid lié. Les admins accèdent à tout. En mode demo ou
// auth_mode ∉ {password, xbox}, l'enforcement est désactivé (accès ouvert).
package authz

import (
	"log/slog"

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
//   - membre de groupe → accès aux profils partageant ≥1 groupe avec lui.
//     `accessibleXUIDs` est l'ensemble des xuids co-membres de groupe du user
//     courant (xuid du user inclus, cf. groupstore.CoMemberXUIDs). L'accès exige
//     que le profil demandé en fasse partie — un étranger (xuid hors groupes)
//     ne peut donc pas consulter le parc.
//
// accessibleXUIDs nil/vide → comportement strict d'origine (propriétaire only).
func CanAccessPlayer(enforced bool, user *domain.User, profileXUID string, accessibleXUIDs map[string]bool) bool {
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
	// Accès groupe : le user et le profil partagent au moins un groupe. Comme
	// l'ensemble co-membres inclut le xuid du user lui-même, tester le profil suffit.
	return accessibleXUIDs[profileXUID]
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

// InstanceLocked résout le verrou « instance fermée » (lockdown) en un SEUL
// point de décision (ADR 0035, D5). Le verrou effectif est le OU de deux
// sources : `envLocked` (LEVELUP_INSTANCE_LOCKED, verrou forcé au boot,
// immuable) et la clé `instance_locked` d'app_settings.json (mutable à chaud
// via PATCH /settings), lue par le callback `load`.
//
// Le callback est fourni par le caller : le package authz reste PUR (il ne peut
// pas importer platform/settings ni config sans créer un cycle et perdre sa
// testabilité). `load` nil ⇒ seule la source env compte.
//
// Repli sur erreur : settings illisibles ⇒ WARN puis `false` (non verrouillé).
// C'est le comportement historique de server_apiv1.go, conservé tel quel — un
// app_settings.json corrompu ne doit pas fermer l'instance à son administrateur.
// LOGUE AVANT DE DÉGRADER (CLAUDE.md règle 3).
//
// Trois copies de ce calcul existaient le 2026-09-15 (handlers/setup.go,
// api/server_apiv1.go, service/bootstrap_service.go) : garde-rail
// internal/archlint/no_bare_instance_lock_read_test.go.
func InstanceLocked(envLocked bool, load func() (bool, error)) bool {
	if envLocked {
		return true
	}
	if load == nil {
		return false
	}
	locked, err := load()
	if err != nil {
		slog.Warn("instance_locked: settings illisibles, repli sur non verrouillé", "err", err)
		return false
	}
	return locked
}
