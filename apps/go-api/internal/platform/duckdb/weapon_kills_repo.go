// Package duckdb — weapon_kills_repo.go : implementation DuckDB du loader
// aggregat weapon_kills (port.WeaponKillsRepository).
//
// Source : shared.weapon_kills (1 row par kill effectif). Agregation cote DB
// via GROUP BY (xuid, effective_weapon_id) + COUNT(*) — pour rester aligne
// avec Q16WeaponKills (le repo MatchViewRepo agrege deja ainsi pour 1 match).
//
// Capability gating : on verifie l'existence des tables shared.weapon_kills
// (et shared.match_participants si IncludeGrenadeMelee=true) via
// information_schema.tables. Si absente -> games.ErrCapabilityNotSupported.
//
// Labels EN/FR : jointure sur metadata.weapon_labels en post-traitement Go
// (la metadata DB est separee — ne peut pas etre jointe en SQL pur sans ATTACH).
//
// Note grenade/melee : les types canoniques HighlightEventType ne contiennent
// PAS "grenade_kill" ni "melee_kill" (cf. canonical/enums.go : kill, death,
// assist, medal, finisher, clutch, first_kill, first_death). Les decomptes
// grenade/melee sont stockes dans shared.match_participants comme colonnes
// agregees (grenade_kills, melee_kills). Le repo expose donc ces totaux par
// joueur / par match en UNION ALL avec un weapon_id sentinel (0=grenade, 1=melee)
// quand IncludeGrenadeMelee=true. Decision documentee dans thought_log
// [2026-04-27].
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// Sentinel weapon_id valeurs reservees pour les rangees grenade/melee
// venant de shared.match_participants (IncludeGrenadeMelee=true).
// Choix : 0 et 1 sont deja exclus par Q16WeaponKills (NOT IN (0,1,2)),
// donc ils ne peuvent jamais collisioner avec un weapon_id reel.
const (
	weaponIDGrenadeSentinel int64 = 0
	weaponIDMeleeSentinel   int64 = 1
)

// WeaponKillsRepo implemente port.WeaponKillsRepository.
type WeaponKillsRepo struct {
	pdb *PlayerDB
}

// NewWeaponKillsRepo cree un WeaponKillsRepo lie a un PlayerDB.
func NewWeaponKillsRepo(pdb *PlayerDB) *WeaponKillsRepo {
	return &WeaponKillsRepo{pdb: pdb}
}

