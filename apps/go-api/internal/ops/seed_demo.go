// Package ops — seed_demo.go : génération du jeu de données démo pour demo.levelup.info.
//
// Portage de scripts/prepare_demo_data.py (supprimé au commit c03707aa lors
// du nettoyage Python legacy). Extrait N matchs récents d'un joueur source,
// anonymise son xuid en "0000000000000000", recrée les vues V6, et écrit les
// configs db_profiles/app_settings.
//
// Utilisation :
//
//	docker compose run --rm levelup levelup seed-demo --gamertag JGtm \
//	    --max-matches 50 --service-tag SPTA --out data/demo
//
// Pipeline :
//  1. Résoudre source_xuid depuis db_profiles.json
//  2. Sélectionner les N match_ids les plus récents (ORDER BY start_time DESC)
//  3. Copier metadata.duckdb intégralement (référentiels neutres)
//  4. Extraire shared_matches_v2.duckdb filtré + anonymisé
//  5. Extraire player stats.duckdb filtré + anonymisé
//  6. Extraire médias (5 max) + media_registry.json
//  7. Écrire db_profiles.json + app_settings.json
//
// Idempotent : un 2e run écrase les fichiers de sortie.
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// SeedDemoOptions configure la génération démo.
type SeedDemoOptions struct {
	// Sources (résolues depuis db_profiles.json si vides via ResolveFromProfiles).
	SourcePlayerDB string // ex: data/players/JGtm/stats.duckdb
	SourceSharedDB string // ex: data/warehouse/shared_matches_v2.duckdb
	SourceMetaDB   string // ex: data/warehouse/metadata.duckdb
	SourceXUID     string // xuid réel du joueur source (lu depuis profiles si vide)

	// Cible et anonymisation.
	OutDir       string // ex: data/demo
	MaxMatches   int    // 50 par défaut
	DemoGamertag string // "DEMO" par défaut
	DemoXUID     string // "0000000000000000" par défaut
	SourceLabel  string // gamertag source affiché en log (ex: "JGtm")
	ServiceTag   string // ex: "SPTA" — profile_service_tag dans app_settings
	IncludeMedia bool   // true par défaut côté CLI
	MaxMedia     int    // 5 par défaut
}

// SeedDemoResult résume l'exécution.
type SeedDemoResult struct {
	OutDir         string
	MatchIDs       []string
	MetadataCopied bool
	SharedRows     map[string]int // table → rows insérées
	PlayerRows     map[string]int
	MediaCopied    int
	ConfigsWritten bool
	Duration       time.Duration
}

// Constantes par défaut.
const (
	DefaultDemoXUID     = "0000000000000000"
	DefaultDemoGamertag = "DEMO"
	DefaultMaxMatches   = 50
	DefaultMaxMedia     = 5
)

// matchIDInClause est la clause WHERE templated pour filtrer les tables shared/player
// par match_ids. Factorisé pour éviter la duplication (9 occurrences dans les tables maps).
const matchIDInClause = "match_id IN (%s)"

// Table player principale, référencée aussi par anonymizeXUIDInTables (4 occurrences).
const playerEnrichmentTable = "player_match_enrichment"

// Tables shared/player référencées plus de 3 fois dans seed_demo + tests.
const (
	tableMatchRegistry     = "match_registry"
	tableMatchParticipants = "match_participants"
)

// Tables shared à extraire avec leur clause WHERE (les ? sont les match_ids
// formatés en littéraux SQL via formatIDsLiteral — DuckDB ne supporte pas
// les placeholders dans CREATE TABLE AS SELECT).
//
// xuid_aliases : SELECT * WHERE xuid IN (xuids des match_participants).
var sharedTablesWhere = []struct {
	name  string
	where string
}{
	{tableMatchRegistry, matchIDInClause},
	{tableMatchParticipants, matchIDInClause},
	{"medals_earned", matchIDInClause},
	{"highlight_events", matchIDInClause},
	{"weapon_kills", matchIDInClause},
	{"killer_victim_pairs", matchIDInClause},
	{"xuid_aliases", "xuid IN (SELECT DISTINCT xuid FROM match_participants WHERE match_id IN (%s))"},
}

