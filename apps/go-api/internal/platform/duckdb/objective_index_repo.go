// Package duckdb — objective_index_repo.go : agrégats par (xuid × famille de mode)
// sur match_objective_stats_latest pour l'index de participation aux objectifs
// (narrative.ComputeObjectiveIndex — plan .ai/PLAN_AXE_OBJECTIFS_INDEX.md, étape 4).
//
// Les listes de colonnes NE SONT PAS dupliquées ici : le SELECT est généré depuis
// les tables exportées narrative.ObjectiveFamilyActionWeights /
// ObjectiveFamilyHoldColumns (source unique). La cohérence clés ⊆ schéma est
// verrouillée par TestObjectiveIndexRepo_WeightKeysSubsetOfSchema.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/narrative"
)

// objectiveIndexMinTimePlayedSeconds : filtre population aligné sur la calibration
// des P80 (étape 2 du plan — time_played_seconds > 30, hors lignes fantômes DNF).
const objectiveIndexMinTimePlayedSeconds = 30

// objectiveFamilyCaseSQL retourne l'expression CASE de classification d'une ligne
// vers sa famille (blocs mutuellement exclusifs par mode ; split KOTH/Strongholds
// par zone_scoring_ticks > 0 AU NIVEAU LIGNE — décision 3 du plan). Discriminants
// alignés sur domain.ObjectiveRaw.HasCTF/HasZones/…/HasVip.
func objectiveFamilyCaseSQL() string {
	return fmt.Sprintf(`CASE
		WHEN COALESCE(o.zone_scoring_ticks, 0) > 0 THEN '%s'
		WHEN o.zone_captures IS NOT NULL OR o.zone_scoring_ticks IS NOT NULL THEN '%s'
		WHEN o.flag_grabs IS NOT NULL OR o.flag_captures IS NOT NULL THEN '%s'
		WHEN o.skull_grabs IS NOT NULL OR o.skull_scoring_ticks IS NOT NULL THEN '%s'
		WHEN o.power_seeds_deposited IS NOT NULL OR o.power_seed_carriers_killed IS NOT NULL THEN '%s'
		WHEN o.successful_extractions IS NOT NULL OR o.extraction_initiations_completed IS NOT NULL THEN '%s'
		WHEN o.times_selected_as_vip IS NOT NULL OR o.kills_as_vip IS NOT NULL THEN '%s'
		ELSE NULL
	END`,
		narrative.FamilyZonesKOTH, narrative.FamilyZonesStrongholds, narrative.FamilyCTF,
		narrative.FamilyOddball, narrative.FamilyStockpile, narrative.FamilyExtraction,
		narrative.FamilyVIP)
}

// objectiveIndexSelectColumns retourne les colonnes d'action et de durée à sommer,
// dans un ordre DÉTERMINISTE (l'ordre du SELECT et du scan en dépend), dérivées des
// tables narrative (source unique, pas de 6e liste de colonnes).
func objectiveIndexSelectColumns() (actionCols, holdCols []string) {
	seen := map[string]bool{}
	for _, fam := range narrative.AllObjectiveFamilies() {
		cols := make([]string, 0, len(narrative.ObjectiveFamilyActionWeights[fam]))
		for col := range narrative.ObjectiveFamilyActionWeights[fam] {
			cols = append(cols, col)
		}
		sort.Strings(cols)
		for _, c := range cols {
			if !seen[c] {
				seen[c] = true
				actionCols = append(actionCols, c)
			}
		}
	}
	seen = map[string]bool{}
	for _, fam := range narrative.AllObjectiveFamilies() {
		for _, c := range narrative.ObjectiveFamilyHoldColumns[fam] {
			if !seen[c] {
				seen[c] = true
				holdCols = append(holdCols, c)
			}
		}
	}
	return actionCols, holdCols
}

