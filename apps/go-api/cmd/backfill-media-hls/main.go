// backfill-media-hls : convertit en HLS les vidéos média existantes (MKV/AVI ou
// multipistes) qui n'ont pas encore de version HLS, et reprend les transcodings
// interrompus (transcode_status='processing' laissé sans hls_path après un crash).
//
// Wrapper fin autour de ops.EnsurePendingHLS — la MÊME routine désormais déclenchée
// automatiquement après chaque scan/sync média (cf. internal/service/media_index_service.go).
// Ce CLI reste utile pour un gros rattrapage initial serveur arrêté.
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
	"encoding/json"
	"flag"
	"fmt"
	"os"

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

func main() {
	dbPath := flag.String("db", "", "path vers shared_social.duckdb (requis)")
	capturesBase := flag.String("captures-base", "", "MediaCapturesBaseDir (sinon lu depuis --settings)")
	settingsPath := flag.String("settings", "app_settings.json", "path vers app_settings.json (fallback)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (optionnel)")
	limit := flag.Int("limit", 0, "nombre max de clips à transcoder (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "lister sans transcoder")
	// Rattrapage serveur (disque rare) : supprimer le source après HLS par défaut
	// (comportement legacy). --delete-source=false pour conserver les originaux.
	deleteSource := flag.Bool("delete-source", true, "supprimer le fichier source après transcodage HLS réussi")
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

	fmt.Printf("DB: %s\nCapturesBase: %s\nDryRun: %v\n\n", *dbPath, base, *dryRun)

	st, err := ops.EnsurePendingHLS(context.Background(), ops.EnsureHLSParams{
		DBPath:       *dbPath,
		CapturesBase: base,
		OnlySlug:     *onlySlug,
		Limit:        *limit,
		DryRun:       *dryRun,
		DeleteSource: *deleteSource,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "sweep:", err)
		os.Exit(1)
	}

	fmt.Printf("Résultats:\n  transcodés     : %d\n  servis direct  : %d\n  source absente : %d\n  échecs         : %d\n",
		st.Transcoded, st.SkippedDirect, st.Missing, st.Failed)
	if *dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
}
