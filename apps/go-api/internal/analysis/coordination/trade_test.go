package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// Jeu de reference : A, B, C dans l'equipe 0 ; K, L dans l'equipe 1.
func equipesArene() domain.EquipesParMatch {
	return domain.EquipesParMatch{
		"m1": {"A": 0, "B": 0, "C": 0, "K": 1, "L": 1},
		"m2": {"A": 0, "B": 0, "C": 0, "K": 1, "L": 1},
	}
}

func ev(match, victime, tueur string, t int64) domain.KillEvent {
	return domain.KillEvent{MatchID: match, VictimXUID: victime, KillerXUID: tueur, TimeMs: t}
}

// mort : retrouve le suivi d'une mort par victime et instant.
func mort(t *testing.T, bilan domain.BilanEchanges, victime string, instant int64) domain.MortSuivie {
	t.Helper()
	for _, m := range bilan.Morts {
		if m.VictimeXUID == victime && m.TimeMs == instant {
			return m
		}
	}
	t.Fatalf("mort de %s a %d ms absente de %+v", victime, instant, bilan.Morts)
	return domain.MortSuivie{}
}

// TestEchanges_FenetreDeCinqSecondes : mort a t, tueur abattu a t+3 s -> echange ; a t+9 s ->
// aucun echange. Les bornes exactes sont verifiees aussi : 5 000 ms compte, 5 001 non.
func TestEchanges_FenetreDeCinqSecondes(t *testing.T) {
	cas := []struct {
		nom      string
		instant  int64
		echangee bool
	}{
		{"trois secondes", 3000, true},
		{"neuf secondes", 9000, false},
		{"pile la fenetre", FenetreEchangeMs, true},
		{"une milliseconde de trop", FenetreEchangeMs + 1, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			bilan := Echanges([]domain.KillEvent{
				ev("m1", "A", "K", 0),
				ev("m1", "K", "B", c.instant),
			}, equipesArene())

			m := mort(t, bilan, "A", 0)
			if !m.Vengeable {
				t.Fatalf("la mort de A par K devait etre vengeable")
			}
			if m.Vengee != c.echangee {
				t.Fatalf("Vengee = %v, attendu %v (tueur abattu a %d ms)", m.Vengee, c.echangee, c.instant)
			}
			if !c.echangee {
				return
			}
			if m.VengeurXUID != "B" {
				t.Fatalf("VengeurXUID = %q, attendu B", m.VengeurXUID)
			}
			if m.DelaiMs != c.instant {
				t.Fatalf("DelaiMs = %d, attendu %d", m.DelaiMs, c.instant)
			}
		})
	}
}

// TestEchanges_UnTueurDeDeuxCoequipiersEchangeLesDeux : cas limite (a) du plan. K abat A puis
// B, et tombe a t+4 s : LES DEUX morts sont vengees par le meme kill. Un appariement
// un-pour-un n'en solderait qu'une.
func TestEchanges_UnTueurDeDeuxCoequipiersEchangeLesDeux(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m1", "B", "K", 1000),
		ev("m1", "K", "C", 4000),
	}, equipesArene())

	if bilan.NbVengees != 2 {
		t.Fatalf("NbVengees = %d, attendu 2 : %+v", bilan.NbVengees, bilan.Morts)
	}
	if a := mort(t, bilan, "A", 0); !a.Vengee || a.VengeurXUID != "C" || a.DelaiMs != 4000 {
		t.Fatalf("mort de A = %+v, attendu vengee par C en 4000 ms", a)
	}
	if b := mort(t, bilan, "B", 1000); !b.Vengee || b.VengeurXUID != "C" || b.DelaiMs != 3000 {
		t.Fatalf("mort de B = %+v, attendu vengee par C en 3000 ms", b)
	}

	// Les paires : C venge A et C venge B, une fois chacune.
	if len(bilan.Paires) != 2 {
		t.Fatalf("len(Paires) = %d, attendu 2 : %+v", len(bilan.Paires), bilan.Paires)
	}
	attendu := []domain.PaireEchange{
		{VengeurXUID: "C", VengeXUID: "A", Nombre: 1, DelaiMoyenMs: 4000},
		{VengeurXUID: "C", VengeXUID: "B", Nombre: 1, DelaiMoyenMs: 3000},
	}
	for i, a := range attendu {
		if bilan.Paires[i] != a {
			t.Fatalf("Paires[%d] = %+v, attendu %+v", i, bilan.Paires[i], a)
		}
	}
}

