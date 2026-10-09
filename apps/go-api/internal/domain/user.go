// Package domain — user.go : types pour l'authentification locale (username/password).
package domain

import (
	"net/url"
	"time"
)

// UserRole représente le rôle d'un utilisateur.
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

// User représente un utilisateur enregistré dans le user store.
type User struct {
	Username     string   `json:"username"`
	PasswordHash string   `json:"password_hash"`
	Role         UserRole `json:"role"`
	Gamertag     string   `json:"gamertag,omitempty"`
	XUID         string   `json:"xuid,omitempty"`
	CreatedAt    string   `json:"created_at"`
	LastLoginAt  string   `json:"last_login_at,omitempty"`
	// ProvisionGrant : code de l'invitation qui a CRÉÉ ce compte, tant qu'il n'a
	// pas servi. Il porte un droit à USAGE UNIQUE de créer SON profil joueur,
	// même sur instance verrouillée (D3) — sans quoi un invité atterrit sur le
	// Setup et prend un 403 instance_locked, donc reste coincé.
	//
	// Le droit est porté par le COMPTE, pas par la session : il survit à une
	// déconnexion entre le login SSO et le Setup. Vidé après la création du
	// profil (SetProvisionGrant(username, "")).
	ProvisionGrant string `json:"provision_grant,omitempty"`
}

// LoginRequest est le body de POST /auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest est le body de POST /auth/register.
type RegisterRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code,omitempty"`
}

// LoginResponse est la réponse de POST /auth/login.
type LoginResponse struct {
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	Gamertag string   `json:"gamertag,omitempty"`
}

// SetPasswordRequest est le body de POST /auth/password (self-service, PR-C).
// L'utilisateur connecté définit/change son propre mot de passe (opt-in).
type SetPasswordRequest struct {
	Password string `json:"password"`
}

// RegisterResponse est la réponse de POST /auth/register.
type RegisterResponse struct {
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
}

// InviteCode représente un code d'invitation.
type InviteCode struct {
	Code      string  `json:"code"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	UsedBy    *string `json:"used_by"`
	UsedAt    *string `json:"used_at"`
	ExpiresAt string  `json:"expires_at"`
	// GroupID : groupe que l'invité rejoint après login Xbox SSO. Vide =
	// invitation SANS groupe : elle crée le compte et donne le droit de créer son
	// profil joueur (D3), sans rattacher l'invité à qui que ce soit.
	GroupID string `json:"group_id,omitempty"`
	// JoinURL : lien relatif à présenter à l'invité (`/join?invite=CODE`). Rendu
	// par le serveur pour que le front n'ait pas à réassembler le chemin — il n'y
	// préfixe que son origine.
	JoinURL string `json:"join_url,omitempty"`
}

// InviteJoinURL rend le lien relatif d'acceptation d'une invitation. Source
// unique du chemin `/join?invite=` côté serveur.
func InviteJoinURL(code string) string {
	return "/join?invite=" + url.QueryEscape(code)
}

// IsExpired retourne true si le code a dépassé sa date d'expiration.
func (ic *InviteCode) IsExpired() bool {
	t, err := time.Parse(time.RFC3339, ic.ExpiresAt)
	if err != nil {
		return true
	}
	return time.Now().After(t)
}

// IsUsed retourne true si le code a déjà été consommé.
func (ic *InviteCode) IsUsed() bool {
	return ic.UsedBy != nil
}

// IsValid retourne true si le code est utilisable.
func (ic *InviteCode) IsValid() bool {
	return !ic.IsExpired() && !ic.IsUsed()
}

// AdminUserSummary est le résumé d'un utilisateur pour le panel admin.
type AdminUserSummary struct {
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	Gamertag string   `json:"gamertag,omitempty"`
	// XUID : identité Xbox liée au compte, vide tant qu'aucun SSO ni
	// LinkIdentity ne l'a posée. C'est la SEULE clé qui relie ce compte aux
	// autres registres (profil, credentials, suivi live) — ADR 0035 D1 ; sans
	// elle, l'annuaire ne peut pas rattacher un compte à son profil.
	XUID        string `json:"xuid,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}

// AdminInviteSummary est le résumé d'une invitation pour le panel admin.
type AdminInviteSummary struct {
	Code      string  `json:"code"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt string  `json:"expires_at"`
	UsedBy    *string `json:"used_by"`
	UsedAt    *string `json:"used_at"`
	Valid     bool    `json:"valid"`
}
