// Package ops — data_quality.go : comptages et listes des « inconnus » data
// pour le dashboard monitoring admin (assets UUID bruts, modes sans
// traduction FR, playlists hors catalogue, xuids orphelins, lying bits).
//
// Fonctions pures sur *sql.DB (testables sur DuckDB :memory:) — l'ouverture
// des handles (shared RO + metadata RW partagé) appartient au caller
// (registry). Cross-DB sans ATTACH : les petites tables metadata sont
// chargées en Go puis les agrégats shared sont filtrés (pattern
// countSharedMatchesMissingEnrichment).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
)

// Bits du ledger match_registry.backfill_completed — miroir de
// internal/sync/backfill_flags.go (MBitEvents / MBitWeaponKills). ops ne peut
// pas importer sync (cycle ops → sync → service → ops), garde-fou testé par
// TestDataQuality_LyingBits.
const (
	dqBitEvents      = 1 << 16
	dqBitWeaponKills = 1 << 21
)

// dqTimestampExpr : horodatage canonique d'un match (règle projet : jamais
// start_time brut — cf. pattern media_repo).
var dqTimestampExpr = `` + analysis.SQLStartTimeCanonical("") + ``

// DataQualityCounts agrège les compteurs d'inconnus.
type DataQualityCounts struct {
	RawUUIDPlaylists  int
	RawUUIDMaps       int
	RawUUIDPairs      int
	RawUUIDVariants   int
	UntranslatedModes int
	OrphanPlaylists   int
	OrphanXUIDs       int
	LyingBitsEvents   int
	LyingBitsWeapons  int
}

// RawUUIDTotal retourne la somme des assets UUID bruts (cible de l'action
// « résoudre les noms registry »).
func (c DataQualityCounts) RawUUIDTotal() int {
	return c.RawUUIDPlaylists + c.RawUUIDMaps + c.RawUUIDPairs + c.RawUUIDVariants
}

// DataQualityIssue est une ligne détaillée d'inconnu (alimente les
// formulaires de résolution côté front).
type DataQualityIssue struct {
	// Kind : "raw_uuid" | "untranslated_mode" | "orphan_playlist" | "orphan_xuid".
	Kind string
	// AssetKind : playlist|map|pair|game_variant (raw_uuid uniquement).
	AssetKind string
	// ID : asset_id / clé mode normalisée / playlist_id / xuid.
	ID string
	// Label : valeur d'affichage courante (pair_name brut échantillon, nom de
	// playlist…). Vide si non pertinent.
	Label string
	// Occurrences : nombre de matchs concernés.
	Occurrences int
	// LastSeen : RFC3339 du match le plus récent concerné (vide si inconnu).
	LastSeen string
}

// rawUUIDColumns : colonnes (id, name) de match_registry où name == id
// signale un asset non résolu — même critère que BackfillRegistryNames.
var rawUUIDColumns = []struct{ Kind, IDCol, NameCol string }{
	{"playlist", "playlist_id", "playlist_name"},
	{"map", "map_id", "map_name"},
	{"pair", "pair_id", "pair_name"},
	{"game_variant", "game_variant_id", "game_variant_name"},
}

