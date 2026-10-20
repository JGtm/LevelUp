package replay

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// identity_registry_occupations_test.go — LES LIENS D UN SIEGE STATBORG RECYCLE SONT BORNES (lot R1).
//
// ROUGE AVANT LE LOT : un slot recycle publiait UNE ligne « de 0 a la derniere frame » au nom d'un
// seul occupant (`bcb6d393`, slot 12 : le remplacant, sur tout le match).

// horlogeDesOccupations : frame 0 a 10 s d'horloge film, pas de 100 ms, 1 000 frames.
var horlogeDesOccupations = IdentityClock{OriginUS: 10_000_000, StepUS: 100_000, FrameCount: 1000}

func TestLignesDesOccupationsBornees(t *testing.T) {
	occ := []objectives.Occupation{
		{OpenFrom: true, ToMS: 40000, XUID: "A", Origin: objectives.OriginDeathInstants},
		{FromMS: 40000, ToMS: 55000, XUID: ""},
		{FromMS: 55000, OpenTo: true, XUID: "B", Origin: objectives.OriginSheetTriplet},
	}
	calage := int64(2000) // horloge film = horloge du fil + 2 s
	got := lignesDesOccupations(12, 0, occ, horlogeDesOccupations, &calage)
	want := []IdentityStatborgSlot{
		{Slot: 12, XUID: "A", Link: canonical.Link{Source: canonical.LinkInferred,
			Method: canonical.MethodDeathInstants, From: 0, To: 319}},
		{Slot: 12, Link: canonical.Link{Source: canonical.LinkUnresolved, Method: canonical.MethodNone,
			From: 320, To: 469}},
		{Slot: 12, XUID: "B", Link: canonical.Link{Source: canonical.LinkInferred,
			Method: canonical.MethodSheetTriplet, From: 470, To: 999}},
	}
	if len(got) != len(want) {
		t.Fatalf("%d ligne(s), attendu %d : %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].XUID != want[i].XUID || got[i].Link.Source != want[i].Link.Source ||
			got[i].Link.Method != want[i].Link.Method || got[i].Link.From != want[i].Link.From ||
			got[i].Link.To != want[i].Link.To {
			t.Errorf("ligne %d : %+v, attendu %+v", i, got[i], want[i])
		}
	}
}

// TestLignesDesOccupationsSansCalage : sans calage du fil des morts, aucune borne ne s'exprime —
// le slot se publie en une ligne non resolue, jamais nomme sur tout le match.
func TestLignesDesOccupationsSansCalage(t *testing.T) {
	occ := []objectives.Occupation{{OpenFrom: true, ToMS: 40000, XUID: "A"}, {FromMS: 40000, OpenTo: true, XUID: "B"}}
	got := lignesDesOccupations(12, 0, occ, horlogeDesOccupations, nil)
	if len(got) != 1 || got[0].XUID != "" || got[0].Link.Source != canonical.LinkUnresolved {
		t.Fatalf("attendu une ligne non resolue, obtenu %+v", got)
	}
}