// LoadObjectiveIndexInputs retourne, par xuid, les agrégats par famille attendus
// par narrative.ComputeObjectiveIndex sur le scope fermé de matchs donné.
// Lecture via la vue append-only match_objective_stats_latest UNIQUEMENT (ADR
// 0026), JOIN match_participants pour n_f et Σ time_played. Best-effort : une
// erreur de requête (vue absente sur une DB non migrée, etc.) dégrade en map
// vide + warn — jamais d'échec dur d'une page (l'axe Objectif est alors retiré).
func (r *ObjectiveStatsRepo) LoadObjectiveIndexInputs(
	ctx context.Context, matchIDs, xuids []string,
) (map[string]narrative.ObjectiveIndexInput, error) {
	if len(matchIDs) == 0 || len(xuids) == 0 {
		return map[string]narrative.ObjectiveIndexInput{}, nil
	}
	keyClause := " AND o.xuid IN (" + Placeholders(len(xuids)) + ")"
	keyArgs := make([]any, 0, len(xuids))
	for _, x := range xuids {
		keyArgs = append(keyArgs, x)
	}
	return r.loadObjectiveIndexInputs(ctx, matchIDs, keyClause, keyArgs)
}

// LoadObjectiveIndexInputsByGamertag est la variante gamertag (radar synergie
// Escouade : les coéquipiers NON suivis n'ont pas de player DB — résolution via
// shared.xuid_aliases, même patron que WeaponKillsRepo.appendXUIDFilter).
// Retourne l'input agrégé du joueur (toutes familles), vide si inconnu.
func (r *ObjectiveStatsRepo) LoadObjectiveIndexInputsByGamertag(
	ctx context.Context, matchIDs []string, gamertag string,
) (narrative.ObjectiveIndexInput, error) {
	if len(matchIDs) == 0 || gamertag == "" {
		return narrative.ObjectiveIndexInput{}, nil
	}
	keyClause := ` AND o.xuid IN (SELECT xuid FROM xuid_aliases WHERE gamertag = ?)`
	byXUID, err := r.loadObjectiveIndexInputs(ctx, matchIDs, keyClause, []any{gamertag})
	if err != nil {
		return narrative.ObjectiveIndexInput{}, err
	}
	// Un gamertag peut mapper plusieurs xuid_aliases historiques : fusionner.
	out := narrative.ObjectiveIndexInput{}
	for _, input := range byXUID {
		for fam, st := range input {
			out[fam] = mergeObjectiveFamilyStats(out[fam], st)
		}
	}
	return out, nil
}

// mergeObjectiveFamilyStats additionne deux agrégats d'une même famille.
func mergeObjectiveFamilyStats(a, b narrative.ObjectiveFamilyStats) narrative.ObjectiveFamilyStats {
	if a.Matches == 0 && a.ColumnSums == nil {
		return b
	}
	a.Matches += b.Matches
	a.HoldSeconds += b.HoldSeconds
	a.TimePlayedSeconds += b.TimePlayedSeconds
	a.TimesSelectedAsVIP += b.TimesSelectedAsVIP
	if a.ColumnSums == nil {
		a.ColumnSums = map[string]float64{}
	}
	for col, v := range b.ColumnSums {
		a.ColumnSums[col] += v
	}
	return a
}