// CountDataQuality calcule tous les compteurs d'inconnus. Best-effort par
// section : une requête en échec est remontée en erreur (le caller dégrade).
func CountDataQuality(ctx context.Context, sharedDB, metaDB *sql.DB, titleSlug string) (DataQualityCounts, error) {
	var c DataQualityCounts
	if sharedDB == nil {
		return c, fmt.Errorf("data_quality: sharedDB nil")
	}

	targets := []*int{&c.RawUUIDPlaylists, &c.RawUUIDMaps, &c.RawUUIDPairs, &c.RawUUIDVariants}
	for i, col := range rawUUIDColumns {
		q := fmt.Sprintf(
			`SELECT COUNT(DISTINCT %s) FROM match_registry WHERE %s IS NOT NULL AND %s = %s`,
			col.IDCol, col.IDCol, col.NameCol, col.IDCol)
		if err := sharedDB.QueryRowContext(ctx, q).Scan(targets[i]); err != nil {
			return c, fmt.Errorf("count raw uuid %s: %w", col.Kind, err)
		}
	}

	untranslated, err := listUntranslatedModes(ctx, sharedDB, metaDB, 0)
	if err != nil {
		return c, err
	}
	c.UntranslatedModes = len(untranslated)

	orphanPlaylists, err := listOrphanPlaylists(ctx, sharedDB, metaDB, titleSlug, 0)
	if err != nil {
		return c, err
	}
	c.OrphanPlaylists = len(orphanPlaylists)

	if err := sharedDB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
	`).Scan(&c.OrphanXUIDs); err != nil {
		return c, fmt.Errorf("count orphan xuids: %w", err)
	}

	if err := sharedDB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, dqBitEvents)).Scan(&c.LyingBitsEvents); err != nil {
		return c, fmt.Errorf("count lying bits events: %w", err)
	}
	if err := sharedDB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id)
	`, dqBitWeaponKills)).Scan(&c.LyingBitsWeapons); err != nil {
		return c, fmt.Errorf("count lying bits weapons: %w", err)
	}

	return c, nil
}

// ListDataQualityIssues retourne les lignes détaillées d'un kind donné, les
// plus fréquentes d'abord. kind ∈ {raw_uuids, untranslated_modes,
// orphan_playlists, orphan_xuids}. limit <= 0 → 50.
func ListDataQualityIssues(
	ctx context.Context, sharedDB, metaDB *sql.DB, titleSlug, kind string, limit int,
) ([]DataQualityIssue, error) {
	if sharedDB == nil {
		return nil, fmt.Errorf("data_quality: sharedDB nil")
	}
	if limit <= 0 {
		limit = 50
	}
	switch kind {
	case "raw_uuids":
		return listRawUUIDs(ctx, sharedDB, limit)
	case "untranslated_modes":
		return listUntranslatedModes(ctx, sharedDB, metaDB, limit)
	case "orphan_playlists":
		return listOrphanPlaylists(ctx, sharedDB, metaDB, titleSlug, limit)
	case "orphan_xuids":
		return listOrphanXUIDs(ctx, sharedDB, limit)
	default:
		return nil, fmt.Errorf("data_quality: kind inconnu %q", kind)
	}
}

// listRawUUIDs liste les assets UUID bruts toutes colonnes confondues.
func listRawUUIDs(ctx context.Context, sharedDB *sql.DB, limit int) ([]DataQualityIssue, error) {
	out := make([]DataQualityIssue, 0, limit)
	for _, col := range rawUUIDColumns {
		q := fmt.Sprintf(`
			SELECT %s, COUNT(*), MAX(%s)
			FROM match_registry
			WHERE %s IS NOT NULL AND %s = %s
			GROUP BY 1 ORDER BY 2 DESC`,
			col.IDCol, dqTimestampExpr, col.IDCol, col.NameCol, col.IDCol)
		rows, err := sharedDB.QueryContext(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("list raw uuid %s: %w", col.Kind, err)
		}
		for rows.Next() {
			var id string
			var n int
			var last sql.NullTime
			if err := rows.Scan(&id, &n, &last); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, DataQualityIssue{
				Kind: "raw_uuid", AssetKind: col.Kind, ID: id,
				Occurrences: n, LastSeen: formatNullTime(last),
			})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Occurrences > out[j].Occurrences })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// metaTableExists indique si une table existe dans la metadata du titre.
// Certains titres ont un schéma metadata PROPRE (PMT-9 : Halo 5 n'a ni
// mode_name_tr ni playlists_catalog — ses référentiels vivent ailleurs,
// asset_translations/pair_name_fr). Un détecteur qui dépend d'un référentiel
// absent du SCHÉMA du titre est NON APPLICABLE : introspection plutôt que
// Catalog Error (qui faisait tomber tout l'endpoint en 500 pour le titre).
func metaTableExists(ctx context.Context, metaDB *sql.DB, table string) (bool, error) {
	var n int
	if err := metaDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table,
	).Scan(&n); err != nil {
		// Une erreur d'introspection (handle fermé, IO, timeout) N'EST PAS une table
		// absente : la remonter (règle n°3 — jamais d'erreur avalée), sinon un faux
		// vert masque des détecteurs data-quality (untranslated_modes = 0 à tort).
		return false, fmt.Errorf("introspection metadata table %s: %w", table, err)
	}
	return n > 0, nil
}

