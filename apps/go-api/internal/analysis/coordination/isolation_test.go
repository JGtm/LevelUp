package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// mortA pose une mort d'un match avec les distances de ses coequipiers VIVANTS.
func mortA(matchID string, distances ...float64) domain.MortAExaminer {
	return domain.MortAExaminer{
		MatchID: matchID, Frame: 100, X: 1, Y: 1,
		DistancesCoequipiers: distances,
	}
}

// rayons18 : un seul match, portee d'Arene.
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
		b := Isolement([]domain.MortAExaminer{mortA("m1", c.distance)}, rayons18("m1"), 1)
		if b.Examinees != 1 {
			t.Fatalf("distance %v : examinees = %d, attendu 1", c.distance, b.Examinees)
		}
		if got := len(b.Isolees) == 1; got != c.isolee {
			t.Fatalf("distance %v m sous un rayon de 18 m : isolee = %v, attendu %v",
				c.distance, got, c.isolee)
		}
	}
}

// TestIsolement_RayonParMatchDansLeMemeUnivers — LE RAYON EST UNE PROPRIETE DU MATCH, pas
// de la lecture.
//
// LA MEME MORT, a 19 m d'un coequipier, est ISOLEE en Arene (18 m) et NE L'EST PAS en BTB
// (24 m). Un rayon unique applique a tout l'univers melangerait deux regles de jeu sous une
// seule mesure — et un filtre qui contient les deux formats est le cas normal.
func TestIsolement_RayonParMatchDansLeMemeUnivers(t *testing.T) {
	morts := []domain.MortAExaminer{mortA("arene", 19), mortA("btb", 19)}
	rayons := map[string]float64{"arene": 18, "btb": 24}
	b := Isolement(morts, rayons, 2)
	if b.Examinees != 2 {
		t.Fatalf("examinees = %d, attendu 2", b.Examinees)
	}
	if len(b.Isolees) != 1 {
		t.Fatalf("isolees = %+v, attendu la seule mort d'Arene", b.Isolees)
	}
	if b.Isolees[0].MatchID != "arene" {
		t.Fatalf("mort isolee = %q, attendu « arene » (19 m > 18 m ; en BTB 19 m < 24 m)",
			b.Isolees[0].MatchID)
	}
}

// TestIsolement_TousCoequipiersMorts_ExclusDuDenominateur — decision produit du plan.
//
// On ne peut pas etre mal accompagne quand personne ne peut accompagner. Compter ces morts
// en « isolees » ferait grimper le taux de l'equipe qui perd un combat entier : on
// mesurerait la defaite, pas le placement.
func TestIsolement_TousCoequipiersMorts_ExclusDuDenominateur(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("m1"),     // aucun coequipier vivant
		mortA("m1", 40), // isolee, celle-la compte
	}
	b := Isolement(morts, rayons18("m1"), 1)
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1 : la mort sans coequipier vivant est EXCLUE", b.Examinees)
	}
	if b.SansCoequipierVivant != 1 {
		t.Fatalf("sans_coequipier_vivant = %d, attendu 1 — l'exclusion se compte, elle ne se tait pas",
			b.SansCoequipierVivant)
	}
	if len(b.Isolees) != 1 || b.Couverture.Taux != 1 {
		t.Fatalf("bilan = %+v, attendu 1 isolee sur 1 examinee", b)
	}
}

// TestIsolement_UnAdversaireProcheNAnnuleRien — la question porte sur le SOUTIEN, pas sur
// la solitude.
//
// Le contrat le dit par construction : `DistancesCoequipiers` ne porte QUE des coequipiers.
// Ce test fige la consequence — mourir a deux metres d'un ennemi et a quarante de son
// equipe, c'est mourir isole, et c'est meme le cas typique.
func TestIsolement_UnAdversaireProcheNAnnuleRien(t *testing.T) {
	// L'adversaire a 2 m n'entre PAS dans les distances : l'appelant l'a ecarte en
	// joignant les equipes. Seul le coequipier a 40 m est la.
	b := Isolement([]domain.MortAExaminer{mortA("m1", 40)}, rayons18("m1"), 1)
	if len(b.Isolees) != 1 {
		t.Fatalf("isolees = %+v, attendu 1 : un adversaire proche n'accompagne personne", b.Isolees)
	}
}

// TestIsolement_VarianteSansRayon — le match sort de la lecture, ET IL SE COMPTE.
//
// Un rayon devine (« 18 m, c'est l'usage ») rendrait une mesure d'apparence normale sur une
// regle de jeu qu'on n'a pas mesuree. Le pied de carte doit pouvoir dire combien de matchs
// sont hors mesure.
func TestIsolement_VarianteSansRayon(t *testing.T) {
	morts := []domain.MortAExaminer{
		mortA("connu", 40),
		mortA("inconnu", 40),
		mortA("inconnu", 2),
	}
	b := Isolement(morts, map[string]float64{"connu": 18}, 1)
	if b.MatchsSansRayon != 1 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 1", b.MatchsSansRayon)
	}
	if b.Examinees != 1 || len(b.Isolees) != 1 {
		t.Fatalf("bilan = %+v : les morts du match sans rayon ne doivent NI etre examinees "+
			"NI compter comme isolees", b)
	}
	// Un rayon a zero ou negatif est traite comme une absence : jamais « tout est isole ».
	b = Isolement([]domain.MortAExaminer{mortA("m1", 2)}, map[string]float64{"m1": 0}, 1)
	if b.MatchsSansRayon != 1 || b.Examinees != 0 {
		t.Fatalf("rayon nul : bilan = %+v, attendu un match sans rayon et aucune mort examinee", b)
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
		morts = append(morts, mortA("m1", d))
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
		t.Fatalf("40 morts examinees : l'echantillon ne doit pas etre faible")
	}
	// Sous le plancher de 30, la mesure existe mais ne classe personne.
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
