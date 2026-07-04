// Package duckdb — SquadMatchProvider pour le titre Halo Infinite.
//
// Implémente prestige.SquadMatchProvider : pour un roster d'escouade, retourne
// les derniers matchs où TOUT le roster a joué, avec tous les participants (pour
// la règle no-overlap) et la métrique du défi par membre. Lecture sur
// shared_matches_v2.duckdb via un SharedReader (coordination RO↔RW).

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/prestige"
)

// PrestigeSquadMatchProvider lit match_participants pour l'évaluation des défis
// d'escouade.
type PrestigeSquadMatchProvider struct {
	reader SharedReader
}

// NewPrestigeSquadMatchProvider construit le provider depuis un SharedReader
// (passer pdb.SharedReadDB() côté caller, comme HaloBaselineProvider).
func NewPrestigeSquadMatchProvider(reader SharedReader) *PrestigeSquadMatchProvider {
	return &PrestigeSquadMatchProvider{reader: reader}
}

var _ prestige.SquadMatchProvider = (*PrestigeSquadMatchProvider)(nil)

// SquadMatchMetrics retourne les `limit` derniers matchs où tout le roster a
// joué, chacun avec ses participants (Xuids) + la métrique par membre (Values).
func (p *PrestigeSquadMatchProvider) SquadMatchMetrics(ctx context.Context, rosterXUIDs []string, _ string, metric string, limit int) ([]prestige.SquadMatchMetric, error) {
	if len(rosterXUIDs) == 0 {
		return nil, nil
	}
	col := mapMetricToColumn(metric)
	if col == "" {
		return nil, nil // métrique non mappée → pas d'évaluation possible
	}
	if limit <= 0 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider: shared reader: %w", err)
	}
	defer release()

	candidates, err := p.candidateMatches(ctx, db, rosterXUIDs, limit)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	return p.participantsWithMetric(ctx, db, candidates, col)
}

// candidateMatches retourne les match_id (les plus récents) où TOUT le roster a
// joué : COUNT(DISTINCT xuid parmi le roster) == taille du roster.
func (p *PrestigeSquadMatchProvider) candidateMatches(ctx context.Context, db *sql.DB, roster []string, limit int) ([]string, error) {
	q := fmt.Sprintf(`
		SELECT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid IN (%s)
		GROUP BY mp.match_id, `+StartTimeCanonicalSQL("mr")+`
		HAVING COUNT(DISTINCT mp.xuid) = %d
		ORDER BY `+StartTimeCanonicalSQL("mr")+` DESC
		LIMIT %d
	`, sqlInPlaceholders(len(roster)), len(roster), limit)

	rows, err := db.QueryContext(ctx, q, toAnyArgs(roster)...)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider.candidateMatches: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// participantsWithMetric retourne, pour les matchs donnés, TOUS les participants
// + la valeur de la colonne métrique par xuid.
func (p *PrestigeSquadMatchProvider) participantsWithMetric(ctx context.Context, db *sql.DB, matchIDs []string, col string) ([]prestige.SquadMatchMetric, error) {
	q := fmt.Sprintf(`
		SELECT mp.match_id, mp.xuid, COALESCE(CAST(%s AS DOUBLE), 0)
		FROM match_participants mp
		WHERE mp.match_id IN (%s)
	`, col, sqlInPlaceholders(len(matchIDs)))

	rows, err := db.QueryContext(ctx, q, toAnyArgs(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("PrestigeSquadMatchProvider.participantsWithMetric: %w", err)
	}
	defer rows.Close()

	byMatch := make(map[string]*prestige.SquadMatchMetric)
	var order []string
	for rows.Next() {
		var matchID, xuid string
		var val float64
		if err := rows.Scan(&matchID, &xuid, &val); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		sm, ok := byMatch[matchID]
		if !ok {
			sm = &prestige.SquadMatchMetric{MatchID: matchID, Values: map[string]float64{}}
			byMatch[matchID] = sm
			order = append(order, matchID)
		}
		sm.Xuids = append(sm.Xuids, xuid)
		sm.Values[xuid] = val
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]prestige.SquadMatchMetric, 0, len(order))
	for _, id := range order {
		out = append(out, *byMatch[id])
	}
	return out, nil
}

// uuidLabelRE détecte un label non résolu (UUID brut) à écarter de l'indice
// d'affichage (miroir du filtre UUID côté front pour les options de cascade).
var uuidLabelRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// SquadUsualContexts retourne les playlists/modes dominants (top 2) parmi les
// `limit` derniers matchs où TOUT le roster a joué ensemble. Labels résolus FR
// via v_match_full ; vides et UUID non résolus écartés. Lecture seule.
func (p *PrestigeSquadMatchProvider) SquadUsualContexts(ctx context.Context, rosterXUIDs []string, _ string, limit int) (playlists, modes []string, err error) {
	if len(rosterXUIDs) == 0 {
		return nil, nil, nil
	}
	if limit <= 0 {
		limit = 60
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := p.reader.Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: shared reader indisponible (indice escouade dégradé)",
			"err", err, "roster_size", len(rosterXUIDs))
		return nil, nil, fmt.Errorf("SquadUsualContexts: shared reader: %w", err)
	}
	defer release()

	q := fmt.Sprintf(`
		WITH cm AS (
			SELECT mp.match_id,
			       MAX(`+StartTimeCanonicalSQL("mr")+`) AS st
			FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid IN (%s)
			GROUP BY mp.match_id
			HAVING COUNT(DISTINCT mp.xuid) = %d
			ORDER BY st DESC
			LIMIT %d
		)
		SELECT
			COALESCE(NULLIF(r.playlist_name_fr, ''), r.playlist_name, '') AS playlist,
			COALESCE(NULLIF(r.pair_name_fr, ''), r.pair_name, '')         AS mode
		FROM cm
		JOIN v_match_full r ON r.match_id = cm.match_id
	`, sqlInPlaceholders(len(rosterXUIDs)), len(rosterXUIDs), limit)

	rows, err := db.QueryContext(ctx, q, toAnyArgs(rosterXUIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "SquadUsualContexts: échec requête (indice escouade dégradé)",
			"err", err, "roster_size", len(rosterXUIDs))
		return nil, nil, fmt.Errorf("SquadUsualContexts: query: %w", err)
	}
	defer rows.Close()

	plCount := map[string]int{}
	mdCount := map[string]int{}
	for rows.Next() {
		var pl, md string
		if err := rows.Scan(&pl, &md); err != nil {
			return nil, nil, fmt.Errorf("SquadUsualContexts: scan: %w", err)
		}
		if pl != "" && !uuidLabelRE.MatchString(pl) {
			plCount[pl]++
		}
		if md != "" && !uuidLabelRE.MatchString(md) {
			mdCount[md]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return topNByFreq(plCount, 2), topNByFreq(mdCount, 2), nil
}

// topNByFreq retourne les n clés les plus fréquentes (fréquence desc, départage
// alphabétique pour un ordre déterministe).
func topNByFreq(counts map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(counts))
	for k, v := range counts {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].v != arr[j].v {
			return arr[i].v > arr[j].v
		}
		return arr[i].k < arr[j].k
	})
	if n > len(arr) {
		n = len(arr)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, arr[i].k)
	}
	return out
}

// sqlInPlaceholders construit "?,?,…" pour une clause IN de n éléments.
func sqlInPlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// toAnyArgs convertit []string en []any pour QueryContext.
func toAnyArgs(xs []string) []any {
	args := make([]any, len(xs))
	for i, x := range xs {
		args[i] = x
	}
	return args
}
