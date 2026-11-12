// Package profilerepo — les lectures DuckDB du profil de progression d'un joueur (Ascension), derrière
// port.ProfileStore.
//
// Les accès croisés (match_registry, match_participants, highlight_events) passent par le SharedReader du
// joueur (ADR 0016), sans préfixe `shared.` ; les tables du joueur (match_skill_rank_latest,
// personal_score_awards_latest, lusr_component_history_latest) se lisent sur sa base. Le paquet est un
// sous-paquet de platform/duckdb parce qu'il compose des lectures qui importent platform/duckdb
// (sync/skill pour les exclusions, platform/duckdb/prestige pour les modèles de défis).
package profilerepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
	duckdbprestige "levelup/go-api/internal/platform/duckdb/prestige"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/prestige"
)

// startTimeCanonicalMR : expression timezone-canonique du début de match pour l'alias `mr` — jamais de
// `start_time` brut dans un filtre/tri temporel (piège start_time_utc NULL / start_time naïf). Source
// unique : duckdb.StartTimeCanonicalSQL.
var startTimeCanonicalMR = duckdb.StartTimeCanonicalSQL("mr")

// Repo lit le profil d'un joueur dans ses bases.
type Repo struct {
	pdb *duckdb.PlayerDB
}

var _ port.ProfileStore = (*Repo)(nil)

// New lie le dépôt aux bases du joueur. pdb.Player est requis ; pdb.Metadata est facultatif (sans elle,
// aucune suggestion de défi).
func New(pdb *duckdb.PlayerDB) *Repo {
	return &Repo{pdb: pdb}
}

// TitleSlug rend le titre des bases du joueur.
func (r *Repo) TitleSlug() string { return r.pdb.TitleSlug }

// campaignExclMR retourne la clause littérale de masquage des matchs Campagne pour l'alias registre `mr`
// (title-aware via le titre du joueur ; no-op Infinite). Source unique analysis.SQLExcludeCampaignVariants :
// sans ce filtre les matchs Campagne (Halo 5) biaiseraient les axes et compteurs de la page Ascension.
func (r *Repo) campaignExclMR() string {
	return analysis.SQLExcludeCampaignVariants(r.pdb.TitleSlug, "mr")
}

// CountWindowMatches retourne le nombre de matchs distincts du joueur dans la fenêtre. Source :
// match_participants côté shared (filtré par xuid), lu via SharedReader (ADR 0016).
func (r *Repo) CountWindowMatches(ctx context.Context, userID string, since, until time.Time) (int, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("countMatchesInWindow: shared reader: %w", err)
	}
	defer release()
	var count int
	err = sharedDB.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT mp.match_id)
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?`+r.campaignExclMR()+`
			  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
		`, userID, since, until).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("countMatchesInWindow: %w", err)
	}
	return count, nil
}

// RadarBase calcule les moyennes match_participants du joueur sur la fenêtre (heuristique V1 du radar).
func (r *Repo) RadarBase(ctx context.Context, userID string, since, until time.Time) (port.ProfileRadarBase, error) {
	var (
		avgKills        sql.NullFloat64
		avgDeaths       sql.NullFloat64
		avgAssists      sql.NullFloat64
		avgScore        sql.NullFloat64
		avgKillingSpree sql.NullFloat64
		matchCount      int
	)
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return port.ProfileRadarBase{}, fmt.Errorf("computeRadarAxesBase: shared reader: %w", err)
	}
	defer release()
	err = sharedDB.QueryRowContext(ctx, `
			SELECT
				AVG(mp.kills),
				AVG(mp.deaths),
				AVG(mp.assists),
				AVG(mp.personal_score),
				AVG(mp.max_killing_spree),
				COUNT(*)
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?`+r.campaignExclMR()+`
			  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
		`, userID, since, until).Scan(&avgKills, &avgDeaths, &avgAssists, &avgScore, &avgKillingSpree, &matchCount)
	if err != nil {
		return port.ProfileRadarBase{}, fmt.Errorf("computeRadarAxesBase: %w", err)
	}
	return port.ProfileRadarBase{
		AvgKills:        nullFloat(avgKills),
		AvgDeaths:       nullFloat(avgDeaths),
		AvgAssists:      nullFloat(avgAssists),
		AvgScore:        nullFloat(avgScore),
		AvgKillingSpree: nullFloat(avgKillingSpree),
		MatchCount:      matchCount,
	}, nil
}

func nullFloat(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}

// WindowMatchIDs retourne les match_id du registre partagé dans la fenêtre (Campagne exclue).
func (r *Repo) WindowMatchIDs(ctx context.Context, since, until time.Time) ([]string, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	matchRows, err := sharedDB.QueryContext(ctx, `
			SELECT match_id FROM match_registry mr
			WHERE `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?`+r.campaignExclMR()+`
		`, since, until)
	if err != nil {
		return nil, err
	}
	defer matchRows.Close()
	var matchIDs []string
	for matchRows.Next() {
		var id string
		if err := matchRows.Scan(&id); err == nil && id != "" {
			matchIDs = append(matchIDs, id)
		}
	}
	return matchIDs, nil
}

