// friends.go — liste d'amis PAR PROFIL JOUEUR (clé : xuid).
//
// Remplace l'ancien réglage global `app_settings.friend_gamertags`, qui était UNE
// liste pour toute l'instance : elle pilotait `is_with_friends` dans toutes les
// player DBs et n'était lisible que par un admin (GET /settings). Les amis sont
// désormais attachés au joueur consulté, pas à l'instance.
//
// Types purs (0 DB, 0 HTTP) : la persistance vit dans platform/friendstore.
package domain

import "strings"

// MaxFriendGamertags borne le nombre d'amis par profil.
const MaxFriendGamertags = 50

// MaxFriendGamertagLen borne la longueur d'un gamertag ami (même borne que la
// création de profil, handlers/setup.go).
const MaxFriendGamertagLen = 50

// PlayerFriends est la liste d'amis d'un profil joueur.
type PlayerFriends struct {
	XUID string `json:"xuid"`
	// Gamertags : jamais nil côté API (contrat OpenAPI non nullable) — les
	// constructeurs de ce fichier garantissent une slice vide plutôt que nil.
	Gamertags []string `json:"gamertags" nullable:"false"`
	UpdatedAt string   `json:"updated_at,omitempty"`
	// CanEdit : vrai si l'appelant peut écrire cette liste (propriétaire direct
	// du profil ou admin). Seule source du mode lecture seule côté front — le
	// front n'interprète jamais un 403 pour décider de l'affichage.
	CanEdit bool `json:"can_edit"`
}

// PutFriendsRequest est le corps de PUT /players/{slug}/friends : remplacement
// complet de la liste.
type PutFriendsRequest struct {
	Gamertags []string `json:"gamertags" nullable:"false"`
}

// NormalizeFriendGamertags applique les règles de la liste d'amis : trim,
// suppression des entrées vides, dédoublonnage insensible à la casse (première
// graphie rencontrée conservée) et exclusion du gamertag du profil lui-même
// (`ownGamertag`, ignoré s'il est vide). L'ordre d'entrée est préservé.
// Retourne toujours une slice non nil.
func NormalizeFriendGamertags(in []string, ownGamertag string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	own := strings.ToLower(strings.TrimSpace(ownGamertag))
	for _, gt := range in {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		key := strings.ToLower(gt)
		if seen[key] || (own != "" && key == own) {
			continue
		}
		seen[key] = true
		out = append(out, gt)
	}
	return out
}

// ValidateFriendGamertags vérifie les bornes d'une liste DÉJÀ normalisée.
// Retourne le premier motif de refus (vide = liste valide), destiné à être
// remonté tel quel dans un 400.
func ValidateFriendGamertags(list []string) string {
	if len(list) > MaxFriendGamertags {
		return "too_many"
	}
	for _, gt := range list {
		if len(gt) > MaxFriendGamertagLen {
			return "gamertag_too_long"
		}
	}
	return ""
}
