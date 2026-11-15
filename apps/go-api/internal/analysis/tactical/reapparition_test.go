package tactical

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func evtTest(match string, reel, mort Point3, ennemis, equipiers []Point3) EvenementReapparition {
	return EvenementReapparition{MatchID: match, CarteID: "c", Reel: reel, Mort: mort, Ennemis: ennemis, Equipiers: equipiers}
}

func TestPointsCandidats_DeuxMatchsDistinctsEtOrdre(t *testing.T) {
	a, b, seul := Point3{0, 0, 0}, Point3{10, 0, 0}, Point3{20, 0, 0}
	evts := []EvenementReapparition{
		evtTest("m1", a, a, nil, nil), evtTest("m1", a, a, nil, nil), evtTest("m2", a, a, nil, nil),
		evtTest("m1", b, b, nil, nil), evtTest("m2", b, b, nil, nil),
		evtTest("m1", seul, seul, nil, nil), evtTest("m1", seul, seul, nil, nil),
	}
	cs := PointsCandidats(evts)
	if len(cs) != 2 {
		t.Fatalf("2 candidats attendus (un point vu dans un seul match n'en est pas un), obtenu %d", len(cs))
	}
	if cs[0].Pos != a || cs[0].Total != 3 || cs[1].Pos != b {
		t.Errorf("ordre attendu par frequence : %+v", cs)
	}
}

func TestPointsCandidats_MailleDemiMetreEtMetreEnHauteur(t *testing.T) {
	if CleDe(Point3{0.1, 0, 0}) != CleDe(Point3{0.2, 0, 0.4}) {
		t.Error("0,1 m et 0,2 m sont une meme case de 0,5 m ; 0,4 m de hauteur est le meme metre")
	}
	if CleDe(Point3{0, 0, 0}) == CleDe(Point3{0, 0, 1}) {
		t.Error("deux etages distincts d'un metre ne se confondent pas")
	}
}

func TestVariables_CasConstruit(t *testing.T) {
	c := Candidat{Pos: Point3{0, 0, 0}}
	e := evtTest("m", c.Pos, Point3{30, 0, 0},
		[]Point3{{10, 0, 0}, {0, 30, 0}}, []Point3{{0, 0, 5}})
	v := Variables(e, c, 3, nil)
	attendu := [NbVariables]float64{
		1,                      // ennemi le plus proche : 10 m
		1,                      // un seul ennemi a moins de 20 m
		0.5,                    // equipier a 5 m
		3,                      // mort a 30 m
		math.Log1p(3),          // frequence
		math.Hypot(5, 15) / 10, // barycentre (5, 15, 0)
		0,                      // pas de geometrie
	}
	for j := range v {
		if math.Abs(v[j]-attendu[j]) > 1e-9 {
			t.Errorf("%s = %v, attendu %v", NomsVariables[j], v[j], attendu[j])
		}
	}
}

func TestVariables_PlafondsEtAbsences(t *testing.T) {
	c := Candidat{Pos: Point3{}}
	v := Variables(evtTest("m", c.Pos, Point3{500, 0, 0}, []Point3{{300, 0, 0}}, nil), c, 0, nil)
	if v[VarEnnemiProche] != DistanceMaxM/DizaineM || v[VarEquipierProche] != DistanceMaxM/DizaineM {
		t.Errorf("plafond de 40 m attendu : %v", v)
	}
	if v[VarDistanceMort] != DistanceMaxMortM/DizaineM || v[VarBarycentreEnnemi] != DistanceMaxMortM/DizaineM {
		t.Errorf("plafond de 60 m attendu : %v", v)
	}
}

func TestVariables_Visibles(t *testing.T) {
	c := Candidat{Pos: Point3{}}
	e := evtTest("m", c.Pos, c.Pos, []Point3{{10, 0, 0}, {20, 0, 0}, {50, 0, 0}}, nil)
	vue := func(en, _ Point3) bool { return en.X != 20 }
	if got := Variables(e, c, 0, vue)[VarVisibles]; got != 1 {
		t.Errorf("seul l'ennemi a 10 m voit le point (20 m bloque, 50 m hors portee), obtenu %v", got)
	}
}

func TestProbabilites(t *testing.T) {
	p := Probabilites([]float64{0, 0, math.Log(2)})
	if math.Abs(p[0]+p[1]+p[2]-1) > 1e-12 || math.Abs(p[2]-0.5) > 1e-12 {
		t.Errorf("softmax = %v", p)
	}
	if Probabilites(nil) != nil {
		t.Error("liste vide : nil")
	}
}

