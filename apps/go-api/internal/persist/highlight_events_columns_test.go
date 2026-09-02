//go:build integration

// Package persist — highlight_events_columns_test.go : LES DEUX COLONNES NE SE
// MELANGENT PAS.
//
// `highlight_events` porte `type_hint INTEGER` (la nature de l event, une quantite
// mesuree du film) et `raw_json VARCHAR` (l identite de la medaille). Avant le
// 2026-09-02 le persister versait `DetailsJSON` dans `type_hint` et n ecrivait
// JAMAIS `raw_json` — d ou 415 matchs de medailles anonymes. Ces tests figent la
// separation ET le canal herite de Halo 5, qui vise bien `type_hint`.
package persist

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/util/pointers"
)

// ligneHighlight est ce qu on relit de la base pour un event.
type ligneHighlight struct {
	typeHint sql.NullInt64
	rawJSON  sql.NullString
}

// lireHighlight relit les events d un match, dans l ordre d insertion.
func lireHighlight(t *testing.T, db *sql.DB, matchID string) []ligneHighlight {
	t.Helper()
	rows, err := db.Query(
		`SELECT type_hint, raw_json FROM highlight_events WHERE match_id = ? ORDER BY id`, matchID)
	if err != nil {
		t.Fatalf("lecture highlight_events: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []ligneHighlight
	for rows.Next() {
		var l ligneHighlight
		if err := rows.Scan(&l.typeHint, &l.rawJSON); err != nil {
			t.Fatalf("scan highlight_events: %v", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration highlight_events: %v", err)
	}
	return out
}

// TestHighlightEvents_ColonnesSeparees — un event medal Halo Infinite pose son
// type_hint numerique ET son document d identite, chacun dans SA colonne.
func TestHighlightEvents_ColonnesSeparees(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	const matchID = "m_hl_colonnes"
	batch := helperBuildSampleBatch(matchID, "1111", "Alice")
	th := 100
	raw := `{"medal_name":"Perfect"}`
	batch.Shared.HighlightEvents = []HighlightEventInsert{
		{MatchID: matchID, XUID: pointers.Ptr("1111"), EventType: "medal", TimeMS: 4200,
			TypeHint: &th, RawJSON: &raw},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	lignes := lireHighlight(t, db, matchID)
	if len(lignes) != 1 {
		t.Fatalf("%d lignes ecrites, 1 attendue", len(lignes))
	}
	if !lignes[0].typeHint.Valid || lignes[0].typeHint.Int64 != 100 {
		t.Errorf("type_hint = %v, attendu 100", lignes[0].typeHint)
	}
	if !lignes[0].rawJSON.Valid || lignes[0].rawJSON.String != raw {
		t.Errorf("raw_json = %v, attendu %q", lignes[0].rawJSON, raw)
	}
}

// TestHighlightEvents_SansIdentiteRestentNull — un event sans identite (kill, ou
// medaille au couple inconnu) laisse `raw_json` a NULL. Pas de chaine vide, pas de
// document sans nom : le lecteur doit voir un trou franc.
func TestHighlightEvents_SansIdentiteRestentNull(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	const matchID = "m_hl_null"
	batch := helperBuildSampleBatch(matchID, "1111", "Alice")
	th := 50
	batch.Shared.HighlightEvents = []HighlightEventInsert{
		// kill : type_hint seul.
		{MatchID: matchID, XUID: pointers.Ptr("1111"), EventType: "kill", TimeMS: 1000, TypeHint: &th},
		// event sans aucun canal : les deux colonnes restent NULL.
		{MatchID: matchID, XUID: pointers.Ptr("1111"), EventType: "mode", TimeMS: 2000},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	lignes := lireHighlight(t, db, matchID)
	if len(lignes) != 2 {
		t.Fatalf("%d lignes ecrites, 2 attendues", len(lignes))
	}
	if !lignes[0].typeHint.Valid || lignes[0].typeHint.Int64 != 50 {
		t.Errorf("kill: type_hint = %v, attendu 50", lignes[0].typeHint)
	}
	if lignes[0].rawJSON.Valid {
		t.Errorf("kill: raw_json = %q, attendu NULL", lignes[0].rawJSON.String)
	}
	if lignes[1].typeHint.Valid {
		t.Errorf("mode: type_hint = %v, attendu NULL", lignes[1].typeHint)
	}
	if lignes[1].rawJSON.Valid {
		t.Errorf("mode: raw_json = %q, attendu NULL", lignes[1].rawJSON.String)
	}
}

// TestHighlightEvents_CanalHeriteHalo5 — NON-REGRESSION. Halo 5 remplit
// `DetailsJSON` avec un identifiant de medaille en chaine, et ce champ vise
// `type_hint`, pas `raw_json` (games/halo_5/ingest/medals.go). L ajout du canal
// canonique ne doit pas avoir devie ce flux.
func TestHighlightEvents_CanalHeriteHalo5(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	const matchID = "m_hl_h5"
	batch := helperBuildSampleBatch(matchID, "1111", "Alice")
	detail := "1633"
	batch.Shared.HighlightEvents = []HighlightEventInsert{
		{MatchID: matchID, XUID: pointers.Ptr("1111"), EventType: "medal", TimeMS: 7000,
			DetailsJSON: &detail},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	lignes := lireHighlight(t, db, matchID)
	if len(lignes) != 1 {
		t.Fatalf("%d lignes ecrites, 1 attendue", len(lignes))
	}
	if !lignes[0].typeHint.Valid || lignes[0].typeHint.Int64 != 1633 {
		t.Errorf("type_hint = %v, attendu 1633 (canal herite Halo 5)", lignes[0].typeHint)
	}
	if lignes[0].rawJSON.Valid {
		t.Errorf("raw_json = %q, attendu NULL — DetailsJSON ne doit PAS y atterrir",
			lignes[0].rawJSON.String)
	}
}

// TestHighlightEvents_ArbitrageTypeHint — si les deux canaux sont renseignes
// (ce qu aucun producteur ne fait), le canal canonique gagne. Le test fige
// l arbitrage documente sur HighlightEventInsert.
func TestHighlightEvents_ArbitrageTypeHint(t *testing.T) {
	db := openSharedTestDB(t)
	p := NewSharedPersister(db)

	const matchID = "m_hl_arbitrage"
	batch := helperBuildSampleBatch(matchID, "1111", "Alice")
	th := 205
	detail := "999"
	batch.Shared.HighlightEvents = []HighlightEventInsert{
		{MatchID: matchID, XUID: pointers.Ptr("1111"), EventType: "medal", TimeMS: 100,
			TypeHint: &th, DetailsJSON: &detail},
	}
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	lignes := lireHighlight(t, db, matchID)
	if len(lignes) != 1 {
		t.Fatalf("%d lignes ecrites, 1 attendue", len(lignes))
	}
	if !lignes[0].typeHint.Valid || lignes[0].typeHint.Int64 != 205 {
		t.Errorf("type_hint = %v, attendu 205 (TypeHint l emporte sur DetailsJSON)", lignes[0].typeHint)
	}
}
