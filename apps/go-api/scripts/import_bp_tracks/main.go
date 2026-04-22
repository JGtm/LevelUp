// Script one-shot : importe les définitions de tracks Battle Pass depuis
// data/investigation/battlepass/*/tracks/*.json dans metadata.duckdb.
//
// La clé `SummaryImagePath` des fichiers locaux correspond à `BattlePassImage`
// dans battlepass_track_definitions (champ côté GameCMS = `BattlePassImage`,
// mais les fichiers locaux ont gardé le nom `SummaryImagePath`).
//
// Usage : cd apps/go-api && go run ./scripts/import_bp_tracks/
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	SummaryImagePath    string         `json:"SummaryImagePath"`    // = BattlePassImage côté GameCMS
	BackgroundImagePath string         `json:"BackgroundImagePath"` // identique
	Name                localizedField `json:"Name"`
	Description         localizedField `json:"Description"`
	// Ranks non utilisé ici (import des items géré séparément)
}

type localizedField struct {
	Value        string            `json:"value"`
	Translations map[string]string `json:"translations"`
}

func main() {
	dbPath := filepath.Join("..", "..", "data", "warehouse", "metadata.duckdb")
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

	// Glob : tous les fichiers tracks pour tous les gamertags.
	glob := filepath.Join("..", "..", "data", "investigation", "battlepass", "*", "tracks", "*.json")
	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		slog.Error("aucun fichier track trouvé", "glob", glob)
		os.Exit(1)
	}
	slog.Info("fichiers tracks trouvés", "count", len(files))

	now := time.Now()
	inserted, skipped, dups := 0, 0, 0

	// Dé-dupliquer : un même track peut exister dans plusieurs sous-répertoires
	// (ex: Chocoboflor/ et JGtm/). On garde la première occurrence.
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
		// Le fichier s'appelle p.ex. "S03BattlePass-f9f98509d7.json"
		// → reward_track_path = "RewardTracks/Operations/S03BattlePass.json"
		base := filepath.Base(fpath) // "S03BattlePass-f9f98509d7.json"
		// Supprimer le hash suffixe "-xxxxxxxxxx"
		namePart := strings.TrimSuffix(base, filepath.Ext(base)) // "S03BattlePass-f9f98509d7"
		// Trouver le dernier "-" suivi d'exactement 10 chars hex
		idx := strings.LastIndex(namePart, "-")
		if idx > 0 && len(namePart)-idx-1 == 10 {
			namePart = namePart[:idx]
		}
		rewardTrackPath := fmt.Sprintf("RewardTracks/Operations/%s.json", namePart)

		// Dé-dupliquer par reward_track_path.
		if _, alreadySeen := seen[rewardTrackPath]; alreadySeen {
			dups++
			continue
		}
		seen[rewardTrackPath] = struct{}{}

		// Calculer le content_hash (SHA256[:8] en hex, comme le serveur Go).
		h := sha256.Sum256(raw)
		contentHash := hex.EncodeToString(h[:8])

		// Invalider les anciens hashes pour ce track.
		_, err = db.Exec(`
			UPDATE battlepass_track_definitions
			SET is_current = FALSE, last_seen_at = ?
			WHERE reward_track_path = ? AND content_hash <> ? AND is_current = TRUE`,
			now, rewardTrackPath, contentHash)
		if err != nil {
			slog.Warn("update is_current", "path", rewardTrackPath, "err", err)
		}

		// UPSERT dans battlepass_track_definitions.
		// SummaryImagePath local = BattlePassImage dans le schema.
		bpImage := strings.TrimSpace(t.SummaryImagePath)
		bgImage := strings.TrimSpace(t.BackgroundImagePath)
		var xpPerRank any
		if t.XpPerRank > 0 {
			xpPerRank = t.XpPerRank
		}

		_, err = db.Exec(`
			INSERT INTO battlepass_track_definitions
				(reward_track_path, content_hash, xp_per_rank,
				 battlepass_image_path, background_image_path,
				 raw_payload_json, is_current, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)
			ON CONFLICT (reward_track_path, content_hash) DO UPDATE SET
				xp_per_rank             = excluded.xp_per_rank,
				battlepass_image_path   = excluded.battlepass_image_path,
				background_image_path   = excluded.background_image_path,
				raw_payload_json        = excluded.raw_payload_json,
				last_seen_at            = excluded.last_seen_at,
				is_current              = TRUE`,
			rewardTrackPath, contentHash, xpPerRank,
			nullStr(bpImage), nullStr(bgImage),
			string(raw), now, now)
		if err != nil {
			slog.Warn("insert track def", "path", rewardTrackPath, "err", err)
			skipped++
			continue
		}

		// Traductions dans battlepass_track_translations.
		langs := map[string]struct{}{"en-US": {}}
		for lang := range t.Name.Translations {
			langs[lang] = struct{}{}
		}
		for lang := range langs {
			name := t.Name.Translations[lang]
			if name == "" {
				name = t.Name.Value
			}
			_, _ = db.Exec(`
				INSERT INTO battlepass_track_translations
					(reward_track_path, content_hash, lang, track_name, first_seen_at, last_seen_at)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT (reward_track_path, content_hash, lang) DO UPDATE SET
					track_name   = excluded.track_name,
					last_seen_at = excluded.last_seen_at`,
				rewardTrackPath, contentHash, lang, nullStr(name), now, now)
		}

		slog.Info("track importé", "path", rewardTrackPath, "hash", contentHash, "name", t.Name.Value)
		inserted++
	}

	// Vérification finale.
	var totalTracks int
	_ = db.QueryRow(`SELECT COUNT(*) FROM battlepass_track_definitions WHERE is_current = TRUE`).Scan(&totalTracks)

	slog.Info("import terminé",
		"inserted", inserted, "skipped", skipped, "dups_ignored", dups,
		"total_tracks_in_db", totalTracks)
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
