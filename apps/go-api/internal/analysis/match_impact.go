// Package analysis — match_impact.go : détection des badges d'impact sur un match.
//
// Portage de src/analysis/match_impact.py.
// Identifie les moments clés : premier sang, kill final, touriste (0 kill), 1ère victime.
package analysis

// ImpactEvent représente un event de kill avec son contexte pour la détection des badges.
type ImpactEvent struct {
	TimeMS     int64
	KillerXUID string
	VictimXUID string
}

// ImpactBadge représente un badge attribué à un joueur sur ce match.
type ImpactBadge struct {
	BadgeKey   string // identifiant technique (first_blood, finisher, tourist, first_victim)
	BadgeFR    string // libellé français
	PlayerXUID string
}

// MatchImpactInput données minimales nécessaires pour ComputeSingleMatchImpact.
type MatchImpactInput struct {
	KillEvents []ImpactEvent
	MyXUID     string
	MyKills    int
}

// ComputeSingleMatchImpact calcule les badges d'impact pour ce match.
// Retourne la liste des badges gagnés par le joueur identifié par MyXUID.
func ComputeSingleMatchImpact(input MatchImpactInput) []ImpactBadge {
	var badges []ImpactBadge

	if input.MyKills == 0 {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "tourist",
			BadgeFR:    "Touriste",
			PlayerXUID: input.MyXUID,
		})
	}

	if len(input.KillEvents) == 0 {
		return badges
	}

	// Tri par TimeMS pour trouver premier et dernier kill
	firstEv := input.KillEvents[0]
	lastEv := input.KillEvents[len(input.KillEvents)-1]
	for _, ev := range input.KillEvents {
		if ev.TimeMS < firstEv.TimeMS {
			firstEv = ev
		}
		if ev.TimeMS > lastEv.TimeMS {
			lastEv = ev
		}
	}

	if firstEv.KillerXUID == input.MyXUID {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "first_blood",
			BadgeFR:    "Premier sang",
			PlayerXUID: input.MyXUID,
		})
	}

	if firstEv.VictimXUID == input.MyXUID {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "first_victim",
			BadgeFR:    "Première victime",
			PlayerXUID: input.MyXUID,
		})
	}

	if lastEv.KillerXUID == input.MyXUID {
		badges = append(badges, ImpactBadge{
			BadgeKey:   "finisher",
			BadgeFR:    "Finisseur",
			PlayerXUID: input.MyXUID,
		})
	}

	return badges
}