// LoadWeaponKillsAggregated charge les kills aggregees par (xuid, weapon_id).
//
// L'appelant DOIT avoir valide les filtres ; le repo re-valide en defense.
// Retourne games.ErrCapabilityNotSupported si shared.weapon_kills n'existe
// pas dans la DB cible.
func (r *WeaponKillsRepo) LoadWeaponKillsAggregated(
	ctx context.Context,
	slug string,
	filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("WeaponKillsRepo.LoadWeaponKillsAggregated: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if !r.weaponKillsTableExists(ctx) {
		slog.DebugContext(ctx, "WeaponKillsRepo: shared.weapon_kills missing",
			"slug", slug, "match_count", len(filters.MatchIDs))
		return nil, games.ErrCapabilityNotSupported
	}

	rows, err := r.queryWeaponKills(ctx, filters)
	if err != nil {
		slog.ErrorContext(ctx, "WeaponKillsRepo: query failed",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("WeaponKillsRepo.LoadWeaponKillsAggregated: %w", err)
	}

	r.attachWeaponLabels(ctx, rows)
	return rows, nil
}

// queryWeaponKills execute le SELECT principal + UNION ALL grenade/melee
// si demande, puis aggrege cote Go (l'aggregation SQL est faite par GROUP BY).
func (r *WeaponKillsRepo) queryWeaponKills(
	ctx context.Context,
	filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	q, args := buildWeaponKillsQuery(filters)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	var out []port.WeaponKillRow
	for dbRows.Next() {
		var (
			xuid     string
			weaponID UBigint // UBIGINT côté DuckDB (cf. ubigint_scanner.go)
			kills    int
			isGM     bool
		)
		if err := dbRows.Scan(&xuid, &weaponID, &kills, &isGM); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if filters.MinKills > 0 && kills < filters.MinKills {
			continue
		}
		out = append(out, port.WeaponKillRow{
			XUID:           xuid,
			WeaponID:       weaponID.Int64(),
			Kills:          kills,
			IsGrenadeMelee: isGM,
		})
	}
	if err := dbRows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// buildWeaponKillsQuery compose le SELECT principal sur shared.weapon_kills,
// optionnellement UNION ALL avec shared.match_participants pour grenade/melee.
//
// Filtres :
//   - MatchIDs (requis) -> wk.match_id IN (?,...)
//   - Gamertag XOR XUIDs -> filtre sur wk.xuid (resolution gamertag->xuid via
//     shared.xuid_aliases si Gamertag fourni)
//
// Note : effective_weapon_id NOT IN (0,1,2) exclu pour rester aligne avec Q16
// (sentinel des armes "no weapon"). Quand IncludeGrenadeMelee=true on injecte
// les sentinels 0 (grenade) et 1 (melee) via UNION ALL — ces rows portent
// le flag is_grenade_melee=true cote SQL.
func buildWeaponKillsQuery(f port.WeaponKillFilters) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)*2+len(f.XUIDs)*2+2)

	matchPlaceholders := Placeholders(len(f.MatchIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}

	// Branche 1 : weapon_kills (armes principales).
	// effective_weapon_id reste en UBIGINT cote SQL (pas de cast ::BIGINT) car
	// certaines armes (filmshell hashes avec bit63=1) ont des IDs hors INT64
	// → "Type UINT64 ... can't be cast ... out of range for INT64". Le scan
	// Go capture en uint64 puis reinterprete bit-a-bit en int64 (cf. queryWeaponKills).
	sb.WriteString(`
SELECT
    wk.xuid,
    wk.effective_weapon_id AS weapon_id,
    COUNT(*) AS kills,
    FALSE AS is_grenade_melee
FROM shared.v_weapon_kills wk
WHERE wk.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)
  AND wk.effective_weapon_id NOT IN (0, 1, 2)`)

	appendXUIDFilter(&sb, &args, "wk", f)

	sb.WriteString(`
GROUP BY wk.xuid, wk.effective_weapon_id`)

	if !f.IncludeGrenadeMelee {
		return sb.String(), args
	}

	// Branche 2 : grenade/melee depuis match_participants.
	// On reduplique les MatchIDs (placeholders distincts).
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	sb.WriteString(`
UNION ALL
SELECT
    mp.xuid,
    `)
	sb.WriteString(strconv.FormatInt(weaponIDGrenadeSentinel, 10))
	sb.WriteString(`::UBIGINT AS weapon_id,
    SUM(COALESCE(mp.grenade_kills, 0))::INTEGER AS kills,
    TRUE AS is_grenade_melee
FROM shared.match_participants mp
WHERE mp.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)
	appendXUIDFilter(&sb, &args, "mp", f)
	sb.WriteString(`
GROUP BY mp.xuid
HAVING SUM(COALESCE(mp.grenade_kills, 0)) > 0`)

	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	sb.WriteString(`
UNION ALL
SELECT
    mp.xuid,
    `)
	sb.WriteString(strconv.FormatInt(weaponIDMeleeSentinel, 10))
	sb.WriteString(`::UBIGINT AS weapon_id,
    SUM(COALESCE(mp.melee_kills, 0))::INTEGER AS kills,
    TRUE AS is_grenade_melee
FROM shared.match_participants mp
WHERE mp.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)
	appendXUIDFilter(&sb, &args, "mp", f)
	sb.WriteString(`
GROUP BY mp.xuid
HAVING SUM(COALESCE(mp.melee_kills, 0)) > 0`)

	return sb.String(), args
}

