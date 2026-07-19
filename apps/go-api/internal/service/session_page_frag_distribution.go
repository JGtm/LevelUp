// Package service — session_page_frag_distribution.go : construction de la
// répartition hiérarchique des frags (sunburst v2) pour la page détail de session
// (P5, PLAN_FRAG_DISTRIBUTION_V2). Nouveau chemin de données : l'endpoint
// sessions/detail ne chargeait ni weapon_kills ni compteurs kill-type. On agrège,
// sur le scope de la session sélectionnée :
//   - les compteurs kill-type API (melee/grenade/spartan + total) depuis les rows
//     canoniques, via le builder Synthesis partagé buildSynthesisDetailedStatsFromCanonical ;
//   - les classes/rôles d'arme (gun) + le top armes depuis le registre (weapon_kills).
//
// Puis RÉUTILISE le builder partagé buildFragDistribution (aucune duplication —
// règle ≤2 copies). hasMechanics est capability-gated (titleHasNativeKillMechanics,
// jamais slug==).
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/fragdist"
)

// attachSessionFragDistribution renseigne FragDistribution + TopWeaponKills sur
// l'entry de session. No-op si entry nil.
func (s *SessionPageService) attachSessionFragDistribution(
	ctx context.Context,
	entry *domain.SessionCompareEntry,
	canonRows []canonical.PlayerMatchRow,
	matchIDs []string,
) {
	if entry == nil {
		return
	}
	fd, top := s.sessionFragDistribution(ctx, canonRows, matchIDs)
	entry.FragDistribution = fd
	entry.TopWeaponKills = top
}

// sessionFragDistribution construit la FragDistribution + le top armes de la session
// délimitée par matchIDs. Best-effort : nil/nil si le scope est vide (total 0 → le
// front rend null). Les classes API (melee/grenade/spartan + total) proviennent des
// rows canoniques du scope ; les classes gun + rôles d'arme + top armes du registre.
func (s *SessionPageService) sessionFragDistribution(
	ctx context.Context,
	canonRows []canonical.PlayerMatchRow,
	matchIDs []string,
) (*domain.FragDistribution, []domain.SynthesisWeaponKillEntry) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	idSet := make(map[string]struct{}, len(matchIDs))
	for _, id := range matchIDs {
		idSet[id] = struct{}{}
	}
	sessionCanon := make([]canonical.PlayerMatchRow, 0, len(matchIDs))
	totalKills := 0
	for i := range canonRows {
		if _, ok := idSet[canonRows[i].Summary.MatchID]; !ok {
			continue
		}
		sessionCanon = append(sessionCanon, canonRows[i])
		if canonRows[i].Self.Kills != nil {
			totalKills += *canonRows[i].Self.Kills
		}
	}
	if totalKills <= 0 {
		return nil, nil
	}
	// Compteurs kill-type API canoniques du scope session (même agrégation que
	// Synthesis). provideSpree non pertinent ici (spree non utilisée pour les frags).
	stats := buildSynthesisDetailedStatsFromCanonical(sessionCanon, false)
	counts := domain.FragKillTypeCounts{
		Melee:         stats.TotalMeleeKills,
		Grenade:       stats.TotalGrenadeKills,
		Assassination: stats.TotalAssassinations,
		GroundPound:   stats.TotalGroundPoundKills,
		ShoulderBash:  stats.TotalShoulderBashKills,
		Total:         totalKills,
	}
	rows := s.loadSessionWeaponKillRows(ctx, matchIDs)
	fd := fragdist.Build(rows, counts, titleHasNativeKillMechanics(s.titleSlug))
	logFragDistribution(ctx, "session page", s.titleSlug, s.gamertag, fd)
	return &fd, buildTopWeaponKills(rows, synthesisWeaponChartTopN)
}

// loadSessionWeaponKillRows charge les rows agrégées d'armes de la session
// (ResolveRoles=true → Role+Class dans la même passe). Best-effort : nil si repo
// absent / gamertag vide, ou erreur (loggée, jamais avalée — parité loadWeaponKillRows
// Synthesis : capability absente = Debug ; anomalie SQL/conn = Warn).
func (s *SessionPageService) loadSessionWeaponKillRows(
	ctx context.Context, matchIDs []string,
) []port.WeaponKillRow {
	if s.weaponKillsRepo == nil || s.gamertag == "" {
		return nil
	}
	wf := port.WeaponKillFilters{MatchIDs: matchIDs, Gamertag: s.gamertag, ResolveRoles: true}
	rows, err := s.weaponKillsRepo.LoadWeaponKillsAggregated(ctx, s.titleSlug, wf)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "session page: weapon kills capability absente",
				"title", s.titleSlug, "gamertag", s.gamertag)
		} else {
			slog.WarnContext(ctx, "session page: weapon kills query failed (best-effort, fallback nil)",
				"title", s.titleSlug, "gamertag", s.gamertag,
				"match_count", len(matchIDs), "err", err)
		}
		return nil
	}
	return rows
}

// matchIDsFromStatsRows projette les match_id d'un lot de StatsMatchRow (scope d'une
// session) → clés de filtrage pour weapon_kills et le sous-ensemble canonique.
func matchIDsFromStatsRows(rows []legacymatch.StatsMatchRow) []string {
	ids := make([]string, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].MatchID)
	}
	return ids
}
