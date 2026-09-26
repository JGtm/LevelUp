package tactical

import (
	"errors"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// ─── LE SEUIL N, ET POURQUOI CETTE VALEUR-LA ───────────────────────────────────

// TestCellulesLisiblesMinEstLeSeuilOuLEchelleCesseDEtreDicteeParUneSeuleCellule mesure la
// propriete qui FIXE N, sur la fonction qui la porte (quantile.go) : en dessous, la borne
// haute de la rampe (p95) est une interpolation entre les DEUX plus grandes valeurs, donc
// la cellule la plus extreme decide seule de l'echelle — exactement ce que la doc de
// OrdreP95 dit vouloir eviter (« une seule cellule extreme aplatit toutes les autres si
// elle borne l'echelle »).
//
// N est le PLUS PETIT compte de cellules lisibles pour lequel au moins DEUX cellules
// depassent la borne. Le test verifie les deux sens : la propriete tient a N, et elle ne
// tient pas a N-1 (sans quoi N ne serait pas minimal, et on aurait grossi la grille plus
// que necessaire).
func TestCellulesLisiblesMinEstLeSeuilOuLEchelleCesseDEtreDicteeParUneSeuleCellule(t *testing.T) {
	auDessusDuP95 := func(n int) int {
		valeurs := make([]float64, n)
		for i := range valeurs {
			valeurs[i] = float64(i + 1)
		}
		borne := quantile(valeurs, OrdreP95)
		k := 0
		for _, v := range valeurs {
			if v > borne {
				k++
			}
		}
		return k
	}
	if got := auDessusDuP95(CellulesLisiblesMin); got < 2 {
		t.Fatalf("a %d cellules, %d cellule(s) au-dessus du p95 ; l'echelle est encore dictee par l'extreme", CellulesLisiblesMin, got)
	}
	if got := auDessusDuP95(CellulesLisiblesMin - 1); got >= 2 {
		t.Fatalf("a %d cellules la propriete tient deja : N=%d n'est pas minimal", CellulesLisiblesMin-1, CellulesLisiblesMin)
	}
}

// TestPasAdaptatifsPartentDuPasParDefautEtDoublent verrouille la suite arretee par la
// decision D6 : 0,5 -> 1 -> 2, en partant du pas par defaut.
func TestPasAdaptatifsPartentDuPasParDefautEtDoublent(t *testing.T) {
	if len(PasAdaptatifsM) == 0 || PasAdaptatifsM[0] != PasParDefautM {
		t.Fatalf("la suite doit partir du pas par defaut (%v), recu %v", PasParDefautM, PasAdaptatifsM)
	}
	for i := 1; i < len(PasAdaptatifsM); i++ {
		if PasAdaptatifsM[i] != 2*PasAdaptatifsM[i-1] {
			t.Fatalf("le pas doit DOUBLER : %v apres %v", PasAdaptatifsM[i], PasAdaptatifsM[i-1])
		}
	}
}

// ─── LE CHOIX DU PAS ───────────────────────────────────────────────────────────

// zonesDenses fabrique `zones` amas dont chacun tient dans UNE cellule de 0,5 m et compte
// `PlancherMatchsParCellule` matchs distincts : la carte est lisible des le pas le plus fin.
func zonesDenses(zones int) ([]string, []domain.PositionSample) {
	matchs := []string{"m1", "m2", "m3"}
	points := make([]domain.PositionSample, 0, zones*len(matchs))
	for z := 0; z < zones; z++ {
		for _, m := range matchs {
			points = append(points, domain.PositionSample{MatchID: m, X: 2*float64(z) + 0.1, Y: 0.1})
		}
	}
	return matchs, points
}

// zonesClairsemees fabrique `zones` amas dont les trois matchs tombent dans TROIS cellules
// de 0,5 m differentes, dans DEUX cellules de 1 m, et dans UNE SEULE cellule de 2 m. Aucune
// cellule ne passe le plancher avant le pas de 2 m — c'est la forme d'Illusion apres le
// backfill : les matchs existent, la densite par cellule ne suit pas.
func zonesClairsemees(zones int) ([]string, []domain.PositionSample) {
	matchs := []string{"m1", "m2", "m3"}
	decalages := []float64{0.1, 0.6, 1.1}
	points := make([]domain.PositionSample, 0, zones*len(matchs))
	for z := 0; z < zones; z++ {
		for i, m := range matchs {
			points = append(points, domain.PositionSample{MatchID: m, X: 4*float64(z) + decalages[i], Y: 0.1})
		}
	}
	return matchs, points
}

// essayeurDe rend l'essai standard d'une lecture NON signee : rasteriser les points sur la
// grille proposee, et compter les cellules qui passent le plancher.
func essayeurDe(matchs []string, points []domain.PositionSample) func(Grille) (*Raster, int, error) {
	return func(g Grille) (*Raster, int, error) {
		r, err := Rasterise(g, matchs, points)
		if err != nil {
			return nil, 0, err
		}
		return r, len(r.Cellules()), nil
	}
}

func TestChoisirPasRetientLePlusFinQuandIlSuffit(t *testing.T) {
	matchs, points := zonesDenses(CellulesLisiblesMin)

	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe(matchs, points))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if !lecture.Suffisante {
		t.Fatalf("la carte est lisible au pas le plus fin, la lecture devrait etre suffisante")
	}
	if lecture.PasM() != PasParDefautM {
		t.Fatalf("pas retenu = %v, attendu %v", lecture.PasM(), PasParDefautM)
	}
	if len(lecture.Tentatives) != 1 {
		t.Fatalf("un seul pas devait etre essaye, %d tentative(s) : %+v", len(lecture.Tentatives), lecture.Tentatives)
	}
}

