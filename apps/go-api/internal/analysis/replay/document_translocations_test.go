package replay

// document_translocations_test.go — la projection des téléportations sur l'axe du document.
// Le golden d'assemblage ne couvre pas ce calque (le film de référence n'a pas de
// translocateur) : sans ces tests, la conversion en frames et les deux règles d'écartement
// n'auraient aucune couverture de non-régression.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestBuildTranslocations(t *testing.T) {
	tracks := []Track{{Slot: 535, Points: []Point{{}, {}}}}
	in := []filmdec.TranslocatorTeleport{
		{TimestampUS: ecOrigin - 1, Slot: 535},         // avant l'origine : écartée
		{TimestampUS: ecOrigin + 2_000_000, Slot: 535}, // publiée, frame 20
		{TimestampUS: ecOrigin + 3_000_000, Slot: 999}, // slot sans piste : écartée
	}
	got, cov := buildTranslocations(in, tracks, ecOrigin, ecStep)
	if len(got) != 1 || got[0].T != 20 || got[0].Slot != 535 {
		t.Fatalf("publiées = %+v, attendu une seule {t:20, slot:535}", got)
	}
	if cov.Events != 3 || cov.Published != 1 || cov.BeforeOrigin != 1 || cov.Unpublished != 1 {
		t.Errorf("couverture = %+v, attendu events=3 published=1 beforeOrigin=1 unpublished=1", cov)
	}
	// Aucun événement : le calque est absent, la couverture dit zéro partout.
	if out, cov := buildTranslocations(nil, tracks, ecOrigin, ecStep); out != nil || cov.Events != 0 {
		t.Errorf("sans événement : out=%v cov=%+v, attendu nil et zéros", out, cov)
	}
}
