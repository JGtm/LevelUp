// Script one-shot : importe les traductions de tracks Battle Pass depuis
// data/investigation/battlepass/*/tracks/*.json dans metadata.duckdb.
//
// Les fichiers locaux contiennent les noms multilingues (Name.translations).
// Ce script NE touche PAS à battlepass_track_definitions (index DuckDB fragile) :
// il lit le content_hash existant depuis la DB et n'écrit que dans battlepass_track_translations.
//
// Usage : cd apps/go-api && go run ./scripts/import_bp_tracks/
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// trackJSON correspond à la structure des fichiers locaux dans
// data/investigation/battlepass/*/tracks/*.json
type trackJSON struct {
	TrackID             string         `json:"TrackId"`
	XpPerRank           int            `json:"XpPerRank"`
	SummaryImagePath    string         `json:"SummaryImagePath"`
	BackgroundImagePath string         `json:"BackgroundImagePath"`
	Name                localizedField `json:"Name"`
	Description         localizedField `json:"Description"`
}

type localizedField struct {
	Value        string            `json:"value"`
	Translations map[string]string `json:"translations"`
}

func main() {
	dbPath := filepath.Join("..", "..", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")
	if _, err := os.Stat(dbPath); err != nil {
		slog.Error("metadata.duckdb introuvable", "path", dbPath)
		os.Exit(1)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Charger les content_hash existants depuis battlepass_track_definitions.
	// Clé : reward_track_path → content_hash (is_current=TRUE).
	dbHashes := map[string]string{}
	rows, err := db.Query(`SELECT reward_track_path, content_hash FROM battlepass_track_definitions WHERE is_current = TRUE`)
	if err != nil {
		slog.Error("lecture battlepass_track_definitions", "err", err)
		os.Exit(1)
	}
	for rows.Next() {
		var path, hash string
		rows.Scan(&path, &hash)
		dbHashes[path] = hash
	}
	rows.Close()
	slog.Info("tracks en DB", "count", len(dbHashes))

	// Glob : tous les fichiers tracks pour tous les gamertags.
	glob := filepath.Join("..", "..", "data", "investigation", "battlepass", "*", "tracks", "*.json")
	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		slog.Error("aucun fichier track trouvé", "glob", glob)
		os.Exit(1)
	}
	slog.Info("fichiers tracks trouvés", "count", len(files))

	now := time.Now()
	written, skipped, dups, notInDB := 0, 0, 0, 0

	// Dé-dupliquer : un même track peut exister dans plusieurs sous-répertoires.
	seen := map[string]struct{}{}

	for _, fpath := range files {
		raw, err := os.ReadFile(fpath)
		if err != nil {
			slog.Warn("lecture", "file", fpath, "err", err)
			skipped++
			continue
		}

		var t trackJSON
		if err := json.Unmarshal(raw, &t); err != nil {
			slog.Warn("parse", "file", fpath, "err", err)
			skipped++
			continue
		}

		// Déduire le reward_track_path depuis le nom de fichier.
		base := filepath.Base(fpath)
		namePart := strings.TrimSuffix(base, filepath.Ext(base))
		idx := strings.LastIndex(namePart, "-")
		if idx > 0 && len(namePart)-idx-1 == 10 {
			namePart = namePart[:idx]
		}
		rewardTrackPath := fmt.Sprintf("RewardTracks/Operations/%s.json", namePart)

		if _, alreadySeen := seen[rewardTrackPath]; alreadySeen {
			dups++
			continue
		}
		seen[rewardTrackPath] = struct{}{}

		// Lire le content_hash depuis la DB (ne pas recalculer depuis le fichier local
		// dont le contenu peut différer du payload GameCMS).
		contentHash, ok := dbHashes[rewardTrackPath]
		if !ok {
			slog.Warn("track absent de battlepass_track_definitions, ignoré", "path", rewardTrackPath)
			notInDB++
			continue
		}

		// Construire la liste des langues à écrire (toujours inclure en-US).
		langs := map[string]struct{}{"en-US": {}}
		for lang := range t.Name.Translations {
			langs[lang] = struct{}{}
		}

		trackWritten := 0
		for lang := range langs {
			name := t.Name.Translations[lang]
			if name == "" {
				name = t.Name.Value
			}
			if name == "" {
				continue
			}
			// INSERT only — battlepass_track_translations est vide au premier passage.
			// En cas de re-run, l'INSERT silencieusement ignore les conflits via le fallback UPDATE.
			resT, _ := db.Exec(`
				UPDATE battlepass_track_translations
				SET track_name=?, last_seen_at=?
				WHERE reward_track_path=? AND content_hash=? AND lang=?`,
				name, now, rewardTrackPath, contentHash, lang)
			if n, _ := resT.RowsAffected(); n == 0 {
				_, _ = db.Exec(`
					INSERT INTO battlepass_track_translations
						(reward_track_path, content_hash, lang, track_name, first_seen_at, last_seen_at)
					VALUES (?, ?, ?, ?, ?, ?)`,
					rewardTrackPath, contentHash, lang, name, now, now)
			}
			trackWritten++
		}

		slog.Info("traductions écrites", "path", rewardTrackPath, "langs", trackWritten, "name_en", t.Name.Value)
		written++
	}

	// Vérification finale.
	var totalTransl int
	_ = db.QueryRow(`SELECT COUNT(*) FROM battlepass_track_translations`).Scan(&totalTransl)

	slog.Info("import terminé",
		"tracks_written", written,
		"skipped", skipped,
		"dups_ignored", dups,
		"not_in_db", notInDB,
		"total_translations", totalTransl)
}
