package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// flag_carries_test.go — LES REGLES DU DRAPEAU, sans film.
//
// Chaque test fige UNE regle et la fait TOMBER si elle disparait. Les entrees sont synthetiques :
// des evenements de statistique dates, un fil des morts, deux socles et des pistes publiees —
// exactement ce que le constructeur passera. La verite terrain sur films reels vit dans
// l'instrument sous garde `OBJ_FILM` (`objectifs_phase1_drapeau_test.go`), qui appelle CE code.

// flagTestCtx fabrique un contexte a l'echelle 100 ms/frame, sans decalage d'horloge : la frame
// d'un instant du match est donc `ms / 100`, ce qui rend les attentes lisibles.
func flagTestCtx(tracks []Track, deaths []Death, frames int) flagCarryCtx {
	return flagCarryCtx{matchClock: matchClock{origin: 0, step: 100_000, frames: frames}, tracks: tracks, deaths: deaths}
}

// flagTestTrack fabrique une piste d'un joueur, un point toutes les frames de from a to.
func flagTestTrack(slot uint32, xuid string, from, to int, x, y float32) Track {
	tr := Track{Slot: slot, Team: TeamNeutral, XUID: xuid, StartFrame: from, EndFrame: to}
	for t := from; t <= to; t++ {
		tr.Points = append(tr.Points, Point{T: t, X: x, Y: y})
	}
	return tr
}

// flagTestSignals rend des signaux qui tiennent la regle de mode (le verdict n'est pas le sujet
// de ces tests-ci : il a les siens, dans `objectiveevents/flagfilm_test.go`).
func flagTestSignals() objectiveevents.FlagFilmSignals {
	return objectiveevents.FlagFilmSignals{Bursts: 3, Captures: 3, Steals: 2, Grabs: 4}
}

// flagStateCarrying dit si un etat publie est un etat PORTE — ferme ou non.
func flagStateCarrying(state string) bool {
	return state == FlagStateCarried || state == FlagStateCarriedOpen
}

// flagOfTeam rend le drapeau d'une equipe, ou echoue.
func flagOfTeam(t *testing.T, carries []FlagCarry, team int) FlagCarry {
	t.Helper()
	for _, c := range carries {
		if c.Team == team {
			return c
		}
	}
	t.Fatalf("aucun drapeau d'equipe %d parmi %d publies", team, len(carries))
	return FlagCarry{}
}

// assertFlagStates compare la suite d'etats d'un drapeau a celle attendue.
func assertFlagStates(t *testing.T, f FlagCarry, want []string) {
	t.Helper()
	got := make([]string, 0, len(f.Spans))
	for _, s := range f.Spans {
		got = append(got, s.State)
	}
	if len(got) != len(want) {
		t.Fatalf("etats %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("etats %v, attendu %v", got, want)
		}
	}
	for i := 1; i < len(f.Spans); i++ {
		if f.Spans[i].T0 != f.Spans[i-1].T1+1 {
			t.Errorf("trou ou recouvrement entre les spans %d et %d : [%d,%d] puis [%d,%d]",
				i-1, i, f.Spans[i-1].T0, f.Spans[i-1].T1, f.Spans[i].T0, f.Spans[i].T1)
		}
	}
}
