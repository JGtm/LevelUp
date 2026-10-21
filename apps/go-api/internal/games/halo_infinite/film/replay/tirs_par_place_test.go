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

// TestLeTireurSuitLaPlaceDeLIntervalle — T-RETOUR de `sieges_a_la_taille_symetrie_test.go` (`400bb628`) :
// R part, revient sur une AUTRE fiche que la premiere, et la sienne a ete reprise par H. La place se
// publie par intervalle (`Presence[k].Seat`) : un tir de la fiche 1 pendant que H la tient est celui de
// H, un tir de la fiche 0 apres le depart de A est celui de R. A chaque frame, le tireur d'une fiche
// est l'occupant de cette fiche.
func TestLeTireurSuitLaPlaceDeLIntervalle(t *testing.T) {
	roster := []RosterEntry{
		entree(0, "100"),  // A, part a 50
		entree(1, "110"),  // R, part a 30, revient a 70 sur la fiche liberee par A
		entree(40, "140"), // H, arrive a 40 : la fiche de R est libre
	}
	retour := []intervalleDePresence{{de: 0, a: 30, aMax: 30}, {de: 70, a: 99, aMax: 99}}
	occ := occupantsFabriques([]int{0, 0, 0}, iv(0, 50), retour, iv(40, 99))
	poserALaTailleDeDeux(roster, occ, 0, 1)
	places := nouveauxTireursParPlace(roster)

	for f := range siegeFrames {
		for _, e := range roster {
			for _, p := range e.Presence {
				if f < p.From || f > p.To {
					continue
				}
				place := e.Seat
				if p.Seat != nil {
					place = *p.Seat
				}
				if idx, _ := places.tireur(place, f); idx != e.FilmIndex {
					t.Errorf("frame %d, fiche %d : tireur %d, attendu l'occupant %s (%d)", f, place, idx, e.Name, e.FilmIndex)
				}
			}
		}
	}
}
