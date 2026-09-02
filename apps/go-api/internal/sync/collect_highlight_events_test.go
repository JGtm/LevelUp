// Package sync — collect_highlight_events_test.go : LE COLLECTEUR NE JETTE PLUS
// L IDENTITE DES MEDAILLES.
//
// Ce que ces tests figent : `type_hint` part pour TOUS les events, `raw_json` ne
// part que pour les `medal` dont le couple (type_hint, medal_type) est mesure, et
// un couple inconnu ne produit AUCUN nom — il incremente un compteur.
package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestHighlightEventInserts_MedailleConnue — le couple mesure (50,26) est Killjoy :
// la ligne part avec son type_hint ET son document d identite.
func TestHighlightEventInserts_MedailleConnue(t *testing.T) {
	events := []analysis.HighlightEvent{
		{XUID: 2533274792574872, Gamertag: "JGtm", EventType: analysis.EventTypeMedal,
			TypeHint: 50, IsMedal: true, TimeMS: 12345, MedalType: 26},
	}
	inserts, sansNom := highlightEventInserts(context.Background(), "m1", events)
	if sansNom != 0 {
		t.Fatalf("medailles sans nom = %d, attendu 0", sansNom)
	}
	if len(inserts) != 1 {
		t.Fatalf("%d lignes, 1 attendue", len(inserts))
	}
	got := inserts[0]
	if got.TypeHint == nil || *got.TypeHint != 50 {
		t.Errorf("TypeHint = %v, attendu 50", got.TypeHint)
	}
	if got.RawJSON == nil {
		t.Fatal("RawJSON absent — la medaille (50,26) est dans la table mesuree")
	}
	if want := `{"medal_name":"Killjoy"}`; *got.RawJSON != want {
		t.Errorf("RawJSON = %q, attendu %q", *got.RawJSON, want)
	}
	if got.MatchID != "m1" || got.TimeMS != 12345 || got.XUID == nil || *got.XUID != "2533274792574872" {
		t.Errorf("champs de base alteres: %+v", got)
	}
	if got.DetailsJSON != nil {
		t.Errorf("DetailsJSON = %q, attendu nil — canal reserve a Halo 5", *got.DetailsJSON)
	}
}

// TestHighlightEventInserts_KillSansRawJSON — un kill porte son type_hint et RIEN
// dans raw_json : l identite n existe que pour les medailles.
func TestHighlightEventInserts_KillSansRawJSON(t *testing.T) {
	events := []analysis.HighlightEvent{
		{XUID: 2533274792574872, EventType: analysis.EventTypeKill, TypeHint: 50, TimeMS: 900},
		{XUID: 2533274792574872, EventType: analysis.EventTypeDeath, TypeHint: 20, TimeMS: 950},
		{XUID: 2533274792574872, EventType: analysis.EventTypeMode, TypeHint: 10, TimeMS: 980},
	}
	inserts, sansNom := highlightEventInserts(context.Background(), "m1", events)
	if sansNom != 0 {
		t.Fatalf("medailles sans nom = %d, attendu 0 (aucun event medal)", sansNom)
	}
	if len(inserts) != 3 {
		t.Fatalf("%d lignes, 3 attendues", len(inserts))
	}
	attendus := []int{50, 20, 10}
	for i, ins := range inserts {
		if ins.TypeHint == nil || *ins.TypeHint != attendus[i] {
			t.Errorf("ligne %d: TypeHint = %v, attendu %d", i, ins.TypeHint, attendus[i])
		}
		if ins.RawJSON != nil {
			t.Errorf("ligne %d: RawJSON = %q, attendu nil", i, *ins.RawJSON)
		}
	}
}

// TestHighlightEventInserts_CoupleInconnuCompte — DEGRADATION MESUREE : un couple
// hors table ne recoit pas un nom voisin, il ne recoit RIEN, et le compteur le dit.
func TestHighlightEventInserts_CoupleInconnuCompte(t *testing.T) {
	events := []analysis.HighlightEvent{
		{XUID: 2533274792574872, EventType: analysis.EventTypeMedal,
			TypeHint: 50, IsMedal: true, TimeMS: 100, MedalType: 255},
		{XUID: 2533274792574872, EventType: analysis.EventTypeMedal,
			TypeHint: 100, IsMedal: true, TimeMS: 200, MedalType: 254},
		{XUID: 2533274792574872, EventType: analysis.EventTypeMedal,
			TypeHint: 150, IsMedal: true, TimeMS: 300, MedalType: 1}, // Triple Kill, connu
	}
	inserts, sansNom := highlightEventInserts(context.Background(), "m1", events)
	if sansNom != 2 {
		t.Fatalf("medailles sans nom = %d, attendu 2", sansNom)
	}
	if len(inserts) != 3 {
		t.Fatalf("%d lignes, 3 attendues — un couple inconnu reste une ligne", len(inserts))
	}
	for i := 0; i < 2; i++ {
		if inserts[i].RawJSON != nil {
			t.Errorf("ligne %d: RawJSON = %q, attendu nil (couple inconnu)", i, *inserts[i].RawJSON)
		}
		if inserts[i].TypeHint == nil {
			t.Errorf("ligne %d: TypeHint absent — il reste une quantite mesuree du film", i)
		}
	}
	if inserts[2].RawJSON == nil || *inserts[2].RawJSON != `{"medal_name":"Triple Kill"}` {
		t.Errorf("ligne 2: RawJSON = %v, attendu Triple Kill", inserts[2].RawJSON)
	}
}
