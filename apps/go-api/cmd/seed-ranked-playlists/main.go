//go:build cgo

// cmd/seed-ranked-playlists — alimente playlists_catalog depuis les CSR snapshots
// de tous les joueurs synchronisés.
//
// Le catalogue est peuplé réactivement depuis l'historique de matchs. Les playlists
// ranked jamais jouées par les joueurs (ex. Ranked Snipers) n'apparaissent donc
// jamais — même si l'API Waypoint retourne un CSR pour ces playlists lors du sync
// CSR. Ce CLI comble ce gap en lisant player_csr_snapshots (qui contient les
// playlist_id reçus de l'API Waypoint) et en insérant les entrées manquantes dans
// playlists_catalog.
//
// IMPORTANT : stopper le serveur API avant de lancer. metadata.duckdb est ouvert
// en RW au boot serveur (DuckDB interdit deux writers simultanés sur Windows).
//
// Usage :
//
//	go run cmd/seed-ranked-playlists/main.go [-dry-run]
//	go run cmd/seed-ranked-playlists/main.go -metadata-db /chemin/metadata.duckdb -players-dir /chemin/players
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

var (
	metaDBPath = flag.String("metadata-db",
		filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb"),
		"chemin metadata.duckdb (RW requis — stopper le serveur avant)")
	playersDirFlag = flag.String("players-dir",
		filepath.Join("..", "..", "data", "titles", "halo_infinite", "players"),
		"dossier racine des joueurs (contient {GT}/stats.duckdb)")
	titleSlugFlag = flag.String("title", "halo_infinite", "title slug (utilisé dans playlists_catalog)")
	dryRun        = flag.Bool("dry-run", false, "affiche sans insérer dans la DB")
)

type playlistEntry struct {
	PlaylistID   string
	PlaylistName string
}

func main() {
	flag.Parse()

	discovered := collectFromPlayerDBs(*playersDirFlag)
	if len(discovered) == 0 {
		fmt.Println("Aucun playlist_id ranked trouvé dans les CSR snapshots joueurs.")
		return
	}

	fmt.Printf("Découverts : %d playlist(s) dans les CSR snapshots\n\n", len(discovered))
	for _, e := range discovered {
		fmt.Printf("  %-38s  %s\n", e.PlaylistID, e.PlaylistName)
	}
	fmt.Println()

	if *dryRun {
		fmt.Println("[dry-run] Rien inséré.")
		return
	}

	meta, err := sql.Open("duckdb", *metaDBPath)
	if err != nil {
		log.Fatalf("open metadata: %v", err)
	}
	defer meta.Close()

	inserted, skipped, errored := 0, 0, 0
	now := time.Now().UTC()

	for _, e := range discovered {
		name := e.PlaylistName
		if name == "" || isUUID(name) {
			name = e.PlaylistID
		}

		res, execErr := meta.Exec(`
			INSERT INTO playlists_catalog
			  (title_slug, playlist_asset_id, current_version_id, name_canonical,
			   experience, is_ranked, is_active, first_seen_at, last_seen_at)
			VALUES (?, ?, '', ?, 'ranked', TRUE, TRUE, ?, ?)
			ON CONFLICT (title_slug, playlist_asset_id) DO NOTHING`,
			*titleSlugFlag, e.PlaylistID, name, now, now,
		)
		if execErr != nil {
			log.Printf("insert %s: %v", e.PlaylistID, execErr)
			errored++
			continue
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
			fmt.Printf("  + %s  (%s)\n", name, e.PlaylistID)
		} else {
			skipped++
		}
	}

	fmt.Printf("\nRésultat : %d ajoutés, %d déjà présents, %d erreurs\n", inserted, skipped, errored)
}

// collectFromPlayerDBs scanne tous les dossiers joueurs et agrège les
// playlist_id ranked depuis player_csr_snapshots (ouverture en read-only).
func collectFromPlayerDBs(playersRoot string) []playlistEntry {
	entries, err := os.ReadDir(playersRoot)
	if err != nil {
		log.Fatalf("ReadDir %s: %v", playersRoot, err)
	}

	seen := make(map[string]playlistEntry)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbPath := filepath.Join(playersRoot, e.Name(), "stats.duckdb")
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		extractFromPlayerDB(dbPath, e.Name(), seen)
	}

	result := make([]playlistEntry, 0, len(seen))
	for _, v := range seen {
		result = append(result, v)
	}
	return result
}

func extractFromPlayerDB(dbPath, gamertag string, seen map[string]playlistEntry) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		log.Printf("open %s: %v", gamertag, err)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT DISTINCT
		  playlist_id,
		  COALESCE(NULLIF(TRIM(playlist_name), ''), '')
		FROM player_csr_snapshots
		WHERE playlist_id IS NOT NULL
		  AND TRIM(playlist_id) != ''`)
	if err != nil {
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = playlistEntry{
				PlaylistID:   id,
				PlaylistName: strings.TrimSpace(name),
			}
		}
		count++
	}
	if count > 0 {
		fmt.Printf("  [%s] %d snapshot(s) CSR trouvés\n", gamertag, count)
	}
}

// isUUID retourne true si s ressemble à un UUID v4 (36 chars, 4 tirets).
func isUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}
