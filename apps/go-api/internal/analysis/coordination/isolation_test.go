package coordination

// isolation_test.go — LES TROIS SORTIES, et le rayon PAR MATCH.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func metres(v float64) *float64 { return &v }

// mort pose une mort a examiner : la distance au coequipier visible le plus proche, et combien
// de coequipiers etaient en mesure d'accompagner.
func mortIsolement(matchID string, proche *float64, visibles, horsDeVue int) domain.MortAExaminer {
	return domain.MortAExaminer{
		MatchID: matchID, X: 1, Y: 1,
		PlusProcheM: proche, Visibles: visibles, HorsDeVue: horsDeVue,
	}
}

func rayons18(ids ...string) map[string]float64 {
	out := map[string]float64{}
	for _, id := range ids {
		out[id] = 18
	}
	return out
}

// TestIsolement_LaBorneEstInclusive — le rayon est la PORTEE DU RADAR : se voir juste a la
// limite, c'est se voir.
//
// Une borne stricte ferait basculer en « isolee » toutes les morts pile a 18 m, et ce sont
// justement celles ou la mesure est la plus fragile.
func TestIsolement_LaBorneEstInclusive(t *testing.T) {
	for _, c := range []struct {
		nom      string
		distance float64
		isolee   bool
	}{
		{"sous la borne", 17.9, false},
		{"pile sur la borne", 18, false},
		{"juste au-dela", 18.01, true},
	} {
		t.Run(c.nom, func(t *testing.T) {
			b := Isolement([]domain.MortAExaminer{mortIsolement("m1", metres(c.distance), 1, 0)},
				rayons18("m1"), 1)
			if got := len(b.Isolees) == 1; got != c.isolee {
				t.Fatalf("distance %v : isolee = %v, attendu %v", c.distance, got, c.isolee)
			}
			if b.Examinees != 1 {
				t.Fatalf("examinees = %d, attendu 1 : la mort est examinable dans les trois cas",
					b.Examinees)
			}
		})
	}
}

// TestIsolement_LeRayonEstParMatch — LA MEME MORT, a 19 m d'un coequipier, est ISOLEE en Arene
// (18 m) et NE L'EST PAS en BTB (24 m).
//
// Un rayon unique melangerait deux regles de jeu sous une seule mesure, et un filtre qui
// contient les deux formats est le cas normal.
func TestIsolement_LeRayonEstParMatch(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{
		mortIsolement("arene", metres(19), 1, 0),
		mortIsolement("btb", metres(19), 1, 0),
	}, map[string]float64{"arene": 18, "btb": 24}, 2)

	if b.Examinees != 2 {
		t.Fatalf("examinees = %d, attendu 2", b.Examinees)
	}
	if len(b.Isolees) != 1 || b.Isolees[0].MatchID != "arene" {
		t.Fatalf("isolees = %+v, attendu la seule mort d'Arene : 19 m depasse 18 m mais pas 24 m",
			b.Isolees)
	}
}

// TestIsolement_EquipeATerre_ExclueEtComptee — personne ne pouvait accompagner.
//
// ON NE PEUT PAS ETRE MAL ACCOMPAGNE QUAND PERSONNE NE PEUT ACCOMPAGNER. Compter ces morts au
// denominateur ferait monter le taux avec les hecatombes de l'equipe, c'est-a-dire avec quelque
// chose que le placement du joueur ne commande pas. Le COMPTE, lui, sort : un denominateur
// ampute ressemble sinon a un denominateur complet.
func TestIsolement_EquipeATerre_ExclueEtComptee(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{
		mortIsolement("m1", nil, 0, 0),        // tous morts ou partis
		mortIsolement("m1", metres(40), 1, 0), // un coequipier visible, trop loin
	}, rayons18("m1"), 1)

	if b.EquipeATerre != 1 {
		t.Fatalf("equipe_a_terre = %d, attendu 1", b.EquipeATerre)
	}
	if b.Examinees != 1 {
		t.Fatalf("examinees = %d, attendu 1 : la mort sans personne pour accompagner est ECARTEE",
			b.Examinees)
	}
	if len(b.Isolees) != 1 {
		t.Fatalf("isolees = %d, attendu 1", len(b.Isolees))
	}
}

