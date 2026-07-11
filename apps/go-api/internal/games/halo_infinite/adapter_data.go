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
	"fmt"
	"log/slog"
	"strings"
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

// ParticipantStatsSource est la surface de lecture de l'agrégat de stats d'un
// joueur sur un set de matchs (Explorer sample, HIGH-B). Implémentée par ExplorerRepo.
type ParticipantStatsSource interface {
	GetParticipantStatsForMatches(ctx context.Context, xuid string, matchIDs []string) (*domain.ParticipantStatsAggregate, error)
}

// CrossPlayerSource est la surface de lecture de l'intersection 2-joueurs (matchs
// communs + kills croisés ; Explorer, HIGH-B). Implémentée par ExplorerRepo.
type CrossPlayerSource interface {
	GetCommonMatches(ctx context.Context, xuid1, xuid2 string) ([]domain.CommonMatchRaw, error)
	GetKillerVictimBetween(ctx context.Context, xuid1, xuid2 string) (domain.KillerVictimAggregate, error)
}

// EventsSource est la surface minimale de lecture des events filmés bruts d'un
// match + sa timeline T0, pour reconstituer la timeline canonique (Canonical
// MatchEvents, Phase 2). Implémentée par internal/platform/duckdb.MatchEventsSource
// (highlight_events + match_registry). nil → LoadMatchEvents retourne
// ErrCapabilityNotSupported.
type EventsSource interface {
	LoadHighlightEvents(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error)
	GetMatchTimeline(ctx context.Context, matchID string) (domain.MatchTimeline, error)
}

// DataAdapter est l'implémentation HI de games.TitleDataAdapter.
//
// Phase B : seules les méthodes nécessaires aux endpoints prévus en Phase C
// sont câblées. Les autres retournent ErrCapabilityNotSupported pour rendre
// les absences explicites côté caller (cf. plan §5.7).
type DataAdapter struct {
	career      CareerSource
	recent      RecentSource           // Explorer profil de combat (Phase 2 HIGH-B). nil → ErrCapabilityNotSupported.
	participant ParticipantStatsSource // Explorer sample stats (HIGH-B). nil → ErrCapabilityNotSupported.
	cross       CrossPlayerSource      // Explorer intersection 2-joueurs (HIGH-B). nil → ErrCapabilityNotSupported.
	events      EventsSource           // Canonical MatchEvents (Phase 2). nil → ErrCapabilityNotSupported.
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

// WithParticipantSource câble la source d'agrégat sample stats (Explorer, HIGH-B).
// nil → LoadParticipantStats retourne ErrCapabilityNotSupported. Chaînable.
func (a *DataAdapter) WithParticipantSource(src ParticipantStatsSource) *DataAdapter {
	a.participant = src
	return a
}

// WithCrossPlayerSource câble la source d'intersection 2-joueurs (Explorer, HIGH-B).
// nil → LoadPlayerIntersection retourne ErrCapabilityNotSupported. Chaînable.
func (a *DataAdapter) WithCrossPlayerSource(src CrossPlayerSource) *DataAdapter {
	a.cross = src
	return a
}

// WithEventsSource câble la source de la timeline d'events (Canonical MatchEvents,
// Phase 2). nil → LoadMatchEvents retourne ErrCapabilityNotSupported. Chaînable.
func (a *DataAdapter) WithEventsSource(src EventsSource) *DataAdapter {
	a.events = src
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
		games.CapCareerRankCatalog:  games.CapSupported,
		games.CapPveFirefight:       games.CapSupported,
		games.CapTimeseries:         games.CapNotExposed,
		games.CapScoreboardExtra:    games.CapNotExposed,
		games.CapCitationsEngine:    games.CapNotExposed,
		games.CapEngagement:         games.CapSupported,
		games.CapBattlePass:         games.CapSupported,
		games.CapChallenges:         games.CapSupported,
		// Canonical MatchEvents (Phase 2 câblée) : timeline reconstruite depuis
		// highlight_events (degraded) ; arme-par-kill best-effort/absente (degraded) ;
		// positions monde non extraites (not_exposed). Cf. events.go + capabilities.toml.
		games.CapMatchEventsTimeline:  games.CapDegraded,
		games.CapMatchKillfeedPerKill: games.CapDegraded,
		games.CapMatchEventsSpatial:   games.CapNotExposed,
		// Précision par arme : pas d'events weapon_drop dans la timeline
		// reconstruite → table weapon_accuracy non peuplée (cf. capabilities.toml).
		games.CapWeaponAccuracy: games.CapNotExposed,
		// Libellés de playlist préfixés d'une catégorie matchmaking à retirer pour
		// l'affichage (analysis.NormalizePlaylistLabel) — trait Halo Infinite,
		// absent des autres titres (cf. capabilities.toml).
		games.CapPlaylistCategoryStrip: games.CapSupported,
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

// LoadParticipantStats wrappe ParticipantStatsSource.GetParticipantStatsForMatches
// et projette vers le canonique (Phase 2 HIGH-B). source nil →
// ErrCapabilityNotSupported. nil agg (set vide) → nil canonique.
func (a *DataAdapter) LoadParticipantStats(ctx context.Context, xuid string, matchIDs []string) (*canonical.PlayerMatchSetStats, error) {
	if a.participant == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "explorer.participant_stats",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	agg, err := a.participant.GetParticipantStatsForMatches(ctx, xuid, matchIDs)
	if err != nil {
		return nil, err
	}
	if agg == nil {
		return nil, nil
	}
	return projectParticipantStats(agg), nil
}

// projectParticipantStats projette l'agrégat domaine → canonique (DamageDealt/
// DamageTaken restent float64).
func projectParticipantStats(a *domain.ParticipantStatsAggregate) *canonical.PlayerMatchSetStats {
	return &canonical.PlayerMatchSetStats{
		Kills:             a.Kills,
		Deaths:            a.Deaths,
		Assists:           a.Assists,
		Wins:              a.Wins,
		Losses:            a.Losses,
		Draws:             a.Draws,
		ShotsFired:        a.ShotsFired,
		ShotsHit:          a.ShotsHit,
		DamageDealt:       a.DamageDealt,
		DamageTaken:       a.DamageTaken,
		HeadshotKills:     a.HeadshotKills,
		MeleeKills:        a.MeleeKills,
		PowerWeaponKills:  a.PowerWeaponKills,
		GrenadeKills:      a.GrenadeKills,
		TimePlayedSeconds: a.TimePlayedSeconds,
		PersonalScore:     a.PersonalScore,
	}
}

// LoadPlayerIntersection wrappe CrossPlayerSource (matchs communs + kills croisés)
// et projette vers le canonique (Phase 2 HIGH-B). source nil →
// ErrCapabilityNotSupported. Échec matchs communs = fatal (propagé) ; échec kills
// croisés = dégradation gracieuse (CrossKills vide + warn) — réplique le service legacy.
func (a *DataAdapter) LoadPlayerIntersection(ctx context.Context, selfXUID, otherXUID string) (*canonical.PlayerIntersection, error) {
	if a.cross == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", "explorer.cross_player",
		)
		return nil, games.ErrCapabilityNotSupported
	}

	matches, err := a.cross.GetCommonMatches(ctx, selfXUID, otherXUID)
	if err != nil {
		return nil, err // matchs communs = fatal
	}
	kv, kvErr := a.cross.GetKillerVictimBetween(ctx, selfXUID, otherXUID)
	if kvErr != nil {
		a.logger.WarnContext(ctx, "explorer_kv_between_failed",
			"xuid1", selfXUID, "xuid2", otherXUID, "err", kvErr)
		kv = domain.KillerVictimAggregate{} // gracieux : badges non calculés
	}

	out := &canonical.PlayerIntersection{
		CrossKills: canonical.CrossKillTally{KillsBySelf: kv.KillsDealt, KillsByOther: kv.DeathsSuffered},
	}
	if len(matches) > 0 {
		out.Matches = make([]canonical.CommonMatchRow, 0, len(matches))
		for _, m := range matches {
			out.Matches = append(out.Matches, projectCommonMatchRow(m))
		}
	}
	return out, nil
}

