// Package domain — group.go : types pour les groupes/familles (accès mutuel aux données).
//
// Un groupe est un ensemble nommé de membres qui partagent l'accès à leurs profils
// joueur respectifs (cf. authz : deux users ont accès mutuel s'ils partagent ≥1 groupe).
// Distinct de `friend_gamertags` (settings) qui ne pilote QUE l'affichage Escouade.
package domain

// GroupRole représente le rôle d'un membre dans un groupe.
type GroupRole string

const (
	GroupRoleOwner  GroupRole = "owner"
	GroupRoleMember GroupRole = "member"
)

// GroupMember représente un membre d'un groupe.
//
// Tag `enum` (V72-01 / H7) : le contrat OpenAPI est GÉNÉRÉ depuis ce type ; sans enum le
// front recevait `role: string` et perdait l'union `'owner' | 'member'`. Type de SORTIE
// uniquement (les corps entrants sont décodés à la main via RawBody) → aucun risque de
// 422 de validation Huma.
type GroupMember struct {
	XUID     string    `json:"xuid"`
	Gamertag string    `json:"gamertag"`
	Role     GroupRole `json:"role" enum:"owner,member"`
	JoinedAt string    `json:"joined_at"`
}

// Group représente un groupe/famille (accès mutuel aux données).
//
// `Members` est NON NULLABLE au contrat (`nullable:"false"`) : l'invariant est tenu à la
// source par groupstore (normalisation nil → slice vide à chaque lecture du fichier), ce
// qui garantit `[]` et jamais `null` sur le fil.
type Group struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	OwnerXUID string        `json:"owner_xuid"`
	Members   []GroupMember `json:"members" nullable:"false"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// HasMember retourne true si le xuid est membre du groupe.
func (g *Group) HasMember(xuid string) bool {
	if xuid == "" {
		return false
	}
	for _, m := range g.Members {
		if m.XUID == xuid {
			return true
		}
	}
	return false
}

// IsOwner retourne true si le xuid est le propriétaire du groupe.
func (g *Group) IsOwner(xuid string) bool {
	return xuid != "" && g.OwnerXUID == xuid
}

// CreateGroupRequest est le body de POST /groups.
type CreateGroupRequest struct {
	Name string `json:"name"`
}

// RenameGroupRequest est le body de PATCH /groups/{id}.
type RenameGroupRequest struct {
	Name string `json:"name"`
}