// listUntranslatedModes liste les modes (clé normalisée via
// analysis.NormalizeModeLabel) absents de mode_name_tr[lang='fr']. Dégradations :
//   - metaDB nil (metadata absente/illisible) → tout est considéré non traduit
//     (dégradation explicite plutôt que faux vert) ;
//   - table mode_name_tr ABSENTE DU SCHÉMA du titre → détecteur non applicable
//     (le titre gère ses traductions autrement) → liste vide, jamais une erreur.
//
// limit <= 0 → pas de limite (usage comptage).
func listUntranslatedModes(ctx context.Context, sharedDB, metaDB *sql.DB, limit int) ([]DataQualityIssue, error) {
	if metaDB != nil {
		exists, err := metaTableExists(ctx, metaDB, "mode_name_tr")
		if err != nil {
			return nil, err
		}
		if !exists {
			slog.DebugContext(ctx, "data_quality: mode_name_tr absente du schéma metadata — détecteur untranslated_modes non applicable",
				"module", "monitoring")
			return []DataQualityIssue{}, nil
		}
	}
	frSet := map[string]struct{}{}
	if metaDB != nil {
		rows, err := metaDB.QueryContext(ctx, `SELECT mode_en FROM mode_name_tr WHERE lang = 'fr'`)
		if err != nil {
			return nil, fmt.Errorf("load mode_name_tr: %w", err)
		}
		for rows.Next() {
			var k string
			if scanErr := rows.Scan(&k); scanErr == nil {
				frSet[strings.TrimSpace(k)] = struct{}{}
			}
		}
		rows.Close()
	}

	mapLabels, err := loadResolvedMapNames(ctx, sharedDB)
	if err != nil {
		return nil, err
	}

	byMode, err := aggregateUntranslatedModes(ctx, sharedDB, frSet, mapLabels)
	if err != nil {
		return nil, err
	}

	out := make([]DataQualityIssue, 0, len(byMode))
	for mode, a := range byMode {
		issue := DataQualityIssue{
			Kind: "untranslated_mode", ID: mode, Label: a.sample, Occurrences: a.total,
		}
		if a.hasLast {
			issue.LastSeen = a.lastSeen.UTC().Format(time.RFC3339)
		}
		out = append(out, issue)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Occurrences != out[j].Occurrences {
			return out[i].Occurrences > out[j].Occurrences
		}
		return out[i].ID < out[j].ID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// untranslatedModeAgg agrège les pair_names bruts d'un même mode normalisé.
type untranslatedModeAgg struct {
	sample   string
	sampleN  int
	total    int
	lastSeen time.Time
	hasLast  bool
}

// aggregateUntranslatedModes groupe les pair_name de match_registry par mode
// normalisé (NormalizeModeLabel) et écarte ceux déjà traduits (frSet).
func aggregateUntranslatedModes(
	ctx context.Context, sharedDB *sql.DB, frSet map[string]struct{}, mapLabels []string,
) (map[string]*untranslatedModeAgg, error) {
	rows, err := sharedDB.QueryContext(ctx, fmt.Sprintf(`
		SELECT pair_name, COUNT(*), MAX(%s)
		FROM match_registry
		WHERE pair_name IS NOT NULL AND pair_name <> ''
		  AND (pair_id IS NULL OR pair_name <> pair_id)
		GROUP BY pair_name`, dqTimestampExpr))
	if err != nil {
		return nil, fmt.Errorf("list pair_names: %w", err)
	}
	defer rows.Close()

	byMode := map[string]*untranslatedModeAgg{}
	for rows.Next() {
		var raw string
		var n int
		var last sql.NullTime
		if scanErr := rows.Scan(&raw, &n, &last); scanErr != nil {
			return nil, scanErr
		}
		mode := strings.TrimSpace(analysis.NormalizeModeLabel(raw, mapLabels...))
		if mode == "" {
			continue
		}
		if _, ok := frSet[mode]; ok {
			continue
		}
		a := byMode[mode]
		if a == nil {
			a = &untranslatedModeAgg{}
			byMode[mode] = a
		}
		a.total += n
		if n >= a.sampleN {
			a.sample, a.sampleN = raw, n
		}
		if last.Valid && (!a.hasLast || last.Time.After(a.lastSeen)) {
			a.lastSeen, a.hasLast = last.Time, true
		}
	}
	return byMode, rows.Err()
}

// listOrphanPlaylists liste les playlist_id de match_registry absents de
// metadata.playlists_catalog pour ce titre. Dégradations :
//   - metaDB nil (metadata absente/illisible) → catalogue vide (tout orphelin) ;
//   - table playlists_catalog ABSENTE DU SCHÉMA du titre → détecteur non
//     applicable (pas de catalogue de playlists pour ce titre) → liste vide.
//
// limit <= 0 → pas de limite (usage comptage).
func listOrphanPlaylists(
	ctx context.Context, sharedDB, metaDB *sql.DB, titleSlug string, limit int,
) ([]DataQualityIssue, error) {
	if metaDB != nil {
		exists, err := metaTableExists(ctx, metaDB, "playlists_catalog")
		if err != nil {
			return nil, err
		}
		if !exists {
			slog.DebugContext(ctx, "data_quality: playlists_catalog absente du schéma metadata — détecteur orphan_playlists non applicable",
				"module", "monitoring", "title", titleSlug)
			return []DataQualityIssue{}, nil
		}
	}
	catalog := map[string]struct{}{}
	if metaDB != nil {
		rows, err := metaDB.QueryContext(ctx,
			`SELECT playlist_asset_id FROM playlists_catalog WHERE title_slug = ?`, titleSlug)
		if err != nil {
			return nil, fmt.Errorf("load playlists_catalog: %w", err)
		}
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr == nil {
				catalog[id] = struct{}{}
			}
		}
		rows.Close()
	}

	rows, err := sharedDB.QueryContext(ctx, fmt.Sprintf(`
		SELECT playlist_id, COALESCE(MAX(playlist_name), ''), COUNT(*), MAX(%s)
		FROM match_registry
		WHERE playlist_id IS NOT NULL AND playlist_id <> ''
		GROUP BY playlist_id`, dqTimestampExpr))
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var out []DataQualityIssue
	for rows.Next() {
		var id, name string
		var n int
		var last sql.NullTime
		if scanErr := rows.Scan(&id, &name, &n, &last); scanErr != nil {
			return nil, scanErr
		}
		if _, ok := catalog[id]; ok {
			continue
		}
		out = append(out, DataQualityIssue{
			Kind: "orphan_playlist", ID: id, Label: name,
			Occurrences: n, LastSeen: formatNullTime(last),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Occurrences > out[j].Occurrences })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// listOrphanXUIDs liste les xuids de match_participants sans alias gamertag.
func listOrphanXUIDs(ctx context.Context, sharedDB *sql.DB, limit int) ([]DataQualityIssue, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mp.xuid, COUNT(DISTINCT mp.match_id)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
		GROUP BY mp.xuid
		ORDER BY 2 DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list orphan xuids: %w", err)
	}
	defer rows.Close()
	var out []DataQualityIssue
	for rows.Next() {
		var xuid string
		var n int
		if scanErr := rows.Scan(&xuid, &n); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, DataQualityIssue{Kind: "orphan_xuid", ID: xuid, Occurrences: n})
	}
	return out, rows.Err()
}

// loadResolvedMapNames charge les noms de maps résolus (≠ UUID) pour aider
// NormalizeModeLabel à stripper " on {map}" / " sur {map}".
func loadResolvedMapNames(ctx context.Context, sharedDB *sql.DB) ([]string, error) {
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT DISTINCT map_name FROM match_registry
		WHERE map_name IS NOT NULL AND map_name <> ''
		  AND (map_id IS NULL OR map_name <> map_id)`)
	if err != nil {
		return nil, fmt.Errorf("load map names: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr == nil && name != "" {
			out = append(out, name)
		}
	}
	return out, rows.Err()
}

func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}
