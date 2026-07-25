package halo_5

import (
	"testing"
)

// carnageWithCommendations construit un carnage 2 joueurs avec des deltas de
// commendations natives (Progressive + Meta) pour les tests d'extraction.
func carnageWithCommendations() *H5CarnageResponse {
	return &H5CarnageResponse{
		IsTeamGame: true,
		PlayerStats: []H5CarnagePlayer{
			{
				Player: H5PlayerRef{Gamertag: "Madina97294"},
				ProgressiveCommendationDeltas: []H5CommendationDelta{
					{Id: "uuid-kills", PreviousProgress: 100, Progress: 117}, // +17
					{Id: "uuid-flat", PreviousProgress: 50, Progress: 50},    // +0 → ignoré
					{Id: "uuid-one", PreviousProgress: 9, Progress: 10},      // +1
				},
				MetaCommendationDeltas: []H5CommendationDelta{
					{Id: "uuid-meta", PreviousProgress: 2, Progress: 5}, // +3
				},
			},
			{
				Player: H5PlayerRef{Gamertag: "JGtm"},
				ProgressiveCommendationDeltas: []H5CommendationDelta{
					{Id: "uuid-kills", PreviousProgress: 7608, Progress: 7611}, // +3
					{Id: "", PreviousProgress: 0, Progress: 4},                 // Id vide → ignoré
				},
			},
		},
	}
}

func commendResolver() func(string) string {
	return func(gt string) string {
		switch gt {
		case "Madina97294":
			return "xA"
		case "JGtm":
			return "xB"
		}
		return "" // inconnu → resolve-or-skip
	}
}

// TestMapCarnageCommendations_DeltaCountAndFilters : count = Progress − Prev,
// deltas ≤ 0 et Id vides ignorés, resolve-or-skip sur xuid non résolu, Meta inclus.
func TestMapCarnageCommendations_DeltaCountAndFilters(t *testing.T) {
	rows := mapCarnageCommendations("m1", carnageWithCommendations(), commendResolver())

	// Madina : 17 (kills) + 1 (one) + 3 (meta) = 3 rows (flat +0 ignoré).
	// JGtm   : 3 (kills) = 1 row (Id vide ignoré).
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4 — %+v", len(rows), rows)
	}

	type key struct{ xuid, cid string }
	got := map[key]int{}
	for _, r := range rows {
		if r.MatchID != "m1" {
			t.Errorf("MatchID = %q, want m1", r.MatchID)
		}
		got[key{r.XUID, r.CommendationID}] = r.Count
	}

	want := map[key]int{
		{"xA", "uuid-kills"}: 17,
		{"xA", "uuid-one"}:   1,
		{"xA", "uuid-meta"}:  3,
		{"xB", "uuid-kills"}: 3,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("count[%v] = %d, want %d", k, got[k], v)
		}
	}
	// flat (+0) ne doit JAMAIS produire de row.
	if _, ok := got[key{"xA", "uuid-flat"}]; ok {
		t.Error("uuid-flat (+0) ne doit pas produire de row")
	}

	// Progress = total à vie ABSOLU au match (carnage Progress), propagé tel quel
	// pour le calcul des totaux courants (dernier progress par commendation).
	progress := map[key]int{}
	for _, r := range rows {
		progress[key{r.XUID, r.CommendationID}] = r.Progress
	}
	wantProgress := map[key]int{
		{"xA", "uuid-kills"}: 117,
		{"xA", "uuid-one"}:   10,
		{"xA", "uuid-meta"}:  5,
		{"xB", "uuid-kills"}: 7611,
	}
	for k, v := range wantProgress {
		if progress[k] != v {
			t.Errorf("progress[%v] = %d, want %d (total à vie absolu)", k, progress[k], v)
		}
	}
}

// TestMapCarnageCommendations_ResolveOrSkip : un joueur dont l'xuid ne résout pas
// est sauté (PK xuid=” collisionnerait, parité mapCarnageParticipants).
func TestMapCarnageCommendations_ResolveOrSkip(t *testing.T) {
	c := &H5CarnageResponse{
		PlayerStats: []H5CarnagePlayer{
			{
				Player: H5PlayerRef{Gamertag: "Unknown"},
				ProgressiveCommendationDeltas: []H5CommendationDelta{
					{Id: "uuid-x", PreviousProgress: 0, Progress: 5},
				},
			},
		},
	}
	rows := mapCarnageCommendations("m1", c, func(string) string { return "" })
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0 (xuid non résolu → skip)", len(rows))
	}
}

func TestMapCarnageCommendations_NilCarnage(t *testing.T) {
	if rows := mapCarnageCommendations("m1", nil, commendResolver()); rows != nil {
		t.Errorf("carnage nil → want nil, got %+v", rows)
	}
	empty := &H5CarnageResponse{}
	if rows := mapCarnageCommendations("m1", empty, commendResolver()); rows != nil {
		t.Errorf("carnage vide → want nil, got %+v", rows)
	}
}
