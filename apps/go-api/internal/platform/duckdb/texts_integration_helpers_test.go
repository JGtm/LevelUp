//go:build integration

package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/i18n"
)

// insertMedalTexts écrit les textes d'une médaille dans medal_translations, une ligne par langue
// qui porte un texte (étiquette Halo principale de la langue) : la forme que les lecteurs de
// médailles attendent depuis que medal_definitions ne porte plus de colonne de texte.
func insertMedalTexts(t *testing.T, db *DB, id int64, names, descriptions i18n.Text) {
	t.Helper()
	for _, l := range i18n.Locales() {
		name, desc := names[l], descriptions[l]
		if name == "" && desc == "" {
			continue
		}
		if _, err := db.Exec(context.Background(),
			`INSERT INTO medal_translations (medal_name_id, lang, name, description) VALUES (?, ?, ?, ?)`,
			id, i18n.HaloTags(l)[0], name, desc); err != nil {
			t.Fatalf("insertMedalTexts(%d, %s): %v", id, l, err)
		}
	}
}

// putCuratedWeaponNames écrit les noms curés d'une arme (weapon_key) dans weapon_name_labels, une
// ligne par langue renseignée.
func putCuratedWeaponNames(t *testing.T, db *DB, slug, key string, names i18n.Text) {
	t.Helper()
	for lang, name := range names {
		if name == "" {
			continue
		}
		if _, err := db.Exec(context.Background(),
			`INSERT INTO weapon_name_labels (title_slug, weapon_key, lang, name) VALUES (?, ?, ?, ?)`,
			slug, key, string(lang), name); err != nil {
			t.Fatalf("putCuratedWeaponNames(%s/%s, %s): %v", slug, key, lang, err)
		}
	}
}
