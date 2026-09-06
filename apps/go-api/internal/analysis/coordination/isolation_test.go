package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// vivant / mort / inconnu : un coequipier dans chacun des trois etats.
func coeqVivant(d float64) domain.StatutCoequipier {
	return domain.StatutCoequipier{Statut: domain.StatutVoisinVivant, DistanceM: d}
}
func coeqMort() domain.StatutCoequipier {
	return domain.StatutCoequipier{Statut: domain.StatutVoisinMort}
}
func coeqInconnu() domain.StatutCoequipier {
	return domain.StatutCoequipier{Statut: domain.StatutVoisinInconnu}
}

// mortA pose une mort d'un match avec les statuts de ses coequipiers.
func mortA(matchID string, coeq ...domain.StatutCoequipier) domain.MortAExaminer {
	return domain.MortAExaminer{MatchID: matchID, X: 1, Y: 1, Coequipiers: coeq}
}

// rayons18 : la portee d'Arene pour les matchs nommes.
func rayons18(matchs ...string) map[string]float64 {
	out := map[string]float64{}
	for _, m := range matchs {
		out[m] = 18
	}
	return out
}

// TestIsolement_LaBorneEstInclusive — 17 m -> accompagne ; 18 m (la borne EXACTE) ->
// accompagne ; 19 m -> isole.
//
// La borne est le rayon du radar : se voir juste a la limite, c'est se voir. Un `<` a la
// place du `<=` ferait basculer toutes les morts pile a la portee.
func TestIsolement_LaBorneEstInclusive(t *testing.T) {
	cas := []struct {
		distance float64
		isolee   bool
	}{
		{17, false},
		{18, false}, // LA BORNE EXACTE
		{19, true},
	}
	for _, c := range cas {
		b := Isolement([]domain.MortAExaminer{mortA("m1", coeqVivant(c.distance))}, rayons18("m1"), 1)
		if b.Examinees != 1 {
			t.Fatalf("distance %v : examinees = %d, attendu 1", c.distance, b.Examinees)
		}
		if got := len(b.Isolees) == 1; got != c.isolee {
			t.Fatalf("distance %v m sous un rayon de 18 m : isolee = %v, attendu %v",
				c.distance, got, c.isolee)
		}
	}
}

// TestIsolement_RayonParMatchDansLeMemeUnivers — LE RAYON EST UNE PROPRIETE DU MATCH.
//
// LA MEME MORT, a 19 m d'un coequipier, est ISOLEE en Arene (18 m) et NE L'EST PAS en BTB
// (24 m). Un rayon unique melangerait deux regles de jeu sous une seule mesure — et un
// filtre qui contient les deux formats est le cas normal.
func TestIsolement_RayonParMatchDansLeMemeUnivers(t *testing.T) {
	morts := []domain.MortAExaminer{mortA("arene", coeqVivant(19)), mortA("btb", coeqVivant(19))}
	b := Isolement(morts, map[string]float64{"arene": 18, "btb": 24}, 2)
	if b.Examinees != 2 {
		t.Fatalf("examinees = %d, attendu 2", b.Examinees)
	}
	if len(b.Isolees) != 1 || b.Isolees[0].MatchID != "arene" {
		t.Fatalf("isolees = %+v, attendu la seule mort d'Arene", b.Isolees)
	}
}

// TestIsolement_UnCoequipierINVISIBLE_NEstPasUnCoequipierMORT — LE DEFAUT P0-1.
//
// Un occupant de vehicule non attribue (la primitive n'apparie que 15,6 a 21,1 % des vies)
// et un survivant de fin de partie (derniere vie anonyme) sont VIVANTS et invisibles. Les
// compter morts rendait « isolee » une mort survenue a trois metres d'un coequipier.
func TestIsolement_UnCoequipierINVISIBLE_NEstPasUnCoequipierMORT(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1", coeqInconnu())}, rayons18("m1"), 1)
	if len(b.Isolees) != 0 {
		t.Fatalf("isolees = %+v : un coequipier INVISIBLE n'est pas un coequipier MORT", b.Isolees)
	}
	if b.Examinees != 0 {
		t.Fatalf("examinees = %d, attendu 0 : la mort est indeterminee, pas mesuree", b.Examinees)
	}
	if b.Indeterminees != 1 {
		t.Fatalf("indeterminees = %d, attendu 1 — l'incertitude se compte, elle ne se tait pas",
			b.Indeterminees)
	}
	if b.SansCoequipierVivant != 0 {
		t.Fatalf("sans_coequipier_vivant = %d : « invisible » n'est pas « toute l'equipe a terre »",
			b.SansCoequipierVivant)
	}
}

// TestIsolement_UnCoequipierAPorteeTranche — un coequipier VU A PORTEE decide, quel que
// soit le reste.
//
// Sans cette priorite, une mort survenue a deux metres d'un coequipier serait rangee
// « indeterminee » parce qu'un TROISIEME joueur etait en vehicule : on perdrait une mesure
// CERTAINE a cause d'une incertitude sans effet.
func TestIsolement_UnCoequipierAPorteeTranche(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1", coeqVivant(2), coeqInconnu(), coeqMort())}, rayons18("m1"), 1)
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1", b.Examinees)
	}
	if len(b.Isolees) != 0 || b.Indeterminees != 0 {
		t.Fatalf("bilan = %+v, attendu une mort ACCOMPAGNEE : un coequipier a 2 m tranche", b)
	}
}

