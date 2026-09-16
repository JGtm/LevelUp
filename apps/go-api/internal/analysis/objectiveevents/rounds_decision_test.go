package objectiveevents

// rounds_decision_test.go — LE VERDICT DE LA LECTURE DES DESIGNATEURS, ET SA MUTATION.
//
// Le lot 1.9.11 ne change AUCUNE manche retenue : il rend LISIBLE ce que la chaine decidait en
// silence. Ces tests tiennent les deux bouts — le verdict publie, et le fait que la regle
// d'ordre soit bien ce qui produit la contradiction (mutation).

import (
	"reflect"
	"testing"
)

// fixtureManche2SansManche1 reproduit la forme mesuree sur `fb1a1a72` et sur 23 autres films du
// cache (lot 1.9.11, 2026-09-16) : une manche 0 pleine, AUCUN enregistrement en manche 1, et un
// designateur 2 MATERIEL — 148 enregistrements, le compte exact de `fb1a1a72`.
//
// C'est le cas que l'item 1.9.11 voulait publier comme une seconde manche. La mesure l'a refute :
// 23 des 24 films qui portent ce motif ont fini de 38 a 442 s DANS leur temps reglementaire sur
// un mode SANS manche. Le designateur est donc ECRIT, MATERIEL, et CONTREDIT par l'ordre — et
// c'est cela que l'artefact doit dire.
func fixtureManche2SansManche1() []StatRecord {
	var recs []StatRecord
	recs = append(recs, joueurSerie(0, 1_000, 900, 200)...)
	// Pas de manche 1 : aucun enregistrement, d'aucune sorte — comme sur `fb1a1a72`.
	recs = append(recs, joueurSerie(2, 60_000, 148, 0)...)
	return recs
}

// TestResolveRoundsPublieLeDesignateurEcritEtLaContradiction — LE VERDICT COMPLET.
func TestResolveRoundsPublieLeDesignateurEcritEtLaContradiction(t *testing.T) {
	d := ResolveRounds(fixtureManche2SansManche1())

	if want := []int{0, 2}; !reflect.DeepEqual(d.Written, want) {
		t.Errorf("designateurs ECRITS = %v, attendu %v : la grammaire doit etre publiee telle "+
			"que le film l'ecrit, avant tout jugement", d.Written, want)
	}
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v", d.Real, want)
	}
	if want := []int{2}; !reflect.DeepEqual(d.Contradicted, want) {
		t.Errorf("contradictions = %v, attendu %v : un designateur MATERIEL que l'ordre refuse "+
			"se compte et se publie, il ne se tait pas (D14 c)", d.Contradicted, want)
	}
	if d.ContradictedRecords != 148 {
		t.Errorf("enregistrements contredits = %d, attendu 148 : sans ce denominateur, "+
			"« une manche refusee » ne se juge pas", d.ContradictedRecords)
	}
	if d.Decreed {
		t.Error("manche 0 DECRETEE alors qu'elle est admise par la matiere : le repli s'est " +
			"declenche devant une lecture disponible, ce que D14 (b) interdit")
	}
}

// TestResolveRoundsMutationDeLaRegleDOrdre — LA MUTATION, ET C'EST LA REGLE D'ORDRE QU'ELLE VISE.
//
// Neutraliser la regle d'ordre, c'est admettre toute manche que l'UN des deux criteres retient.
// Sur la fixture, cela publie DEUX manches et ZERO contradiction : le test ci-dessus rougirait.
// La mutation est jouee ici sur une COPIE de la regle — le verdict de production reste celui de
// [ResolveRounds], et la restauration est le fait que les deux ne coincident pas.
func TestResolveRoundsMutationDeLaRegleDOrdre(t *testing.T) {
	recs := fixtureManche2SansManche1()
	runs, material := modeScoreRunsByRound(recs), materialRounds(recs)

	var sansOrdre []int
	for round := 0; round <= statMaxRound; round++ {
		if admissibleRound(runs, material, round) {
			sansOrdre = append(sansOrdre, round)
		}
	}
	if want := []int{0, 2}; !reflect.DeepEqual(sansOrdre, want) {
		t.Fatalf("sans la regle d'ordre : %v, attendu %v — la fixture ne porte pas le motif "+
			"qu'elle doit porter", sansOrdre, want)
	}

	d := ResolveRounds(recs)
	if reflect.DeepEqual(d.Real, sansOrdre) {
		t.Error("la regle d'ordre ne change RIEN sur la fixture : le test du verdict ne prouve " +
			"alors plus rien, et la mesure des 24 films du cache n'a plus de garde-rail")
	}
}

