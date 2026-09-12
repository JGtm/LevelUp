package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// build_score_test.go — LE BRANCHEMENT DU CALQUE DANS LE DOCUMENT (revue R1, 2026-08-18).
//
// POURQUOI CE TEST EXISTE. `score_timeline_test.go` prouve que l'ASSEMBLAGE est juste ; aucun
// test ne prouvait qu'il est APPELE. Retirer la ligne de cablage de `BuildFromPositions` laissait
// toute la suite verte et l'artefact sans courbe — exactement le defaut que le contrat de champs
// attrape cote OpenAPI, mais pas cote assemblage.

// TestDocumentCarriesScoreLayerWhenInputIsGiven — avec une entree de score, le document PORTE la
// courbe ET sa couverture. Sans entree, il ne porte NI l'une NI l'autre.
//
// Les deux sens comptent : le premier attrape un cablage retire, le second attrape une couverture
// posee par defaut, qui ferait croire a une lecture qui n'a pas eu lieu.
func TestDocumentCarriesScoreLayerWhenInputIsGiven(t *testing.T) {
	pos := positionsPourOrigine()

	avec := BuildFromPositions("m", "halo_infinite", pos, nil, Options{
		FilmClockOriginUS: 1_000_000,
		Score:             scoreEntreeSynthetique(),
	})
	if avec.ScoreTimeline == nil {
		t.Fatal("document SANS scoreTimeline alors que l'entree en porte : le calque n'est pas cable")
	}
	if avec.Coverage == nil || avec.Coverage.Score == nil {
		t.Fatal("document sans coverage.score : la couverture du calque n'est pas cablee")
	}
	if len(avec.ScoreTimeline.Teams) == 0 {
		t.Errorf("aucune courbe d'equipe publiee : %+v", avec.ScoreTimeline)
	}
	if avec.Coverage.Score.Points == 0 {
		t.Error("coverage.score.points nul alors que des courbes sont publiees")
	}
	if avec.Coverage.Score.Oracle != ScoreOracleDisplayed {
		t.Errorf("oracle = %q, attendu %q", avec.Coverage.Score.Oracle, ScoreOracleDisplayed)
	}

	sans := BuildFromPositions("m", "halo_infinite", pos, nil, Options{FilmClockOriginUS: 1_000_000})
	if sans.ScoreTimeline != nil {
		t.Errorf("calque publie sans entree : %+v", sans.ScoreTimeline)
	}
	if sans.Coverage != nil && sans.Coverage.Score != nil {
		t.Errorf("couverture de score publiee sans entree : %+v — l'absence dit « rien n'a ete "+
			"fourni a lire », un bloc vide dirait « rien n'existait »", sans.Coverage.Score)
	}
}

// scoreEntreeSynthetique : une manche, deux camps, des scores de registre qui les departagent.
// Les instants tombent DANS la fenetre : origine du film a 1 000 ms, trois frames de 100 ms.
func scoreEntreeSynthetique() *ScoreInput {
	var recs []objectiveevents.StatRecord
	for i, v := range []int64{1, 2, 3} {
		recs = append(recs, objectiveevents.StatRecord{
			TimeMS: 1_000 + i*100, Slot: 6, Round: 0,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}},
		})
	}
	scores := [2]int{3, 0}
	return &ScoreInput{Records: recs, TeamScores: &scores}
}