// TestIsolement_TousCoequipiersMorts_ExclusDuDenominateur — decision produit du plan.
//
// « Toute l'equipe a terre » exige que TOUS soient SUS morts. On ne peut pas etre mal
// accompagne quand personne ne peut accompagner ; l'y compter mesurerait la defaite.
func TestIsolement_TousCoequipiersMorts_ExclusDuDenominateur(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("m1", coeqMort(), coeqMort()),
		mortA("m1", coeqVivant(40)),
	}
	b := Isolement(morts, rayons18("m1"), 1)
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1 : la mort a equipe a terre est EXCLUE", b.Examinees)
	}
	if b.SansCoequipierVivant != 1 {
		t.Fatalf("sans_coequipier_vivant = %d, attendu 1", b.SansCoequipierVivant)
	}
	if len(b.Isolees) != 1 || b.Couverture.Taux != 1 {
		t.Fatalf("bilan = %+v, attendu 1 isolee sur 1 examinee", b)
	}
}

// TestIsolement_UnMortInvisibleEtUnVuHorsPortee — un vu HORS portee ne suffit pas a
// trancher quand un autre est invisible : celui-la pourrait etre a deux metres.
func TestIsolement_UnMortInvisibleEtUnVuHorsPortee(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1", coeqVivant(40), coeqInconnu())}, rayons18("m1"), 1)
	if b.Indeterminees != 1 || len(b.Isolees) != 0 {
		t.Fatalf("bilan = %+v, attendu indeterminee : l'invisible pourrait etre a portee", b)
	}
}

// TestIsolement_PositionInconnue — une mort sans lieu n'est ni peinte ni examinee.
func TestIsolement_PositionInconnue(t *testing.T) {
	m := mortA("m1", coeqVivant(40))
	m.PositionInconnue = true
	b := Isolement([]domain.MortAExaminer{m}, rayons18("m1"), 1)
	if b.PositionInconnue != 1 {
		t.Fatalf("position_inconnue = %d, attendu 1", b.PositionInconnue)
	}
	if b.Examinees != 0 || len(b.Isolees) != 0 {
		t.Fatalf("bilan = %+v : une mort sans lieu ne se mesure pas", b)
	}
}

// TestIsolement_AucunCoequipierDuTout — un joueur seul de son camp au registre : pour le
// placement, c'est « on ne peut pas etre mal accompagne ».
func TestIsolement_AucunCoequipierDuTout(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1")}, rayons18("m1"), 1)
	if b.SansCoequipierVivant != 1 || b.Examinees != 0 {
		t.Fatalf("bilan = %+v, attendu une exclusion « equipe a terre »", b)
	}
}

// TestIsolement_VarianteSansRayon — les morts du match sont ECARTEES. Le COMPTE des matchs,
// lui, est pose par l'appelant : ce paquet ne voit que des morts, et un match sans mort du
// joueur ne passerait jamais ici (correction P0-2).
func TestIsolement_VarianteSansRayon(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("connu", coeqVivant(40)),
		mortA("inconnu", coeqVivant(40)),
		mortA("inconnu", coeqVivant(2)),
	}
	b := Isolement(morts, map[string]float64{"connu": 18}, 1)
	if b.Examinees != 1 || len(b.Isolees) != 1 {
		t.Fatalf("bilan = %+v : les morts du match sans rayon ne sont NI examinees NI isolees", b)
	}
	// Un rayon a zero ou negatif est traite comme une absence : jamais « tout est isole ».
	b = Isolement([]domain.MortAExaminer{mortA("m1", coeqVivant(2))}, map[string]float64{"m1": 0}, 1)
	if b.Examinees != 0 {
		t.Fatalf("rayon nul : examinees = %d, attendu 0", b.Examinees)
	}
}

// TestIsolement_LaCouvertureEstLaFormeCanonique — jamais un taux nu : le brut, le
// denominateur, la quantite par match et le drapeau d'echantillon faible voyagent avec.
func TestIsolement_LaCouvertureEstLaFormeCanonique(t *testing.T) {
	morts := make([]domain.MortAExaminer, 0, 40)
	for i := 0; i < 40; i++ {
		d := 40.0 // isolee
		if i%4 == 0 {
			d = 5.0 // accompagnee
		}
		morts = append(morts, mortA("m1", coeqVivant(d)))
	}
	b := Isolement(morts, rayons18("m1"), 4)
	if b.Examinees != 40 || b.Couverture.N != 40 {
		t.Fatalf("denominateur = %d / %d, attendu 40", b.Examinees, b.Couverture.N)
	}
	if b.Couverture.Brut != 30 || b.Couverture.Taux != 0.75 {
		t.Fatalf("couverture = %+v, attendu 30 isolees sur 40 (0,75)", b.Couverture)
	}
	if b.Couverture.ParMatch != 7.5 {
		t.Fatalf("par match = %v, attendu 7,5 (30 / 4 matchs)", b.Couverture.ParMatch)
	}
	if b.Couverture.EchantillonFaible {
		t.Fatal("40 morts examinees : l'echantillon ne doit pas etre faible")
	}
	petit := Isolement(morts[:8], rayons18("m1"), 1)
	if !petit.Couverture.EchantillonFaible {
		t.Fatalf("8 morts examinees : l'echantillon doit etre faible (seuil %d)", SeuilEchantillonFaible)
	}
}

// TestIsolement_AucuneMort — aucune donnee n'est pas « zero pour cent ».
func TestIsolement_AucuneMort(t *testing.T) {
	b := Isolement(nil, rayons18("m1"), 3)
	if len(b.Isolees) != 0 || b.Examinees != 0 {
		t.Fatalf("bilan = %+v", b)
	}
	if b.Couverture.Taux != 0 || !b.Couverture.EchantillonFaible {
		t.Fatalf("couverture = %+v, attendu un taux nul ET un echantillon faible", b.Couverture)
	}
}
