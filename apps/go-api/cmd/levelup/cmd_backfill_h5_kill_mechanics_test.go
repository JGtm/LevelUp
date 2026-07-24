package main

// cmd_backfill_h5_kill_mechanics_test.go — logique PURE du backfill : projection carnage
// -> verite par gamertag, plan de correction (update/unchanged/unmatched) et idempotence.
// Aucune DB ni reseau : les helpers testes sont stateless (modele cmd/h5-roster-refetch).

import (
	"testing"
	"time"

	halo5 "levelup/go-api/internal/games/halo_5"
)

func TestCarnageMechByGamertag(t *testing.T) {
	carnage := &halo5.H5CarnageResponse{
		PlayerStats: []halo5.H5CarnagePlayer{
			{Player: halo5.H5PlayerRef{Gamertag: "JGtm"}, TotalAssassinations: 3, TotalGroundPoundKills: 1, TotalShoulderBashKills: 2},
			{Player: halo5.H5PlayerRef{Gamertag: "Ally"}, TotalAssassinations: 0, TotalGroundPoundKills: 0, TotalShoulderBashKills: 0},
			{Player: halo5.H5PlayerRef{Gamertag: ""}, TotalAssassinations: 9}, // gamertag vide -> ignore
		},
	}
	got := carnageMechByGamertag(carnage)
	if len(got) != 2 {
		t.Fatalf("map len = %d, want 2 (gamertag vide ignore)", len(got))
	}
	if got["JGtm"] != (mechTriple{assassination: 3, groundPound: 1, shoulderBash: 2}) {
		t.Errorf("JGtm = %+v, want {3 1 2}", got["JGtm"])
	}
	if got["Ally"] != (mechTriple{}) {
		t.Errorf("Ally = %+v, want zero triple", got["Ally"])
	}
	if _, ok := got[""]; ok {
		t.Errorf("gamertag vide ne doit pas etre une cle")
	}
}

func TestCarnageMechByGamertag_Nil(t *testing.T) {
	if got := carnageMechByGamertag(nil); got == nil || len(got) != 0 {
		t.Errorf("carnage nil -> map vide non-nil, got %v", got)
	}
}

func TestPlanMechanicUpdates_CorrigeSeulementLesDivergences(t *testing.T) {
	stored := []storedMech{
		// JGtm : corrompu 0/0/0 -> a corriger ; Ally : vrai zero -> inchange ;
		// Ghost : absent du carnage -> unmatched.
		{xuid: "xJ", gamertag: "JGtm", values: mechTriple{0, 0, 0}},
		{xuid: "xA", gamertag: "Ally", values: mechTriple{0, 0, 0}},
		{xuid: "xG", gamertag: "Ghost", values: mechTriple{0, 0, 0}},
	}
	truth := map[string]mechTriple{
		"JGtm": {assassination: 3, groundPound: 1, shoulderBash: 2},
		"Ally": {}, // vrai 0/0/0
	}
	updates, unchanged, unmatched := planMechanicUpdates(stored, truth)
	if len(updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(updates))
	}
	if updates[0].xuid != "xJ" || updates[0].values != (mechTriple{3, 1, 2}) {
		t.Errorf("update = %+v, want {xJ {3 1 2}}", updates[0])
	}
	if unchanged != 1 {
		t.Errorf("unchanged = %d, want 1 (Ally deja a 0/0/0)", unchanged)
	}
	if unmatched != 1 {
		t.Errorf("unmatched = %d, want 1 (Ghost absent du carnage)", unmatched)
	}
}

// TestPlanMechanicUpdates_Idempotence : une 2e passe (lignes deja a la verite carnage) ne
// planifie AUCUNE correction -> re-lancable sans effet de bord.
func TestPlanMechanicUpdates_Idempotence(t *testing.T) {
	truth := map[string]mechTriple{"JGtm": {assassination: 3, groundPound: 1, shoulderBash: 2}}
	corrected := []storedMech{{xuid: "xJ", gamertag: "JGtm", values: mechTriple{3, 1, 2}}}
	updates, unchanged, unmatched := planMechanicUpdates(corrected, truth)
	if len(updates) != 0 {
		t.Errorf("2e passe -> updates = %d, want 0 (idempotent)", len(updates))
	}
	if unchanged != 1 || unmatched != 0 {
		t.Errorf("unchanged=%d unmatched=%d, want 1/0", unchanged, unmatched)
	}
}

func TestParseH5MechCutoff(t *testing.T) {
	// Vide -> time zero (pas de borne).
	if got, err := parseH5MechCutoff(""); err != nil || !got.IsZero() {
		t.Errorf("vide -> (%v, %v), want (zero, nil)", got, err)
	}
	if got, err := parseH5MechCutoff("   "); err != nil || !got.IsZero() {
		t.Errorf("blancs -> (%v, %v), want (zero, nil)", got, err)
	}
	// RFC3339 valide.
	got, err := parseH5MechCutoff("2026-06-26T15:00:00Z")
	if err != nil {
		t.Fatalf("cutoff valide: err = %v", err)
	}
	if want := time.Date(2026, 6, 26, 15, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("cutoff = %v, want %v", got, want)
	}
	// Invalide -> erreur.
	if _, err := parseH5MechCutoff("2026-06-26 15:00"); err == nil {
		t.Errorf("format non-RFC3339 doit renvoyer une erreur")
	}
	// Le defaut du fichier doit parser.
	if _, err := parseH5MechCutoff(defaultH5MechCutoff); err != nil {
		t.Errorf("defaultH5MechCutoff %q ne parse pas: %v", defaultH5MechCutoff, err)
	}
}
