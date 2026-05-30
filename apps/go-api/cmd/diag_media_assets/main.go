// diag_media_assets — inspecte (lecture seule) la résolution des noms d'assets
// média : map_images_registry, maps_catalog, mode_name_tr, asset_translations
// playlist, et l'état de quelques match_registry (map_id/map_name/pair_name).
//
// Usage :
//
//	go run ./cmd/diag_media_assets [sharedPath] [metadataPath]
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openRO(path string) *sql.DB {
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		fmt.Printf("open %s: %v (skip)\n\n", path, err)
		return nil
	}
	// Force une vraie connexion pour détecter le verrou tout de suite.
	if err := db.Ping(); err != nil {
		fmt.Printf("ping %s: %v (skip — base verrouillée par le serveur)\n\n", path, err)
		_ = db.Close()
		return nil
	}
	return db
}

func dumpQuery(db *sql.DB, title, q string, args ...any) {
	fmt.Printf("--- %s ---\n", title)
	if db == nil {
		fmt.Printf("  (base indisponible)\n\n")
		return
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		fmt.Printf("  ERR: %v\n\n", err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	n := 0
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Printf("  scan: %v\n", err)
			continue
		}
		line := "  "
		for i, c := range cols {
			line += fmt.Sprintf("%s=%v  ", c, vals[i])
		}
		fmt.Println(line)
		n++
	}
	if n == 0 {
		fmt.Println("  (aucune ligne)")
	}
	fmt.Println()
}

func main() {
	base := "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/"
	sharedPath := base + "shared_matches_v2.duckdb"
	metaPath := base + "metadata.duckdb"
	if len(os.Args) > 1 {
		sharedPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		metaPath = os.Args[2]
	}

	meta := openRO(metaPath)
	if meta != nil {
		defer meta.Close()
	}
	shared := openRO(sharedPath)
	if shared != nil {
		defer shared.Close()
	}

	fmt.Println("=== METADATA ===")
	dumpQuery(meta, "mode_name_tr (sous-modes du picker)",
		`SELECT mode_en, lang, name FROM mode_name_tr WHERE mode_en IN ('Slayer','Team Slayer','Neutral Flag CTF') ORDER BY mode_en, lang`)
	dumpQuery(meta, "asset_translations playlist Quick Play",
		`SELECT asset_id, lang, name FROM asset_translations WHERE asset_type='playlist' AND (asset_id='1b1691dc-d8b9-4b1f-825d-cb1c065184c1' OR name ILIKE '%quick%') ORDER BY asset_id, lang`)
	dumpQuery(meta, "map_images_registry count par title",
		`SELECT title_id, COUNT(*) AS n FROM map_images_registry GROUP BY title_id`)
	dumpQuery(meta, "map_images_registry exemples",
		`SELECT title_id, map_id, local_path FROM map_images_registry LIMIT 5`)
	dumpQuery(meta, "maps_catalog Cliffhanger (5324364b)",
		`SELECT title_slug, map_asset_id, name_canonical FROM maps_catalog WHERE map_asset_id='5324364b-39a8-4f93-96a6-b80a1f18ce8a'`)
	dumpQuery(meta, "asset_translations map 5324364b",
		`SELECT asset_id, lang, name FROM asset_translations WHERE asset_type='map' AND asset_id='5324364b-39a8-4f93-96a6-b80a1f18ce8a' ORDER BY lang`)

	fmt.Println("=== SHARED match_registry ===")
	dumpQuery(shared, "Domicile (cd89b091 — match du picker) + GUID (4cb4a8d0)",
		`SELECT match_id, COALESCE(map_id,'<NULL>') AS map_id, COALESCE(map_name,'<NULL>') AS map_name,
		        COALESCE(map_name_fr,'<NULL>') AS map_name_fr, COALESCE(pair_name,'<NULL>') AS pair_name,
		        COALESCE(playlist_name,'<NULL>') AS playlist_name, COALESCE(playlist_id,'<NULL>') AS playlist_id
		 FROM match_registry
		 WHERE match_id IN ('cd89b091-6fb9-457d-96dd-67f11585fcfa','4cb4a8d0-962b-436e-a6fc-98c1a5e36ca9')`)
	dumpQuery(shared, "map_id vide vs rempli sur match_registry (global)",
		`SELECT (map_id IS NULL OR map_id='') AS map_id_empty, COUNT(*) AS n FROM match_registry GROUP BY 1`)
}
