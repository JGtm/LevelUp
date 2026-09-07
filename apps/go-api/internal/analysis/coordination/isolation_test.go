package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// coeqVu / coeqAbsent / coeqVivantSansPosition : les trois etats qu'un coequipier peut
// prendre a l'instant d'une mort, tels que le SERVICE les resout.
func coeqVu(d float64) domain.EtatCoequipier {
	return domain.EtatCoequipier{Vivant: true, PositionConnue: true, DistanceM: d}
}
func coeqAbsent() domain.EtatCoequipier {
	return domain.EtatCoequipier{}
}
func coeqVivantSansPosition() domain.EtatCoequipier {
	return domain.EtatCoequipier{Vivant: true}
}

// mortA pose une mort d'un match avec l'etat de ses coequipiers.
func mortA(matchID string, coeq ...domain.EtatCoequipier) domain.MortAExaminer {
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
		b := Isolement([]domain.MortAExaminer{mortA("m1", coeqVu(c.distance))}, rayons18("m1"), 1)
		if b.Examinees != 1 {
			t.Fatalf("distance %v : examinees = %d, attendu 1", c.distance, b.Examinees)
		}
		if got := len(b.Isolees) == 1; got != c.isolee {
			t.Fatalf("distance %v m sous un rayon de 18 m : isolee = %v, attendu %v",
				c.distance, got, c.isolee)
		}
	}
}

// TestIsolement_RayonParMatchDansLeMemeUnivers — LA MEME MORT, a 19 m d'un coequipier, est
// ISOLEE en Arene (18 m) et NE L'EST PAS en BTB (24 m).
func TestIsolement_RayonParMatchDansLeMemeUnivers(t *testing.T) {
	morts := []domain.MortAExaminer{mortA("arene", coeqVu(19)), mortA("btb", coeqVu(19))}
	b := Isolement(morts, map[string]float64{"arene": 18, "btb": 24}, 2)
	if b.Examinees != 2 {
		t.Fatalf("examinees = %d, attendu 2", b.Examinees)
	}
	if len(b.Isolees) != 1 || b.Isolees[0].MatchID != "arene" {
		t.Fatalf("isolees = %+v, attendu la seule mort d'Arene", b.Isolees)
	}
}

// TestIsolement_UnCoequipierAPorteeTranche — un coequipier VU A PORTEE decide, quel que
// soit le reste. Sans cette priorite, une mort survenue a deux metres d'un coequipier
// serait rangee ailleurs a cause d'un TROISIEME joueur : on perdrait une mesure CERTAINE.
func TestIsolement_UnCoequipierAPorteeTranche(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{
		mortA("m1", coeqVu(2), coeqVivantSansPosition(), coeqAbsent()),
	}, rayons18("m1"), 1)
	if b.Examinees != 1 || len(b.Isolees) != 0 {
		t.Fatalf("bilan = %+v, attendu une mort ACCOMPAGNEE : un coequipier a 2 m tranche", b)
	}
}

// TestIsolement_EquipeATerre_ExclueEtPUBLIEE — decision produit du plan, et le compte SORT.
//
// « Toute l'equipe a terre » exige que TOUS les coequipiers soient morts ou partis. On ne
// peut pas etre mal accompagne quand personne ne peut accompagner ; l'y compter mesurerait
// la defaite. Et le compte est PUBLIE : il etait mesure sans jamais sortir, si bien qu'un
// denominateur ampute ressemblait a un denominateur complet.
func TestIsolement_EquipeATerre_ExclueEtPubliee(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("m1", coeqAbsent(), coeqAbsent()),
		mortA("m1", coeqVu(40)),
	}
	b := Isolement(morts, rayons18("m1"), 1)
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1 : la mort a equipe a terre est EXCLUE", b.Examinees)
	}
	if b.EquipeATerre != 1 {
		t.Fatalf("equipe_a_terre = %d, attendu 1 — l'exclusion se compte ET se publie", b.EquipeATerre)
	}
	if len(b.Isolees) != 1 || b.Couverture.Taux != 1 {
		t.Fatalf("bilan = %+v, attendu 1 isolee sur 1 examinee", b)
	}
}

// TestIsolement_AucunCoequipierDuTout — un joueur seul de son camp : pour le placement,
// c'est « on ne peut pas etre mal accompagne ».
func TestIsolement_AucunCoequipierDuTout(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1")}, rayons18("m1"), 1)
	if b.EquipeATerre != 1 || b.Examinees != 0 {
		t.Fatalf("bilan = %+v, attendu une exclusion « equipe a terre »", b)
	}
}

// TestIsolement_VivantSansPosition_NeProuveRien — un coequipier vivant dont le film n'a
// jamais donne de position compte comme PRESENCE (la mort est examinee), jamais comme
// SOUTIEN (il ne peut pas rendre la mort accompagnee).
func TestIsolement_VivantSansPosition_NeProuveRien(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortA("m1", coeqVivantSansPosition())}, rayons18("m1"), 1)
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1 : un coequipier vivant est une presence", b.Examinees)
	}
	if len(b.Isolees) != 1 {
		t.Fatalf("isolees = %+v, attendu 1 : sans position, il ne peut pas accompagner", b.Isolees)
	}
}

// TestIsolement_VarianteSansRayon — les morts du match sont ECARTEES. Le COMPTE des matchs
// est pose par l'appelant : ce paquet ne voit que des morts, et un match sans mort du
// joueur ne passerait jamais ici.
func TestIsolement_VarianteSansRayon(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("connu", coeqVu(40)),
		mortA("inconnu", coeqVu(40)),
		mortA("inconnu", coeqVu(2)),
	}
	b := Isolement(morts, map[string]float64{"connu": 18}, 1)
	if b.Examinees != 1 || len(b.Isolees) != 1 {
		t.Fatalf("bilan = %+v : les morts du match sans rayon ne sont NI examinees NI isolees", b)
	}
	b = Isolement([]domain.MortAExaminer{mortA("m1", coeqVu(2))}, map[string]float64{"m1": 0}, 1)
	if b.Examinees != 0 {
		t.Fatalf("rayon nul : examinees = %d, attendu 0", b.Examinees)
	}
}

// TestIsolement_LaCouvertureEstLaFormeCanonique — jamais un taux nu.
func TestIsolement_LaCouvertureEstLaFormeCanonique(t *testing.T) {
	morts := make([]domain.MortAExaminer, 0, 40)
	for i := 0; i < 40; i++ {
		d := 40.0 // isolee
		if i%4 == 0 {
			d = 5.0 // accompagnee
		}
		morts = append(morts, mortA("m1", coeqVu(d)))
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
	if petit := Isolement(morts[:8], rayons18("m1"), 1); !petit.Couverture.EchantillonFaible {
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
