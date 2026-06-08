// Package domain — user.go : types pour l'authentification locale (username/password).
package domain

import "time"

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
	Username    string   `json:"username"`
	Role        UserRole `json:"role"`
	Gamertag    string   `json:"gamertag,omitempty"`
	CreatedAt   string   `json:"created_at"`
	LastLoginAt string   `json:"last_login_at,omitempty"`
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
