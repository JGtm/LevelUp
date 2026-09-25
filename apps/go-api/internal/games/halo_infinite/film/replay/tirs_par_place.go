package replay

// tirs_par_place.go — LE TIREUR D UN TIR EST L OCCUPANT DE SA PLACE (lot M4b.4 de la campagne
// « retours rejeu », 2026-09-24 ; regle des places de l utilisateur du 2026-09-23).
//
// # LE FAIT
//
// L index de tireur d un record de tir (`grammar.FireEvent.FilmIndex`, cinq bits depuis M4b) et
// l index de controle d une rafale (`types.ContinuousFireBurst.PlayerIndex`) ne sont pas l index
// de participant : c est la PLACE, et le remplacant en herite (sonde P4, `sieges_tirs.go` : 3
// remplacements sur 3). Le rattachement d avant cherchait les slots du JOUEUR de cet index — ceux
// du partant — et perdait les tirs du remplacant (sans slot) ou les rendait a un absent.
//
// # LA REGLE
//
// Un tir de la place P a la frame F appartient a l entree du roster ASSISE a P (`RosterEntry.Seat`)
// dont la presence couvre F — jusqu a `toMax`, la borne d affichage : deux occupants successifs
// d une place ne se recouvrent pas (`sieges_places.go`). Sans occupant unique a cet instant, la
// place reste l index : c est exactement le rattachement d avant, juste pour tout joueur sans
// relais. Le compte des tirs rendus a un AUTRE index que la place est publie (`coverage.seats`
// pour les tirs, `coverage.continuousFire.byPlace` pour les rafales).
//
// PUR : aucune I/O.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// occupantDePlace est une entree assise a une place : son index de joueur et sa presence.
type occupantDePlace struct {
	index    int
	presence []PresenceInterval
}

// tireursParPlace indexe les occupants du roster par place.
type tireursParPlace map[int][]occupantDePlace

// nouveauxTireursParPlace indexe le roster POSE (apres `poserLesSieges`).
func nouveauxTireursParPlace(roster []RosterEntry) tireursParPlace {
	out := tireursParPlace{}
	for _, e := range roster {
		if len(e.Presence) == 0 {
			continue // aucune presence : rien ne dit quand cette entree tient sa place
		}
		out[e.Seat] = append(out[e.Seat], occupantDePlace{index: e.FilmIndex, presence: e.Presence})
	}
	return out
}

// tireur rend l index de joueur de l occupant de la place a la frame, et vrai quand il differe de
// la place. Sans occupant UNIQUE qui couvre la frame, la place elle-meme.
func (t tireursParPlace) tireur(place, fr int) (int, bool) {
	trouve, n := place, 0
	for _, o := range t[place] {
		if presenceCouvre(o.presence, fr) {
			trouve = o.index
			n++
		}
	}
	if n != 1 {
		return place, false
	}
	return trouve, trouve != place
}

// presenceCouvre dit si une presence couvre la frame, jusqu a sa borne d affichage.
func presenceCouvre(ivs []PresenceInterval, fr int) bool {
	for _, iv := range ivs {
		fin := iv.To
		if iv.ToMax != nil {
			fin = *iv.ToMax
		}
		if fr >= iv.From && fr <= fin {
			return true
		}
	}
	return false
}

// tirsParPlace rend une COPIE des evenements de tir dont l index est celui de l occupant de la
// place a l instant du tir, et le nombre de tirs rendus a un autre index. Les evenements de
// l appelant ne sont pas modifies (le registre et la lecture des places les relisent tels que le
// film les ecrit).
func (t tireursParPlace) tirsParPlace(fire []grammar.FireEvent, horloge replayClock) ([]grammar.FireEvent, int) {
	if len(t) == 0 || len(fire) == 0 {
		return fire, 0
	}
	out := make([]grammar.FireEvent, len(fire))
	copy(out, fire)
	autres := 0
	for i := range out {
		if !out[i].HasShooter {
			continue
		}
		if idx, autre := t.tireur(out[i].FilmIndex, horloge.frame(out[i].TimestampUS)); autre {
			out[i].FilmIndex = idx
			autres++
		}
	}
	return out, autres
}
