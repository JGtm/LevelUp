// backfill-media-hls : convertit en HLS les vidéos média existantes (MKV/AVI ou
// multipistes) qui n'ont pas encore de version HLS, et reprend les transcodings
// interrompus (transcode_status='processing' laissé sans hls_path après un crash).
//
// Réutilise exactement la pipeline de l'upload (ops.DetectHLSNeeded +
// ops.RunHLSTranscode) : copy si possible (Opus/H.264/AV1 conservés), suppression
// du source après succès, remux WebM legacy conservé en cas d'échec.
//
// IMPORTANT : à lancer serveur ARRÊTÉ (écritures RW sur shared_social.duckdb).
//
// Usage :
//
//	backfill-media-hls --db data/.../shared_social.duckdb [--captures-base C:\Captures]
//	                   [--slug JGtm] [--limit 0] [--dry-run]
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ops"
)

type appSettings struct {
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
}

func loadCapturesBase(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var s appSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	return s.MediaCapturesBaseDir
}

type candidate struct {
	id         int64
	playerSlug string
	filePath   string
}

// selectCandidates ouvre la DB en lecture seule, liste les vidéos sans hls_path,
// puis ferme la connexion — RunHLSTranscode rouvrira la sienne en écriture
// (DuckDB n'autorise qu'un handle RW par fichier dans le process).
func selectCandidates(dbPath, onlySlug string) ([]candidate, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, err
	}
	defer db.Close() //nolint:errcheck

	q := `SELECT id, COALESCE(player_slug, ''), COALESCE(file_path, '')
	      FROM media_files
	      WHERE kind = 'video' AND hls_path IS NULL`
	args := []any{}
	if onlySlug != "" {
		q += ` AND player_slug = ?`
		args = append(args, onlySlug)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.playerSlug, &c.filePath); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type stats struct{ transcoded, skippedDirect, missing, failed int }

// processCandidate transcode un candidat si nécessaire, met à jour les stats.
func processCandidate(ctx context.Context, c candidate, base, dbPath string, dryRun bool, st *stats) {
	owner := c.playerSlug
	if owner == "" {
		owner = strings.SplitN(filepath.ToSlash(c.filePath), "/", 2)[0]
	}
	abs := ops.MediaPathStore{CapturesBase: base}.ToAbs(c.filePath)
	if _, err := os.Stat(abs); err != nil {
		st.missing++
		return
	}
	needed, err := ops.DetectHLSNeeded(ctx, abs)
	if err != nil {
		fmt.Printf("  [probe échec] %s: %v\n", c.filePath, err)
		st.failed++
		return
	}
	if !needed {
		st.skippedDirect++
		return
	}
	outDir, hlsRel := ops.HLSPathsFor(filepath.Join(base, owner), base, owner, abs)
	if dryRun {
		fmt.Printf("  [serait transcodé] %s -> %s\n", c.filePath, hlsRel)
		st.transcoded++
		return
	}
	if err := ops.RunHLSTranscode(ctx, ops.HLSTranscodeParams{
		SourceAbs: abs, OutDir: outDir, DBPath: dbPath, FileRel: c.filePath, HLSRel: hlsRel,
	}); err != nil {
		fmt.Printf("  [échec] %s: %v\n", c.filePath, err)
		st.failed++
		return
	}
	fmt.Printf("  [ok] %s -> %s\n", c.filePath, hlsRel)
	st.transcoded++
}

func main() {
	dbPath := flag.String("db", "", "path vers shared_social.duckdb (requis)")
	capturesBase := flag.String("captures-base", "", "MediaCapturesBaseDir (sinon lu depuis --settings)")
	settingsPath := flag.String("settings", "app_settings.json", "path vers app_settings.json (fallback)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (optionnel)")
	limit := flag.Int("limit", 0, "nombre max de clips à transcoder (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "lister sans transcoder")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "--db requis")
		os.Exit(2)
	}
	base := *capturesBase
	if base == "" {
		base = loadCapturesBase(*settingsPath)
	}
	if base == "" {
		fmt.Fprintln(os.Stderr, "captures base introuvable (ni --captures-base ni app_settings.json)")
		os.Exit(2)
	}

	candidates, err := selectCandidates(*dbPath, *onlySlug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "select:", err)
		os.Exit(1)
	}
	fmt.Printf("DB: %s\nCapturesBase: %s\nCandidats (vidéos sans HLS): %d\nDryRun: %v\n\n",
		*dbPath, base, len(candidates), *dryRun)

	ctx := context.Background()
	var st stats
	for _, c := range candidates {
		if *limit > 0 && st.transcoded >= *limit {
			break
		}
		processCandidate(ctx, c, base, *dbPath, *dryRun, &st)
	}

	fmt.Printf("\nRésultats:\n  transcodés     : %d\n  servis direct  : %d\n  source absente : %d\n  échecs         : %d\n",
		st.transcoded, st.skippedDirect, st.missing, st.failed)
	if *dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
}
