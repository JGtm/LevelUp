// Package service — explorer_target_frag_distribution.go : construction de la
// « Répartition des frags » v2 (sunburst hiérarchique classe→rôle) + « Outils de
// destruction » de l'encart adversaire de l'Explorer, sur les matchs communs.
//
// MIROIR de session_page_frag_distribution.go / squadFragClassesByPlayer : RÉUTILISE
// le builder partagé fragdist.Build (aucune duplication — règle ≤2 copies). Provenance
// (anti-double-source, ADR frags v2) :
//   - classes gun shoulder/sidearm/heavy + rôles d'arme = registre (weapon_kills de la
//     cible sur les matchs communs, résolus dans la même passe ResolveRoles) ;
//   - Mêlée / Grenade / total = agrégat cible déjà calculé (sample, match_participants) ;
//   - mécaniques natives H5 (assassinats + capacités spartanes) = LoadKillMechanicsAggregated
//     (capability OPTIONNELLE via type-assertion) — gate native_kill_mechanics, jamais slug==.
//
// Best-effort partout : nil → le front retombe sur le donut kill-type + top armes legacy.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/fragdist"
)

// explorerKillMechanicsLoader est la capability OPTIONNELLE (type-assertion, même
// pattern que lobbySizeProvider/objectiveScoreProvider côté session page) permettant de
// charger les mécaniques de kill NATIVES Halo 5 (assassinats + capacités spartanes)
// agrégées par xuid. Le repo weapon_kills DuckDB concret (duckdb.WeaponKillsRepo)
// l'implémente ; un loader qui ne l'implémente pas dégrade proprement (Mêlée non
// splittée, pas de classe « Capacités spartanes »).
type explorerKillMechanicsLoader interface {
	LoadKillMechanicsAggregated(ctx context.Context, filters port.WeaponKillFilters) ([]port.KillMechanicsRow, error)
}

// targetFragDistribution construit la « Répartition des frags » v2 (sunburst classe→rôle)
// + le top armes enrichi de la cible sur les matchs communs. Best-effort : nil/nil si pas
// de repo weapon_kills, pas de kills, ou aucune arme résolue (cible sans données d'arme —
// un sunburst réduit à Mêlée/Grenade + « Non attribué » serait moins informatif que le
// donut kill-type legacy, donc on laisse le front basculer sur le repli).
func (s *ExplorerService) targetFragDistribution(
	ctx context.Context,
	targetXUID string,
	matchIDs []string,
	sample *domain.ExplorerTargetSampleStats,
) (*domain.FragDistribution, []domain.SynthesisWeaponKillEntry) {
	if sample == nil || sample.Kills <= 0 {
		return nil, nil
	}
	rows := s.loadTargetWeaponKillRows(ctx, targetXUID, matchIDs)
	if len(rows) == 0 {
		return nil, nil
	}
	slug := ctxkeys.TitleSlug(ctx)
	// hasMechanics capability-gated (native_kill_mechanics, jamais slug== — ratchet
	// no_slug_comparison_test.go) : sur H5 on charge les mécaniques natives de la cible
	// (assassinats + capacités spartanes) pour splitter Mêlée et alimenter la classe
	// « Capacités spartanes » ; sur Infinite (cap off) → Mêlée feuille, pas de spartan.
	hasMechanics := titleHasNativeKillMechanics(slug)
	counts := domain.FragKillTypeCounts{
		Melee:   sample.MeleeKills,
		Grenade: sample.GrenadeKills,
		Total:   sample.Kills,
	}
	if hasMechanics {
		if m := s.loadTargetKillMechanics(ctx, targetXUID, matchIDs); m != nil {
			counts.Assassination = m.Assassinations
			counts.GroundPound = m.GroundPound
			counts.ShoulderBash = m.ShoulderBash
		}
	}
	fd := fragdist.Build(rows, counts, hasMechanics)
	logFragDistribution(ctx, "explorer target", slug, targetXUID, fd)
	return &fd, buildTopWeaponKills(rows, synthesisWeaponChartTopN)
}

// loadTargetWeaponKillRows charge les rows agrégées d'armes de la cible sur les matchs
// communs (XUIDs filtre + ResolveRoles=true → Role+Class dans la même passe registre).
// Best-effort : nil si repo absent / xuid ou scope vide, ou erreur (loggée, jamais
// avalée — MIROIR de loadSessionWeaponKillRows : capability absente = Debug ; anomalie
// SQL/conn = Warn).
func (s *ExplorerService) loadTargetWeaponKillRows(
	ctx context.Context, targetXUID string, matchIDs []string,
) []port.WeaponKillRow {
	if s.weaponKillsRepo == nil || targetXUID == "" || len(matchIDs) == 0 {
		return nil
	}
	slug := ctxkeys.TitleSlug(ctx)
	wf := port.WeaponKillFilters{MatchIDs: matchIDs, XUIDs: []string{targetXUID}, ResolveRoles: true}
	rows, err := s.weaponKillsRepo.LoadWeaponKillsAggregated(ctx, slug, wf)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "explorer target: weapon kills capability absente",
				"title", slug, "xuid", targetXUID)
		} else {
			slog.WarnContext(ctx, "explorer target: weapon kills query failed (best-effort, fallback nil)",
				"title", slug, "xuid", targetXUID, "match_count", len(matchIDs), "err", err)
		}
		return nil
	}
	return rows
}

// loadTargetKillMechanics charge les mécaniques de kill NATIVES Halo 5 (assassinats +
// capacités spartanes) de la cible agrégées sur les matchs communs, via la capability
// OPTIONNELLE explorerKillMechanicsLoader (type-assertion sur le repo weapon_kills).
// nil best-effort si le repo ne fournit pas la capability, si le scope est vide, ou en
// cas d'erreur/aucune donnée (loggée Debug, jamais avalée — parité loadSquadMechanicsByGT).
func (s *ExplorerService) loadTargetKillMechanics(
	ctx context.Context, targetXUID string, matchIDs []string,
) *port.KillMechanicsRow {
	loader, ok := s.weaponKillsRepo.(explorerKillMechanicsLoader)
	if !ok || targetXUID == "" || len(matchIDs) == 0 {
		return nil
	}
	rows, err := loader.LoadKillMechanicsAggregated(ctx, port.WeaponKillFilters{
		MatchIDs: matchIDs, XUIDs: []string{targetXUID},
	})
	if err != nil {
		slog.DebugContext(ctx, "explorer target: kill mechanics skipped (best-effort)",
			"xuid", targetXUID, "err", err)
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	// Au plus une ligne par xuid ; on somme par sûreté (parité loadSquadMechanicsByGT).
	agg := port.KillMechanicsRow{XUID: targetXUID}
	for _, r := range rows {
		agg.Assassinations += r.Assassinations
		agg.GroundPound += r.GroundPound
		agg.ShoulderBash += r.ShoulderBash
	}
	return &agg
}
