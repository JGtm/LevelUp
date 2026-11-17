package tactical

import "testing"

// Un point vu seulement dans le match evalue et dans un autre n'est PAS candidat pour ce match :
// le point reel ne devient pas candidat parce que le match evalue l'a vu.
func TestConstruireEchantillons_CandidatsHorsDuMatchEvalue(t *testing.T) {
	p, q := Point3{X: 0}, Point3{X: 50}
	vu := func(carte Point3, matchs ...string) []EvenementReapparition {
		var out []EvenementReapparition
		for _, m := range matchs {
			out = append(out, evtTest(m, carte, carte, []Point3{{X: 200}}, nil))
		}
		return out
	}
	// p : m1 et m2 seulement. q : m2, m3, m4. Un troisieme point r, vu par m3 et m4 seulement.
	r := Point3{X: 90}
	evts := append(append(vu(p, "m1", "m2"), vu(q, "m2", "m3", "m4")...), vu(r, "m3", "m4")...)
	cands := CandidatsParCarte(evts)
	if len(cands["c"]) != 3 {
		t.Fatalf("l'ensemble maximal a 3 candidats, obtenu %d", len(cands["c"]))
	}
	echs := ConstruireEchantillons(evts, cands, ReglagesEchantillons{MatchsMinCarte: 1})

	// Cherche, par match et point reel, l'echantillon.
	trouve := func(match string, reel Point3) Echantillon {
		for i, e := range evts {
			if e.MatchID == match && e.Reel == reel {
				return echs[i]
			}
		}
		t.Fatalf("pas d'echantillon pour %s en %v", match, reel)
		return Echantillon{}
	}
	if len(echs) != len(evts) {
		t.Fatalf("un echantillon par evenement attendu, %d pour %d", len(echs), len(evts))
	}
	// m1 : p n'a qu'un AUTRE match (m2) ; q a trois autres ; r deux autres -> {q, r}, reel hors candidats.
	s := trouve("m1", p)
	if len(s.Feats) != 2 || len(s.Pos) != 2 || s.Reel != -1 {
		t.Errorf("m1 : p ne doit pas etre candidat (vu seulement par m1 et m2) : %d candidats, reel %d", len(s.Feats), s.Reel)
	}
	for _, c := range s.Pos {
		if c == p {
			t.Error("m1 : le point reel de m1 figure parmi ses candidats")
		}
	}
	// m3, reel q : p (m1, m2) reste candidat ; q reste candidat (m2, m4) ; r (m4 seul) ne l'est plus.
	s = trouve("m3", q)
	if len(s.Pos) != 2 || s.Pos[0] == r || s.Pos[1] == r || s.Reel < 0 || s.Pos[s.Reel] != q {
		t.Errorf("m3 : candidats %v, reel %d", s.Pos, s.Reel)
	}
}

// La frequence d'un candidat ne compte pas le match evalue.
func TestConstruireEchantillons_FrequenceHorsDuMatchEvalue(t *testing.T) {
	p, q := Point3{X: 0}, Point3{X: 50}
	var evts []EvenementReapparition
	for _, m := range []string{"m1", "m2", "m3", "m4"} {
		evts = append(evts, evtTest(m, p, p, []Point3{{X: 200}}, nil), evtTest(m, p, p, []Point3{{X: 200}}, nil),
			evtTest(m, q, q, []Point3{{X: 200}}, nil))
	}
	echs := ConstruireEchantillons(evts, CandidatsParCarte(evts), ReglagesEchantillons{MatchsMinCarte: 1})
	// p : 8 reapparitions au total dont 2 en m1 -> 6 hors m1 ; log1p(6).
	attendu := Variables(evts[0], Candidat{Pos: p}, 6)[VarFrequence]
	if got := echs[0].Feats[0][VarFrequence]; got != attendu {
		t.Errorf("frequence de p pour m1 = %v, attendu %v (hors m1)", got, attendu)
	}
}

// echantillonsDeuxPlis : deux candidats, la variable 0 les separe ; `reel` dit lequel est reel.
func echantillonDeuxCandidats(pli, reel int) Echantillon {
	return Echantillon{
		Pli: pli, Feats: [][NbVariables]float64{{VarEnnemiProche: 1}, {VarEnnemiProche: 0}},
		Pos: []Point3{{X: 0}, {X: 100}}, Reel: reel, ZoneReelle: -1,
	}
}

// Un poids appris a le signe de la preference observee : positif quand le point reel a la plus
// grande valeur de la variable, negatif quand il a la plus petite.
func TestAjuster_SigneDuGradient(t *testing.T) {
	tout := func(Echantillon) bool { return true }
	positif := []Echantillon{echantillonDeuxCandidats(0, 0), echantillonDeuxCandidats(1, 0)}
	if w := Ajuster(positif, tout, ToutesLesVariables); w[VarEnnemiProche] <= 0 {
		t.Errorf("le reel a la variable la plus haute : poids %v, attendu strictement positif", w[VarEnnemiProche])
	}
	negatif := []Echantillon{echantillonDeuxCandidats(0, 1), echantillonDeuxCandidats(1, 1)}
	if w := Ajuster(negatif, tout, ToutesLesVariables); w[VarEnnemiProche] >= 0 {
		t.Errorf("le reel a la variable la plus basse : poids %v, attendu strictement negatif", w[VarEnnemiProche])
	}
	// Une variable hors du masque reste a zero, meme quand elle separe.
	if w := Ajuster(positif, tout, FrequenceSeule); w[VarEnnemiProche] != 0 {
		t.Errorf("variable hors masque ajustee : %v", w[VarEnnemiProche])
	}
}

// Un echantillon est note par un modele ajuste SANS son pli : les scores valides sont ceux du
// poids appris sur les autres plis, jamais ceux du poids appris avec lui.
func TestScoresValides_ChaqueEchantillonEstNoteSansSonPli(t *testing.T) {
	// Le pli 0 prefere le candidat bas (reel = 1), les autres plis le candidat haut (reel = 0).
	var echs []Echantillon
	for pli := 0; pli < PlisValidation; pli++ {
		reel := 0
		if pli == 0 {
			reel = 1
		}
		for i := 0; i < 3; i++ {
			echs = append(echs, echantillonDeuxCandidats(pli, reel))
		}
	}
	pred := ScoresValides(echs, ToutesLesVariables)
	for i := range echs {
		if len(pred[i]) != len(echs[i].Feats) {
			t.Fatalf("echantillon %d non note", i)
		}
	}
	tout := func(Echantillon) bool { return true }
	complet := Ajuster(echs, tout, ToutesLesVariables)
	for pli := 0; pli < PlisValidation; pli++ {
		sans := Ajuster(echs, func(s Echantillon) bool { return s.Pli != pli }, ToutesLesVariables)
		if sans == complet {
			t.Fatalf("pli %d : le jeu d'essai ne distingue pas l'ajustement sans le pli de l'ajustement complet", pli)
		}
		for i, s := range echs {
			if s.Pli != pli {
				continue
			}
			attendu := s.Scores(sans)
			for j := range attendu {
				if pred[i][j] != attendu[j] {
					t.Fatalf("echantillon %d (pli %d) : score %v, attendu %v (poids hors pli)", i, pli, pred[i], attendu)
				}
			}
		}
	}
}