// Tables player à extraire. sessions/career_progression copiés intégralement
// (pas filtrés par match_id car données globales joueur).
var playerTablesWhere = []struct {
	name  string
	where string
}{
	{playerEnrichmentTable, matchIDInClause},
	{"match_citations", matchIDInClause},
	{"sessions", "1=1"},
	{"career_progression", "1=1"},
	{"sync_meta", "key NOT IN ('msal_token_cache')"},
	{"match_skill_rank", matchIDInClause},
}

// SeedDemo exécute le pipeline complet.
func SeedDemo(ctx context.Context, opts SeedDemoOptions) (SeedDemoResult, error) {
	start := time.Now()
	res := SeedDemoResult{OutDir: opts.OutDir, SharedRows: map[string]int{}, PlayerRows: map[string]int{}}

	if err := validateSeedDemoOpts(&opts); err != nil {
		return res, fmt.Errorf("seed-demo: %w", err)
	}

	slog.InfoContext(ctx, "seed-demo: démarrage",
		"source_gamertag", opts.SourceLabel,
		"source_xuid", opts.SourceXUID,
		"max_matches", opts.MaxMatches,
		"out_dir", opts.OutDir,
		"include_media", opts.IncludeMedia,
	)

	// 1. Sélectionner les match_ids
	matchIDs, err := selectRecentMatchIDs(ctx, opts.SourceSharedDB, opts.SourceXUID, opts.MaxMatches)
	if err != nil {
		return res, fmt.Errorf("seed-demo: select matches: %w", err)
	}
	if len(matchIDs) == 0 {
		return res, fmt.Errorf("seed-demo: aucun match trouvé pour xuid=%s dans %s",
			opts.SourceXUID, opts.SourceSharedDB)
	}
	res.MatchIDs = matchIDs
	slog.InfoContext(ctx, "seed-demo: matchs sélectionnés", "count", len(matchIDs))

	// 2. Copie metadata.duckdb
	outMeta := filepath.Join(opts.OutDir, "warehouse", "metadata.duckdb")
	if err := copyMetadataFile(opts.SourceMetaDB, outMeta); err != nil {
		return res, fmt.Errorf("seed-demo: copy metadata: %w", err)
	}
	res.MetadataCopied = true
	slog.InfoContext(ctx, "seed-demo: metadata copiée", "out", outMeta)

	// 3. Extraction shared
	outShared := filepath.Join(opts.OutDir, "warehouse", "shared_matches_v2.duckdb")
	sharedRows, err := extractSharedTables(ctx, opts.SourceSharedDB, outShared, matchIDs,
		opts.SourceXUID, opts.DemoXUID)
	if err != nil {
		return res, fmt.Errorf("seed-demo: extract shared: %w", err)
	}
	res.SharedRows = sharedRows
	slog.InfoContext(ctx, "seed-demo: shared extraite", "out", outShared, "rows", sharedRows)

	// 4. Extraction player
	outPlayer := filepath.Join(opts.OutDir, "players", opts.DemoGamertag, "stats.duckdb")
	playerRows, err := extractPlayerTables(ctx, opts.SourcePlayerDB, outPlayer, matchIDs,
		opts.SourceXUID, opts.DemoXUID)
	if err != nil {
		return res, fmt.Errorf("seed-demo: extract player: %w", err)
	}
	res.PlayerRows = playerRows
	slog.InfoContext(ctx, "seed-demo: player extraite", "out", outPlayer, "rows", playerRows)

	// 5. Médias (optionnel)
	if opts.IncludeMedia {
		mediaDir := filepath.Join(opts.OutDir, "players", opts.DemoGamertag, "media")
		mediaCount, mediaErr := extractDemoMedia(ctx, opts.SourcePlayerDB, opts.SourceSharedDB,
			outPlayer, mediaDir, matchIDs, opts.DemoXUID, opts.MaxMedia)
		if mediaErr != nil {
			// Non bloquant : la démo fonctionne sans média.
			slog.WarnContext(ctx, "seed-demo: extraction média partielle",
				"err", mediaErr, "copied", mediaCount)
		}
		res.MediaCopied = mediaCount
	}

	// 6. Configs
	if err := writeDemoConfigs(opts.OutDir, opts.DemoXUID, opts.SourceLabel,
		opts.ServiceTag, res.MediaCopied > 0); err != nil {
		return res, fmt.Errorf("seed-demo: write configs: %w", err)
	}
	res.ConfigsWritten = true

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "seed-demo: terminé",
		"duration", res.Duration,
		"matches", len(matchIDs),
		"media", res.MediaCopied,
	)
	return res, nil
}