// TestChoisirPasNeChangeRienQuandLePasParDefautSuffit est l'invariant du lot : sur une carte
// deja lisible a 0,5 m, le plan rendu est EXACTEMENT celui d'avant le pas adaptatif —
// memes cellules, memes valeurs, memes bornes.
func TestChoisirPasNeChangeRienQuandLePasParDefautSuffit(t *testing.T) {
	matchs, points := zonesDenses(CellulesLisiblesMin)

	temoin, err := Rasterise(GrilleParDefaut(), matchs, points)
	if err != nil {
		t.Fatalf("Rasterise: %v", err)
	}
	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe(matchs, points))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if !reflect.DeepEqual(lecture.Raster.Cellules(), temoin.Cellules()) {
		t.Fatalf("le plan a change alors que 0,5 m suffisait :\n adaptatif %+v\n temoin    %+v",
			lecture.Raster.Cellules(), temoin.Cellules())
	}
	if lecture.Raster.Bornes() != temoin.Bornes() {
		t.Fatalf("bornes changees : %+v vs %+v", lecture.Raster.Bornes(), temoin.Bornes())
	}
}

func TestChoisirPasDoubleJusquACeQueLaDensiteSuffise(t *testing.T) {
	matchs, points := zonesClairsemees(CellulesLisiblesMin)

	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe(matchs, points))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if !lecture.Suffisante {
		t.Fatalf("le pas de 2 m devait suffire : %+v", lecture.Tentatives)
	}
	if lecture.PasM() != 2 {
		t.Fatalf("pas retenu = %v, attendu 2 m", lecture.PasM())
	}
	if len(lecture.Tentatives) != 3 {
		t.Fatalf("les trois pas devaient etre essayes, recu %+v", lecture.Tentatives)
	}
	if lecture.Tentatives[0].CellulesLisibles != 0 {
		t.Fatalf("au pas de 0,5 m aucune cellule ne devait passer le plancher, recu %d",
			lecture.Tentatives[0].CellulesLisibles)
	}
	if got := len(lecture.Raster.Cellules()); got < CellulesLisiblesMin {
		t.Fatalf("le raster retenu doit porter au moins %d cellules lisibles, recu %d",
			CellulesLisiblesMin, got)
	}
}

// TestChoisirPasLePlancherNeBaisseJamais : le plancher de matchs distincts est le meme a
// tous les pas. Deux matchs par amas ne suffisent a AUCUN pas — la grille grossit, la regle
// de fiabilite ne bouge pas (decision D6 : « abaisser ment »).
func TestChoisirPasLePlancherNeBaisseJamais(t *testing.T) {
	matchs := []string{"m1", "m2"}
	points := make([]domain.PositionSample, 0, 2*CellulesLisiblesMin)
	for z := 0; z < CellulesLisiblesMin; z++ {
		for _, m := range matchs {
			points = append(points, domain.PositionSample{MatchID: m, X: 4*float64(z) + 0.1, Y: 0.1})
		}
	}

	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe(matchs, points))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if lecture.Suffisante {
		t.Fatalf("deux matchs distincts ne passent pas un plancher de %d, quel que soit le pas", PlancherMatchsParCellule)
	}
	if got := len(lecture.Raster.Cellules()); got != 0 {
		t.Fatalf("aucune cellule ne devait etre lisible, recu %d", got)
	}
}

// TestChoisirPasSansAucunPasSuffisantRendLaMeilleureTentative : rien n'est cache. Faute de
// pas suffisant, on garde la tentative la plus FOURNIE, et a egalite la plus FINE — la
// moins agregee des lectures egales.
func TestChoisirPasSansAucunPasSuffisantRendLaMeilleureTentative(t *testing.T) {
	// Trois amas clairsemes : lisibles a 2 m seulement, mais trois cellules < N.
	matchs, points := zonesClairsemees(3)

	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe(matchs, points))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if lecture.Suffisante {
		t.Fatalf("trois cellules ne font pas %d", CellulesLisiblesMin)
	}
	if lecture.PasM() != 2 {
		t.Fatalf("la tentative la plus fournie est celle de 2 m, pas retenu = %v", lecture.PasM())
	}
	if got := len(lecture.Raster.Cellules()); got != 3 {
		t.Fatalf("le raster retenu devait porter les 3 cellules mesurees, recu %d", got)
	}
}