// TestIsolement_HorsDeVue_ExamineeEtIsolee — LE VEHICULE.
//
// Un coequipier vivant mais non repliqué PEUT accompagner (la mort reste examinable), mais on ne
// sait pas ou il est — il ne peut donc pas etre « a portee ». Les deux erreurs symetriques ont
// ete commises : le compter mort faisait sortir « equipe a terre » une mort a trois metres d'un
// Warthog ; lui inventer une position aurait fabrique un accompagnement.
func TestIsolement_HorsDeVue_ExamineeEtIsolee(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortIsolement("m1", nil, 0, 2)}, rayons18("m1"), 1)

	if b.EquipeATerre != 0 {
		t.Fatalf("equipe_a_terre = %d, attendu 0 : deux coequipiers VIVANTS, simplement hors de vue",
			b.EquipeATerre)
	}
	if b.Examinees != 1 || len(b.Isolees) != 1 {
		t.Fatalf("examinees = %d, isolees = %d, attendu 1 et 1 : sans position, ils ne peuvent "+
			"pas etre a portee", b.Examinees, len(b.Isolees))
	}
}

// TestIsolement_MatchSansRayon_NEntrePas — un match dont la variante n'a pas de portee mesuree
// ne fournit NI numerateur NI denominateur.
//
// Le laisser au denominateur diviserait la mesure par des matchs qu'on a refuse de lire : deux
// matchs dont un Husky Raid rendraient 0,5 mort isolee par match au lieu de 1.
func TestIsolement_MatchSansRayon_NEntrePas(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{
		mortIsolement("connu", metres(40), 1, 0),
		mortIsolement("inconnu", metres(40), 1, 0),
	}, rayons18("connu"), 1)

	if b.Examinees != 1 || len(b.Isolees) != 1 {
		t.Fatalf("examinees = %d, isolees = %d, attendu 1 et 1", b.Examinees, len(b.Isolees))
	}
	if b.Couverture.ParMatch != 1 {
		t.Fatalf("par match = %v, attendu 1 : diviser par 2 ferait varier la mesure avec les "+
			"matchs qu'on refuse de lire", b.Couverture.ParMatch)
	}
}

// TestIsolement_RayonNul_EstUnRayonInconnu — une entree a zero ou negative n'est pas « personne
// n'est jamais a portee », c'est une table mal remplie. Le match sort.
func TestIsolement_RayonNul_EstUnRayonInconnu(t *testing.T) {
	b := Isolement([]domain.MortAExaminer{mortIsolement("m1", metres(2), 1, 0)},
		map[string]float64{"m1": 0}, 1)
	if b.Examinees != 0 {
		t.Fatalf("examinees = %d, attendu 0 : un rayon nul est un rayon INCONNU", b.Examinees)
	}
}

// TestIsolement_LeTauxSortSousSaFormeCanonique — jamais un nombre seul.
func TestIsolement_LeTauxSortSousSaFormeCanonique(t *testing.T) {
	morts := make([]domain.MortAExaminer, 0, 40)
	for i := 0; i < 40; i++ {
		morts = append(morts, mortIsolement("m1", metres(40), 1, 0)) // toutes isolees
	}
	b := Isolement(morts, rayons18("m1"), 4)

	if b.Couverture.Taux != 1 || b.Couverture.Brut != 40 || b.Couverture.N != 40 {
		t.Fatalf("couverture = %+v, attendu taux 1 sur 40/40", b.Couverture)
	}
	if b.Couverture.ParMatch != 10 {
		t.Fatalf("par match = %v, attendu 10 (40 isolees / 4 matchs)", b.Couverture.ParMatch)
	}
	if b.Couverture.EchantillonFaible {
		t.Fatal("echantillon faible a 40 morts, alors que le plancher est 30")
	}
	petit := Isolement(morts[:8], rayons18("m1"), 1)
	if !petit.Couverture.EchantillonFaible {
		t.Fatal("8 morts : l'echantillon faible doit etre pose — 100 % sur huit morts est un tirage")
	}
}
