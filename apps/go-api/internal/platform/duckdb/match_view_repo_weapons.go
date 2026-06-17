// Package duckdb — match_view_repo_weapons.go : kills par arme (joueur +
// bulk scoreboard) + helpers lookup généraux. Découpé de match_view_repo.go
// (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

type weaponMetaEntry struct {
	label  string
	nameEN string
}

// lookupWeaponMeta résout label (FR>EN) + name_en depuis weapon_labels.
// name_en est nécessaire pour construire l'URL image via AssetURLAdapter.WeaponImageURL.
func (r *MatchViewRepo) lookupWeaponMeta(ctx context.Context, weaponIDs []int64) map[int64]weaponMetaEntry {
	result := map[int64]weaponMetaEntry{}
	if len(weaponIDs) == 0 || r.pdb.Metadata == nil {
		return result
	}
	unique := uniqueInt64s(weaponIDs)
	parts := make([]string, len(unique))
	for i, id := range unique {
		parts[i] = fmt.Sprintf("%d", uint64(id)) //nolint:gosec
	}
	query := fmt.Sprintf( //nolint:gosec
		`SELECT weapon_id,
		        COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS label,
		        COALESCE(name_en, '') AS name_en
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		strings.Join(parts, ","),
	)
	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var id UBigint
		var label, nameEN string
		if err := rows.Scan(&id, &label, &nameEN); err == nil && label != "" {
			result[id.Int64()] = weaponMetaEntry{label: label, nameEN: nameEN}
		}
	}
	return result
}

func (r *MatchViewRepo) lookupWeaponLabels(ctx context.Context, weaponIDs []int64) map[int64]string {
	labels := map[int64]string{}
	if len(weaponIDs) == 0 || r.pdb.Metadata == nil {
		return labels
	}
	// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
	// On injecte les IDs comme littéraux décimaux (valeurs internes, pas user input).
	unique := uniqueInt64s(weaponIDs)
	parts := make([]string, len(unique))
	for i, id := range unique {
		parts[i] = fmt.Sprintf("%d", uint64(id)) //nolint:gosec
	}
	query := fmt.Sprintf( //nolint:gosec
		`SELECT weapon_id, COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS weapon_label
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		strings.Join(parts, ","),
	)
	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return labels
	}
	defer rows.Close()
	for rows.Next() {
		// weapon_id UBIGINT scanné via UBigint (cf. ubigint_scanner.go) — sinon
		// overflow silencieux pour les hash filmshell bit63=1 (Mutilateur,
		// MK50 Sidekick, Fuel Rod SPNKr…).
		var id UBigint
		var label string
		if err := rows.Scan(&id, &label); err == nil && label != "" {
			labels[id.Int64()] = label
		}
	}
	return labels
}

func lookupLabelsByID(ctx context.Context, db *DB, queryTemplate string, ids []int64) map[int64]string {
	labels := map[int64]string{}
	query, args, ok := buildLookupQuery(queryTemplate, ids)
	if !ok || db == nil {
		return labels
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return labels
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label string
		if err := rows.Scan(&id, &label); err == nil && label != "" {
			labels[id] = label
		}
	}
	return labels
}

func buildLookupQuery(queryTemplate string, ids []int64) (string, []interface{}, bool) {
	uniqueIDs := uniqueInt64s(ids)
	if len(uniqueIDs) == 0 {
		return "", nil, false
	}

	placeholders := make([]string, 0, len(uniqueIDs))
	args := make([]interface{}, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(queryTemplate, strings.Join(placeholders, ",")), args, true
}

func uniqueInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// GetMatchBulkWeaponKills retourne les kills par arme de tous les joueurs (Q28).
// Applique la fusion variante→canonique par xuid (regroupe Duelist Energy Sword
// + Elite Bloodblade + Energy Sword sous le même canonique pour chaque joueur).
// Exécutée sur SharedReader (ADR 0016) — Q28 lit weapon_kills (shared-only).
func (r *MatchViewRepo) GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, Q28BulkWeaponKills, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	type key struct {
		xuid string
		wid  int64
	}
	killsByKey := make(map[key]int)
	ordered := make([]key, 0, 32)
	for rows.Next() {
		var xuid string
		var widU uint64
		var kills int
		if err := rows.Scan(&xuid, &widU, &kills); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills scan: %w", err)
		}
		canonicalU := widU
		if canon, ok := analysis.WeaponFusionMapID[widU]; ok {
			canonicalU = canon
		}
		k := key{xuid: xuid, wid: int64(canonicalU)} //nolint:gosec
		if _, seen := killsByKey[k]; !seen {
			ordered = append(ordered, k)
		}
		killsByKey[k] += kills
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]domain.BulkWeaponKillRaw, 0, len(ordered))
	weaponIDs := make([]int64, 0, len(ordered))
	for _, k := range ordered {
		results = append(results, domain.BulkWeaponKillRaw{
			XUID:     k.xuid,
			WeaponID: k.wid,
			Kills:    killsByKey[k],
		})
		weaponIDs = append(weaponIDs, k.wid)
	}

	weapMeta := r.lookupWeaponMeta(ctx, weaponIDs)
	for i := range results {
		if m, ok := weapMeta[results[i].WeaponID]; ok {
			results[i].WeaponLabel = m.label
			results[i].NameEN = m.nameEN
			continue
		}
		// Fallback : weapon_id en string pour les variantes absentes de
		// metadata.weapon_labels (cohérent avec GetMatchWeaponKills L428).
		// Évite que le frontend ait à gérer un weapon_label vide (`??` ne
		// fallback pas sur "").
		results[i].WeaponLabel = strconv.FormatInt(results[i].WeaponID, 10)
	}
	return results, nil
}
