// Command tmp_datemap : outil jetable. Pour chaque dossier de film (8 hex = 1er
// segment du match_id), résout date + map + mode depuis shared_matches_v2.duckdb
// (READ-ONLY) en attachant metadata.duckdb pour les traductions FR.
//
// Build (cache Go isolé obligatoire, cf. consigne) :
//
//	export GOCACHE="$TEMP/gc-datemap" GOTMPDIR="$TEMP/gt-datemap"
//	CC=/c/msys64/ucrt64/bin/gcc CGO_ENABLED=1 go build -o "$TEMP/datemap.exe" ./cmd/tmp_datemap
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	repoRoot   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
	filmDir    = repoRoot + `/data/cache/film_chunks`
	sharedDB   = repoRoot + `/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
	metadataDB = repoRoot + `/data/titles/halo_infinite/warehouse/metadata.duckdb`
	limitFilms = 40
)

type result struct {
	FilmID   string `json:"filmId"`
	MatchID  string `json:"matchId"`
	Date     string `json:"date"`
	Map      string `json:"map"`
	Mode     string `json:"mode"`
	PairName string `json:"pairName"`
	MapEN    string `json:"mapEN"`
	Note     string `json:"note"`
}

func main() {
	films, err := listFilms()
	if err != nil {
		fmt.Fprintln(os.Stderr, "listFilms:", err)
		os.Exit(1)
	}

	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open shared:", err)
		os.Exit(1)
	}
	defer db.Close()

	// Attache metadata en read-only pour les traductions FR (best-effort).
	metaOK := true
	if _, err := db.Exec(fmt.Sprintf("ATTACH '%s' AS meta (READ_ONLY)", metadataDB)); err != nil {
		fmt.Fprintln(os.Stderr, "attach meta (traductions FR indisponibles):", err)
		metaOK = false
	}

	out := make([]result, 0, len(films))
	for _, film := range films {
		out = append(out, resolveFilm(db, film, metaOK))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}

func listFilms() ([]string, error) {
	entries, err := os.ReadDir(filmDir)
	if err != nil {
		return nil, err
	}
	var films []string
	for _, e := range entries {
		if e.IsDir() {
			films = append(films, e.Name())
		}
	}
	sort.Strings(films)
	if len(films) > limitFilms {
		films = films[:limitFilms]
	}
	return films, nil
}

func resolveFilm(db *sql.DB, film string, metaOK bool) result {
	r := result{FilmID: film}

	const q = `
		SELECT match_id,
		       COALESCE(CAST(start_time AS VARCHAR), ''),
		       COALESCE(map_id, ''),
		       COALESCE(map_name, ''),
		       COALESCE(pair_name, ''),
		       COALESCE(playlist_id, '')
		FROM match_registry
		WHERE match_id LIKE ? || '%'
		ORDER BY start_time
		LIMIT 1`

	var matchID, startTime, mapID, mapName, pairName, playlistID string
	err := db.QueryRow(q, film).Scan(&matchID, &startTime, &mapID, &mapName, &pairName, &playlistID)
	if err == sql.ErrNoRows {
		r.Note = "aucun match_registry pour ce prefixe"
		return r
	}
	if err != nil {
		r.Note = "erreur requete: " + err.Error()
		return r
	}

	r.MatchID = matchID
	r.Date = startTime
	r.PairName = pairName
	r.MapEN = mapName

	// Résolution FR de la map via metadata.asset_translations (map_name_fr est NULL en base).
	mapFR := ""
	if metaOK && mapID != "" {
		mapFR = lookupMapFR(db, mapID)
	}

	// Map affichée : FR si dispo, sinon EN, sinon dérivée du pair_name, sinon map_id.
	displayMap := firstNonEmpty(mapFR, mapName)
	// Mode + map dérivés du pair_name ("Mode on Map" / "Mode sur Map").
	mode, mapFromPair := splitPair(pairName)
	if displayMap == "" {
		displayMap = mapFromPair
	}
	if displayMap == "" {
		displayMap = mapID
		if displayMap != "" {
			r.Note = strings.TrimSpace(r.Note + " map non resolue: map_id brut")
		}
	}

	// Mode FR si traduction dispo.
	if metaOK && mode != "" {
		if fr := lookupModeFR(db, mode); fr != "" {
			mode = fr
		}
	}

	r.Map = displayMap
	r.Mode = mode
	if r.Mode == "" && pairName == "" {
		r.Note = strings.TrimSpace(r.Note + " pair_name vide: mode indeterminable")
	}
	return r
}

// splitPair extrait (mode, map) d'un pair_name "Mode on Map" ou "Mode sur Map".
func splitPair(pair string) (mode, mapName string) {
	if pair == "" {
		return "", ""
	}
	for _, sep := range []string{" on ", " sur "} {
		if idx := strings.LastIndex(pair, sep); idx >= 0 {
			return strings.TrimSpace(pair[:idx]), strings.TrimSpace(pair[idx+len(sep):])
		}
	}
	return strings.TrimSpace(pair), ""
}

func lookupMapFR(db *sql.DB, mapID string) string {
	const q = `
		SELECT name FROM meta.asset_translations
		WHERE asset_type = 'map' AND asset_id = ? AND lang IN ('fr-FR','fr')
		ORDER BY CASE WHEN lang = 'fr-FR' THEN 0 ELSE 1 END
		LIMIT 1`
	var name string
	if err := db.QueryRow(q, mapID).Scan(&name); err != nil {
		return ""
	}
	return name
}

func lookupModeFR(db *sql.DB, modeEN string) string {
	const q = `SELECT name FROM meta.mode_name_tr WHERE lang = 'fr' AND mode_en = ? LIMIT 1`
	var name string
	if err := db.QueryRow(q, modeEN).Scan(&name); err != nil {
		return ""
	}
	return name
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
