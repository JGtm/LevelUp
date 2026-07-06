// Package api — registry_weapon_coverage.go : runner monitoring « couverture de
// résolution d'arme » (admin). Lecture seule, dual-DB (shared v_weapon_kills +
// metadata registre/weapon_labels), réutilise dataQualityHandles.
package wire

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

const weaponCoverageTopUnresolved = 15

// WeaponCoverage calcule la couverture de résolution d'arme d'un titre.
func (r *ServiceRegistry) WeaponCoverage(ctx context.Context, titleSlug string) (domain.AdminWeaponCoverage, error) {
	resp := domain.AdminWeaponCoverage{
		TitleSlug:     titleSlug,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		TopUnresolved: []domain.AdminWeaponCoverageItem{},
	}
	sharedSQL, metaSQL, closeAll, err := r.dataQualityHandles(ctx, titleSlug)
	if err != nil {
		return resp, err
	}
	defer closeAll()

	// 1. weapon_id distincts (hors sentinels 0/1/2) + kills, depuis v_weapon_kills.
	killsByID := map[int64]int{}
	ids := make([]int64, 0, 64)
	rows, err := sharedSQL.QueryContext(ctx,
		`SELECT effective_weapon_id, COUNT(*) FROM v_weapon_kills WHERE effective_weapon_id NOT IN (0,1,2) GROUP BY 1`)
	if err != nil {
		return resp, fmt.Errorf("v_weapon_kills: %w", err)
	}
	for rows.Next() {
		var id duckdb.UBigint
		var n int
		if scanErr := rows.Scan(&id, &n); scanErr != nil {
			_ = rows.Close()
			return resp, scanErr
		}
		killsByID[id.Int64()] = n
		ids = append(ids, id.Int64())
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return resp, err
	}
	resp.DistinctWeapons = len(ids)
	if len(ids) == 0 {
		return resp, nil
	}

	// 2. classification via metadata (registre + weapon_labels). metaSQL nil → tout non résolu.
	inReg, inLabel := map[int64]bool{}, map[int64]bool{}
	if metaSQL != nil {
		classifyWeaponCoverage(ctx, metaSQL, titleSlug, ids, inReg, inLabel)
	}
	for _, id := range ids {
		switch {
		case inReg[id]:
			resp.ResolvedRegistry++
		case inLabel[id]:
			resp.ResolvedLabel++
		default:
			resp.Unresolved++
			resp.TopUnresolved = append(resp.TopUnresolved,
				domain.AdminWeaponCoverageItem{WeaponID: strconv.FormatInt(id, 10), Kills: killsByID[id]})
		}
	}
	sort.Slice(resp.TopUnresolved, func(i, j int) bool { return resp.TopUnresolved[i].Kills > resp.TopUnresolved[j].Kills })
	if len(resp.TopUnresolved) > weaponCoverageTopUnresolved {
		resp.TopUnresolved = resp.TopUnresolved[:weaponCoverageTopUnresolved]
	}
	if resp.DistinctWeapons > 0 {
		d := float64(resp.DistinctWeapons)
		resp.CoveragePercent = float64(resp.ResolvedRegistry+resp.ResolvedLabel) / d * 100
		resp.RegistryPercent = float64(resp.ResolvedRegistry) / d * 100
	}
	return resp, nil
}

// classifyWeaponCoverage marque chaque id comme présent au registre (weapon_ids/
// weapons) et/ou dans weapon_labels. Requête unifiée ; si le registre est absent
// (vieux schéma) → fallback weapon_labels seul (silencieux, metaSQL brut ne logue pas).
func classifyWeaponCoverage(ctx context.Context, metaSQL *sql.DB, slug string, ids []int64, inReg, inLabel map[int64]bool) {
	vals := make([]string, len(ids))
	for i, id := range ids {
		vals[i] = "('" + strconv.FormatUint(uint64(id), 10) + "')" //nolint:gosec — ids numériques internes
	}
	// slug PARAMÉTRÉ (K1h) — plus de concaténation d'une string dans le SQL. Les ids
	// (VALUES) restent inlinés : entiers internes validés (nolint:gosec ci-dessus).
	q := "SELECT ids.v," +
		" CASE WHEN w.weapon_key IS NOT NULL THEN 1 ELSE 0 END," +
		" CASE WHEN wl.weapon_id IS NOT NULL THEN 1 ELSE 0 END" +
		" FROM (VALUES " + strings.Join(vals, ",") + ") AS ids(v)" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug=? AND wi.id_value=ids.v" +
		" LEFT JOIN weapons w ON w.title_slug=wi.title_slug AND w.weapon_key=wi.weapon_key" +
		" LEFT JOIN weapon_labels wl ON wl.weapon_id=CAST(ids.v AS UBIGINT)"
	rows, err := metaSQL.QueryContext(ctx, q, slug)
	if err != nil {
		classifyWeaponLabelsOnly(ctx, metaSQL, ids, inLabel)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		var reg, lab int
		if scanErr := rows.Scan(&v, &reg, &lab); scanErr != nil {
			continue
		}
		u, perr := strconv.ParseUint(v, 10, 64)
		if perr != nil {
			continue
		}
		id := int64(u)
		if reg == 1 {
			inReg[id] = true
		}
		if lab == 1 {
			inLabel[id] = true
		}
	}
}

// classifyWeaponLabelsOnly — fallback quand le registre est absent.
func classifyWeaponLabelsOnly(ctx context.Context, metaSQL *sql.DB, ids []int64, inLabel map[int64]bool) {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(uint64(id), 10) //nolint:gosec
	}
	rows, err := metaSQL.QueryContext(ctx,
		"SELECT weapon_id FROM weapon_labels WHERE weapon_id IN ("+strings.Join(parts, ",")+")")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id duckdb.UBigint
		if scanErr := rows.Scan(&id); scanErr == nil {
			inLabel[id.Int64()] = true
		}
	}
}
