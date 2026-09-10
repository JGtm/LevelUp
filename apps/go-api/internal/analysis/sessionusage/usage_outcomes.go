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
// « UTILISÉ » A TROIS DÉFINITIONS (décision P2, amendée par le lot 5.5) et c'est le
// seul endroit du paquet qui les distingue : les deux bonus servent quand ils sont
// ACTIVÉS (leur compte d'épisodes), le MUR quand il est POSÉ (`deployed_json`, seul
// déployable qui engendre une pièce), tout autre déployable quand il CONSOMME UNE
// CHARGE (`spent_json`). Le lecteur ne voit pas la différence — c'est un détail de
// calcul.
func equipmentOutcomeOf(p *PlayerRow, family string) outcomeCounts {
	return outcomeCounts{
		used:    equipmentUsedOf(p, family),
		kept:    p.KeptByFamily[family],
		dropped: p.DroppedByFamily[family],
		taken:   p.TakenByFamily[family],
	}
}

// equipmentUsedOf — le côté « utilisé » d'une famille, sur une ligne de BASE.
//
// MÊME RÈGLE que `usageUsedOf` (internal/analysis/replay/usage_summary_outcomes.go),
// qui décide de la même chose à la projection : deuxième et dernière copie tolérée de
// cette BASCULE (règle CLAUDE.md n°6). La fonction elle-même ne peut pas être
// partagée — là-bas elle lit une ligne de projection, ici une ligne de base — mais LA
// CONNAISSANCE, elle, l'est : les familles à pièce engendrée viennent de
// [replay.UsageFamilySpawnsPiece], jamais d'une liste réécrite ici. Garde-rail :
// usage_outcomes_guard_test.go.
//
// CORRIGÉ LE 2026-09-10 (constat C1 de la revue de la vague 5). Cette fonction
// lisait `DeployedByFamily` pour TOUTES les familles, alors que le résumé était
// passé aux consommations en `us6` : sur un capteur `taken=3, spent=2, dropped=1,
// deployed=0`, la page Sessions affichait « utilisé 0 · gardé 0 · lâché 1 » quand la
// vue match affichait « utilisé 2 », et la famille disparaissait même des grandeurs
// dès que ses poses étaient nulles (metricKeys lit `total()`). `SpentByFamily` était
// chargée par le repo et n'avait aucun lecteur.
func equipmentUsedOf(p *PlayerRow, family string) int {
	switch {
	case family == replay.EquipmentFamilyPowerupCamo:
		return p.CamoEpisodes
	case family == replay.EquipmentFamilyPowerupOvershield:
		return p.OvershieldEpisodes
	case replay.UsageFamilySpawnsPiece(family):
		// Le MUR seul : son `spent` tombe sur la pose de PANNEAU, jamais sur la
		// création de l'appareil porté (rapport E0 du 2026-09-10, question 5).
		return p.DeployedByFamily[family]
	default:
		// Tout autre déployable : une pose `deployed` y mesure un LÂCHER VOLONTAIRE
		// à mi-vie, pas un déploiement. Son usage se lit sur les charges consommées.
		return p.SpentByFamily[family]
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
	out := computeOutcomes(playerXUID, []string{family}, measured)
	m.Outcomes = &out
}

// computeOutcomes — les trois issues du SUJET sur un ensemble de familles, et les
// deux taux de référence qui l'EXCLUENT (décision P7). SOURCE UNIQUE du
// remplissage de barre : la page Sessions la lit par famille (attachOutcomes), le
// bloc de période par famille pour la Synthèse et toutes familles confondues pour
// l'Escouade (usage_overview.go).
//
// LE SUJET N'EST PAS TOUJOURS LE JOUEUR DE LA ROUTE : sur une ligne de coéquipier,
// « le reste de mon équipe » est mon camp moins CE coéquipier. Le camp de
// référence reste celui du joueur de la route (MatchInput.PlayerTeam) — c'est le
// seul que le scope connaisse, et les sujets suivis y sont tous alliés.
//
// Un participant sans camp connu dans un match à camp connu tombe du côté « eux » :
// on sait qu'il n'est pas dans mon camp, on ne sait rien de plus.
func computeOutcomes(subjectXUID string, families []string, measured []MatchInput) domain.SessionUsageOutcomes {
	var mine, teammates, opponents outcomeCounts
	teamKnown := false
	for i := range measured {
		mi := &measured[i]
		for j := range mi.Players {
			p := &mi.Players[j]
			c := outcomeCountsOf(p, families)
			if p.XUID == subjectXUID {
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
				teammates.add(c) // mon camp MOINS le sujet (décision P7)
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
	return out
}

// outcomeCountsOf — le cumul des issues d'une ligne joueur sur plusieurs familles.
func outcomeCountsOf(p *PlayerRow, families []string) outcomeCounts {
	var c outcomeCounts
	for _, family := range families {
		c.add(equipmentOutcomeOf(p, family))
	}
	return c
}

// subjectBilanFamilies — les familles du bilan que LE SUJET a lui-même touchées
// (une de ses trois issues non nulle), sur tout le scope mesuré.
//
// SOURCE UNIQUE DU CRITÈRE D'ENTRÉE d'une ligne "equipment_<famille>" (lot 6.4
// point 4) : [metricKeys] (usage.go, page Sessions) ET [overviewFamilies]
// (usage_overview.go, pages Synthèse/Escouade) l'appellent TOUTES LES DEUX. Les
// deux pages affichent la même barre SUJET SEUL (`attachOutcomes` /
// `computeOutcomes` réduisent déjà au sujet) : leur critère d'entrée doit être le
// même, sans quoi l'une peut ouvrir une ligne entièrement vide — un « reproche
// sans objet » — sur la seule foi d'un coéquipier ou d'un adversaire. La question
// posée est « qu'est-ce QUE JE gâche », jamais « qu'est-ce que le lobby gâche ».
func subjectBilanFamilies(playerXUID string, measured []MatchInput) map[string]bool {
	out := map[string]bool{}
	for i := range measured {
		for j := range measured[i].Players {
			p := &measured[i].Players[j]
			if p.XUID != playerXUID {
				continue
			}
			for _, family := range equipmentBilanFamilies {
				if equipmentOutcomeOf(p, family).total() > 0 {
					out[family] = true
				}
			}
		}
	}
	return out
}
