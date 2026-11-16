package tactical

import "testing"

// pisteTest construit une vie de `debut` a `fin` (frames), a 33 ms la frame (la cadence du parc), avec un point au
// debut et un a la fin, a la position (x, 0, 0) puis (x+1, 0, 0).
func pisteTest(xuid string, equipe, debut, fin int, x float64, lue *FinLue) Piste {
	return Piste{
		XUID: xuid, Equipe: equipe, Fin: lue,
		Points: []PointPiste{{T: debut, X: x}, {T: fin, X: x + 1}},
	}
}

func entreeTest(comptes []DebutCompte, pistes ...Piste) EntreeOccupation {
	return EntreeOccupation{MatchID: "m", IntervalleFrameMs: 33, Pistes: pistes, DebutsDeCompte: comptes}
}

// spawnDe rend le SpawnPiste de la piste de rang `i` de l'entree.
func spawnDe(t *testing.T, e EntreeOccupation, i int) SpawnPiste {
	t.Helper()
	s, ok := NouvelIndexVies(e).Spawns()[i]
	if !ok {
		t.Fatalf("piste %d sans spawn", i)
	}
	return s
}

func TestReapparition_FilmLiberation(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 103}),
		pisteTest("a", 0, 150, 300, 50, nil),
		pisteTest("b", 1, 0, 400, 20, nil),
	)
	s := spawnDe(t, e, 1)
	if !s.Reapparition || s.Source != SourceFilm || s.DebutDeManche {
		t.Fatalf("reapparition film attendue, obtenu %+v", s)
	}
	if s.Mort == nil || s.Mort.Frame != 100 || s.Mort.X != 11 {
		t.Errorf("mort = %+v, attendu la fin de la piste precedente", s.Mort)
	}
	if len(s.Instantane) != 1 || s.Instantane[0].XUID != "b" || s.Instantane[0].Equipe != 1 {
		t.Errorf("instantane = %+v, attendu b seul", s.Instantane)
	}
}

func TestReapparition_FilmCompteARebours(t *testing.T) {
	e := entreeTest([]DebutCompte{{XUID: "a", Frame: 98}},
		pisteTest("a", 0, 0, 100, 10, nil),
		pisteTest("a", 0, 150, 300, 50, nil),
	)
	if s := spawnDe(t, e, 1); !s.Reapparition || s.Source != SourceFilm {
		t.Fatalf("un compte qui commence a la fin de la piste prouve la mort, obtenu %+v", s)
	}
}

func TestReapparition_FinDeManche(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonSupprimee, Frame: 100}),
		pisteTest("a", 0, 150, 300, 50, nil),
	)
	s := spawnDe(t, e, 1)
	if s.Reapparition || !s.DebutDeManche || s.Mort != nil || s.Instantane != nil {
		t.Fatalf("debut de manche attendu, ni reapparition ni instantane, obtenu %+v", s)
	}
}

func TestReapparition_ToleranceDeLaPreuve(t *testing.T) {
	cas := []struct {
		nom     string
		frame   int
		attendu bool
	}{
		{"deux frames avant la fin", 98, true},
		{"trois frames avant la fin", 97, false},
		{"six frames apres la fin", 106, true},
		{"sept frames apres la fin", 107, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			e := entreeTest(nil,
				pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: c.frame}),
				pisteTest("a", 0, 150, 300, 50, nil),
			)
			if got := spawnDe(t, e, 1).Reapparition; got != c.attendu {
				t.Errorf("Reapparition = %v, attendu %v", got, c.attendu)
			}
		})
	}
}

