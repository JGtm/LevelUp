package sessionusage

// usage_outcomes.go — LES TROIS ISSUES D'UN OBJET PRIS, au grain SESSION (étape E3
// du PLAN_EQUIPEMENT_GACHIS_2026-09-09, décisions P1, P2, P6, P7).
//
// # CE QUE CE FICHIER AJOUTE, ET CE QU'IL NE REFAIT PAS
//
// Les parts, cadences, points par match et lignes d'escouade d'une grandeur
// "equipment_<famille>" sont déjà calculés par [computeMetric] : cette clé est une
// grandeur comme les autres, sa valeur est simplement la somme des trois issues.
// Ce fichier ne pose que ce que le tronc commun ne sait pas dire — LE REMPLISSAGE
// de la barre (utilisé / gardé / lâché) et LES DEUX TAUX DE RÉFÉRENCE.
//
// # LES RÉFÉRENCES M'EXCLUENT (décision P7)
//
// « Le reste de mon équipe » est mon camp MOINS moi ; « eux » est le lobby MOINS
// mon camp. Se comparer à une moyenne qui vous contient amortit le signal : un
// joueur qui gâche tout tire vers le bas la moyenne à laquelle on le compare.
//
// # LA RÈGLE DE SCOPE EST CELLE DE computeMetric, SANS EXCEPTION
//
// Les deux taux de référence sont des grandeurs D'ÉQUIPE : leurs numérateurs ET
// leurs dénominateurs ne portent que sur les matchs mesurés à CAMP CONNU. Une
// session entièrement FFA n'a pas de « reste de mon équipe » — les deux taux y
// restent nil, jamais un 0 % qui se lirait « ils n'utilisent rien ». Les trois
// issues du joueur, elles, portent sur TOUT le scope mesuré, comme PlayerTotal.

import (
	"strings"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
)

// equipmentBilanFamilies — les familles qui portent une ligne d'issue. SOURCE
// UNIQUE : la table de reconnaissance du résumé (`replay.EquipmentOutcomeFamilies`),
// jamais une seconde liste écrite ici. C'est elle qui garantit que le répulseur n'a
// pas de ligne (décision P4 : son usage n'est mesuré par aucun canal, une ligne
// dirait « 0 utilisation » là où la vérité est « non mesuré »).
var equipmentBilanFamilies = replay.EquipmentOutcomeFamilies()

// outcomeCounts — les trois issues d'une famille, pour un compteur quelconque
// (un joueur, un camp, un lobby), plus les prises qui servent de dénominateur
// d'honnêteté.
type outcomeCounts struct{ used, kept, dropped, taken int }

func (o outcomeCounts) total() int { return o.used + o.kept + o.dropped }

func (o *outcomeCounts) add(other outcomeCounts) {
	o.used += other.used
	o.kept += other.kept
	o.dropped += other.dropped
	o.taken += other.taken
}

// usedRatePct — la part UTILISÉE de la barre, en pourcentage. nil quand la barre
// est vide : 0/0 n'est pas 0 %, et un repère posé à zéro serait une affirmation.
func (o outcomeCounts) usedRatePct() *float64 {
	return sharePct(float64(o.used), float64(o.total()))
}

// equipmentOutcomeOf — les trois issues d'UNE famille pour UNE ligne (match,
// joueur).
//
// « UTILISÉ » A DEUX DÉFINITIONS (décision P2) et c'est le seul endroit du paquet
// qui les distingue : les deux bonus servent quand ils sont ACTIVÉS (leur compte
// d'épisodes), tout le reste sert quand il est POSÉ (`deployed_json`). Le lecteur
// ne voit pas la différence — c'est un détail de calcul.
func equipmentOutcomeOf(p *PlayerRow, family string) outcomeCounts {
	return outcomeCounts{
		used:    equipmentUsedOf(p, family),
		kept:    p.KeptByFamily[family],
		dropped: p.DroppedByFamily[family],
		taken:   p.TakenByFamily[family],
	}
}

// equipmentUsedOf — le côté « utilisé » d'une famille. JUMEAU EXACT de
// `usageUsedOf` (internal/analysis/replay/usage_summary_outcomes.go), qui décide de
// la même chose à la projection : deuxième et dernière copie tolérée de cette
// bascule (règle CLAUDE.md n°6). Elle ne peut pas être partagée — là-bas elle lit
// une ligne de projection, ici une ligne de base.
func equipmentUsedOf(p *PlayerRow, family string) int {
	switch family {
	case replay.EquipmentFamilyPowerupCamo:
		return p.CamoEpisodes
	case replay.EquipmentFamilyPowerupOvershield:
		return p.OvershieldEpisodes
	default:
		return p.DeployedByFamily[family]
	}
}

// attachOutcomes pose les trois issues et les deux taux de référence sur une
// grandeur "equipment_<famille>". Sur toute autre clé : no-op — le champ reste
// absent du contrat (`omitempty`), il ne se publie pas à zéro.
func attachOutcomes(m *domain.SessionUsageMetric, playerXUID string, measured []MatchInput) {
	family, ok := strings.CutPrefix(m.Key, MetricEquipmentPrefix)
	if !ok {
		return
	}
	var mine, teammates, opponents outcomeCounts
	teamKnown := false
	for i := range measured {
		mi := &measured[i]
		for j := range mi.Players {
			p := &mi.Players[j]
			c := equipmentOutcomeOf(p, family)
			if p.XUID == playerXUID {
				mine.add(c)
				continue
			}
			// LES DEUX RÉFÉRENCES SONT DES GRANDEURS D'ÉQUIPE : hors d'un match à
			// camp connu, on ne sait dire ni « mon camp » ni « eux ».
			if mi.PlayerTeam == nil {
				continue
			}
			teamKnown = true
			if teamID, inLobby := mi.TeamOf[p.XUID]; inLobby && teamID == *mi.PlayerTeam {
				teammates.add(c) // mon camp MOINS moi (décision P7)
			} else {
				opponents.add(c) // le lobby MOINS mon camp
			}
		}
	}
	out := domain.SessionUsageOutcomes{
		Used: float64(mine.used), Kept: float64(mine.kept),
		Dropped: float64(mine.dropped), Taken: float64(mine.taken),
		UsedRatePct: mine.usedRatePct(),
	}
	if teamKnown {
		out.TeammatesUsedRatePct = teammates.usedRatePct()
		out.OpponentsUsedRatePct = opponents.usedRatePct()
	}
	m.Outcomes = &out
}
