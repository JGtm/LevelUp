// Package ops — season_catalog_refresh.go : persiste la liste des saisons CSR
// exposée par la page classement Waypoint (numéro + nom d'Operation + traduction
// FR) dans shared.season_catalog.
//
// ART-safe : écritures via duckdb.UpsertRowNoConflict (SELECT-then-write,
// canonique K1d/K1j), JAMAIS d'`INSERT ... ON CONFLICT DO UPDATE`. Table PK-only
// (pas d'index secondaire) → l'UPDATE ne touche pas de surface ART #23046. Même
// politique que refreshPlaylistsCatalog (cf. ADR 0019).
//
// Écrit dans la SHARED DB (pas metadata) : la source est le scrape Waypoint et le
// seul writer sanctionné détenu par world_leaderboard_cron est le writer shared —
// écrire metadata depuis ce cron contredirait le writer mono-process (contention
// sync, cf. A3). Le *sql.DB passé est donc le writer shared acquis.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// RefreshSeasonCatalog upsert (ART-safe) chaque saison dans shared.season_catalog.
// Idempotent : re-jouable à chaque cycle cron. Retourne le nombre de lignes traitées.
// db = writer shared acquis par l'appelant (fenêtre RW). NameFR est déjà résolu par
// le scraper (fallback DisplayName) ; ici on ne fait que persister + parser le n°.
func RefreshSeasonCatalog(ctx context.Context, db *sql.DB, titleSlug string, seasons []domain.WorldSeasonRef) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("RefreshSeasonCatalog: db nil")
	}
	n := 0
	for _, s := range seasons {
		id := strings.TrimSpace(s.SeasonID)
		if id == "" {
			continue
		}
		nameFR := strings.TrimSpace(s.NameFR)
		if nameFR == "" {
			nameFR = s.DisplayName
		}
		major, minor := parseWaypointSeasonNumber(id)
		if err := duckdb.UpsertRowNoConflict(ctx, db,
			`SELECT 1 FROM season_catalog WHERE title_slug = ? AND season_id = ?`,
			[]any{titleSlug, id},
			`UPDATE season_catalog SET
				display_name = ?, name_fr = ?, season_major = ?, season_minor = ?,
				last_fetched_at = CURRENT_TIMESTAMP
			 WHERE title_slug = ? AND season_id = ?`,
			[]any{s.DisplayName, nameFR, major, minor, titleSlug, id},
			`INSERT INTO season_catalog
				(title_slug, season_id, display_name, name_fr, season_major, season_minor, first_seen_at, last_fetched_at)
			 VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			[]any{titleSlug, id, s.DisplayName, nameFR, major, minor},
		); err != nil {
			slog.WarnContext(ctx, "upsert season_catalog", "season_id", id, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

// parseWaypointSeasonNumber extrait (major, minor) depuis "csrseason{major}-{minor}"
// (préfixe insensible à la casse ; le payload Waypoint est en minuscules). (0,0) si
// non parsable.
func parseWaypointSeasonNumber(seasonID string) (int, int) {
	s := strings.TrimSpace(seasonID)
	const prefix = "csrseason"
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return 0, 0
	}
	s = s[len(prefix):]
	majorStr, minorStr, _ := strings.Cut(s, "-")
	major, err := strconv.Atoi(strings.TrimSpace(majorStr))
	if err != nil {
		return 0, 0
	}
	minor, _ := strconv.Atoi(strings.TrimSpace(minorStr))
	return major, minor
}
