// Package duckdb — persist_sink_items.go : persistance des item definitions GameCMS
// (battlepass_item_definitions + translations). Extrait de persist_sink.go (K3f god-file
// split, 2026-07-06), meme package. INSERT-only ART-safe inchange (ADR 0019).
package duckdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/platform/dblease"
)

// ---------------------------------------------------------------------------
// Item Definitions (battlepass_item_definitions + battlepass_item_translations)
// ---------------------------------------------------------------------------

// itemDefCommonData est le sous-arbre CommonData d'un item GameCMS.
// GameCMS utilise "Type" (pas "ItemType") pour la rareté fonctionnelle — les deux
// champs sont lus pour couvrir toutes les versions de l'API.
type itemDefCommonData struct {
	Title       any    `json:"Title"`
	Description any    `json:"Description"`
	Quality     string `json:"Quality"`
	Type        string `json:"Type"`
	ItemType    string `json:"ItemType"`
	DisplayPath struct {
		Media struct {
			MediaURL struct {
				Path string `json:"Path"`
			} `json:"MediaUrl"`
		} `json:"Media"`
	} `json:"DisplayPath"`
}

type itemDefRaw struct {
	CommonData itemDefCommonData `json:"CommonData"`
}

// itemDefLocalizedText extrait le texte localisé depuis un champ Halo polymorphe.
// GameCMS retourne soit une string, soit {"value":"…","status":"Resolved"}, soit
// {"translations":{"fr-FR":"…","en-US":"…"}}.
func itemDefLocalizedText(v any, preferLang string) string {
	if v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if trans, ok := typed["translations"].(map[string]any); ok {
			for _, lang := range []string{preferLang, LangCodeFR, LangCodeEN, "en"} {
				if s, ok := trans[lang].(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		if s, ok := typed["value"].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// UpsertItemDefinition implémente halo.ItemDefinitionPersister.
// Persiste la définition d'un item BP dans battlepass_item_definitions et
// battlepass_item_translations (fr-FR + en-US) de metadata.duckdb.
// Utilise UPDATE-first + INSERT-if-zero (idiome DuckDB) pour garantir l'idempotence.
func (s *PersistSink) UpsertItemDefinition(ctx context.Context, itemPath string, raw []byte) error {
	var def itemDefRaw
	_ = json.Unmarshal(raw, &def) // best-effort

	cd := def.CommonData
	itemType := strings.TrimSpace(cd.Type)
	if itemType == "" {
		itemType = strings.TrimSpace(cd.ItemType)
	}
	displayPath := strings.TrimSpace(cd.DisplayPath.Media.MediaURL.Path)

	var qualityArg, itemTypeArg, displayPathArg any
	if q := strings.TrimSpace(cd.Quality); q != "" {
		qualityArg = q
	}
	if itemType != "" {
		itemTypeArg = itemType
	}
	if displayPath != "" {
		displayPathArg = displayPath
	}

	relMeta, err := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
	if err != nil {
		return fmt.Errorf("UpsertItemDefinition lease meta: %w", err)
	}
	defer relMeta()

	db, err := OpenReadWrite(s.MetaPath)
	if err != nil {
		return fmt.Errorf("open meta rw: %w", err)
	}
	defer db.Close()

	hash := persistHash(raw)
	now := time.Now()

	// Invalider les anciennes entrées de cet item (hash différent).
	_, _ = db.Exec(ctx, `
		UPDATE battlepass_item_definitions
		SET is_current = FALSE, last_seen_at = ?
		WHERE inventory_item_path = ? AND content_hash <> ? AND is_current = TRUE`,
		now, itemPath, hash)

	res, err := db.Exec(ctx, `
		UPDATE battlepass_item_definitions
		SET quality = ?, item_type = ?, display_path = ?,
		    raw_payload_json = ?, last_seen_at = ?, is_current = TRUE
		WHERE inventory_item_path = ? AND content_hash = ?`,
		qualityArg, itemTypeArg, displayPathArg, string(raw), now, itemPath, hash)
	if err != nil {
		return fmt.Errorf("update battlepass_item_definitions: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.DebugContext(ctx, "persist_sink: bp item definition updated", "path", itemPath, "hash", hash)
		return s.upsertItemTranslations(ctx, db, itemPath, hash, cd, now)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type,
			 display_path, raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?)`,
		itemPath, hash, qualityArg, itemTypeArg, displayPathArg, string(raw), now, now)
	if err != nil {
		return fmt.Errorf("insert battlepass_item_definitions: %w", err)
	}
	slog.DebugContext(ctx, "persist_sink: bp item definition inserted", "path", itemPath, "hash", hash)
	return s.upsertItemTranslations(ctx, db, itemPath, hash, cd, now)
}

// upsertItemTranslations persiste les traductions fr-FR et en-US d'un item.
func (s *PersistSink) upsertItemTranslations(
	ctx context.Context,
	db *DB,
	itemPath, hash string,
	cd itemDefCommonData,
	now time.Time,
) error {
	type langEntry struct {
		lang        string
		title       string
		description string
	}

	entries := []langEntry{
		{
			lang:        LangCodeFR,
			title:       itemDefLocalizedText(cd.Title, LangCodeFR),
			description: itemDefLocalizedText(cd.Description, LangCodeFR),
		},
		{
			lang:        LangCodeEN,
			title:       itemDefLocalizedText(cd.Title, LangCodeEN),
			description: itemDefLocalizedText(cd.Description, LangCodeEN),
		},
	}

	for _, e := range entries {
		if e.title == "" && e.description == "" {
			continue
		}
		var titleArg, descArg any
		if e.title != "" {
			titleArg = e.title
		}
		if e.description != "" {
			descArg = e.description
		}
		res, err := db.Exec(ctx, `
			UPDATE battlepass_item_translations
			SET title = ?, description = ?, last_seen_at = ?
			WHERE inventory_item_path = ? AND content_hash = ? AND lang = ?`,
			titleArg, descArg, now, itemPath, hash, e.lang)
		if err != nil {
			return fmt.Errorf("update battlepass_item_translations %s: %w", e.lang, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			continue
		}
		_, err = db.Exec(ctx, `
			INSERT INTO battlepass_item_translations
				(inventory_item_path, content_hash, lang, title, description,
				 first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemPath, hash, e.lang, titleArg, descArg, now, now)
		if err != nil {
			return fmt.Errorf("insert battlepass_item_translations %s: %w", e.lang, err)
		}
	}
	return nil
}
