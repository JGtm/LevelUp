// Package halo_infinite — implémentation des adapters multi-titres pour HI.
//
// Phase B du plan : ce package wrappe les repos existants
// (internal/platform/duckdb/*) sans réécrire leurs requêtes. La migration
// physique des queries arrivera en Phase F (cleanup) une fois la bascule
// endpoint par endpoint validée par golden parity.
//
// Le data adapter utilise des interfaces minimales (CareerSource, etc.) pour
// rester découplé des structs concrets et permettre un mock simple en tests
// unitaires.
package halo_infinite

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// CareerSource est l'interface minimale consommée par le data adapter pour
// la lecture de la progression carrière. Implémentée par
// internal/platform/duckdb/CareerRepo.
type CareerSource interface {
	GetLatestRank(ctx context.Context) (*domain.CareerRankData, error)
	GetEncounters(ctx context.Context) ([]domain.EncounterRawRow, error)
	// GetXPHistory : historique XP (Phase 2 HIGH-C) — alimente CareerSnapshot.History
	// quand CareerOptions.IncludeHistory. Déjà implémenté par duckdb.CareerRepo.
	GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error)
	// GetLUSRHistory : historique des checkpoints de rating LUSR/CSR (Phase 2 HIGH-C).
	GetLUSRHistory(ctx context.Context) ([]domain.LUSRCheckpointDTO, error)
	// GetTopMatches : meilleurs/pires matchs carrière (Phase 2 HIGH-C).
	GetTopMatches(ctx context.Context) ([]domain.TopMatchRawRow, error)
}

// RecentSource est la surface minimale de lecture du profil de combat récent d'un
// joueur (Explorer, Phase 2 HIGH-B). Implémentée par internal/platform/duckdb.ExplorerRepo.
type RecentSource interface {
	GetTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error)
}

// DataAdapter est l'implémentation HI de games.TitleDataAdapter.
//
// Phase B : seules les méthodes nécessaires aux endpoints prévus en Phase C
// sont câblées. Les autres retournent ErrCapabilityNotSupported pour rendre
// les absences explicites côté caller (cf. plan §5.7).
type DataAdapter struct {
	career CareerSource
	recent RecentSource // Explorer profil de combat (Phase 2 HIGH-B). nil → ErrCapabilityNotSupported.
	// staticCaps : CapabilityMap chargée depuis capabilities.toml (Phase 1.7a),
	// injectée via WithCapabilities. nil → fallbackCapabilities (sécurité boot).
	staticCaps games.CapabilityMap
	logger     *slog.Logger
}

// NewDataAdapter construit un data adapter HI.
//
// career peut être nil si le titre est déclaré sans capability career — dans
// ce cas, LoadCareerSnapshot retourne ErrCapabilityNotSupported.
func NewDataAdapter(career CareerSource, logger *slog.Logger) *DataAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &DataAdapter{career: career, logger: logger}
}

// WithCapabilities injecte la CapabilityMap statique chargée depuis
// capabilities.toml (via mappings.Registry → games.CapabilityMapFromMappings).
// C'est le chemin nominal (Phase 1.7a) : la map TOML remplace le fallback codé.
func (a *DataAdapter) WithCapabilities(caps games.CapabilityMap) *DataAdapter {
	a.staticCaps = caps
	return a
}

// WithRecentSource câble la source du profil de combat récent (Explorer, HIGH-B).
// nil → LoadTargetRecentMatches retourne ErrCapabilityNotSupported. Chaînable.
func (a *DataAdapter) WithRecentSource(src RecentSource) *DataAdapter {
	a.recent = src
	return a
}

// TitleSlug retourne le slug HI canonique.
//
// Décision multi-titre (MT-20, PMT-13) : retourner DefaultSlug est l'IDENTITÉ
// PROPRE de l'adapter Halo Infinite (cet adapter EST Halo) — ce n'est pas un
// gating de titre. Le routage/gating par titre passe par le registre
// (HasCapability) et le Resolver d'adapters, jamais par cette valeur. Un futur
// adapter d'un autre jeu retournera SON slug ici, par le même pattern.
func (a *DataAdapter) TitleSlug() string { return titlePkg.DefaultSlug }

// Capabilities décrit l'état des capabilities HI exposées par cet adapter.
//
// Source nominale (Phase 1.7a) : la map injectée depuis capabilities.toml via
// WithCapabilities. À défaut (TOML non chargé), fallbackCapabilities sert de
// filet de sécurité boot. Dans les deux cas, career.progression est rétrogradée
// à not_exposed au runtime si aucune source carrière n'est câblée (on ne force
// jamais au-dessus de l'intention déclarée).
func (a *DataAdapter) Capabilities() games.CapabilityMap {
	base := a.staticCaps
	if base == nil {
		base = fallbackCapabilities()
	}
	out := make(games.CapabilityMap, len(base))
	for k, v := range base {
		out[k] = v
	}
	if a.career == nil && out[games.CapCareerProgression] != games.CapNotExposed {
		out[games.CapCareerProgression] = games.CapNotExposed
	}
	return out
}

