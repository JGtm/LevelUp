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

// TestBuildTranslocationsVaEtVient fige la publication du saut : les six coordonnées sont
// armées ENSEMBLE quand la charge a été lue, ABSENTES EN BLOC sinon — jamais six zéros qui
// se liraient comme un saut vers l'origine du monde. `positioned` porte le dénominateur.
func TestBuildTranslocationsVaEtVient(t *testing.T) {
	tracks := []Track{{Slot: 535, Points: []Point{{}, {}}}, {Slot: 560, Points: []Point{{}, {}}}}
	in := []filmdec.TranslocatorTeleport{
		{TimestampUS: ecOrigin + 2_000_000, Slot: 535, HasPositions: true,
			From: [3]float32{2.789, 152.174, 3.5}, To: [3]float32{17.341, 135.502, 1.25}},
		{TimestampUS: ecOrigin + 3_000_000, Slot: 560}, // charge non lue : sans va-et-vient
	}
	got, cov := buildTranslocations(in, tracks, ecOrigin, ecStep)
	if len(got) != 2 {
		t.Fatalf("publiées = %+v, attendu 2", got)
	}
	if got[0].FX == nil || got[0].TZ == nil {
		t.Fatalf("le saut lu doit porter ses six coordonnées : %+v", got[0])
	}
	// Arrondi au centimètre, comme les points de piste (coordScale).
	if *got[0].FX != 2.79 || *got[0].FY != 152.17 || *got[0].TX != 17.34 || *got[0].TY != 135.5 {
		t.Errorf("va-et-vient publié (%.3f,%.3f) -> (%.3f,%.3f), attendu (2.79,152.17) -> (17.34,135.50)",
			*got[0].FX, *got[0].FY, *got[0].TX, *got[0].TY)
	}
	if *got[0].FZ != 3.5 || *got[0].TZ != 1.25 {
		t.Errorf("altitudes publiées %.2f / %.2f, attendu 3.50 / 1.25", *got[0].FZ, *got[0].TZ)
	}
	for _, p := range []*float32{got[1].FX, got[1].FY, got[1].FZ, got[1].TX, got[1].TY, got[1].TZ} {
		if p != nil {
			t.Fatalf("une téléportation sans charge lue a publié une coordonnée : %+v", got[1])
		}
	}
	if cov.Published != 2 || cov.Positioned != 1 {
		t.Errorf("couverture = %+v, attendu published=2 positioned=1", cov)
	}
}