// echantillonsSynthetiques : le point reel est toujours le plus loin de l'ennemi.
func echantillonsSynthetiques() []Echantillon {
	var evts []EvenementReapparition
	pts := []Point3{{0, 0, 0}, {20, 0, 0}, {40, 0, 0}, {60, 0, 0}}
	for m := 0; m < 40; m++ {
		id := string(rune('a'+m%26)) + string(rune('A'+m/26))
		for _, p := range pts {
			evts = append(evts, evtTest(id, p, Point3{500, 0, 0}, []Point3{{-5, 0, 0}}, nil))
		}
	}
	return ConstruireEchantillons(evts, CandidatsParCarte(evts), ReglagesEchantillons{MatchsMinCarte: 8})
}

func TestAjuster_RetrouveLaPreferenceEtEstDeterministe(t *testing.T) {
	echs := echantillonsSynthetiques()
	if len(echs) == 0 {
		t.Fatal("aucun echantillon")
	}
	tout := func(Echantillon) bool { return true }
	w1 := Ajuster(echs, tout, ToutesLesVariables)
	w2 := Ajuster(echs, tout, ToutesLesVariables)
	if w1 != w2 {
		t.Fatalf("ajustement non deterministe : %v puis %v", w1, w2)
	}
	if w1[VarVisibles] != 0 {
		t.Errorf("la septieme variable est hors masque, poids %v", w1[VarVisibles])
	}
	// Les quatre points sont reels a egalite : aucune variable ne doit separer, la
	// vraisemblance est deja maximale a poids nuls pour une distribution uniforme.
	m := Evaluer(echs, scoresDe(echs, w1))
	if m.N != len(echs) || m.Top3 != 100 {
		t.Errorf("quatre candidats a egalite : tous dans les trois premiers, obtenu %+v", m)
	}
}

func scoresDe(echs []Echantillon, w [NbVariables]float64) [][]float64 {
	out := make([][]float64, len(echs))
	for i, s := range echs {
		out[i] = s.Scores(w)
	}
	return out
}

func TestEvaluer_CasConstruit(t *testing.T) {
	pos := []Point3{{0, 0, 0}, {5, 0, 0}, {100, 0, 0}, {200, 0, 0}}
	zones := []int{0, 0, 1, 2}
	s := Echantillon{
		Feats: make([][NbVariables]float64, 4), Pos: pos, Zones: zones,
		Reel: 1, Au: pos[1], ZoneReelle: 0,
	}
	// Le reel (indice 1) est second au score ; le premier est a 5 m, donc dans le rayon.
	m := Evaluer([]Echantillon{s}, [][]float64{{3, 2, 1, 0}})
	if m.Top1 != 0 || m.Top3 != 100 || m.Rayon1 != 100 || m.Rayon3 != 100 {
		t.Errorf("metriques exactes = %+v", m)
	}
	if m.Hasard1 != 25 || m.Hasard3 != 75 {
		t.Errorf("hasard = %v / %v", m.Hasard1, m.Hasard3)
	}
	// Zone 0 : elle porte les deux meilleurs scores, donc la plus probable.
	if m.Zone1 != 100 || m.ZonesMesurees != 1 || math.Abs(m.HasardZone1-100.0/3) > 1e-9 {
		t.Errorf("zones = %+v", m)
	}
}

func TestEvaluer_EgalitesEtHorsCandidats(t *testing.T) {
	pos := []Point3{{0, 0, 0}, {100, 0, 0}}
	egal := Echantillon{Feats: make([][NbVariables]float64, 2), Pos: pos, Reel: 0, Au: pos[0], ZoneReelle: -1}
	hors := Echantillon{Feats: make([][NbVariables]float64, 2), Pos: pos, Reel: -1, Au: Point3{500, 0, 0}, ZoneReelle: -1}
	m := Evaluer([]Echantillon{egal, hors}, [][]float64{{1, 1}, {1, 0}})
	if m.HorsCandidats != 1 || m.N != 2 {
		t.Fatalf("comptes = %+v", m)
	}
	if m.Top1 != 25 { // egalite : 0,5 pour le premier echantillon, 0 pour l'autre, sur 2
		t.Errorf("Top1 = %v, attendu 25", m.Top1)
	}
}

func TestPliDuMatch_StableEtBorne(t *testing.T) {
	for _, id := range []string{"a", "match-2", "zz"} {
		p := PliDuMatch(id, PlisValidation)
		if p < 0 || p >= PlisValidation || p != PliDuMatch(id, PlisValidation) {
			t.Errorf("pli %d invalide pour %q", p, id)
		}
	}
}