// fallbackCapabilities est la CapabilityMap par défaut, utilisée UNIQUEMENT si
// capabilities.toml n'a pas pu être chargé/injecté (sécurité boot). Le chemin
// nominal passe par WithCapabilities (TOML). La parité fallback ⟷ TOML est
// garantie par capabilities_parity_test.go (toute divergence casse le test).
func fallbackCapabilities() games.CapabilityMap {
	return games.CapabilityMap{
		games.CapMatchHistory:       games.CapSupported,
		games.CapMatchDetailCore:    games.CapSupported,
		games.CapMatchSkillSnapshot: games.CapDegraded,
		games.CapCareerProgression:  games.CapSupported,
		games.CapPveFirefight:       games.CapSupported,
		games.CapTimeseries:         games.CapNotExposed,
		games.CapScoreboardExtra:    games.CapNotExposed,
		games.CapCitationsEngine:    games.CapNotExposed,
		games.CapEngagement:         games.CapSupported,
	}
}

// LoadMatchSummaries n'est pas câblée en Phase B. Elle remontera en Phase C.
func (a *DataAdapter) LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchDetail n'est pas câblée en Phase B.
func (a *DataAdapter) LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadPlayerStats n'est pas câblée en Phase B.
func (a *DataAdapter) LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, games.ErrCapabilityNotSupported
}

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

func projectEncounterRow(r domain.EncounterRawRow) canonical.EncounterRow {
	row := canonical.EncounterRow{
		Identity:   canonical.PlayerIdentity{XUID: r.XUID, Gamertag: r.Gamertag},
		MatchCount: r.MatchCount,
		AsTeammate: r.AsTeammate,
		AsEnemy:    r.AsEnemy,
	}
	if r.AvgKDA != nil {
		v := *r.AvgKDA
		row.AvgKDA = &v
	}
	return row
}

// LoadTimeseries n'est pas câblée en Phase B.
func (a *DataAdapter) LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadMatchScoreboard n'est pas encore câblée (Phase B+).
func (a *DataAdapter) LoadMatchScoreboard(ctx context.Context, matchID string) ([]canonical.MatchParticipant, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadHighlightEvents n'est pas encore câblée (Phase B+).
func (a *DataAdapter) LoadHighlightEvents(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}

// LoadFriendsXUIDs n'est pas encore câblée (Phase B+).
func (a *DataAdapter) LoadFriendsXUIDs(ctx context.Context, xuid string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}

// projectCareerSnapshot transforme un domain.CareerRankData en canonical.
// Garde la conversion isolée pour tester la projection sans toucher la DB.
func projectCareerSnapshot(xuid string, row *domain.CareerRankData) *canonical.CareerSnapshot {
	if row == nil {
		return &canonical.CareerSnapshot{Player: canonical.PlayerIdentity{XUID: xuid}}
	}

	xp := row.CurrentXP
	recordedAt := row.RecordedAt
	snap := &canonical.CareerSnapshot{
		Player:     canonical.PlayerIdentity{XUID: xuid},
		CurrentXP:  &xp,
		RankNumber: row.RankNumber,
		IsMaxRank:  row.IsMaxRank,
		RecordedAt: &recordedAt,
	}
	if row.XPForNextRank != nil {
		v := *row.XPForNextRank
		snap.XPForNextRank = &v
	}
	if row.XPTotal != nil {
		v := *row.XPTotal
		snap.XPTotal = &v
	}
	if row.RankTier != nil {
		v := *row.RankTier
		snap.RankTier = &v
	}
	if row.RankName != nil {
		v := *row.RankName
		snap.RankName = &v
	}
	if row.RankLabel != nil || row.RankName != nil {
		snap.CurrentRank = &canonical.AssetReference{
			Kind:         "career_rank",
			ID:           rankID(row),
			DefaultLabel: stringDeref(row.RankName, stringDeref(row.RankLabel, "")),
		}
	}
	return snap
}

func rankID(row *domain.CareerRankData) string {
	if row.RankLabel != nil {
		return *row.RankLabel
	}
	if row.RankName != nil {
		return *row.RankName
	}
	return ""
}

func stringDeref(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

// isNoRowsErr détecte les "pas de résultat" sans coupler le package au driver.
// La string match est volontairement permissive : sql.ErrNoRows s'imprime ainsi.
func isNoRowsErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, errSentinelNoRows) || matchesNoRowsString(err.Error())
}

// errSentinelNoRows est une sentinel locale comparable.
// On accepte aussi sql.ErrNoRows via duck-typing string pour ne pas importer
// database/sql dans games/halo_infinite (le projet utilise duckdb-go directement).
var errSentinelNoRows = errors.New("sql: no rows in result set")

func matchesNoRowsString(s string) bool {
	return s == "sql: no rows in result set"
}
