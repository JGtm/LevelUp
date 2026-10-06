package sessionusage

// usage_outcomes.go — LES ISSUES D'UN OBJET PRIS (utilisé, gardé sans servir, lâché en mourant,
// et les prises), décidées sur UNE ligne (match, joueur) du résumé d'usage. Les comptes publiés
// sont dans usage_outcomes_counts.go (lus par l'Emprise).

import "levelup/go-api/internal/domain/equipmentusage"

// outcomeCounts — les trois issues d'une famille, plus les prises qui servent de
// dénominateur d'honnêteté.
type outcomeCounts struct{ used, kept, dropped, taken int }

func (o *outcomeCounts) add(other outcomeCounts) {
	o.used += other.used
	o.kept += other.kept
	o.dropped += other.dropped
	o.taken += other.taken
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
// MÊME RÈGLE que `usageUsedOf` (internal/games/halo_infinite/film/replay/usage_summary_outcomes.go),
// qui décide de la même chose à la projection : deuxième et dernière copie tolérée de
// cette BASCULE (règle CLAUDE.md n°6). La fonction elle-même ne peut pas être
// partagée — là-bas elle lit une ligne de projection, ici une ligne de base — mais LA
// CONNAISSANCE, elle, l'est : les familles à pièce engendrée viennent de
// [equipmentusage.UsageFamilySpawnsPiece], jamais d'une liste réécrite ici. Garde-rail :
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
	case family == equipmentusage.EquipmentFamilyPowerupCamo:
		return p.CamoEpisodes
	case family == equipmentusage.EquipmentFamilyPowerupOvershield:
		return p.OvershieldEpisodes
	case equipmentusage.UsageFamilySpawnsPiece(family):
		// Le MUR seul : son `spent` tombe sur la pose de PANNEAU, jamais sur la
		// création de l'appareil porté (rapport E0 du 2026-09-10, question 5).
		return p.DeployedByFamily[family]
	default:
		// Tout autre déployable : une pose `deployed` y mesure un LÂCHER VOLONTAIRE
		// à mi-vie, pas un déploiement. Son usage se lit sur les charges consommées.
		return p.SpentByFamily[family]
	}
}

// outcomeCountsOf — le cumul des issues d'une ligne joueur sur plusieurs familles.
func outcomeCountsOf(p *PlayerRow, families []string) outcomeCounts {
	var c outcomeCounts
	for _, family := range families {
		c.add(equipmentOutcomeOf(p, family))
	}
	return c
}