// TestEchanges_TueurMortSansKillDeCoequipier : cas limite (b) du plan. Le tueur meurt bien
// dans la fenetre, mais aucun coequipier de la victime ne l'a abattu : rien n'est echange.
func TestEchanges_TueurMortSansKillDeCoequipier(t *testing.T) {
	cas := []struct {
		nom    string
		tueur  string
		relire string
	}{
		{"environnement, chute ou grenade perdue", "", "personne ne revendique la mort du tueur"},
		{"suicide du tueur", "K", "le tueur s'est tue lui-meme"},
		{"tir ami dans le camp du tueur", "L", "le tueur est abattu par SON coequipier"},
		{"la victime elle-meme, deja morte", "A", "une anomalie de donnees, pas une vengeance"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			bilan := Echanges([]domain.KillEvent{
				ev("m1", "A", "K", 0),
				ev("m1", "K", c.tueur, 2000),
			}, equipesArene())

			m := mort(t, bilan, "A", 0)
			if !m.Vengeable {
				t.Fatalf("la mort de A par K devait rester vengeable")
			}
			if m.Vengee {
				t.Fatalf("Vengee = true alors que %s", c.relire)
			}
			if bilan.NbVengees != 0 {
				t.Fatalf("NbVengees = %d, attendu 0", bilan.NbVengees)
			}
		})
	}
}

// TestEchanges_MortSansTueurNEstPasVengeable : une mort que personne ne revendique sort du
// denominateur. La compter comme un echec ferait chuter le taux d'echange d'une equipe pour
// des chutes et des hors-limites.
func TestEchanges_MortSansTueurNEstPasVengeable(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "", 0),
		ev("m1", "B", "B", 1000),
	}, equipesArene())

	if bilan.NbVengeables != 0 {
		t.Fatalf("NbVengeables = %d, attendu 0 : %+v", bilan.NbVengeables, bilan.Morts)
	}
	for _, m := range bilan.Morts {
		if m.Vengeable {
			t.Fatalf("mort %+v marquee vengeable", m)
		}
	}
}

// TestEchanges_EquipesInconnues : sans table d'equipes, personne n'est coequipier de
// personne — la mesure rend zero, elle ne devine pas les camps. Et une mort dont les camps
// sont inconnus n'est pas VENGEABLE : elle sort du denominateur au lieu d'y compter comme
// un echec.
func TestEchanges_EquipesInconnues(t *testing.T) {
	kills := []domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m1", "K", "B", 2000),
	}
	cas := []struct {
		nom            string
		equipes        domain.EquipesParMatch
		vengeablesAttn int
	}{
		{"table absente", nil, 0},
		{"match absent de la table", domain.EquipesParMatch{"autre": {"A": 0, "K": 1}}, 0},
		// A et K sont connus : leur duel reste vengeable. La mort de K par B ne l'est pas,
		// l'equipe de B etant inconnue.
		{"vengeur hors table", domain.EquipesParMatch{"m1": {"A": 0, "K": 1}}, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			bilan := Echanges(kills, c.equipes)
			if bilan.NbVengees != 0 {
				t.Fatalf("NbVengees = %d, attendu 0", bilan.NbVengees)
			}
			if bilan.NbVengeables != c.vengeablesAttn {
				t.Fatalf("NbVengeables = %d, attendu %d : %+v", bilan.NbVengeables, c.vengeablesAttn, bilan.Morts)
			}
		})
	}
}

// TestEchanges_MortParUnCoequipierNEstPasVengeable : un tir ami n'ouvre pas d'echange — il
// n'y a pas d'adversaire a abattre. La mort sort du denominateur.
func TestEchanges_MortParUnCoequipierNEstPasVengeable(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "B", 0),
		ev("m1", "B", "K", 1000),
	}, equipesArene())

	if m := mort(t, bilan, "A", 0); m.Vengeable {
		t.Fatalf("mort de A par son coequipier B marquee vengeable : %+v", m)
	}
	if bilan.NbVengeables != 1 {
		t.Fatalf("NbVengeables = %d, attendu 1 (seule la mort de B par K l'est) : %+v", bilan.NbVengeables, bilan.Morts)
	}
}

