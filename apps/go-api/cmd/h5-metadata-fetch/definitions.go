package main

// definitions.go — médailles de Halo 5 : colonnes non textuelles dans medal_definitions, noms et
// descriptions dans medal_translations, une ligne par langue de config/locales.toml pour laquelle
// le seed a un texte (ADR 0040 D-5). Commendations et couleurs d'équipe : definitions_assets.go.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/ops"
)

// acceptLanguage donne l'en-tête Accept-Language qui fait servir la langue `l` par l'API Metadata
// (première étiquette Halo de la langue). La langue de repli est celle que l'API sert sans
// en-tête : elle n'en reçoit pas.
func acceptLanguage(l i18n.Locale) string {
	if l == i18n.Fallback {
		return ""
	}
	tags := i18n.HaloTags(l)
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

// setText inscrit `v` (sans espaces de bord) dans `t` pour la langue `l` ; une valeur vide n'inscrit
// rien : une langue sans texte reste absente, jamais inventée (D-4).
func setText(t i18n.Text, l i18n.Locale, v string) {
	if v = strings.TrimSpace(v); v != "" {
		t[l] = v
	}
}

type apiMedal struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Classification string `json:"classification"`
	Difficulty     int    `json:"difficulty"`
	ID             string `json:"id"`
	SpriteLocation struct {
		SpriteSheetURI string `json:"spriteSheetUri"`
		Left           int    `json:"left"`
		Top            int    `json:"top"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
	} `json:"spriteLocation"`
}

// medalTexts rassemble le nom et la description d'une médaille par langue : la langue de l'API
// vient de l'API, les autres des tables de asset_labels.toml (clé = nom de l'API). Une langue
// sans entrée n'écrit rien (la lecture se replie sur la langue de l'API).
func medalTexts(m apiMedal, labels assetLabels) (names, descs i18n.Text) {
	names, descs = i18n.Text{}, i18n.Text{}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			setText(names, l, m.Name)
			setText(descs, l, m.Description)
			continue
		}
		setText(names, l, labels.Medals[m.Name][l])
		setText(descs, l, labels.MedalDescriptions[m.Name][l])
	}
	return names, descs
}

// seedMedals peuple medal_definitions (sprite, catégorie, difficulté) et medal_translations.
func seedMedals(ctx context.Context, db *sql.DB, key string, labels assetLabels) {
	body, err := fetchMeta(key, "medals")
	if err != nil {
		fmt.Printf("medals: SKIP (%v)\n", err)
		return
	}
	var medals []apiMedal
	if err := json.Unmarshal(body, &medals); err != nil {
		fmt.Printf("medals: parse %v\n", err)
		return
	}
	n := 0
	for _, m := range medals {
		id, perr := strconv.ParseInt(m.ID, 10, 64)
		if perr != nil {
			continue // id non numérique → ignoré (medal_name_id = BIGINT)
		}
		if err := persistMedal(ctx, db, id, m, labels); err != nil {
			fmt.Printf("medals: %s: %v\n", m.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("medals: %d seedées (sur %d)\n", n, len(medals))
}

// persistMedal écrit une médaille : ses colonnes non textuelles puis ses textes par langue.
func persistMedal(ctx context.Context, db *sql.DB, id int64, m apiMedal, labels assetLabels) error {
	if err := insertMedalDefinition(db, id, m); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	names, descs := medalTexts(m, labels)
	if err := ops.UpsertMedalTexts(ctx, db, id, names, descs); err != nil {
		return fmt.Errorf("textes: %w", err)
	}
	return nil
}

// insertMedalDefinition écrit les colonnes non textuelles d'une médaille. h5 : `difficulty`
// (0..245) n'est PAS l'enum 0-3 d'Infinite → stocké brut en VARCHAR (difficulty),
// difficulty_index laissé à 0 (non applicable). medal_type = classification normalisée vers les
// clés canoniques inter-titres (multikill/spree/skill/style/mode/proficiency/other) via
// canonical.NormalizeMedalCategory — sinon l'enum brut H5 (MultiKill, Style, CaptureTheFlag…)
// court-circuite la traduction frontend (categoryLabels). Icône = sprite (feuille + offset).
func insertMedalDefinition(db *sql.DB, id int64, m apiMedal) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO medal_definitions
		(medal_name_id, difficulty_index, difficulty, medal_type,
		 sprite_sheet_url, sprite_left, sprite_top, sprite_width, sprite_height)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id, 0, strconv.Itoa(m.Difficulty), canonical.NormalizeMedalCategory(m.Classification),
		m.SpriteLocation.SpriteSheetURI, m.SpriteLocation.Left, m.SpriteLocation.Top,
		m.SpriteLocation.Width, m.SpriteLocation.Height)
	return err
}
