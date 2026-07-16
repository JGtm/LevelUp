// Package duckdb — weapon_kills_repo.go : implementation DuckDB du loader
// aggregat weapon_kills (port.WeaponKillsRepository).
//
// Source : weapon_kills (1 row par kill effectif). Agregation cote DB
// via GROUP BY (xuid, effective_weapon_id) + COUNT(*) — pour rester aligne
// avec Q16WeaponKills (le repo MatchViewRepo agrege deja ainsi pour 1 match).
//
// ADR 0016 : queries executees via SharedReader.Get(ctx) — connexion directe
// au fichier shared_matches_v2.duckdb, pas de prefixe `shared.` (l'ATTACH a
// ete retire). Toutes les tables/vues vivent dans le schema `main` par defaut.
//
// Capability gating : si la table cible est absente (ou la vue
// v_weapon_kills), DuckDB remonte une erreur "Table with name ... does
// not exist" — interceptee via isTableNotFoundErr et convertie en
// games.ErrCapabilityNotSupported. Plus pérenne qu'une introspection
// information_schema (les CATALOG/SCHEMA varient entre prod RO direct,
// sharedprovider, et tests in-memory avec attaches simulees).
//
// Labels EN/FR : jointure sur metadata.weapon_labels en post-traitement Go
// (la metadata DB est separee — ne peut pas etre jointe en SQL pur sans ATTACH).
//
// Note grenade/melee : les types canoniques HighlightEventType ne contiennent
// PAS "grenade_kill" ni "melee_kill" (cf. canonical/enums.go : kill, death,
// assist, medal, finisher, clutch, first_kill, first_death). Les decomptes
// grenade/melee sont stockes dans match_participants comme colonnes
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
// venant de match_participants (IncludeGrenadeMelee=true).
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
// Retourne games.ErrCapabilityNotSupported si weapon_kills n'existe
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

	rows, err := r.queryWeaponKills(ctx, filters)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "WeaponKillsRepo: weapon_kills missing",
				"slug", slug, "match_count", len(filters.MatchIDs))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "WeaponKillsRepo: query failed",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("WeaponKillsRepo.LoadWeaponKillsAggregated: %w", err)
	}

	r.attachWeaponMeta(ctx, slug, rows, filters.ResolveRoles)
	return rows, nil
}

// LoadKillMechanicsAggregated agrège les mécaniques de kill NATIVES Halo 5 par
// xuid (assassination / ground_pound / shoulder_bash) depuis match_participants,
// sur les matchs filtrés (GROUP BY xuid). Mêmes filtres que les armes (MatchIDs +
// XUIDs). Retourne games.ErrCapabilityNotSupported si match_participants est
// absente. Les colonnes mécaniques sont garanties présentes (migration
// add_h5_kill_mechanics_columns) ; valent 0 pour les titres qui ne les peuplent pas.
func (r *WeaponKillsRepo) LoadKillMechanicsAggregated(
	ctx context.Context,
	filters port.WeaponKillFilters,
) ([]port.KillMechanicsRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("WeaponKillsRepo.LoadKillMechanicsAggregated: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	q, args := buildKillMechanicsQuery(filters, r.pdb.TitleSlug)
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	var out []port.KillMechanicsRow
	for dbRows.Next() {
		var row port.KillMechanicsRow
		if err := dbRows.Scan(&row.XUID, &row.Assassinations, &row.GroundPound, &row.ShoulderBash); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, row)
	}
	return out, dbRows.Err()
}

// buildKillMechanicsQuery construit le SELECT agrégé des mécaniques par xuid
// (même forme que la branche grenade/melee de buildWeaponKillsQuery).
func buildKillMechanicsQuery(f port.WeaponKillFilters, titleSlug string) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs))
	matchPlaceholders := Placeholders(len(f.MatchIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	sb.WriteString(`
SELECT
    mp.xuid,
    SUM(COALESCE(mp.assassination_kills, 0))::INTEGER AS assassinations,
    SUM(COALESCE(mp.ground_pound_kills, 0))::INTEGER  AS ground_pound,
    SUM(COALESCE(mp.shoulder_bash_kills, 0))::INTEGER AS shoulder_bash
FROM match_participants mp
WHERE mp.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "mp.match_id"))
	appendXUIDFilter(&sb, &args, "mp", f)
	sb.WriteString(`
GROUP BY mp.xuid`)
	return sb.String(), args
}

// queryWeaponKills execute le SELECT principal + UNION ALL grenade/melee
// si demande, puis aggrege cote Go (l'aggregation SQL est faite par GROUP BY).
func (r *WeaponKillsRepo) queryWeaponKills(
	ctx context.Context,
	filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	q, args := buildWeaponKillsQuery(filters, r.pdb.TitleSlug)
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

// buildWeaponKillsQuery compose le SELECT principal sur v_weapon_kills,
// optionnellement UNION ALL avec match_participants pour grenade/melee.
//
// Filtres :
//   - MatchIDs (requis) -> wk.match_id IN (?,...)
//   - Gamertag XOR XUIDs -> filtre sur wk.xuid (resolution gamertag->xuid via
//     xuid_aliases si Gamertag fourni)
//
// Note : effective_weapon_id NOT IN (0,1,2) exclu pour rester aligne avec Q16
// (sentinel des armes "no weapon"). Quand IncludeGrenadeMelee=true on injecte
// les sentinels 0 (grenade) et 1 (melee) via UNION ALL — ces rows portent
// le flag is_grenade_melee=true cote SQL.
func buildWeaponKillsQuery(f port.WeaponKillFilters, titleSlug string) (string, []any) {
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
FROM v_weapon_kills wk
WHERE wk.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)
  AND wk.effective_weapon_id NOT IN (0, 1, 2)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "wk.match_id"))

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
FROM match_participants mp
WHERE mp.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "mp.match_id"))
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
FROM match_participants mp
WHERE mp.match_id IN (`)
	sb.WriteString(matchPlaceholders)
	sb.WriteString(`)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "mp.match_id"))
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
      SELECT xuid FROM xuid_aliases WHERE gamertag = ?
  )`)
	*args = append(*args, f.Gamertag)
}

// attachWeaponMeta renseigne Label (toujours, parité weapon_labels) + Role (si
// withRoles) via le resolver unifié resolveWeaponMeta — PASSAGE PRINCIPAL P4 :
// une seule requête metadata (registre + weapon_labels) remplace l'ancien duo
// attachWeaponLabels + attachWeaponRoles. Best-effort : meta absent → no-op.
// Sentinels grenade/melee (0/1) : Label vient de weapon_labels ("Grenade"/"Mêlée"),
// Role reste "" (absents du registre) — identique à l'ancien comportement.
func (r *WeaponKillsRepo) attachWeaponMeta(ctx context.Context, slug string, rows []port.WeaponKillRow, withRoles bool) {
	if r.pdb == nil || r.pdb.Metadata == nil || len(rows) == 0 {
		return
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.WeaponID)
	}
	meta := resolveWeaponMeta(ctx, r.pdb.Metadata, slug, ids)
	for i := range rows {
		m, ok := meta[rows[i].WeaponID]
		if !ok {
			continue
		}
		if m.label != "" {
			rows[i].Label = m.label
		}
		if withRoles && m.role != "" {
			rows[i].Role = m.role
		}
	}
}
