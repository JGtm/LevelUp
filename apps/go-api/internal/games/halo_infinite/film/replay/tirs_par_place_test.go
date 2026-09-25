package replay

// tirs_par_place_test.go — LE TIREUR D UN TIR EST L OCCUPANT DE SA PLACE (lot M4b.4,
// `tirs_par_place.go`).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TestLeTirDuRemplacantLuiRevient — la place 5 est tenue par l index 5 jusqu a la frame 30, puis par
// son remplacant d index 9 : un tir de la place 5 a la frame 40 est le sien, un tir a la frame 10
// reste au partant, et un tir sans tireur n est pas touche.
func TestLeTirDuRemplacantLuiRevient(t *testing.T) {
	fin := 32
	places := nouveauxTireursParPlace([]RosterEntry{
		{XUID: "1", FilmIndex: 5, Seat: 5, Presence: []PresenceInterval{{From: 0, To: 30, ToMax: &fin}}},
		{XUID: "2", FilmIndex: 9, Seat: 5, Presence: []PresenceInterval{{From: 33, To: 90}}},
	})
	fire := []grammar.FireEvent{
		{TimestampUS: 4_000_000, FilmIndex: 5, HasShooter: true},
		{TimestampUS: 1_000_000, FilmIndex: 5, HasShooter: true},
		{TimestampUS: 4_000_000, FilmIndex: -1},
	}
	out, autres := places.tirsParPlace(fire, vsClock())
	if autres != 1 || out[0].FilmIndex != 9 || out[1].FilmIndex != 5 || out[2].FilmIndex != -1 {
		t.Errorf("index = %d, %d, %d (%d rendus), attendu 9, 5, -1 (1)", out[0].FilmIndex, out[1].FilmIndex,
			out[2].FilmIndex, autres)
	}
	if fire[0].FilmIndex != 5 {
		t.Error("les evenements de l appelant ont ete modifies")
	}
}

// TestUnePlaceSansPresenceGardeSonIndex — sans presence publiee (artefact ancien, film sans
// section d identification), la place reste l index : le rattachement d avant.
func TestUnePlaceSansPresenceGardeSonIndex(t *testing.T) {
	places := nouveauxTireursParPlace([]RosterEntry{{XUID: "1", FilmIndex: 5, Seat: 5}})
	if idx, autre := places.tireur(5, 10); idx != 5 || autre {
		t.Errorf("tireur = %d (%v), attendu 5 (faux)", idx, autre)
	}
}