// projectCommonMatchRow projette une ligne de match commun domaine → canonique
// (team IDs deep-copiés ; SelfOutcomeCode = code BRUT).
func projectCommonMatchRow(m domain.CommonMatchRaw) canonical.CommonMatchRow {
	c := canonical.CommonMatchRow{
		MatchID:         m.MatchID,
		StartTime:       m.StartTime,
		MapUI:           m.MapUI,
		ModeUI:          m.ModeUI,
		SelfOutcomeCode: m.Player1Outcome,
		SelfKills:       m.Player1Kills,
		SelfDeaths:      m.Player1Deaths,
		SelfKDA:         m.Player1KDA,
	}
	if m.Player1TeamID != nil {
		v := *m.Player1TeamID
		c.SelfTeamID = &v
	}
	if m.Player2TeamID != nil {
		v := *m.Player2TeamID
		c.OtherTeamID = &v
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

// LoadMatchEvents reconstitue la timeline canonique d'events d'un match Infinite
// depuis highlight_events (+ appariement killer/victim, correction T0). Phase 2 du
// plan PLAN_CANONICAL_MATCH_EVENTS — arme-par-kill degraded, positions not_exposed
// (cf. infiniteEventLimitations + mapInfiniteEvents).
//
// Comportement :
//   - events source nil → ErrCapabilityNotSupported (adapter global non player-scopé) ;
//   - matchID vide → ErrCapabilityNotSupported ;
//   - échec lecture highlight_events → erreur propagée (problème DB) ;
//   - T0 indisponible → dégradation gracieuse (timeline non corrigée, T0=0) ;
//   - match sans events → timeline vide (Events=[]), pas d'erreur.
func (a *DataAdapter) LoadMatchEvents(ctx context.Context, matchID string, opts canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	if a.events == nil {
		a.logger.Warn("capability_not_supported",
			"title_slug", a.TitleSlug(),
			"capability", games.CapMatchEventsTimeline,
		)
		return nil, games.ErrCapabilityNotSupported
	}
	if strings.TrimSpace(matchID) == "" {
		return nil, games.ErrCapabilityNotSupported
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	raw, err := a.events.LoadHighlightEvents(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("hi LoadMatchEvents(%s): highlight events: %w", matchID, err)
	}

	tl, err := a.events.GetMatchTimeline(ctx, matchID)
	if err != nil {
		// T0 indisponible ne doit pas casser la timeline : on dégrade en T0=0
		// (events non corrigés du countdown) plutôt que d'échouer durement.
		a.logger.WarnContext(ctx, "match_timeline_unavailable",
			"title_slug", a.TitleSlug(), "match_id", matchID, "err", err)
		tl = domain.MatchTimeline{}
	}

	return &canonical.MatchEventTimeline{
		MatchID:     matchID,
		Events:      mapInfiniteEvents(raw, tl, opts),
		Limitations: infiniteEventLimitations(),
	}, nil
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
