// Package halo_infinite — adapter_data_career.go : methodes DataAdapter carriere/historique
// (rang, encounters, LUSR, top matches, matchs recents). Extrait de adapter_data.go
// (K3f god-file split, 2026-07-06), meme package.
package halo_infinite

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// LoadCareerSnapshot wrappe CareerSource.GetLatestRank et projette le résultat
// vers le canonique services.
//
// Comportement :
//   - career source nil → ErrCapabilityNotSupported ;
//   - aucune entrée trouvée (sql.ErrNoRows) → CareerSnapshot vide, pas d'erreur ;
//   - autre erreur → propagée avec contexte.
func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", games.CapCareerProgression,
		)
		return nil, games.ErrCapabilityNotSupported
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	row, err := a.career.GetLatestRank(ctx)
	var snap *canonical.CareerSnapshot
	switch {
	case err == nil:
		snap = projectCareerSnapshot(xuid, row)
	case isNoRowsErr(err):
		snap = &canonical.CareerSnapshot{Player: canonical.PlayerIdentity{XUID: xuid}}
	default:
		return nil, err
	}

	// HIGH-C : l'historique XP est INDÉPENDANT du rang courant (le legacy
	// repo.GetXPHistory ne dépend pas de GetLatestRank) → fetch séparé quand demandé.
	if opts.IncludeHistory {
		hist, herr := a.career.GetXPHistory(ctx)
		if herr != nil {
			return nil, herr
		}
		snap.History = projectCareerHistory(hist)
	}

	return snap, nil
}

// projectCareerHistory projette l'historique XP domaine → canonique. Retourne nil
// pour une entrée vide (préserve la sémantique nil du legacy GetXPHistory : le
// service ré-initialise nil→[] APRÈS les projections, cf. GetCareerPage).
func projectCareerHistory(rows []domain.XPHistoryPoint) []canonical.CareerHistoryEntry {
	if len(rows) == 0 {
		return nil
	}
	out := make([]canonical.CareerHistoryEntry, 0, len(rows))
	for _, p := range rows {
		cur, tot := p.CurrentXP, p.XPTotal
		out = append(out, canonical.CareerHistoryEntry{
			RecordedAt: p.RecordedAt,
			RankNumber: p.Rank,
			CurrentXP:  &cur,
			XPTotal:    &tot,
		})
	}
	return out
}

// LoadEncounters wrappe CareerSource.GetEncounters et projette le résultat
// vers le canonique services.
//
// Comportement :
//   - career source nil → ErrCapabilityNotSupported ;
//   - aucune entrée trouvée → slice vide, pas d'erreur ;
//   - autre erreur → propagée avec contexte.
//
// L'argument xuid est ignoré : le CareerRepo HI résout déjà l'identité du
// joueur courant via son PlayerDB. Le paramètre est conservé dans la
// signature canonique pour permettre à un futur titre B de s'en servir.
func (a *DataAdapter) LoadEncounters(ctx context.Context, xuid string) ([]canonical.EncounterRow, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "career.encounters",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := a.career.GetEncounters(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]canonical.EncounterRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectEncounterRow(r))
	}
	return out, nil
}

// LoadLUSRHistory wrappe CareerSource.GetLUSRHistory et projette vers le canonique
// (Phase 2 HIGH-C). career source nil → ErrCapabilityNotSupported. Retourne nil sur
// historique vide (préserve la sémantique nil du legacy GetLUSRHistory).
func (a *DataAdapter) LoadLUSRHistory(ctx context.Context, xuid string) ([]canonical.LUSRCheckpoint, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "career.lusr_history",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := a.career.GetLUSRHistory(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]canonical.LUSRCheckpoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectLUSRCheckpoint(r))
	}
	return out, nil
}

