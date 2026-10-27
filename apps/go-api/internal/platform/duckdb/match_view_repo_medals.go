// Package duckdb — match_view_repo_medals.go : médailles (joueur + bulk
// scoreboard) pour la vue Match. Découpé de match_view_repo.go
// (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// GetMatchMedals retourne les médailles du joueur dans ce match (Q14).
// Exécutée sur SharedReader (ADR 0016) — Q14 lit medals_earned (shared-only).
//
// Les médailles sont servies même si la résolution de leurs libellés échoue ; elle est alors dite par
// une *domain.SectionDegradedError (libellés repliés sur l'identifiant).
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

	meta, metaErr := r.lookupMedalMeta(ctx, medalIDs)
	for index := range results {
		if m, ok := meta[results[index].MedalID]; ok {
			results[index].Label = m.label
			results[index].Description = m.description
			results[index].Difficulty = m.difficulty
		} else {
			results[index].Label = strconv.FormatInt(results[index].MedalID, 10)
		}
	}
	return results, medalLabelsDegraded(metaErr)
}

// medalLabelsDegraded : une résolution de libellés en échec laisse les médailles servies (comptes
// exacts, libellés repliés sur l'identifiant) mais DIT la panne, au lieu de la faire passer pour un
// référentiel sans nom. nil quand la résolution a réussi.
func medalLabelsDegraded(err error) error {
	if err == nil {
		return nil
	}
	return &domain.SectionDegradedError{Section: "medal_labels", Err: err}
}

type medalMeta struct {
	label       string
	description string
	difficulty  string
}

// lookupMedalMeta résout label + description + difficulty : difficulty vient de medal_definitions,
// label et description de medal_translations dans la langue de requête (ctxkeys.Locale ← header
// X-LevelUp-Locale, repli `en`, jamais la langue par défaut : source unique loadMedalLabels).
// Fallback citation_mappings.citation_name_display si la médaille n'a aucun nom.
//
// Rend les libellés résolus ET, si une lecture du référentiel a échoué, l'erreur : l'appelant
// garde ce qui est résolu mais doit dire la panne. Un référentiel non configuré (`Metadata` nil)
// ou sans identifiant à résoudre rend une map vide sans erreur.
func (r *MatchViewRepo) lookupMedalMeta(ctx context.Context, medalIDs []int64) (map[int64]medalMeta, error) {
	result := make(map[int64]medalMeta, len(medalIDs))
	if len(medalIDs) == 0 || r.pdb.Metadata == nil {
		return result, nil
	}
	difficulties, err := r.medalDifficulties(ctx, medalIDs)
	if err != nil {
		return result, err
	}
	labels, err := loadMedalLabels(ctx, r.pdb.Metadata, medalIDs, ctxkeys.Locale(ctx))
	if err != nil {
		return result, fmt.Errorf("lookupMedalMeta: medal_translations: %w", err)
	}
	for id, diff := range difficulties {
		if t := labels[id]; t.label != "" {
			result[id] = medalMeta{label: t.label, description: t.description, difficulty: diff}
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
	citations, err := lookupMedalCitationLabels(ctx, r.pdb.Metadata, missing)
	for id, label := range citations {
		result[id] = medalMeta{label: label, difficulty: "Normal"}
	}
	if err != nil {
		return result, fmt.Errorf("lookupMedalMeta: %w", err)
	}
	return result, nil
}

// medalDifficulties lit la difficulté de medal_definitions pour les médailles `medalIDs` (colonne
// non textuelle) ; le curseur est fermé au retour.
func (r *MatchViewRepo) medalDifficulties(ctx context.Context, medalIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(medalIDs))
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id, COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty
		 FROM medal_definitions md
		 WHERE md.medal_name_id IN (%s)`,
		medalIDs,
	)
	if !ok {
		return out, nil
	}
	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return out, fmt.Errorf("lookupMedalMeta: medal_definitions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var diff string
		if err := rows.Scan(&id, &diff); err != nil {
			return out, fmt.Errorf("lookupMedalMeta: medal_definitions scan: %w", err)
		}
		out[id] = diff
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("lookupMedalMeta: medal_definitions rows: %w", err)
	}
	return out, nil
}

// LookupMedalMetaByName résout l'identité de médailles depuis leur nom ANGLAIS
// (medal_translations, étiquettes Halo de la langue de repli) — le pont entre les events `medal`
// du film (raw_json.medal_name, ex. « Odin's Raven ») et le référentiel du jeu.
// Label et description suivent la langue de la requête par loadMedalLabels, comme lookupMedalMeta.
// Metadata non configurée → map vide sans erreur (absence constatée) ; une requête ou une lecture
// en échec REMONTE : l'appelant (décoration du kill feed) la journalise avant de laisser le feed
// sur le nom brut. La comparaison est
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
	idsByName, err := r.medalIDsByEnglishName(ctx, unique)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(idsByName))
	for _, id := range idsByName {
		ids = append(ids, id)
	}
	labels, err := loadMedalLabels(ctx, r.pdb.Metadata, ids, ctxkeys.Locale(ctx))
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.LookupMedalMetaByName: textes: %w", err)
	}
	for name, id := range idsByName {
		t := labels[id]
		result[name] = domain.MedalNameMeta{MedalNameID: id, Label: t.label, Description: t.description}
	}
	return result, nil
}

// medalIDsByEnglishName associe chaque nom anglais de `names` au medal_name_id de la médaille qui
// le porte dans medal_translations (étiquettes Halo de la langue de repli).
func (r *MatchViewRepo) medalIDsByEnglishName(ctx context.Context, names []string) (map[string]int64, error) {
	tags := i18n.HaloTags(i18n.Fallback)
	args := make([]interface{}, 0, len(tags)+len(names))
	for _, tag := range tags {
		args = append(args, tag)
	}
	for _, n := range names {
		args = append(args, n)
	}
	rows, err := r.pdb.Metadata.Query(ctx,
		`SELECT mt.name, mt.medal_name_id
		 FROM medal_translations mt
		 WHERE mt.lang IN (`+Placeholders(len(tags))+`)
		   AND mt.name IN (`+Placeholders(len(names))+`)
		   AND EXISTS (SELECT 1 FROM medal_definitions md WHERE md.medal_name_id = mt.medal_name_id)
		 ORDER BY mt.medal_name_id`, args...)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.LookupMedalMetaByName: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int64, len(names))
	for rows.Next() {
		var name string
		var id int64
		if err := rows.Scan(&name, &id); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.LookupMedalMetaByName scan: %w", err)
		}
		out[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("MatchViewRepo.LookupMedalMetaByName rows: %w", err)
	}
	return out, nil
}

// GetMatchBulkMedals retourne les médailles de tous les joueurs du match (Q27).
// Exécutée sur SharedReader (ADR 0016) — Q27 lit medals_earned (shared-only).
func (r *MatchViewRepo) GetMatchBulkMedals(ctx context.Context, matchID string) ([]domain.BulkMedalRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkMedals: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q27BulkMedals, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkMedals: Q27: %w", err)
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

	metas, metaErr := r.lookupMedalMeta(ctx, medalIDs)
	for i := range results {
		if m, ok := metas[results[i].MedalID]; ok {
			results[i].Label = m.label
			results[i].Difficulty = m.difficulty
		} else {
			results[i].Label = strconv.FormatInt(results[i].MedalID, 10)
		}
	}
	return results, medalLabelsDegraded(metaErr)
}