func TestObstacles_Voit(t *testing.T) {
	mur := Boite{X0: 9, Y0: -5, X1: 11, Y1: 5, ZBas: 0, ZHaut: 4}
	o := NouveauxObstacles([]Boite{mur})
	a, b := Point3{0, 0, 0}, Point3{20, 0, 0}
	if o.Voit(a, b) {
		t.Error("un mur de 4 m entre les deux bloque la vue")
	}
	if !o.Voit(a, Point3{0, 20, 0}) {
		t.Error("hors du mur, la vue est libre")
	}
	haut := NouveauxObstacles([]Boite{{X0: 9, Y0: -5, X1: 11, Y1: 5, ZBas: 3, ZHaut: 4}})
	if !haut.Voit(a, b) {
		t.Error("une boite au-dessus de la tete ne bloque pas")
	}
	// Une boite qui contient une extremite est ignoree (emprise alignee d'une piece tournee).
	enferme := NouveauxObstacles([]Boite{{X0: -1, Y0: -1, X1: 1, Y1: 1, ZBas: -1, ZHaut: 3}})
	if !enferme.Voit(a, b) {
		t.Error("une boite qui enferme le point de depart est ignoree")
	}
	// Un mur plus bas que la cible : sous la ligne de vue (yeux 1,6 m, cible 1 m).
	bas := NouveauxObstacles([]Boite{{X0: 9, Y0: -5, X1: 11, Y1: 5, ZBas: 0, ZHaut: 0.5}})
	if !bas.Voit(a, b) {
		t.Error("un muret de 0,5 m ne bloque pas la ligne des yeux au torse")
	}
	if o.NbBoites() != 1 || NouveauxObstacles([]Boite{{X0: 1, X1: 1}}).NbBoites() != 0 {
		t.Error("une emprise vide n'est pas une boite")
	}
}

func TestPoidsNommesEtPoidsDeFamille_AllerRetour(t *testing.T) {
	w := [NbVariables]float64{1, -2, 3, 0.5, -0.25, 0, 0}
	f := domain.PoidsFamille{Poids: PoidsNommes(w)}
	if got := PoidsDeFamille(f); got != w {
		t.Errorf("aller-retour = %v, attendu %v", got, w)
	}
	// Un nom inconnu est ignore, une variable absente vaut zero.
	f.Poids["variable_disparue"] = 9
	delete(f.Poids, NomsVariables[VarFrequence])
	got := PoidsDeFamille(f)
	if got[VarFrequence] != 0 || got[VarEnnemiProche] != 1 {
		t.Errorf("poids = %v", got)
	}
}

func TestPoidsPourFamille_ChoixEtRepliSurTous(t *testing.T) {
	propre := [NbVariables]float64{VarEnnemiProche: 2}
	general := [NbVariables]float64{VarEnnemiProche: 1}
	fichier := domain.PoidsReapparition{Familles: map[string]domain.PoidsFamille{
		"assassin":                {Poids: PoidsNommes(propre)},
		"vide":                    {}, // famille sans poids propres
		domain.FamilleToutesModes: {Poids: PoidsNommes(general)},
	}}
	cas := []struct {
		nom, famille, attendu string
		w                     [NbVariables]float64
	}{
		{"famille ajustee", "assassin", "assassin", propre},
		{"famille inconnue du fichier", "drapeau", domain.FamilleToutesModes, general},
		{"famille sans poids propres", "vide", domain.FamilleToutesModes, general},
		{"match sans famille", "", domain.FamilleToutesModes, general},
	}
	for _, c := range cas {
		nom, w, ok := PoidsPourFamille(fichier, c.famille)
		if !ok || nom != c.attendu || w != c.w {
			t.Errorf("%s : (%q, %v, %v), attendu (%q, %v, true)", c.nom, nom, w, ok, c.attendu, c.w)
		}
	}
	// Fichier sans « tous » : une famille inconnue n'a pas de modele.
	sansTous := domain.PoidsReapparition{Familles: map[string]domain.PoidsFamille{"assassin": {Poids: PoidsNommes(propre)}}}
	if _, _, ok := PoidsPourFamille(sansTous, "drapeau"); ok {
		t.Error("ni la famille ni « tous » : pas de modele")
	}
	// Fichier absent (valeur zero) : pas de modele, jamais de panique.
	if _, _, ok := PoidsPourFamille(domain.PoidsReapparition{}, "assassin"); ok {
		t.Error("fichier vide : pas de modele")
	}
}