// TestChoisirPasSurUneCarteVideRendLePasLePlusFin : aucune tentative ne rend rien, toutes
// sont a egalite de zero — c'est la plus fine qui est retenue, et rien n'est peint.
func TestChoisirPasSurUneCarteVideRendLePasLePlusFin(t *testing.T) {
	lecture, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, essayeurDe([]string{"m1"}, nil))
	if err != nil {
		t.Fatalf("ChoisirPas: %v", err)
	}
	if lecture.Suffisante {
		t.Fatalf("une carte sans point n'est jamais suffisante")
	}
	if lecture.PasM() != PasParDefautM {
		t.Fatalf("pas retenu = %v, attendu le plus fin (%v)", lecture.PasM(), PasParDefautM)
	}
	if len(lecture.Tentatives) != len(PasAdaptatifsM) {
		t.Fatalf("tous les pas devaient etre essayes, recu %d", len(lecture.Tentatives))
	}
}

func TestChoisirPasRefuseUneSuiteVide(t *testing.T) {
	_, err := ChoisirPas(nil, CellulesLisiblesMin, essayeurDe([]string{"m1"}, nil))
	if !errors.Is(err, ErrAucunPas) {
		t.Fatalf("attendu ErrAucunPas, recu %v", err)
	}
}

func TestChoisirPasPropageLErreurDEssai(t *testing.T) {
	sentinelle := errors.New("essai en echec")
	_, err := ChoisirPas(PasAdaptatifsM, CellulesLisiblesMin, func(Grille) (*Raster, int, error) {
		return nil, 0, sentinelle
	})
	if !errors.Is(err, sentinelle) {
		t.Fatalf("l'erreur d'essai doit remonter telle quelle, recu %v", err)
	}
}

func TestChoisirPasRefuseUnPasInvalide(t *testing.T) {
	_, err := ChoisirPas([]float64{0}, CellulesLisiblesMin, essayeurDe([]string{"m1"}, nil))
	if !errors.Is(err, ErrPasInvalide) {
		t.Fatalf("attendu ErrPasInvalide, recu %v", err)
	}
}

// ─── LE READRESSAGE D'UNE CELLULE D'UNE GRILLE A L'AUTRE ───────────────────────

func TestReadresserRegroupeLesCellulesFinesDansLaGrosse(t *testing.T) {
	fine := grilleOuPanique(t, 0.5)
	grosse := grilleOuPanique(t, 2)

	// Les quatre colonnes 0..3 de la grille de 0,5 m couvrent [0, 2) : elles tombent
	// toutes dans la colonne 0 de la grille de 2 m.
	for col := 0; col < 4; col++ {
		got := Readresser(Cellule{Col: col, Lig: col}, fine, grosse)
		if got != (Cellule{Col: 0, Lig: 0}) {
			t.Fatalf("cellule (%d,%d) readressee en %+v, attendu (0,0)", col, col, got)
		}
	}
	// Cote negatif : l'ancrage est l'ORIGINE DU MONDE, jamais les bornes de la lecture —
	// la colonne -1 de la grille fine couvre [-0,5 ; 0), donc la colonne -1 de la grosse.
	if got := Readresser(Cellule{Col: -1, Lig: -4}, fine, grosse); got != (Cellule{Col: -1, Lig: -1}) {
		t.Fatalf("cellule (-1,-4) readressee en %+v, attendu (-1,-1)", got)
	}
}

func TestReadresserComptesGarderLesOccurrencesEtLesMatchs(t *testing.T) {
	fine := grilleOuPanique(t, 0.5)
	grosse := grilleOuPanique(t, 2)
	comptes := []CompteCellule{
		{Cellule: Cellule{Col: 0, Lig: 0}, MatchID: "m1", Occurrences: 2},
		{Cellule: Cellule{Col: 3, Lig: 1}, MatchID: "m2", Occurrences: 5},
	}

	got := ReadresserComptes(comptes, fine, grosse)
	attendu := []CompteCellule{
		{Cellule: Cellule{Col: 0, Lig: 0}, MatchID: "m1", Occurrences: 2},
		{Cellule: Cellule{Col: 0, Lig: 0}, MatchID: "m2", Occurrences: 5},
	}
	if !reflect.DeepEqual(got, attendu) {
		t.Fatalf("comptes readresses %+v, attendu %+v", got, attendu)
	}
}

func grilleOuPanique(t *testing.T, pasM float64) Grille {
	t.Helper()
	g, err := NouvelleGrille(pasM)
	if err != nil {
		t.Fatalf("NouvelleGrille(%v): %v", pasM, err)
	}
	return g
}