// projectLUSRCheckpoint projette un checkpoint LUSR domaine → canonique (copie
// profonde des pointeurs pour éviter tout aliasing avec la slice source).
func projectLUSRCheckpoint(r domain.LUSRCheckpointDTO) canonical.LUSRCheckpoint {
	c := canonical.LUSRCheckpoint{
		MatchID:      r.MatchID,
		RatingType:   r.RatingType,
		RatingValue:  r.RatingValue,
		PlaylistName: r.PlaylistName,
		PlaylistID:   r.PlaylistID,
	}
	if r.TierLabel != nil {
		v := *r.TierLabel
		c.TierLabel = &v
	}
	if r.PlaylistGroup != nil {
		v := *r.PlaylistGroup
		c.PlaylistGroup = &v
	}
	if r.RecordedAt != nil {
		v := *r.RecordedAt
		c.RecordedAt = &v
	}
	if r.RatingDelta != nil {
		v := *r.RatingDelta
		c.RatingDelta = &v
	}
	if r.BadgeImageURL != nil {
		v := *r.BadgeImageURL
		c.BadgeImageURL = &v
	}
	return c
}

// LoadTopMatches wrappe CareerSource.GetTopMatches et projette vers le canonique
// (Phase 2 HIGH-C). career source nil → ErrCapabilityNotSupported. Retourne nil
// sur entrée vide (préserve la sémantique du legacy GetTopMatches).
func (a *DataAdapter) LoadTopMatches(ctx context.Context, xuid string) ([]canonical.CareerTopMatch, error) {
	if a.career == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "career.top_matches",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := a.career.GetTopMatches(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]canonical.CareerTopMatch, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectCareerTopMatch(r))
	}
	return out, nil
}

// projectCareerTopMatch projette un top-match domaine → canonique (copie profonde
// des pointeurs). OutcomeCode = code BRUT (jamais via canonical.Outcome string).
func projectCareerTopMatch(r domain.TopMatchRawRow) canonical.CareerTopMatch {
	c := canonical.CareerTopMatch{
		MatchID:          r.MatchID,
		PerformanceScore: r.PerformanceScore,
		OutcomeCode:      r.Outcome,
		Kills:            r.Kills,
		Deaths:           r.Deaths,
		DominanceFlag:    r.DominanceFlag,
	}
	if r.StartTime != nil {
		v := *r.StartTime
		c.StartTime = &v
	}
	if r.MapName != nil {
		v := *r.MapName
		c.MapName = &v
	}
	if r.PairName != nil {
		v := *r.PairName
		c.PairName = &v
	}
	if r.PlaylistName != nil {
		v := *r.PlaylistName
		c.PlaylistName = &v
	}
	if r.KDA != nil {
		v := *r.KDA
		c.KDA = &v
	}
	if r.TeamMMR != nil {
		v := *r.TeamMMR
		c.TeamMMR = &v
	}
	if r.EnemyMMR != nil {
		v := *r.EnemyMMR
		c.EnemyMMR = &v
	}
	return c
}

// LoadTargetRecentMatches wrappe RecentSource.GetTargetRecentMatches et projette
// vers le canonique (Phase 2 HIGH-B). source nil → ErrCapabilityNotSupported.
// Retourne nil sur entrée vide (préserve la sémantique du legacy).
func (a *DataAdapter) LoadTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]canonical.RecentMatchRow, error) {
	if a.recent == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "explorer.recent_matches",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := a.recent.GetTargetRecentMatches(ctx, xuid, limit)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]canonical.RecentMatchRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, projectRecentMatchRow(r))
	}
	return out, nil
}

// projectRecentMatchRow projette un match récent domaine → canonique. Outcome =
// code BRUT ; Rank deep-copié ; ModePairAssetID (transient live) NON porté.
func projectRecentMatchRow(r domain.ExplorerTargetRecentMatch) canonical.RecentMatchRow {
	c := canonical.RecentMatchRow{
		MatchID:         r.MatchID,
		StartTime:       r.StartTime,
		MapUI:           r.MapUI,
		ModeUI:          r.ModeUI,
		Outcome:         r.Outcome,
		Kills:           r.Kills,
		Deaths:          r.Deaths,
		Assists:         r.Assists,
		KDA:             r.KDA,
		Score:           r.Score,
		DamageDealt:     r.DamageDealt,
		DamageTaken:     r.DamageTaken,
		MaxKillingSpree: r.MaxKillingSpree,
		PerfectKills:    r.PerfectKills,
	}
	if r.Rank != nil {
		v := *r.Rank
		c.Rank = &v
	}
	return c
}
