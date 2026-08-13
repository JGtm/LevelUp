// Package duckdb — match_view_repo_medals.go : médailles (joueur + bulk
// scoreboard) pour la vue Match. Découpé de match_view_repo.go
// (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
// Exécutée sur SharedReader (ADR 0016) — Q14 lit medals_earned (shared-only).
func (r *MatchViewRepo) GetMatchMedals(ctx context.Context, xuid, matchID string) ([]domain.MedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q14MatchMedals, xuid, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals: %w", err)
	}
	defer rows.Close()

	var results []domain.MedalRaw
	var medalIDs []int64
	for rows.Next() {
		var m domain.MedalRaw
		if err := rows.Scan(&m.MedalID, &m.Count); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchMedals scan: %w", err)
		}
		results = append(results, m)
		medalIDs = append(medalIDs, m.MedalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	meta := r.lookupMedalMeta(ctx, medalIDs)
	for index := range results {
		if m, ok := meta[results[index].MedalID]; ok {
			results[index].Label = m.label
			results[index].Description = m.description
			results[index].Difficulty = m.difficulty
		} else {
			results[index].Label = strconv.FormatInt(results[index].MedalID, 10)
		}
	}
	return results, nil
}

type medalMeta struct {
	label       string
	description string
	difficulty  string
}

// lookupMedalMeta résout label + description + difficulty depuis medal_definitions
// (chaîne BCP-47 medal_translations > medal_definitions), LOCALE-AWARE via la locale
// de requête (ctxkeys.Locale ← header X-LevelUp-Locale). En UI EN, ne jamais injecter
// les colonnes FR (name_fr/description_fr) — le drawer/résumé affichaient des noms FR
// sous UI EN (GH-5b). Source unique de la chaîne : medalLabelDescCoalesceSQL.
// Fallback citation_mappings.citation_name_display si la médaille n'est pas dans medal_definitions.
func (r *MatchViewRepo) lookupMedalMeta(ctx context.Context, medalIDs []int64) map[int64]medalMeta {
	result := make(map[int64]medalMeta, len(medalIDs))
	if len(medalIDs) == 0 || r.pdb.Metadata == nil {
		return result
	}
	locale := ctxkeys.Locale(ctx)
	labelExpr, descExpr := medalLabelDescCoalesceSQL(locale)
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        `+labelExpr+` AS label,
		        `+descExpr+` AS description,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty
		 FROM medal_definitions md
		 `+medalTranslationJoinsSQL(locale)+`
		 WHERE md.medal_name_id IN (%s)`,
		medalIDs,
	)
	if !ok {
		return result
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var label, desc, diff string
		if err := rows.Scan(&id, &label, &desc, &diff); err == nil && label != "" {
			result[id] = medalMeta{label: label, description: desc, difficulty: diff}
		}
	}
	// Fallback citation_mappings pour les IDs absents de medal_definitions
	// (source unique partagée, cf. medal_citation_fallback.go).
	missing := make([]int64, 0)
	for _, id := range medalIDs {
		if _, ok := result[id]; !ok {
			missing = append(missing, id)
		}
	}
	for id, label := range lookupMedalCitationLabels(ctx, r.pdb.Metadata, missing) {
		result[id] = medalMeta{label: label, difficulty: "Normal"}
	}
	return result
}

// LookupMedalMetaByName résout l'identité de médailles depuis leur nom ANGLAIS
// (medal_definitions.name_en, metadata.duckdb) — le pont entre les events `medal`
// du film (raw_json.medal_name, ex. « Odin's Raven ») et le référentiel du jeu.
// Locale-aware par la même chaîne que lookupMedalMeta (medalLabelDescCoalesceSQL).
// Best-effort : metadata absente ou requête en échec → map vide, jamais d'erreur
// (le kill feed affiche alors le nom brut plutôt que rien). La comparaison est
// stricte au caractère près : le nom du film EST la clé du référentiel (mesuré
// 21 noms/21 résolus sur 000d5950, apostrophe et esperluette comprises).
func (r *MatchViewRepo) LookupMedalMetaByName(
	ctx context.Context,
	namesEN []string,
) (map[string]domain.MedalNameMeta, error) {
	result := make(map[string]domain.MedalNameMeta, len(namesEN))
	if len(namesEN) == 0 || r.pdb.Metadata == nil {
		return result, nil
	}
	unique := make([]string, 0, len(namesEN))
	seen := make(map[string]struct{}, len(namesEN))
	for _, n := range namesEN {
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		unique = append(unique, n)
	}
	if len(unique) == 0 {
		return result, nil
	}
	locale := ctxkeys.Locale(ctx)
	labelExpr, descExpr := medalLabelDescCoalesceSQL(locale)
	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, n := range unique {
		placeholders[i] = "?"
		args[i] = n
	}
	q := `SELECT md.name_en, md.medal_name_id,
	             ` + labelExpr + ` AS label,
	             ` + descExpr + ` AS description
	      FROM medal_definitions md
	      ` + medalTranslationJoinsSQL(locale) + `
	      WHERE md.name_en IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "match_view: lookup médailles par nom en échec",
			"noms", len(unique), "err", err)
		return result, nil
	}
	defer rows.Close()
	for rows.Next() {
		var nameEN, label, desc string
		var id int64
		if err := rows.Scan(&nameEN, &id, &label, &desc); err != nil {
			slog.WarnContext(ctx, "match_view: scan médaille par nom en échec", "err", err)
			continue
		}
		result[nameEN] = domain.MedalNameMeta{MedalNameID: id, Label: label, Description: desc}
	}
	return result, nil
}

// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
// Exécutée sur SharedReader (ADR 0016) — Q27 lit medals_earned (shared-only).
func (r *MatchViewRepo) GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q27BulkMedals, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var results []domain.BulkMedalRaw
	var medalIDs []int64
	for rows.Next() {
		var m domain.BulkMedalRaw
		if err := rows.Scan(&m.XUID, &m.MedalID, &m.Count); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkMedals scan: %w", err)
		}
		results = append(results, m)
		medalIDs = append(medalIDs, m.MedalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	metas := r.lookupMedalMeta(ctx, medalIDs)
	for i := range results {
		if m, ok := metas[results[i].MedalID]; ok {
			results[i].Label = m.label
			results[i].Difficulty = m.difficulty
		} else {
			results[i].Label = strconv.FormatInt(results[i].MedalID, 10)
		}
	}
	return results, nil
}
