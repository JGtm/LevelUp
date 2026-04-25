// Script one-shot : importe les items Battle Pass depuis
// data/investigation/battlepass/*/items/*.json dans metadata.duckdb.
// Usage : go run ./scripts/import_bp_items/  (depuis apps/go-api)
package main

import (
	"crypto/sha1"
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

type itemCommonData struct {
	ID          string          `json:"Id"`
	Quality     string          `json:"Quality"`
	Type        string          `json:"Type"`     // champ GameCMS réel (prioritaire)
	ItemType    string          `json:"ItemType"` // ancien format, fallback
	DisplayPath itemDisplayPath `json:"DisplayPath"`
	Title       localizedValue  `json:"Title"`
	Description localizedValue  `json:"Description"`
}

type itemDisplayPath struct {
	Media itemMedia `json:"Media"`
}
type itemMedia struct {
	MediaURL itemMediaURL `json:"MediaUrl"`
}
type itemMediaURL struct {
	Path string `json:"Path"`
}

type localizedValue struct {
	Value        string            `json:"value"`
	Translations map[string]string `json:"translations"`
}

type itemJSON struct {
	CommonData itemCommonData `json:"CommonData"`
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

	glob := filepath.Join("..", "..", "data", "investigation", "battlepass", "*", "items", "*.json")
	files, err := filepath.Glob(glob)
	if err != nil || len(files) == 0 {
		slog.Error("aucun fichier trouvé", "glob", glob)
		os.Exit(1)
	}
	slog.Info("fichiers trouvés", "count", len(files))

	now := time.Now()
	inserted, skipped := 0, 0

	for _, fpath := range files {
		raw, err := os.ReadFile(fpath)
		if err != nil {
			slog.Warn("lecture", "file", fpath, "err", err)
			skipped++
			continue
		}

		var item itemJSON
		if err := json.Unmarshal(raw, &item); err != nil {
			slog.Warn("parse", "file", fpath, "err", err)
			skipped++
			continue
		}

		cd := item.CommonData
		inventoryItemPath := strings.TrimSpace(cd.ID)
		if inventoryItemPath == "" {
			skipped++
			continue
		}

		h := sha1.Sum(raw)
		contentHash := fmt.Sprintf("%x", h[:8])

		displayPath := cd.DisplayPath.Media.MediaURL.Path
		quality := cd.Quality
		itemType := cd.Type
		if itemType == "" {
			itemType = cd.ItemType
		}

		_, err = db.Exec(`
			INSERT INTO battlepass_item_definitions
				(inventory_item_path, content_hash, quality, item_type, display_path,
				 raw_payload_json, first_seen_at, last_seen_at, is_current)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE)
			ON CONFLICT (inventory_item_path, content_hash) DO UPDATE SET
				quality        = excluded.quality,
				item_type      = excluded.item_type,
				display_path   = excluded.display_path,
				last_seen_at   = excluded.last_seen_at,
				is_current     = TRUE`,
			inventoryItemPath, contentHash, nullStr(quality), nullStr(itemType),
			nullStr(displayPath), string(raw), now, now)
		if err != nil {
			slog.Warn("insert item", "path", inventoryItemPath, "err", err)
			skipped++
			continue
		}

		// Traductions
		langs := map[string]struct{}{"en-US": {}}
		for l := range cd.Title.Translations {
			langs[l] = struct{}{}
		}
		for l := range cd.Description.Translations {
			langs[l] = struct{}{}
		}
		for lang := range langs {
			title := cd.Title.Translations[lang]
			if title == "" {
				title = cd.Title.Value
			}
			desc := cd.Description.Translations[lang]
			if desc == "" {
				desc = cd.Description.Value
			}
			_, _ = db.Exec(`
				INSERT INTO battlepass_item_translations
					(inventory_item_path, content_hash, lang, title, description,
					 first_seen_at, last_seen_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (inventory_item_path, content_hash, lang) DO UPDATE SET
					title        = excluded.title,
					description  = excluded.description,
					last_seen_at = excluded.last_seen_at`,
				inventoryItemPath, contentHash, lang,
				nullStr(title), nullStr(desc), now, now)
		}

		inserted++
	}

	slog.Info("import terminé", "inserted", inserted, "skipped", skipped)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
