// Package squadimpact — LES RÔLES D'IMPACT DE L'ESCOUADE : ceux qu'un match attribue aux membres
// de l'escouade, leur barème, et leur cumul par soirée.
//
// Deux surfaces de la page Escouade les lisent : la matrice d'impact (un match par colonne) et
// les points d'impact par soirée. Les deux passent par RolesOfMatch et par le même barème
// (Weight) : un match ne peut pas donner un rôle à l'une et pas à l'autre.
//
// Pur : aucune ouverture de base, aucune horloge, aucune chaîne de langue.
package squadimpact

import (
	"levelup/go-api/internal/analysis"
)

// Les clés des rôles comptés (celles d'analysis.ComputeMatchImpactFull et du badge « Voleur »).
const (
	RoleFirstBlood      = "first_blood"
	RoleClutchFinisher  = "clutch_finisher"
	RoleLastCasualty    = "last_casualty"
	RoleLastGroupKill   = "last_group_kill"
	RoleFirstGroupDeath = "first_group_death"
	RoleSilentHero      = "silent_hero"
	RoleFalseBrother    = "false_brother"
	RoleKamikaze        = "kamikaze"
	RoleTopKiller       = "top_killer"
	RoleThief           = analysis.BadgeKeyThief
)

// roleOrder — l'ordre canonique des colonnes agrégées de la matrice d'impact.
var roleOrder = []string{
	RoleFirstBlood, RoleClutchFinisher, RoleLastCasualty, RoleLastGroupKill, RoleFirstGroupDeath,
	RoleSilentHero, RoleFalseBrother, RoleKamikaze, RoleThief, RoleTopKiller,
}

// stackOrder — l'ordre d'empilement du barème publié : gains du plus fort au plus faible, puis
// pertes de la plus forte à la plus faible ; à barème égal, l'ordre de lecture de la maquette
// validée (Finisseur avant Premier sang ; Première victime, Touriste, Kamikaze, Voleur).
var stackOrder = []string{
	RoleClutchFinisher, RoleFirstBlood, RoleSilentHero, RoleTopKiller,
	RoleLastCasualty, RoleFalseBrother, RoleFirstGroupDeath, RoleLastGroupKill, RoleKamikaze, RoleThief,
}

// weights — les points d'une occurrence de chaque rôle. Un rôle absent ne compte pas : le
// « Top Gun » d'analysis.ComputeMatchImpactFull est calculé mais hors barème.
var weights = map[string]float64{
	RoleClutchFinisher:  2.0,
	RoleFirstBlood:      2.0,
	RoleLastCasualty:    -2.0,
	RoleSilentHero:      1.5,
	RoleFalseBrother:    -1.5,
	RoleLastGroupKill:   -1.0,
	RoleFirstGroupDeath: -1.0,
	RoleKamikaze:        -1.0,
	RoleThief:           -1.0,
	RoleTopKiller:       1.0,
}

// RoleOrder rend l'ordre canonique des colonnes de la matrice (une copie).
func RoleOrder() []string { return append([]string(nil), roleOrder...) }

// Weight rend les points d'une occurrence du rôle ; faux si le rôle n'est pas compté.
func Weight(role string) (float64, bool) {
	w, ok := weights[role]
	return w, ok
}

// MatchInput — ce qu'un match apporte au calcul de ses rôles.
type MatchInput struct {
	// Events : les frags et morts horodatés du match (référentiel gameplay).
	Events []analysis.ImpactEvent
	// Allies : l'équipe alliée COMPLÈTE du joueur principal (pas seulement l'escouade) : les
	// rôles sont attribués à l'échelle de l'équipe, puis seuls ceux des membres sont gardés.
	Allies []analysis.ParticipantSnap
	// Thief : le badge « Voleur » du match, calculé sur l'escouade seule (nil = aucun vol).
	Thief *analysis.ImpactBadge
}

// Attribution — un rôle compté tombé sur un membre de l'escouade.
type Attribution struct {
	// Player : le nom de la ligne du membre (la valeur de squad pour son xuid).
	Player string
	Role   string
}

// RolesOfMatch rend les rôles comptés d'un match qui tombent sur un membre de l'escouade, dans
// l'ordre d'émission d'analysis.ComputeMatchImpactFull, le « Voleur » en dernier. squad associe
// le xuid de chaque membre au nom de sa ligne : l'appartenance se décide par xuid, jamais par
// nom. Un match sans allié ni événement n'attribue rien (même avec un « Voleur »).
func RolesOfMatch(in MatchInput, squad map[string]string) []Attribution {
	if len(in.Allies) == 0 && len(in.Events) == 0 {
		return nil
	}
	badges := analysis.ComputeMatchImpactFull(analysis.MatchImpactInput{
		Events: in.Events, Participants: in.Allies,
	})
	if in.Thief != nil {
		badges = append(badges, *in.Thief)
	}
	var out []Attribution
	for _, b := range badges {
		if _, scored := weights[b.BadgeKey]; !scored {
			continue
		}
		player, ok := squad[b.PlayerXUID]
		if !ok {
			continue
		}
		out = append(out, Attribution{Player: player, Role: b.BadgeKey})
	}
	return out
}