// Un document sans aucune lecture de fin de vie du film n'a aucune reapparition, quel que soit
// l'ecart entre deux pistes : il n'y a pas de repli sur l'ecart.
func TestReapparition_SansLectureDuFilmAucuneReapparition(t *testing.T) {
	for _, debut := range []int{100 + 118, 100 + 122, 100 + 454, 100 + 458} {
		e := entreeTest(nil,
			pisteTest("a", 0, 0, 100, 10, nil),
			pisteTest("a", 0, debut, debut+100, 50, nil),
		)
		if LectureDuFilm(e) {
			t.Fatal("l'entree ne porte aucune lecture du film")
		}
		s := spawnDe(t, e, 1)
		if s.Reapparition || s.DebutDeManche || s.Mort != nil || s.Instantane != nil {
			t.Errorf("debut a la frame %d : aucune reapparition attendue sans lecture du film, obtenu %+v", debut, s)
		}
	}
}

func TestLectureDuFilm(t *testing.T) {
	sans := entreeTest(nil, pisteTest("a", 0, 0, 100, 10, nil))
	if LectureDuFilm(sans) {
		t.Error("aucune fin lue, aucun compte a rebours")
	}
	if !LectureDuFilm(entreeTest(nil, pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}))) {
		t.Error("une fin lue est une lecture du film")
	}
	if !LectureDuFilm(entreeTest([]DebutCompte{{XUID: "a", Frame: 100}}, pisteTest("a", 0, 0, 100, 10, nil))) {
		t.Error("un debut de compte a rebours rattache a un xuid est une lecture du film")
	}
	if LectureDuFilm(entreeTest([]DebutCompte{{XUID: "", Frame: 100}}, pisteTest("a", 0, 0, 100, 10, nil))) {
		t.Error("un compte a rebours sans xuid ne prouve rien")
	}
}

// Un document qui porte une lecture du film, meme pour un seul joueur : une piste sans preuve n'y est
// pas une reapparition.
func TestReapparition_CasMixte(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", 0, 150, 300, 50, nil),
		pisteTest("b", 1, 0, 100, 20, nil),
		pisteTest("b", 1, 150, 300, 60, nil),
	)
	if s := spawnDe(t, e, 1); !s.Reapparition || s.Source != SourceFilm {
		t.Errorf("a : reapparition film attendue, obtenu %+v", s)
	}
	if s := spawnDe(t, e, 3); s.Reapparition || s.DebutDeManche {
		t.Errorf("b : aucune preuve, le repli est interdit, obtenu %+v", s)
	}
}

func TestReapparition_LaMortEmporteSurLaFinDeManche(t *testing.T) {
	e := entreeTest([]DebutCompte{{XUID: "a", Frame: 100}},
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonSupprimee, Frame: 100}),
		pisteTest("a", 0, 150, 300, 50, nil),
	)
	if s := spawnDe(t, e, 1); !s.Reapparition || s.DebutDeManche {
		t.Fatalf("un compte a rebours prouve la mort, obtenu %+v", s)
	}
}

func TestReapparition_PremiereVieJamais(t *testing.T) {
	e := entreeTest(nil, pisteTest("a", 0, 0, 100, 10, nil))
	if s := spawnDe(t, e, 0); s.Reapparition || s.DebutDeManche {
		t.Fatalf("la premiere vie n'est pas une reapparition, obtenu %+v", s)
	}
}

func TestInstantane_Regles(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", 0, 150, 300, 50, nil),
		pisteTest("vivant", 1, 0, 400, 20, nil),
		pisteTest("mort", 1, 0, 120, 30, nil),
		pisteTest("simultane", 0, 148, 400, 40, nil),
		pisteTest("tard", 0, 152, 400, 45, nil),
		pisteTest("", 1, 0, 400, 70, nil),
	)
	s := spawnDe(t, e, 1)
	noms := map[string]bool{}
	for _, v := range s.Instantane {
		noms[v.XUID] = true
	}
	// "simultane" (2 frames = 200 ms) et "tard" (200 ms) commencent dans la meme seconde ;
	// "mort" est finie a la frame 150 ; le sujet est exclu ; la vie anonyme compte.
	if len(s.Instantane) != 2 || !noms["vivant"] || !noms[""] {
		t.Fatalf("instantane = %+v, attendu vivant et la vie anonyme", s.Instantane)
	}
}

