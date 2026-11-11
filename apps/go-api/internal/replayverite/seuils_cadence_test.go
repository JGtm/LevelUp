package replayverite

import "testing"

// cadencesEssayees : le pas historique du rejeu et le pas de 30 images/s. Chaque seuil temporel du
// banc est une DUREE : elle doit tenir la meme decision sur l'une et l'autre grille.
var cadencesEssayees = []int{100, 33}

// documentAuPas rend le document juste, regrille au pas donne : les temps d'image sont des durees
// divisees par le pas, jamais des images posees en dur.
func documentAuPas(pas int) *Document {
	d := documentJuste()
	d.FrameIntervalMs = pas
	return d
}

// TestImages_ConvertitUneDureeAvecLePasDuDocument : la conversion arrondit vers le bas (une
// tolerance ne s'elargit jamais) ; un document sans pas n'a AUCUN repli : LireDocument le refuse, et un Document
// construit a la main sans pas rend 0 image (jamais une tolerance elargie).
func TestImages_ConvertitUneDureeAvecLePasDuDocument(t *testing.T) {
	cas := []struct{ pas, ms, veut int }{
		{100, 200, 2}, {100, 500, 5}, {100, 100, 1},
		{33, 200, 6}, {33, 500, 15}, {33, 100, 3},
		{0, 500, 0}, // pas absent : aucune image, pas de repli
	}
	for _, c := range cas {
		d := &Document{FrameIntervalMs: c.pas}
		if got := d.images(c.ms); got != c.veut {
			t.Errorf("pas %d ms : images(%d ms) = %d, veut %d", c.pas, c.ms, got, c.veut)
		}
	}
}

// TestSeuils_MemeDureeReelleAuxDeuxCadences : pour chaque seuil, le nombre d'images rendu couvre
// au plus la duree voulue, et l'image suivante la depasse (aucun seuil ne derive d'une cadence a
// l'autre) ; a 100 ms, les valeurs sont celles d'avant la conversion.
func TestSeuils_MemeDureeReelleAuxDeuxCadences(t *testing.T) {
	seuils := map[string]struct {
		ms       int
		historic int
	}{
		"translocation": {toleranceTranslocationMS, 2},
		"vie":           {toleranceVieMS, 5},
		"echantillon":   {toleranceEchantillonMS, 1},
		"mort":          {toleranceMortMS, 2},
		"fin de film":   {margeFinDeFilmMS, 2},
	}
	for nom, s := range seuils {
		for _, pas := range cadencesEssayees {
			n := documentAuPas(pas).images(s.ms)
			if n*pas > s.ms || (n+1)*pas <= s.ms {
				t.Errorf("%s a %d ms : %d images = %d ms pour un seuil de %d ms", nom, pas, n, n*pas, s.ms)
			}
		}
		if got := documentAuPas(100).images(s.ms); got != s.historic {
			t.Errorf("%s a 100 ms : %d images, veut la valeur historique %d", nom, got, s.historic)
		}
	}
}

// TestSaut_ToleranceDeTranslocationEstUneDuree : un saut a la limite de la tolerance d'une
// translocation est exempte, un de plus non, sur les deux grilles.
func TestSaut_ToleranceDeTranslocationEstUneDuree(t *testing.T) {
	for _, pas := range cadencesEssayees {
		tol := documentAuPas(pas).images(toleranceTranslocationMS)
		for _, c := range []struct {
			ecart   int
			exempte int
		}{{tol, 1}, {tol + 1, 0}} {
			d := documentAuPas(pas)
			saut := 11
			d.Tracks[0].Points = []Point{{T: 10, X: 0, Y: 0}, {T: saut, X: 20, Y: 0}, {T: saut + 1, X: 20.01, Y: 0}}
			d.Translocations = []Translocation{{T: saut + c.ecart}}
			b := Noter(d, faitsJustes(), nil)
			if got := b.Violations[ViolSaut].Exemptees; got != c.exempte {
				t.Errorf("pas %d ms, ecart %d images : exemptees = %d, veut %d", pas, c.ecart, got, c.exempte)
			}
		}
	}
}

