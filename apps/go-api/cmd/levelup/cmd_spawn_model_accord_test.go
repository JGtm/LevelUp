package main

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func vieAccord(xuid string, debut, fin int, f *tactical.FinLue) tactical.Piste {
	return tactical.Piste{
		XUID: xuid, StartFrame: debut, EndFrame: fin, Equipe: 0, Fin: f,
		Points: []tactical.PointPiste{{T: debut, X: 1, Y: 1}, {T: fin, X: 2, Y: 2}},
	}
}

// A 100 ms par frame : 111 reapparait 3 s apres un lacher (film seul), 222 6 s apres une
// suppression (repli seul, debut de manche au film), 333 10 s apres un compte a rebours (accord).
func TestAccordDefinitions_CompareLesDeuxVerdicts(t *testing.T) {
	e := tactical.EntreeOccupation{
		MatchID: "m", IntervalleFrameMs: 100,
		Pistes: []tactical.Piste{
			vieAccord("111", 0, 100, &tactical.FinLue{Raison: tactical.RaisonLiberee, Frame: 100}),
			vieAccord("111", 130, 300, nil),
			vieAccord("222", 0, 100, &tactical.FinLue{Raison: tactical.RaisonSupprimee, Frame: 100}),
			vieAccord("222", 160, 300, nil),
			vieAccord("333", 0, 100, nil),
			vieAccord("333", 200, 300, nil),
		},
		DebutsDeCompte: []tactical.DebutCompte{{XUID: "333", Frame: 101}},
	}
	var a accordDefinitions
	a.comparer(e)
	want := accordDefinitions{documentsFilm: 1, accord: 1, filmSeul: 1, repliSeul: 1, repliSeulManche: 1}
	if a != want {
		t.Fatalf("accord = %+v, attendu %+v", a, want)
	}
	if e.Pistes[0].Fin == nil || len(e.DebutsDeCompte) != 1 {
		t.Error("la comparaison ne touche pas l'entree d'origine")
	}
}

func TestAccordDefinitions_DocumentSansLectureNEntrePas(t *testing.T) {
	e := tactical.EntreeOccupation{
		MatchID: "m", IntervalleFrameMs: 100,
		Pistes: []tactical.Piste{vieAccord("111", 0, 100, nil), vieAccord("111", 160, 300, nil)},
	}
	var a accordDefinitions
	a.comparer(e)
	if a != (accordDefinitions{}) {
		t.Fatalf("accord = %+v, attendu vide", a)
	}
}
