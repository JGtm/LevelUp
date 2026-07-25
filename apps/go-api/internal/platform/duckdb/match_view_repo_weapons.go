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
	// class / role / family : dimensions du registre (axe manipulation + fonction de
	// combat + famille), résolues dans la même passe que le label (resolveWeaponMeta les
	// porte déjà). Vides si l'arme est absente du registre. Alimentent la FragDistribution
	// par-match (family = TYPE de grenade au niveau 2, V72-15.2).
	class  string
	role   string
	family string
}

// lookupWeaponMeta résout label (FR>EN) + name_en + class/role depuis le registre.
// name_en est nécessaire pour construire l'URL image via AssetURLAdapter.WeaponImageURL ;
// class/role alimentent la FragDistribution par-match (sunburst v2).
func (r *MatchViewRepo) lookupWeaponMeta(ctx context.Context, weaponIDs []int64) map[int64]weaponMetaEntry {
	result := map[int64]weaponMetaEntry{}
	if len(weaponIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result
	}
	// PASSAGE PRINCIPAL P4 : résolution via le registre + weapon_labels (parité du
	// nom). On ne garde que les ids résolus (label non vide), comme l'ancien lookup.
	for id, m := range resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponIDs) {
		if m.label != "" {
			result[id] = weaponMetaEntry{label: m.label, nameEN: m.nameEN, class: m.class, role: m.role, family: m.family}
		}
	}
	return result
}

func (r *MatchViewRepo) lookupWeaponLabels(ctx context.Context, weaponIDs []int64) map[int64]string {
	labels := map[int64]string{}
	if len(weaponIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return labels
	}
	for id, m := range resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponIDs) {
		if m.label != "" {
			labels[id] = m.label
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

	sharedDB, release, err := r.sharedRead().Get(ctx)
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
	mechanicByKey := make(map[key]int) // sous-ensemble non-arme (kill_kind <> 'weapon'), V72-15.3
	ordered := make([]key, 0, 32)
	for rows.Next() {
		var xuid string
		var widU uint64
		var kills, mechanicKills int
		if err := rows.Scan(&xuid, &widU, &kills, &mechanicKills); err != nil {
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
		mechanicByKey[k] += mechanicKills
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]domain.BulkWeaponKillRaw, 0, len(ordered))
	weaponIDs := make([]int64, 0, len(ordered))
	for _, k := range ordered {
		results = append(results, domain.BulkWeaponKillRaw{
			XUID:          k.xuid,
			WeaponID:      k.wid,
			Kills:         killsByKey[k],
			MechanicKills: mechanicByKey[k],
		})
		weaponIDs = append(weaponIDs, k.wid)
	}

	weapMeta := r.lookupWeaponMeta(ctx, weaponIDs)
	for i := range results {
		if m, ok := weapMeta[results[i].WeaponID]; ok {
			results[i].WeaponLabel = m.label
			results[i].NameEN = m.nameEN
			results[i].Class = m.class
			results[i].Role = m.role
			results[i].Family = m.family
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