// PlayerWindowMatchIDs rend les match_id où le joueur a participé dans la fenêtre (Campagne exclue), lus sur
// la base partagée.
func (r *Repo) PlayerWindowMatchIDs(ctx context.Context, userID string, since, until time.Time) ([]string, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
			SELECT DISTINCT mp.match_id
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?`+r.campaignExclMR()+`
			  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
		`, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("requête: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lecture: %w", err)
	}
	return ids, nil
}

// MatchStartTimes rend les instants de début (canoniques) des matchs du joueur dans la fenêtre, croissants.
func (r *Repo) MatchStartTimes(ctx context.Context, userID string, since, until time.Time) ([]time.Time, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("computeEngagementSimple: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
			SELECT `+startTimeCanonicalMR+`
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?`+r.campaignExclMR()+`
			  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
			ORDER BY `+startTimeCanonicalMR+` ASC
		`, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("computeEngagementSimple: %w", err)
	}
	defer rows.Close()
	var times []time.Time
	for rows.Next() {
		var t sql.NullTime
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		if t.Valid {
			times = append(times, t.Time)
		}
	}
	if rerr := rows.Err(); rerr != nil {
		return nil, rerr
	}
	return times, nil
}

// MatchDays rend (match, début canonique) des matchs du joueur dans la fenêtre, pour le calendrier
// d'activité (motif A6 : le bucket jour se calcule en Go, pas par un CAST DATE côté DuckDB).
func (r *Repo) MatchDays(ctx context.Context, userID string, since, until time.Time) ([]port.ProfileMatchTime, error) {
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadActivityCalendar: shared reader: %w", err)
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, `
			SELECT mp.match_id, `+startTimeCanonicalMR+`
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = ?`+r.campaignExclMR()+`
			  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
		`, userID, since, until)
	if err != nil {
		return nil, fmt.Errorf("LoadActivityCalendar: %w", err)
	}
	defer rows.Close()
	var out []port.ProfileMatchTime
	for rows.Next() {
		var (
			matchID string
			at      sql.NullTime
		)
		if err := rows.Scan(&matchID, &at); err != nil {
			return nil, err
		}
		if !at.Valid {
			continue
		}
		out = append(out, port.ProfileMatchTime{MatchID: matchID, At: at.Time})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ObjectiveIndexInputs charge les agrégats de l'index des objectifs (vue match_objective_stats_latest).
func (r *Repo) ObjectiveIndexInputs(ctx context.Context, matchIDs, xuids []string) (map[string]narrative.ObjectiveIndexInput, error) {
	return duckdb.NewObjectiveStatsRepo(r.pdb).LoadObjectiveIndexInputs(ctx, matchIDs, xuids)
}

// KillDeathEvents charge les événements de frag et de mort par le chargeur unifié (highlight_events pour
// Infinite ; repli sur les frags horodatés du titre quand highlight_events n'en porte pas).
func (r *Repo) KillDeathEvents(ctx context.Context, matchIDs []string) ([]canonical.HighlightEvent, error) {
	return duckdb.NewHighlightEventsRepo(r.pdb).Load(ctx, port.HighlightEventFilters{
		MatchIDs:   matchIDs,
		EventTypes: []canonical.HighlightEventType{canonical.EventKill, canonical.EventDeath},
	})
}

// SuggestionTemplates rend les modèles du titre (référentiels), textes par langue compris.
func (r *Repo) SuggestionTemplates(ctx context.Context, titleSlug string) ([]prestige.Template, bool, error) {
	if r.pdb.Metadata == nil {
		return nil, false, nil
	}
	templates, err := duckdbprestige.NewPrestigeTemplateRepo(r.pdb.Metadata).ListByTitle(ctx, titleSlug)
	return templates, true, err
}

// ForPlayer rend le magasin de profil des bases d'un joueur, ou nil quand elles sont inutilisables (pdb nil
// ou sans base joueur) : le service construit dessus refuse alors BuildProfile.
func ForPlayer(pdb *duckdb.PlayerDB) port.ProfileStore {
	if pdb == nil || pdb.Player == nil {
		return nil
	}
	return New(pdb)
}