// TestHorsVie_ToleranceEstUneDuree : un tir a la limite de la tolerance de vie est dans la vie, un
// de plus est hors vie, sur les deux grilles (la piste 512 finit a l'image 100).
func TestHorsVie_ToleranceEstUneDuree(t *testing.T) {
	for _, pas := range cadencesEssayees {
		tol := documentAuPas(pas).images(toleranceVieMS)
		for _, c := range []struct{ ecart, faux int }{{tol, 0}, {tol + 1, 1}} {
			d := documentAuPas(pas)
			d.Shots = append(d.Shots, Action{T: 100 + c.ecart, Slot: ptr(512)})
			b := Noter(d, faitsJustes(), nil)
			if got := b.Violations[ViolHorsVie].Total(); got != c.faux {
				t.Errorf("pas %d ms, ecart %d images : hors vie = %d, veut %d", pas, c.ecart, got, c.faux)
			}
		}
	}
}

// TestMortsSansVie_ToleranceEstUneDuree : une mort du statborg a la limite de la tolerance d'une
// fin de vie est appariee, une image de plus non.
func TestMortsSansVie_ToleranceEstUneDuree(t *testing.T) {
	for _, pas := range cadencesEssayees {
		tol := documentAuPas(pas).images(toleranceMortMS)
		for _, c := range []struct{ ecart, faux int }{{tol, 0}, {tol + 1, 1}} {
			d := documentAuPas(pas)
			d.ScoreTimeline.Players[0].Deaths = serie(Pas{0, 0}, Pas{100 + c.ecart, 1}, Pas{220, 2})
			b := Noter(d, faitsJustes(), nil)
			// 220 est a 70 images de la fin de vie 300 : il est compte dans les deux cas, on isole
			// la mort ecartee de c.ecart de la fin de vie 100.
			if got := b.Violations[ViolMortSansVie].Total() - 1; got != c.faux {
				t.Errorf("pas %d ms, ecart %d images : morts sans vie = %d, veut %d", pas, c.ecart, got, c.faux)
			}
		}
	}
}

// TestFinsDeVie_MargeDeFinDeFilmEstUneDuree : une piste qui finit a la marge de la fin du film finit
// AVEC le film (pas de mort) ; une image de plus avant, c'est une fin de vie.
func TestFinsDeVie_MargeDeFinDeFilmEstUneDuree(t *testing.T) {
	for _, pas := range cadencesEssayees {
		marge := documentAuPas(pas).images(margeFinDeFilmMS)
		for _, c := range []struct{ avantLaFin, fins int }{{marge, 0}, {marge + 1, 1}} {
			d := &Document{FrameCount: 1000, FrameIntervalMs: pas,
				Tracks: []Piste{{Slot: 1, XUID: "111", EndFrame: 1000 - 1 - c.avantLaFin}}}
			b := Noter(d, faitsJustes(), nil)
			if got := finsDeVieNotees(b); got != c.fins {
				t.Errorf("pas %d ms, fin a %d images de la fin : fins de vie = %d, veut %d", pas, c.avantLaFin, got, c.fins)
			}
		}
	}
}

// finsDeVieNotees rend le nombre de fins de vie que O-V2 a comptees (vrais et faux positifs).
func finsDeVieNotees(b Bulletin) int {
	s := b.Scores[ScoreFinsDeVie]
	return s.VP + s.FP
}

// TestTrajet_ToleranceDEchantillonEstUneDuree : le point du passager apparie un echantillon de
// vehicule jusqu'a la tolerance, et pas au-dela, sur les deux grilles ; au-dela, rien n'est juge.
func TestTrajet_ToleranceDEchantillonEstUneDuree(t *testing.T) {
	for _, pas := range cadencesEssayees {
		tol := documentAuPas(pas).images(toleranceEchantillonMS)
		for _, c := range []struct{ ecart, faux int }{{tol, 1}, {tol + 1, 0}} {
			d := documentAuPas(pas)
			d.Tracks[0].Points = []Point{{T: 20, X: 0, Y: 0}}
			d.Vehicles = []Vehicule{{Slot: 700,
				Samples: []Echantillon{{T: 20 + c.ecart, X: rayonTrajetM + 5, Y: 0}},
				Rides:   []Trajet{{T0: 0, T1: 100, Slot: 512}}}}
			exigerViolations(t, Noter(d, faitsJustes(), nil), ViolTrajetLoin, c.faux)
		}
	}
}

// TestLireDocument_RefuseUnPasAbsent : la seule porte des documents lus refuse un artefact sans pas positif ; c'est
// pourquoi la conversion des durees n'a pas de repli.
func TestLireDocument_RefuseUnPasAbsent(t *testing.T) {
	if _, err := LireDocument([]byte(`{"matchId":"m","tracks":[]}`)); err == nil {
		t.Fatal("un artefact sans frameIntervalMs doit etre refuse")
	}
}