// TestResolveRoundsDecreteLaManche0QuandLeFilmEstMuet — LE SEUL REPLI DE LA CHAINE.
//
// Aucun enregistrement : le film ne dit rien, la manche 0 est DECRETEE pour qu'il reste lisible.
// C'est `repli_manche_zero_decretee`, et le drapeau est ce que l'artefact publie
// (`coverage.score.roundsDecreed`) et ce que le compteur de replis compte.
func TestResolveRoundsDecreteLaManche0QuandLeFilmEstMuet(t *testing.T) {
	d := ResolveRounds(nil)

	if !d.Decreed {
		t.Error("le repli ne s'est pas declare alors qu'aucune manche n'etait admise : " +
			"`coverage.score.roundsDecreed` mentirait, et le compteur resterait a zero")
	}
	if want := []int{0}; !reflect.DeepEqual(d.Real, want) {
		t.Errorf("manches retenues = %v, attendu %v : un film muet reste lisible", d.Real, want)
	}
	if len(d.Written) != 0 {
		t.Errorf("designateurs ECRITS = %v sur un film sans enregistrement", d.Written)
	}
	if len(d.Contradicted) != 0 || d.ContradictedRecords != 0 {
		t.Errorf("contradiction %v / %d sur un film muet : un silence n'est pas un desaccord",
			d.Contradicted, d.ContradictedRecords)
	}
}

// TestResolveRoundsNeContreditPasUnAncrageFortuit — LE DENOMINATEUR RESTE LISIBLE.
//
// Un designateur porte par UN enregistrement fortuit (mesure du lot 1.9.11 : 1 506 des 1 535
// designateurs jetes du cache portent 5 enregistrements ou moins, et 950 en portent UN) figure
// dans `Written` — c'est la grammaire — mais il n'est PAS une contradiction : il ne passe aucun
// des deux criteres d'admission, donc l'ordre n'a rien refuse.
func TestResolveRoundsNeContreditPasUnAncrageFortuit(t *testing.T) {
	recs := append(joueurSerie(0, 1_000, 900, 200),
		StatRecord{TimeMS: 42_000, Slot: 12, Round: 5, Comps: map[int]StatValue{modeScoreComp: {A: 3}}})

	d := ResolveRounds(recs)
	if want := []int{0, 5}; !reflect.DeepEqual(d.Written, want) {
		t.Errorf("designateurs ECRITS = %v, attendu %v", d.Written, want)
	}
	if len(d.Contradicted) != 0 {
		t.Errorf("contradictions = %v : un ancrage d'UN enregistrement n'est pas une manche "+
			"refusee par l'ordre, c'est une manche qu'aucun critere n'admet", d.Contradicted)
	}
}

// TestRealRoundsRendLeMemeEnsembleQueResolveRounds — LA NEUTRALITE, TENUE PAR UN TEST.
//
// Le lot ne devait changer AUCUNE manche retenue. Les deux portes doivent donc rendre le meme
// ensemble sur chaque forme du corpus de fixtures du paquet.
func TestRealRoundsRendLeMemeEnsembleQueResolveRounds(t *testing.T) {
	cas := map[string][]StatRecord{
		"manche 2 sans manche 1": fixtureManche2SansManche1(),
		"film muet":              nil,
		"trois manches pleines": append(append(
			joueurSerie(0, 1_000, 300, 200), joueurSerie(1, 200_000, 300, 200)...),
			joueurSerie(2, 400_000, 300, 200)...),
		"manche 0 courte": append(append(
			modeSerie(6, 0, 1_000, 2), modeSerie(6, 1, 20_000, 1, 2, 3)...),
			modeSerie(6, 2, 40_000, 1, 2, 3)...),
	}
	for nom, recs := range cas {
		got, want := RealRounds(recs), ResolveRounds(recs).RealSet()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s : RealRounds = %v, ResolveRounds().RealSet() = %v", nom, got, want)
		}
	}
}