// TestEchanges_RienNeTraverseUnMatch : les memes joueurs, la meme horloge, deux matchs. Le
// kill du tueur dans l'autre match ne venge rien.
func TestEchanges_RienNeTraverseUnMatch(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m2", "K", "B", 1000),
	}, equipesArene())

	if m := mort(t, bilan, "A", 0); m.Vengee {
		t.Fatalf("mort de A vengee par un kill d'un AUTRE match : %+v", m)
	}
	if bilan.NbVengees != 0 {
		t.Fatalf("NbVengees = %d, attendu 0", bilan.NbVengees)
	}
}

// TestEchanges_LePremierVengeurCompte : le tueur reapparait et retombe deux fois dans la
// fenetre ; c'est le premier kill qui solde la mort, et lui seul.
func TestEchanges_LePremierVengeurCompte(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m1", "K", "B", 4000),
		ev("m1", "K", "C", 2000),
	}, equipesArene())

	m := mort(t, bilan, "A", 0)
	if m.VengeurXUID != "C" || m.DelaiMs != 2000 {
		t.Fatalf("mort de A = %+v, attendu vengee par C en 2000 ms", m)
	}
	if bilan.NbVengees != 1 {
		t.Fatalf("NbVengees = %d, attendu 1", bilan.NbVengees)
	}
}

// TestEchanges_PairesEtDelaiMoyen : deux echanges du meme couple, delai moyen exact.
func TestEchanges_PairesEtDelaiMoyen(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m1", "K", "C", 1000),
		ev("m1", "A", "K", 10000),
		ev("m1", "K", "C", 13000),
	}, equipesArene())

	if len(bilan.Paires) != 1 {
		t.Fatalf("len(Paires) = %d, attendu 1 : %+v", len(bilan.Paires), bilan.Paires)
	}
	attendu := domain.PaireEchange{VengeurXUID: "C", VengeXUID: "A", Nombre: 2, DelaiMoyenMs: 2000}
	if bilan.Paires[0] != attendu {
		t.Fatalf("Paires[0] = %+v, attendu %+v", bilan.Paires[0], attendu)
	}
}

// TestEchanges_SortieTriee : le parcours des matchs est un parcours de map ; la sortie ne
// doit pas bouger d'un appel a l'autre.
func TestEchanges_SortieTriee(t *testing.T) {
	kills := []domain.KillEvent{
		ev("m2", "B", "K", 500),
		ev("m1", "A", "K", 900),
		ev("m1", "K", "C", 100),
		ev("m2", "A", "L", 200),
	}
	attendu := []struct {
		match   string
		victime string
		instant int64
	}{
		{"m1", "K", 100},
		{"m1", "A", 900},
		{"m2", "A", 200},
		{"m2", "B", 500},
	}
	for essai := 0; essai < 5; essai++ {
		bilan := Echanges(kills, equipesArene())
		if len(bilan.Morts) != len(attendu) {
			t.Fatalf("len(Morts) = %d, attendu %d", len(bilan.Morts), len(attendu))
		}
		for i, a := range attendu {
			m := bilan.Morts[i]
			if m.MatchID != a.match || m.VictimeXUID != a.victime || m.TimeMs != a.instant {
				t.Fatalf("essai %d, Morts[%d] = (%s, %s, %d), attendu (%s, %s, %d)",
					essai, i, m.MatchID, m.VictimeXUID, m.TimeMs, a.match, a.victime, a.instant)
			}
		}
	}
}

// TestEchanges_CouvertureDepuisLeBilan : le bilan alimente directement une Couverture — le
// taux d'echange ne se recalcule pas a la main chez chaque appelant.
func TestEchanges_CouvertureDepuisLeBilan(t *testing.T) {
	bilan := Echanges([]domain.KillEvent{
		ev("m1", "A", "K", 0),
		ev("m1", "K", "C", 1000),
		ev("m1", "B", "K", 20000),
	}, equipesArene())

	if bilan.NbVengeables != 3 {
		t.Fatalf("NbVengeables = %d, attendu 3 : %+v", bilan.NbVengeables, bilan.Morts)
	}
	c := Mesurer(bilan.NbVengees, bilan.NbVengeables, 1)
	if c.Brut != 1 || c.N != 3 {
		t.Fatalf("Couverture = %+v, attendu Brut 1 et N 3", c)
	}
	if !c.EchantillonFaible {
		t.Fatalf("EchantillonFaible = false sur 3 morts, attendu true")
	}
}