func TestInstantane_EquipeInconnueSansInstantane(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", EquipeInconnue, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", EquipeInconnue, 150, 300, 50, nil),
		pisteTest("b", 1, 0, 400, 20, nil),
	)
	s := spawnDe(t, e, 1)
	if !s.Reapparition || s.Instantane != nil {
		t.Fatalf("reapparition sans instantane attendue, obtenu %+v", s)
	}
}

func TestInstantane_Deterministe(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 0, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", 0, 150, 300, 50, nil),
		pisteTest("z", 1, 0, 400, 20, nil),
		pisteTest("b", 1, 0, 400, 25, nil),
	)
	premier := spawnDe(t, e, 1).Instantane
	for i := 0; i < 20; i++ {
		autre := spawnDe(t, e, 1).Instantane
		if len(autre) != len(premier) || autre[0] != premier[0] || autre[1] != premier[1] {
			t.Fatalf("instantane non deterministe : %+v puis %+v", premier, autre)
		}
	}
	if premier[0].XUID != "b" {
		t.Errorf("ordre par xuid attendu, obtenu %+v", premier)
	}
}

func TestOccupation_PorteLEquipeEtLeSpawn(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 3, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", 3, 230, 380, 50, nil),
	)
	joueurs := Occupation(GrilleParDefaut(), e, PasOccupationMs)
	if len(joueurs) != 1 || joueurs[0].Equipe != 3 {
		t.Fatalf("equipe du joueur = %+v", joueurs)
	}
	sp := joueurs[0].Spawns
	if len(sp) != 2 || !sp[0].PremiereVie || !sp[1].Reapparition {
		t.Fatalf("spawns = %+v", sp)
	}
}

func TestJoindreDebutsDeCompte(t *testing.T) {
	debuts, sans := JoindreDebutsDeCompte(
		[]CompteBrut{{IndexFilm: 1, Debut: 100}, {IndexFilm: 2, Debut: 200}, {IndexFilm: 9, Debut: 300}},
		map[int]string{1: "a", 2: "b"},
	)
	if len(debuts) != 2 || debuts[0] != (DebutCompte{XUID: "a", Frame: 100}) || sans != 1 {
		t.Fatalf("debuts = %+v, sans xuid = %d", debuts, sans)
	}
}

// Une vie sans equipe lue reprend celle de la premiere vie nommee du meme joueur : sa
// reapparition porte un instantane, et ses autres joueurs sont partages en equipiers et
// adversaires de cette equipe.
func TestInstantane_EquipeReprise_DeLaPremiereVieNommee(t *testing.T) {
	e := entreeTest(nil,
		pisteTest("a", 2, 0, 100, 10, &FinLue{Raison: RaisonLiberee, Frame: 100}),
		pisteTest("a", EquipeInconnue, 150, 300, 50, nil),
		pisteTest("b", 1, 0, 400, 20, nil),
	)
	s := spawnDe(t, e, 1)
	if !s.Reapparition || len(s.Instantane) != 1 || s.Instantane[0].XUID != "b" {
		t.Fatalf("la vie sans equipe reprend celle de la vie precedente : instantane attendu, obtenu %+v", s)
	}
}

func TestEquipeDuJoueur(t *testing.T) {
	ix := NouvelIndexVies(entreeTest(nil,
		pisteTest("a", EquipeInconnue, 0, 100, 10, nil),
		pisteTest("a", 2, 150, 250, 20, nil),
		pisteTest("a", 3, 300, 400, 30, nil),
		pisteTest("sans", EquipeInconnue, 0, 100, 10, nil),
	))
	if got := ix.EquipeDuJoueur("a"); got != 2 {
		t.Errorf("equipe de a = %d, attendu 2 (la premiere vie qui en porte une)", got)
	}
	if got := ix.EquipeDuJoueur("sans"); got != EquipeInconnue {
		t.Errorf("equipe de sans = %d, attendu EquipeInconnue", got)
	}
	if got := ix.EquipeDuJoueur("absent"); got != EquipeInconnue {
		t.Errorf("joueur inconnu : %d, attendu EquipeInconnue", got)
	}
}