// validateSeedDemoOpts applique les valeurs par défaut + vérifie les chemins source.
func validateSeedDemoOpts(opts *SeedDemoOptions) error {
	if opts.SourcePlayerDB == "" || opts.SourceSharedDB == "" || opts.SourceMetaDB == "" {
		return fmt.Errorf("source paths required (SourcePlayerDB, SourceSharedDB, SourceMetaDB)")
	}
	if opts.SourceXUID == "" {
		return fmt.Errorf("SourceXUID required (use ResolveSourceXUIDFromProfiles to read from db_profiles.json)")
	}
	if opts.OutDir == "" {
		return fmt.Errorf("OutDir required")
	}
	if opts.MaxMatches <= 0 {
		opts.MaxMatches = DefaultMaxMatches
	}
	if opts.DemoXUID == "" {
		opts.DemoXUID = DefaultDemoXUID
	}
	if opts.DemoGamertag == "" {
		opts.DemoGamertag = DefaultDemoGamertag
	}
	if opts.MaxMedia <= 0 {
		opts.MaxMedia = DefaultMaxMedia
	}
	for _, p := range []string{opts.SourcePlayerDB, opts.SourceSharedDB, opts.SourceMetaDB} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("source DB introuvable: %s: %w", p, err)
		}
	}
	return nil
}

