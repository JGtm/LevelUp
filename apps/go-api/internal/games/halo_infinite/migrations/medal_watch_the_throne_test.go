package migrations

// medal_watch_the_throne_test.go — la médaille VIP « Watch the Throne » (1210968206) sert sa
// description VIP (SpartanRecord), et non celle de Roi de la colline héritée de l'ère Python.

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/i18n"
)

const watchTheThroneID = 1210968206

// TestWatchTheThrone_BaseNeuve : sur une base vierge, la ligne est créée ; rejouée, rien n'est
// dupliqué.
func TestWatchTheThrone_BaseNeuve(t *testing.T) {
	db := ouvrirMemoire(t)
	jouerMetadata(t, db)
	assertWatchTheThrone(t, db)

	jouerMetadata(t, db)
	assertWatchTheThrone(t, db)
}

// TestWatchTheThrone_BaseHeritee : table SANS clé primaire (forme de l'ère Python), ligne qui
// porte la description de Roi de la colline, et traduction FR héritée dans medal_translations
// (lue AVANT les colonnes par l'ancienne chaîne FR).
func TestWatchTheThrone_BaseHeritee(t *testing.T) {
	db := ouvrirMemoire(t)
	for _, q := range []string{
		`CREATE TABLE medal_definitions (
			medal_name_id  BIGINT,
			name_fr        VARCHAR,
			name_en        VARCHAR,
			description_fr VARCHAR,
			description_en VARCHAR,
			is_custom      BOOLEAN
		)`,
		`INSERT INTO medal_definitions
			(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom)
		 VALUES (1210968206, 'Garde du trône', 'Watch the Throne',
		         'Contrôlez la colline du début à la fin lors d''un cycle complet.',
		         'Control the hill from start to finish during a complete cycle.', FALSE)`,
		`CREATE TABLE medal_translations (
			medal_name_id BIGINT NOT NULL,
			lang          VARCHAR NOT NULL,
			name          VARCHAR,
			description   VARCHAR,
			PRIMARY KEY (medal_name_id, lang)
		)`,
		`INSERT INTO medal_translations VALUES (1210968206, 'fr-FR', 'Garde du trône',
		         'Contrôlez la colline du début à la fin lors d''un cycle complet.')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("setup %q: %v", q, err)
		}
	}

	jouerMetadata(t, db)
	assertWatchTheThrone(t, db)
}

// assertWatchTheThrone : une seule ligne de définition, et des textes par langue dans
// medal_translations (la traduction héritée est purgée, puis les colonnes passent dans les tables
// par langue) : ni la description de Roi de la colline, ni un doublon.
func assertWatchTheThrone(t *testing.T, db *sql.DB) {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM medal_definitions WHERE medal_name_id = ?`, watchTheThroneID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("lignes pour %d = %d, want 1", watchTheThroneID, n)
	}
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM medal_translations WHERE medal_name_id = ?`, watchTheThroneID,
	).Scan(&n); err != nil {
		t.Fatalf("count medal_translations: %v", err)
	}
	if n != 2 {
		t.Errorf("medal_translations : %d lignes pour %d, want 2 (fr et en)", n, watchTheThroneID)
	}
	names, descriptions := medalTextsByLanguage(t, db, watchTheThroneID)
	want := [4]string{"Watch the Throne", "Garde du trône", "Save your VIP's life", "Sauvez la vie de votre VIP."}
	got := [4]string{
		names[i18n.LocaleEN], names[i18n.LocaleFR],
		descriptions[i18n.LocaleEN], descriptions[i18n.LocaleFR],
	}
	if got != want {
		t.Errorf("médaille %d = %q, want %q", watchTheThroneID, got, want)
	}
}