// loadObjectiveIndexInputs exécute l'agrégat (xuid × famille) avec le filtre de
// clé fourni et assemble les ObjectiveFamilyStats.
func (r *ObjectiveStatsRepo) loadObjectiveIndexInputs(
	ctx context.Context, matchIDs []string, keyClause string, keyArgs []any,
) (map[string]narrative.ObjectiveIndexInput, error) {
	out := map[string]narrative.ObjectiveIndexInput{}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()

	actionCols, holdCols := objectiveIndexSelectColumns()
	var sb strings.Builder
	sb.WriteString("SELECT o.xuid, ")
	sb.WriteString(objectiveFamilyCaseSQL())
	sb.WriteString(` AS family,
		COUNT(*)::INTEGER AS n,
		COALESCE(SUM(mp.time_played_seconds), 0)::DOUBLE AS tp,
		COALESCE(SUM(o.times_selected_as_vip), 0)::DOUBLE AS vip_sel`)
	for _, c := range actionCols {
		sb.WriteString(",\n\t\tCOALESCE(SUM(o." + c + "), 0)::DOUBLE")
	}
	for _, c := range holdCols {
		sb.WriteString(",\n\t\tCOALESCE(SUM(o." + c + "), 0)::DOUBLE")
	}
	sb.WriteString(`
		FROM match_objective_stats_latest o
		JOIN match_participants mp ON mp.match_id = o.match_id AND mp.xuid = o.xuid
		WHERE o.match_id IN (` + Placeholders(len(matchIDs)) + `)` + keyClause + `
		  AND COALESCE(mp.time_played_seconds, 0) > ` + fmt.Sprintf("%d", objectiveIndexMinTimePlayedSeconds) + `
		GROUP BY o.xuid, family`)

	args := make([]any, 0, len(matchIDs)+len(keyArgs))
	for _, id := range matchIDs {
		args = append(args, id)
	}
	args = append(args, keyArgs...)

	rows, err := db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		slog.WarnContext(ctx, "ObjectiveStatsRepo: index inputs query failed (best-effort)",
			"match_count", len(matchIDs), "err", err)
		return out, nil
	}
	defer rows.Close()

	nCols := len(actionCols) + len(holdCols)
	for rows.Next() {
		row := objectiveIndexRow{sums: make([]float64, nCols)}
		if err := rows.Scan(row.scanPtrs()...); err != nil {
			slog.WarnContext(ctx, "ObjectiveStatsRepo: index inputs scan failed", "err", err)
			continue
		}
		fam, st, ok := row.stats(actionCols, holdCols)
		if !ok {
			continue
		}
		if out[row.xuid] == nil {
			out[row.xuid] = narrative.ObjectiveIndexInput{}
		}
		out[row.xuid][fam] = st
	}
	return out, rows.Err()
}

// objectiveIndexRow porte une ligne brute de l'agrégat (xuid × famille) : les
// sommes arrivent dans l'ordre DÉTERMINISTE du SELECT (actionCols puis holdCols,
// cf. objectiveIndexSelectColumns).
type objectiveIndexRow struct {
	xuid   string
	family *string
	n      int
	tp     float64
	vipSel float64
	sums   []float64
}

// scanPtrs retourne les cibles de rows.Scan dans l'ordre du SELECT. sums doit
// être pré-alloué à la taille attendue (len(actionCols)+len(holdCols)).
func (row *objectiveIndexRow) scanPtrs() []any {
	ptrs := make([]any, 0, 5+len(row.sums))
	ptrs = append(ptrs, &row.xuid, &row.family, &row.n, &row.tp, &row.vipSel)
	for i := range row.sums {
		ptrs = append(ptrs, &row.sums[i])
	}
	return ptrs
}

// stats convertit la ligne en agrégat narrative. ok == false ⇒ ligne hors
// périmètre, à ignorer : famille NULL (aucun bloc objectif), n <= 0, ou famille
// absente de la table de poids.
func (row objectiveIndexRow) stats(actionCols, holdCols []string) (narrative.ObjectiveFamily, narrative.ObjectiveFamilyStats, bool) {
	if row.family == nil || row.n <= 0 {
		return "", narrative.ObjectiveFamilyStats{}, false
	}
	fam := narrative.ObjectiveFamily(*row.family)
	weights, ok := narrative.ObjectiveFamilyActionWeights[fam]
	if !ok {
		return "", narrative.ObjectiveFamilyStats{}, false
	}
	byCol := make(map[string]float64, len(row.sums))
	for i, c := range actionCols {
		byCol[c] = row.sums[i]
	}
	for i, c := range holdCols {
		byCol[c] = row.sums[len(actionCols)+i]
	}
	st := narrative.ObjectiveFamilyStats{
		Matches:           row.n,
		TimePlayedSeconds: row.tp,
		ColumnSums:        make(map[string]float64, len(weights)),
	}
	for col := range weights {
		st.ColumnSums[col] = byCol[col]
	}
	for _, hc := range narrative.ObjectiveFamilyHoldColumns[fam] {
		st.HoldSeconds += byCol[hc]
	}
	if fam == narrative.FamilyVIP {
		st.TimesSelectedAsVIP = row.vipSel
	}
	return fam, st, true
}
