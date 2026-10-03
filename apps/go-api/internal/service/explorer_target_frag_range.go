// Package service — explorer_target_frag_range.go : le bloc « Portée des frags » de la 3e
// rangée de l'encart adversaire, sur les matchs joués ensemble.
//
// # UNE SEULE BANDE : LA CIBLE
//
// Côté FRAGS, sur les matchs que les deux joueurs ont joués ensemble. Le côté du joueur
// courant était publié et superposé sur la même bande jusqu'au 2026-09-19 ; la décision 7 du
// plan d'ajustements pré-v7.5 a ramené le bloc à la cible seule, et sa lecture dédiée est
// partie avec lui — un champ que l'écran ne lit plus ne se calcule pas.
//
// # LA CHAÎNE N'EST PAS RÉÉCRITE ICI
//
// Frags mesurés, clés d'arme traduites en rôles, regroupement, agrégat : tout vit dans
// `weapon_range_by_role.go`, partagé avec le profil d'armes du Face-à-face. Ce fichier ne
// fait que réunir le scope et les dénominateurs, puis appeler.
//
// # BEST-EFFORT
//
// Une cible sans frag mesuré laisse la bande absente, et le front rend un état vide titré.
// Aucune erreur ne remonte : le bloc est additif à une réponse qui se sert sans lui.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// enrichEncounterFragRange remplit FragRangeTarget sur les stats de rencontre. no-op si le
// repo n'est pas câblé, si la cible est inconnue, ou si les deux joueurs n'ont aucun match
// en commun.
func (s *ExplorerService) enrichEncounterFragRange(
	ctx context.Context,
	stats *domain.ExplorerEncounterStats,
	targetXUID string,
	matchIDs []string,
	targetSample *domain.ExplorerTargetSampleStats,
) {
	if stats == nil || s.weaponRangeRepo == nil || targetXUID == "" || len(matchIDs) == 0 {
		return
	}
	slug := ctxkeys.TitleSlug(ctx)

	// Les totaux de la cible sont DÉJÀ calculés par l'encart (même scope, même agrégat) :
	// les relire serait une seconde source, et deux lectures d'une base concurrente
	// peuvent différer.
	targetScope := weaponRangeScopeInfo{matchIDs: matchIDs}
	if targetSample != nil {
		targetScope.totalKills, targetScope.totalDeaths = targetSample.Kills, targetSample.Deaths
	}
	stats.FragRangeTarget = buildWeaponRangeByRole(ctx, s.weaponRangeRepo, "explorer", slug, targetXUID, targetScope)

	slog.DebugContext(ctx, "explorer portee des frags",
		"title", slug, "other_xuid", targetXUID, "match_count", len(matchIDs),
		"cible_lignes", weaponRangeRowCount(stats.FragRangeTarget))
}

// weaponRangeRowCount : nombre de lignes publiées d'un bloc, 0 si le bloc est absent.
func weaponRangeRowCount(block *domain.SynthesisWeaponRange) int {
	if block == nil {
		return 0
	}
	return len(block.Weapons)
}
