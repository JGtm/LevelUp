package duckdb

// weapon_coverage_repo.go : les lectures de la page d'administration « couverture de résolution d'arme »,
// derrière `port.WeaponCoverageRepo`. Lecture seule, deux bases : la base partagée du titre (frags par
// identifiant d'arme ou par source de dégât ; tables append-only lues par leurs vues `_latest`) et metadata
// (registre d'armes et libellés). Le repo n'ouvre ni ne ferme rien.

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"levelup/go-api/internal/port"
)

var _ port.WeaponCoverageRepo = (*WeaponCoverageRepo)(nil)

// WeaponCoverageRepo lit la couverture d'arme d'un titre.
type WeaponCoverageRepo struct {
	shared *sql.DB
	meta   *sql.DB
}

// NewWeaponCoverageRepo crée le repo sur des connexions DÉJÀ ouvertes. meta peut être nil : tout est alors
// non résolu.
func NewWeaponCoverageRepo(shared, meta *sql.DB) *WeaponCoverageRepo {
	return &WeaponCoverageRepo{shared: shared, meta: meta}
}

// HasMetadata dit si metadata est ouverte.
func (r *WeaponCoverageRepo) HasMetadata() bool { return r.meta != nil }

// KillsByWeaponID rend les frags par identifiant d'arme distinct (hors sentinelles 0/1/2), depuis
// v_weapon_kills.
func (r *WeaponCoverageRepo) KillsByWeaponID(ctx context.Context) ([]port.WeaponKillCount, error) {
	rows, err := r.shared.QueryContext(ctx,
		`SELECT effective_weapon_id, COUNT(*) FROM v_weapon_kills WHERE effective_weapon_id NOT IN (0,1,2) GROUP BY 1`)
	if err != nil {
		return nil, fmt.Errorf("v_weapon_kills: %w", err)
	}
	var out []port.WeaponKillCount
	for rows.Next() {
		var id UBigint
		var n int
		if scanErr := rows.Scan(&id, &n); scanErr != nil {
			_ = rows.Close()
			return nil, scanErr
		}
		out = append(out, port.WeaponKillCount{WeaponID: id.Int64(), Kills: n})
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// KillsBySourceTag rend les frags par source de dégât mesurée (résumés par match, vue `_latest`).
func (r *WeaponCoverageRepo) KillsBySourceTag(ctx context.Context) ([]port.KillSourceCount, error) {
	rows, err := r.shared.QueryContext(ctx, `
		SELECT source_tag, SUM(kills)::INTEGER
		FROM match_kill_pairs_latest
		WHERE source_tag IS NOT NULL
		GROUP BY source_tag`)
	if err != nil {
		return nil, fmt.Errorf("match_kill_pairs_latest: %w", err)
	}
	defer rows.Close()
	var out []port.KillSourceCount
	for rows.Next() {
		var tag uint32
		var kills int
		if scanErr := rows.Scan(&tag, &kills); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, port.KillSourceCount{SourceTag: tag, Kills: kills})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ClassifyWeaponIDs marque chaque id comme présent au registre (weapon_ids/weapons) et/ou dans
// weapon_labels. Requête unifiée ; si le registre est absent (vieux schéma), repli sur weapon_labels
// seul (silencieux). Sans metadata, rien n'est marqué.
func (r *WeaponCoverageRepo) ClassifyWeaponIDs(ctx context.Context, titleSlug string, ids []int64) (inRegistry, inLabel map[int64]bool) {
	inRegistry, inLabel = map[int64]bool{}, map[int64]bool{}
	if r.meta == nil {
		return inRegistry, inLabel
	}
	r.classifyWeaponCoverage(ctx, titleSlug, ids, inRegistry, inLabel)
	return inRegistry, inLabel
}

func (r *WeaponCoverageRepo) classifyWeaponCoverage(ctx context.Context, slug string, ids []int64, inReg, inLabel map[int64]bool) {
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
	rows, err := r.meta.QueryContext(ctx, q, slug)
	if err != nil {
		r.classifyWeaponLabelsOnly(ctx, slug, ids, inLabel)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		var reg, lab int
		if scanErr := rows.Scan(&v, &reg, &lab); scanErr != nil {
			signalUnreadableRow(ctx, slug, "weapon_coverage", scanErr)
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

// classifyWeaponLabelsOnly — repli quand le registre est absent.
func (r *WeaponCoverageRepo) classifyWeaponLabelsOnly(ctx context.Context, slug string, ids []int64, inLabel map[int64]bool) {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(uint64(id), 10) //nolint:gosec
	}
	rows, err := r.meta.QueryContext(ctx,
		"SELECT weapon_id FROM weapon_labels WHERE weapon_id IN ("+strings.Join(parts, ",")+")")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id UBigint
		if scanErr := rows.Scan(&id); scanErr != nil {
			signalUnreadableRow(ctx, slug, "weapon_coverage_labels", scanErr)
			continue
		}
		inLabel[id.Int64()] = true
	}
}

// RegistryWeaponKeys rend les clés d'arme du registre du titre (table `weapons`). Une ligne illisible est
// sautée.
func (r *WeaponCoverageRepo) RegistryWeaponKeys(ctx context.Context, titleSlug string) (map[string]bool, error) {
	rows, err := r.meta.QueryContext(ctx,
		"SELECT weapon_key FROM weapons WHERE title_slug = ?", titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connues := map[string]bool{}
	for rows.Next() {
		var k string
		if rows.Scan(&k) == nil {
			connues[k] = true
		}
	}
	return connues, nil
}
