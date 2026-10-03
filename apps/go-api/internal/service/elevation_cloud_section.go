// Package service — elevation_cloud_section.go : L'ASSEMBLAGE DU NUAGE « DÉNIVELÉ » (D25, T5).
//
// # POURQUOI CE FICHIER NE LIT RIEN
//
// Le nuage et la portée par arme décrivent LES MÊMES frags mesurés, sur le MÊME scope. Une
// seconde lecture du repo aurait doublé deux requêtes SQL et un emprunt du lecteur partagé
// (la ressource la plus disputée du process, ADR 0013) pour obtenir exactement les mêmes
// lignes — et, au premier écart de filtre, deux vérités sur la même page. Le chargement reste
// donc chez `buildWeaponRangeSections`, et ce fichier ne fait que mettre en forme.
//
// Les quantiles et le signe du dénivelé vivent dans `internal/analysis` (purs) ; ici, la
// conversion en contrat et l'hydratation des libellés d'arme.
package service

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// buildElevationCloudBlock met le nuage à la forme du contrat, ou rend nil.
//
// NIL QUAND LES DEUX CÔTÉS SONT VIDES, jamais un bloc à zéro : un nuage sans point se lirait
// « aucun engagement », alors que la vérité est « rien n'est décodé sur cette fenêtre ».
func buildElevationCloudBlock(
	kills []analysis.MeasuredKill, scope weaponRangeScopeInfo,
) *domain.ElevationCloudBlock {
	cloud := analysis.BuildElevationCloud(kills)
	if len(cloud.Kills) == 0 && len(cloud.Deaths) == 0 {
		return nil
	}
	return &domain.ElevationCloudBlock{
		Kills:          elevationPoints(cloud.Kills),
		Deaths:         elevationPoints(cloud.Deaths),
		KillsSummary:   elevationSummary(cloud.KillsSummary),
		DeathsSummary:  elevationSummary(cloud.DeathsSummary),
		MeasuredKills:  cloud.KillsSummary.N,
		TotalKills:     scope.totalKills,
		MeasuredDeaths: cloud.DeathsSummary.N,
		TotalDeaths:    scope.totalDeaths,
	}
}

// elevationPoints projette les points. Toujours une tranche NON NIL : le contrat sérialise
// `[]` et le front itère sans garde.
func elevationPoints(pts []analysis.ElevationPoint) []domain.ElevationPoint {
	out := make([]domain.ElevationPoint, 0, len(pts))
	for _, p := range pts {
		out = append(out, domain.ElevationPoint{
			DistanceM: p.DistanceM,
			DeltaZM:   p.DeltaZM,
			MatchID:   p.MatchID,
			TimeMS:    p.TimeMS,
			Weapon:    p.WeaponKey,
		})
	}
	return out
}

// elevationSummary passe les six quantiles en pointeurs. UN CÔTÉ VIDE N'A PAS DE QUANTILE :
// ses six champs restent nil (cf. `domain.ElevationSideSummary`), seul `N` vaut zéro.
func elevationSummary(s analysis.ElevationSideSummary) domain.ElevationSideSummary {
	if s.N == 0 {
		return domain.ElevationSideSummary{}
	}
	return domain.ElevationSideSummary{
		DistanceP25: &s.DistanceP25,
		DistanceP50: &s.DistanceP50,
		DistanceP75: &s.DistanceP75,
		DeltaZP25:   &s.DeltaZP25,
		DeltaZP50:   &s.DeltaZP50,
		DeltaZP75:   &s.DeltaZP75,
		N:           s.N,
	}
}

// applyElevationLabels pose le dictionnaire des noms d'arme du nuage, depuis les libellés
// DÉJÀ RÉSOLUS par `hydrateLabels` — un seul appel au résolveur sert les deux blocs.
//
// BEST-EFFORT : une clé que la metadata ne connaît pas n'a PAS d'entrée, et le front retombe
// sur la clé — jamais un nom inventé côté Go (aucun libellé FR/EN en dur, règle transverse
// multi-titre). PUR : aucune I/O, aucun log.
func applyElevationLabels(block *domain.ElevationCloudBlock, labels map[string]port.WeaponLabel) {
	if block == nil {
		return
	}
	keys := collectElevationWeaponKeys(block)
	if len(keys) == 0 {
		return
	}
	out := make(map[string]domain.ElevationWeaponLabel, len(keys))
	for _, k := range keys {
		l := labels[k]
		if l.Label == "" && l.LabelEN == "" {
			continue
		}
		out[k] = domain.ElevationWeaponLabel{Label: l.Label, LabelEN: l.LabelEN}
	}
	if len(out) > 0 {
		block.WeaponLabels = out
	}
}

// collectElevationWeaponKeys rend les clés à traduire, dédupliquées, dans l'ordre de première
// apparition — l'ordre est stable pour que le log et les tests ne dépendent pas d'une map.
func collectElevationWeaponKeys(block *domain.ElevationCloudBlock) []string {
	if block == nil {
		return nil
	}
	seen := make(map[string]bool)
	keys := make([]string, 0, 16)
	for _, side := range [][]domain.ElevationPoint{block.Kills, block.Deaths} {
		for _, p := range side {
			if p.Weapon != "" && !seen[p.Weapon] {
				seen[p.Weapon] = true
				keys = append(keys, p.Weapon)
			}
		}
	}
	return keys
}
