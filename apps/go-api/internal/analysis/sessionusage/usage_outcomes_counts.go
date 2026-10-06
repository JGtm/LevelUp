package sessionusage

// usage_outcomes_counts.go — LES QUATRE COMPTES D'ISSUE, PUBLIÉS (décision D11 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
//
// L'Emprise a besoin des COMPTES, camp par camp — « bonus perdus : 2 sur 12 » — et des temps et
// frags d'effet des bonus. Ce fichier les expose SANS réécrire la bascule « utilisé » : il passe
// par [outcomeCountsOf], donc par [equipmentOutcomeOf] (garde-rail usage_outcomes_guard_test.go et
// golden usage_outcomes_golden_test.go).

import "levelup/go-api/internal/domain/equipmentusage"

// OutcomeCounts — pris, utilisé, gardé sans servir, lâché en mourant.
type OutcomeCounts struct {
	Taken, Used, Kept, Dropped int
}

// Lost — les objets PERDUS : gardés sans servir ou lâchés en mourant (maquette de l'onglet
// Emprise, « Bonus perdus »).
func (o OutcomeCounts) Lost() int { return o.Kept + o.Dropped }

// Add cumule deux comptes.
func (o *OutcomeCounts) Add(other OutcomeCounts) {
	o.Taken += other.Taken
	o.Used += other.Used
	o.Kept += other.Kept
	o.Dropped += other.Dropped
}

// PlayerOutcomeCounts — les quatre comptes d'une ligne (match, joueur) sur un ensemble de
// familles, par la bascule « utilisé » unique de ce paquet.
func PlayerOutcomeCounts(p *PlayerRow, families []string) OutcomeCounts {
	c := outcomeCountsOf(p, families)
	return OutcomeCounts{Taken: c.taken, Used: c.used, Kept: c.kept, Dropped: c.dropped}
}

// PowerupFamilies — les familles des BONUS (camouflage, surbouclier), dans cet ordre : la
// ressource « bonus » de l'Emprise (décision D4). Source unique : equipmentusage.
func PowerupFamilies() []string {
	return []string{
		equipmentusage.EquipmentFamilyPowerupCamo,
		equipmentusage.EquipmentFamilyPowerupOvershield,
	}
}

// PowerupEffect — le temps d'effet (ms) et les frags pendant l'effet d'UN bonus, pour une ligne
// (match, joueur). (0, 0) pour toute autre famille : seuls les deux bonus ont un état actif
// mesuré (`equipmentEpisodes`).
func PowerupEffect(p *PlayerRow, family string) (ms int64, kills int) {
	switch family {
	case equipmentusage.EquipmentFamilyPowerupCamo:
		return p.CamoMS, p.CamoKills
	case equipmentusage.EquipmentFamilyPowerupOvershield:
		return p.OvershieldMS, p.OvershieldKills
	}
	return 0, 0
}