// appendXUIDFilter ajoute la clause AND sur xuid en fonction de Gamertag ou XUIDs.
// Au moins un des deux est garanti par Validate().
func appendXUIDFilter(sb *strings.Builder, args *[]any, alias string, f port.WeaponKillFilters) {
	if len(f.XUIDs) > 0 {
		sb.WriteString(`
  AND `)
		sb.WriteString(alias)
		sb.WriteString(`.xuid IN (`)
		sb.WriteString(Placeholders(len(f.XUIDs)))
		sb.WriteString(`)`)
		for _, x := range f.XUIDs {
			*args = append(*args, x)
		}
		return
	}
	// Gamertag -> resolution via xuid_aliases
	sb.WriteString(`
  AND `)
	sb.WriteString(alias)
	sb.WriteString(`.xuid IN (
      SELECT xuid FROM shared.xuid_aliases WHERE gamertag = ?
  )`)
	*args = append(*args, f.Gamertag)
}

// attachWeaponLabels enrichit les rows non-grenade/melee avec leur libelle
// EN/FR depuis metadata.weapon_labels. Best-effort : si Metadata est absent
// ou la table manque, les labels restent vides (le service appelant peut
// fallback sur le weapon_id).
func (r *WeaponKillsRepo) attachWeaponLabels(ctx context.Context, rows []port.WeaponKillRow) {
	if r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	// On inclut aussi les sentinels grenade/melee (weapon_id=0/1) — ils ont
	// des labels "Grenade"/"Mêlée" en metadata.weapon_labels (cf. migration
	// add_weapon_labels). Skipper ici laissait Label="" alors que le label
	// est disponible.
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.WeaponID)
	}
	if len(ids) == 0 {
		return
	}

	// Driver workaround : weapon_id est UBIGINT, on injecte les literals decimals
	// (cf. match_view_repo.lookupWeaponLabels).
	unique := uniqueInt64s(ids)
	parts := make([]string, len(unique))
	for i, id := range unique {
		parts[i] = strconv.FormatUint(uint64(id), 10) //nolint:gosec
	}
	query := fmt.Sprintf( //nolint:gosec
		`SELECT weapon_id, COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS weapon_label
		 FROM weapon_labels
		 WHERE weapon_id IN (%s)`,
		strings.Join(parts, ","),
	)
	dbRows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return
	}
	defer dbRows.Close()

	labels := map[int64]string{}
	for dbRows.Next() {
		// weapon_id UBIGINT scanné via UBigint (cf. ubigint_scanner.go) — sinon
		// overflow silencieux pour les hash filmshell bit63=1.
		var id UBigint
		var label string
		if err := dbRows.Scan(&id, &label); err == nil && label != "" {
			labels[id.Int64()] = label
		}
	}
	for i := range rows {
		if label, ok := labels[rows[i].WeaponID]; ok {
			rows[i].Label = label
		}
	}
}

// weaponKillsTableExists verifie que la table shared.weapon_kills (ou la vue
// shared.v_weapon_kills) est presente. Capability check minimal pour la
// Phase 1 — la presence des donnees est consideree equivalente au support
// de la capability "match.detail.weapon_kills".
//
// Accepte 2 configurations (cf. MedalsByXUIDRepo.medalsEarnedTableExists) :
//   - Prod : shared_matches_v2.duckdb attaché sous catalog 'shared' (ATTACH).
//   - Tests : tables exposées sous schema 'shared' (CREATE SCHEMA shared).
func (r *WeaponKillsRepo) weaponKillsTableExists(ctx context.Context) bool {
	if r.pdb == nil {
		return false
	}
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return false
	}
	defer release()

	var count int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_name IN ('weapon_kills', 'v_weapon_kills')
		  AND (table_catalog = 'shared' OR table_schema = 'shared')
	`).Scan(&count)
	if err != nil {
		// Si la requete d'introspection echoue, on considere absent (defensive).
		return false
	}
	return count > 0
}
