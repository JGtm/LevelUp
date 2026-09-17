// Package service — explorer_target_frag_range.go : le bloc « Portée des frags » de la 3e
// rangée de l'encart adversaire, sur les matchs joués ensemble.
//
// # DEUX BANDES, UNE SEULE MESURE
//
// Le joueur courant et la cible, côté FRAGS, sur le MÊME scope : les matchs qu'ils ont joués
// ensemble. C'est ce qui rend les deux bandes comparables — deux portées lues sur deux
// ensembles de matchs différents ne se superposeraient pas honnêtement.
//
// # LA CHAÎNE N'EST PAS RÉÉCRITE ICI
//
// Frags mesurés, clés d'arme traduites en rôles, regroupement, agrégat : tout vit dans
// `weapon_range_by_role.go`, partagé avec le profil d'armes du Face-à-face. Ce fichier ne
// fait que réunir le scope et les dénominateurs, puis appeler.
//
// # BEST-EFFORT, ET LES DEUX CÔTÉS SONT INDÉPENDANTS
//
// Un joueur sans frag mesuré laisse SA bande absente ; l'autre reste publiée. Les deux
// absents = le front rend un état vide titré. Aucune erreur ne remonte : le bloc est additif
// à une réponse qui se sert sans lui.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// enrichEncounterFragRange remplit FragRangeSelf / FragRangeTarget sur les stats de
// rencontre. no-op si le repo n'est pas câblé, si la cible est inconnue, ou si les deux
// joueurs n'ont aucun match en commun.
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

	// Dénominateurs de couverture du joueur courant : ses frags et ses morts SUR CES
	// MATCHS, jamais sa carrière. Une lecture en échec ne fait pas tomber la bande — elle
	// la publie avec des totaux à zéro plutôt que rien, ce que le front ne montre pas
	// (le bloc ne rend que les bandes).
	selfScope := weaponRangeScopeInfo{matchIDs: matchIDs}
	if agg, err := s.loadParticipantStats(ctx, s.xuid, matchIDs); err != nil {
		slog.WarnContext(ctx, "explorer portee: totaux du joueur courant illisibles (couverture a zero)",
			"title", slug, "xuid", s.xuid, "match_count", len(matchIDs), "err", err)
	} else if agg != nil {
		selfScope.totalKills, selfScope.totalDeaths = agg.Kills, agg.Deaths
	}
	stats.FragRangeSelf = buildWeaponRangeByRole(ctx, s.weaponRangeRepo, "explorer", slug, s.xuid, selfScope)

	// Côté cible, les totaux sont DÉJÀ calculés par l'encart (même scope, même agrégat) :
	// les relire serait une seconde source, et deux lectures d'une base concurrente
	// peuvent différer.
	targetScope := weaponRangeScopeInfo{matchIDs: matchIDs}
	if targetSample != nil {
		targetScope.totalKills, targetScope.totalDeaths = targetSample.Kills, targetSample.Deaths
	}
	stats.FragRangeTarget = buildWeaponRangeByRole(ctx, s.weaponRangeRepo, "explorer", slug, targetXUID, targetScope)

	slog.DebugContext(ctx, "explorer portee des frags",
		"title", slug, "other_xuid", targetXUID, "match_count", len(matchIDs),
		"self_lignes", weaponRangeRowCount(stats.FragRangeSelf),
		"cible_lignes", weaponRangeRowCount(stats.FragRangeTarget))
}

// weaponRangeRowCount : nombre de lignes publiées d'un bloc, 0 si le bloc est absent.
func weaponRangeRowCount(block *domain.SynthesisWeaponRange) int {
	if block == nil {
		return 0
	}
	return len(block.Weapons)
}