// ResolveSourceXUIDFromProfiles lit db_profiles.json et retourne le xuid + le
// chemin player DB pour le gamertag donné.
func ResolveSourceXUIDFromProfiles(profilesPath, gamertag string) (xuid, playerDBPath string, err error) {
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return "", "", fmt.Errorf("read profiles: %w", err)
	}
	var doc struct {
		Profiles map[string]struct {
			XUID   string `json:"xuid"`
			DBPath string `json:"db_path"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", "", fmt.Errorf("parse profiles: %w", err)
	}
	p, ok := doc.Profiles[gamertag]
	if !ok {
		return "", "", fmt.Errorf("gamertag %q introuvable dans %s", gamertag, profilesPath)
	}
	if p.XUID == "" {
		return "", "", fmt.Errorf("profile %q sans xuid", gamertag)
	}
	return p.XUID, p.DBPath, nil
}

// selectRecentMatchIDs retourne les N match_ids les plus récents du joueur dans shared.
func selectRecentMatchIDs(ctx context.Context, sharedDBPath, xuid string, limit int) ([]string, error) {
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT mr.match_id
		FROM match_registry mr
		JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ?
		ORDER BY mr.start_time DESC
		LIMIT ?`, xuid, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// copyMetadataFile copie src→dst (file copy bit-à-bit, équivalent shutil.copy2).
// Crée le répertoire parent si nécessaire.
func copyMetadataFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

// formatIDsLiteral retourne "'id1', 'id2', ..." pour interpolation SQL.
// DuckDB ne supporte pas les ? dans CREATE TABLE AS SELECT, donc on inline.
// Les match_ids étant des UUID hex contrôlés (pas user input), pas de SQLi.
func formatIDsLiteral(ids []string) string {
	if len(ids) == 0 {
		return "''" // pas de match → clause vide qui ne matche rien
	}
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		// Sanity check : un match_id Halo est UUID hex (a-f, 0-9, -). Refuser toute
		// chaîne avec apostrophe pour défense en profondeur (même si fournis par DB).
		if strings.ContainsAny(id, "'\"\\;") {
			continue
		}
		quoted = append(quoted, "'"+id+"'")
	}
	if len(quoted) == 0 {
		return "''"
	}
	return strings.Join(quoted, ", ")
}

// extractSharedTables crée out_shared en ATTACHant src_shared et en copiant
// les 7 tables filtrées sur match_ids. Anonymise sourceXUID → demoXUID dans
// match_participants et xuid_aliases. Recrée les vues V6 à la fin.
func extractSharedTables(
	ctx context.Context,
	srcPath, dstPath string,
	matchIDs []string,
	sourceXUID, demoXUID string,
) (map[string]int, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	// Supprimer dst existant pour idempotence (DuckDB échoue sur ATTACH si fichier
	// existe avec schéma différent).
	_ = os.Remove(dstPath)

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return nil, fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS src (READ_ONLY)", srcPath)); err != nil {
		return nil, fmt.Errorf("attach src: %w", err)
	}
	defer func() { _, _ = dst.ExecContext(ctx, "DETACH src") }()

	idsLit := formatIDsLiteral(matchIDs)
	counts := make(map[string]int, len(sharedTablesWhere))
	for _, t := range sharedTablesWhere {
		where := fmt.Sprintf(t.where, idsLit)
		stmt := fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM src.%s WHERE %s`, t.name, t.name, where)
		if _, err := dst.ExecContext(ctx, stmt); err != nil {
			return counts, fmt.Errorf("extract %s: %w", t.name, err)
		}
		var n int
		if err := dst.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)).Scan(&n); err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
		counts[t.name] = n
	}

	// Anonymisation : sourceXUID → demoXUID.
	if err := anonymizeXUIDInTables(ctx, dst, sourceXUID, demoXUID,
		[]string{"match_participants", "xuid_aliases"}); err != nil {
		return counts, fmt.Errorf("anonymize: %w", err)
	}

	// Recréer les vues V6 (v_gamertag_lookup, v_match_full, v_weapon_kills).
	if err := recreateSharedViews(ctx, dst); err != nil {
		// Non-bloquant : log mais continue (vues optionnelles côté demo).
		slog.WarnContext(ctx, "seed-demo: recreate views partielle", "err", err)
	}
	return counts, nil
}

// extractPlayerTables extrait les tables enrichment/citations/sessions/career/sync_meta/skill_rank.
// Anonymise sourceXUID → demoXUID + met à jour sync_meta.xuid si présent.
func extractPlayerTables(
	ctx context.Context,
	srcPath, dstPath string,
	matchIDs []string,
	sourceXUID, demoXUID string,
) (map[string]int, error) {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	_ = os.Remove(dstPath)

	dst, err := sql.Open("duckdb", dstPath)
	if err != nil {
		return nil, fmt.Errorf("open dst: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS src (READ_ONLY)", srcPath)); err != nil {
		return nil, fmt.Errorf("attach src: %w", err)
	}
	defer func() { _, _ = dst.ExecContext(ctx, "DETACH src") }()

	idsLit := formatIDsLiteral(matchIDs)
	counts := make(map[string]int, len(playerTablesWhere))
	for _, t := range playerTablesWhere {
		where := t.where
		if strings.Contains(where, "%s") {
			where = fmt.Sprintf(where, idsLit)
		}
		stmt := fmt.Sprintf(`CREATE TABLE %s AS SELECT * FROM src.%s WHERE %s`, t.name, t.name, where)
		if _, err := dst.ExecContext(ctx, stmt); err != nil {
			// Tolérant : certaines tables peuvent ne pas exister (ex : match_skill_rank
			// sur DB legacy). Log et continue.
			slog.WarnContext(ctx, "seed-demo: extract player table partielle",
				"table", t.name, "err", err)
			counts[t.name] = 0
			continue
		}
		var n int
		if err := dst.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)).Scan(&n); err != nil {
			return counts, fmt.Errorf("count %s: %w", t.name, err)
		}
		counts[t.name] = n
	}

	// Anonymisation : sourceXUID → demoXUID dans tables avec colonne xuid.
	if err := anonymizeXUIDInTables(ctx, dst, sourceXUID, demoXUID,
		[]string{playerEnrichmentTable, "match_skill_rank"}); err != nil {
		// Non bloquant
		slog.WarnContext(ctx, "seed-demo: anonymize player partielle", "err", err)
	}
	// sync_meta.value pour la clé 'xuid' canonique.
	if _, err := dst.ExecContext(ctx,
		`UPDATE sync_meta SET value = ? WHERE key = 'xuid' AND value = ?`,
		demoXUID, sourceXUID); err != nil {
		slog.WarnContext(ctx, "seed-demo: update sync_meta xuid partielle", "err", err)
	}
	return counts, nil
}

// anonymizeXUIDInTables exécute UPDATE ... SET xuid = ? WHERE xuid = ? sur les
// tables fournies. Tolérant : les tables sans colonne xuid sont skip avec warn.
func anonymizeXUIDInTables(
	ctx context.Context,
	db *sql.DB,
	sourceXUID, demoXUID string,
	tables []string,
) error {
	for _, t := range tables {
		stmt := fmt.Sprintf(`UPDATE %s SET xuid = ? WHERE xuid = ?`, t)
		if _, err := db.ExecContext(ctx, stmt, demoXUID, sourceXUID); err != nil {
			return fmt.Errorf("update %s: %w", t, err)
		}
	}
	return nil
}

// recreateSharedViews recrée v_gamertag_lookup, v_match_full, v_weapon_kills
// requis par les pages analytics. Tolère l'absence de tables sources.
func recreateSharedViews(ctx context.Context, db *sql.DB) error {
	// v_gamertag_lookup : FULL OUTER JOIN xuid_aliases + match_participants.
	// Format simplifié (sans CASE bot — la démo n'a pas besoin de noms bots officiels).
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE VIEW v_gamertag_lookup AS
		SELECT
			COALESCE(xa.xuid, mp.xuid) AS xuid,
			CASE
				WHEN xa.gamertag IS NOT NULL AND xa.gamertag != '' THEN xa.gamertag
				WHEN mp.gamertag IS NOT NULL AND mp.gamertag != '' THEN mp.gamertag
				ELSE COALESCE(xa.xuid, mp.xuid)
			END AS gamertag
		FROM xuid_aliases xa
		FULL OUTER JOIN (
			SELECT xuid, MAX(gamertag) AS gamertag
			FROM match_participants GROUP BY xuid
		) mp ON xa.xuid = mp.xuid`); err != nil {
		return fmt.Errorf("v_gamertag_lookup: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`); err != nil {
		return fmt.Errorf("v_match_full: %w", err)
	}
	// v_weapon_kills : tolère table absente.
	_, _ = db.ExecContext(ctx, `
		CREATE OR REPLACE VIEW v_weapon_kills AS
		SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id
		FROM weapon_kills`)
	return nil
}

// writeDemoConfigs écrit db_profiles.json + app_settings.json dans outDir.
func writeDemoConfigs(outDir, demoXUID, gamertag, serviceTag string, mediaEnabled bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// db_profiles.json
	profiles := map[string]any{
		"version":        "2.1",
		"warehouse_path": "data/warehouse",
		"metadata_db":    "data/warehouse/metadata.duckdb",
		"profiles": map[string]any{
			DefaultDemoGamertag: map[string]any{
				"db_path":         "data/players/" + DefaultDemoGamertag + "/stats.duckdb",
				"xuid":            demoXUID,
				"waypoint_player": gamertag,
			},
		},
	}
	if err := writeJSONFile(filepath.Join(outDir, "db_profiles.json"), profiles); err != nil {
		return fmt.Errorf("db_profiles.json: %w", err)
	}
	// app_settings.json
	settings := map[string]any{
		"lang":                                "fr",
		"media_enabled":                       mediaEnabled,
		"spnkr_refresh_on_start":              false,
		"spnkr_refresh_on_manual_refresh":     false,
		"spnkr_refresh_max_matches":           0,
		"spnkr_refresh_with_highlight_events": false,
		"spnkr_refresh_with_backfill":         false,
		"spnkr_refresh_backfill_medals":       false,
		"spnkr_refresh_backfill_events":       false,
		"spnkr_refresh_backfill_skill":        false,
		"discord_notifications_enabled":       false,
		"doppler_enabled":                     false,
		"tailscale_funnel_enabled":            false,
		"profile_assets_download_enabled":     false,
		"profile_api_enabled":                 false,
		"profile_service_tag":                 serviceTag,
		"repository_mode":                     "duckdb",
		"enable_duckdb_analytics":             true,
	}
	if err := writeJSONFile(filepath.Join(outDir, "app_settings.json"), settings); err != nil {
		return fmt.Errorf("app_settings.json: %w", err)
	}
	return nil
}

func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
