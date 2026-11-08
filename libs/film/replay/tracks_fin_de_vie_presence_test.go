package replay

// tracks_fin_de_vie_presence_test.go — LES PROPRIETES DE LA FIN DE VIE D UN JOUEUR PRESENT A LA FIN,
// et la mutation qui fait rougir chacune.
//
// LA FIGURE est celle de tracks_fin_de_vie_test.go : un slot replique de 1 s a 4 s puis de 10 s a
// 12 s, les deux vies du joueur 111 ; le film compte 200 frames, sa derniere frame est la 199. La
// derniere vie finit a son dernier point (frame 110) tant que rien ne la prolonge.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/types"
)

const finDuFilm = finFrames - 1

func bp(v bool) *bool { return &v }
func ip(v int) *int   { return &v }

// presentAvecMorts : la ligne du tableau d un joueur que l API declare present a la fin.
func presentAvecMorts(xuid string, morts int) Participant {
	return Participant{ID: xuid, PresentAtCompletion: bp(true), LeftInProgress: bp(false), Deaths: ip(morts)}
}

func nbRepli(fb *fallback.Compteur, nom fallback.Nom) int { return fb.Compte(nom) }

// TestLeJoueurPresentALaFinResteJusquAuBoutDuFilm — LA REGLE. La derniere vie finit a la derniere
// frame du film, aucun point n est ajoute, la premiere vie (une vie close par un trou) ne change pas.
//
// MUTATION JOUEE : `finParPresence` rend `derniere` -> EndFrame 110, ce test rouge.
func TestLeJoueurPresentALaFinResteJusquAuBoutDuFilm(t *testing.T) {
	tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), nil, calageNul(),
		[]Participant{presentAvecMorts("111", 0)})
	if tracks[1].EndFrame != finDuFilm {
		t.Errorf("EndFrame = %d, attendu %d (la derniere frame du film)", tracks[1].EndFrame, finDuFilm)
	}
	if pts := tracks[1].Points; pts[len(pts)-1].T != 110 {
		t.Errorf("dernier point a T=%d : aucun point ne se fabrique, attendu 110", pts[len(pts)-1].T)
	}
	if tracks[0].EndFrame != 30 {
		t.Errorf("la premiere vie n est pas la derniere du joueur : EndFrame = %d, attendu 30", tracks[0].EndFrame)
	}
	if n := nbRepli(fb, fallback.NomFinDeVieSansTableauAPI) + nbRepli(fb, fallback.NomFinDeVieMortManquante); n != 0 {
		t.Errorf("%d repli(s) compte(s), attendu 0 : le tableau decide", n)
	}
}

// TestLeJoueurPartiResteAuDernierPoint — « a quitte » se lit sur l API : present faux, ou parti en cours, la vie reste
// au dernier point et rien ne se compte (c est la fin que le film montre).
func TestLeJoueurPartiResteAuDernierPoint(t *testing.T) {
	for nom, p := range map[string]Participant{
		"absent a la fin": {ID: "111", PresentAtCompletion: bp(false), LeftInProgress: bp(false), Deaths: ip(0)},
		"parti en cours":  {ID: "111", PresentAtCompletion: bp(true), LeftInProgress: bp(true), Deaths: ip(0)},
	} {
		tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), nil, calageNul(), []Participant{p})
		if tracks[1].EndFrame != 110 {
			t.Errorf("%s : EndFrame = %d, attendu 110", nom, tracks[1].EndFrame)
		}
		if n := nbRepli(fb, fallback.NomFinDeVieSansTableauAPI); n != 0 {
			t.Errorf("%s : %d repli sans tableau, attendu 0 : l API a repondu", nom, n)
		}
	}
}

// TestSansTableauLaFinResteAuDernierPointEtLeRepliSeCompte : un drapeau non declare n est ni vrai ni
// faux. Participant introuvable, chaque champ nil tour a tour, vie sans xuid : dernier point, un
// repli par vie.
//
// MUTATION JOUEE : traiter `left == nil` comme faux -> la ligne « left nil » etend, ce test rouge.
func TestSansTableauLaFinResteAuDernierPointEtLeRepliSeCompte(t *testing.T) {
	for nom, tableau := range map[string][]Participant{
		"tableau vide":       nil,
		"participant absent": {presentAvecMorts("222", 0)},
		"present nil":        {{ID: "111", LeftInProgress: bp(false), Deaths: ip(0)}},
		"left nil":           {{ID: "111", PresentAtCompletion: bp(true), Deaths: ip(0)}},
		"deaths nil":         {{ID: "111", PresentAtCompletion: bp(true), LeftInProgress: bp(false)}},
	} {
		tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), nil, calageNul(), tableau)
		if tracks[1].EndFrame != 110 {
			t.Errorf("%s : EndFrame = %d, attendu 110", nom, tracks[1].EndFrame)
		}
		if n := nbRepli(fb, fallback.NomFinDeVieSansTableauAPI); n != 1 {
			t.Errorf("%s : repli sans tableau = %d, attendu 1", nom, n)
		}
	}
	// Une vie sans xuid que le film laisse courir jusqu au bout (cause film_end) se compte aussi ; la
	// premiere, close par une mort, non.
	tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(0), nil, calageNul(), []Participant{presentAvecMorts("111", 0)})
	if tracks[1].EndFrame != 110 || nbRepli(fb, fallback.NomFinDeVieSansTableauAPI) != 1 {
		t.Errorf("vie sans xuid : EndFrame %d, repli %d, attendu 110 et 1", tracks[1].EndFrame,
			nbRepli(fb, fallback.NomFinDeVieSansTableauAPI))
	}
}

// TestUneMortManquanteInterditLaProlongation : l API compte plus de morts que le film n en ecrit.
// Le film ecrit une mort de 111 (6 s, qui clot la premiere vie) : l API en compte deux, une manque —
// la derniere vie reste au dernier point et le cas se compte. A egalite, la prolongation a lieu.
//
// MUTATION JOUEE : retirer la garde `*p.deaths > c.mortsDuFilm[l.xuid]` -> EndFrame 199, ce test rouge.
func TestUneMortManquanteInterditLaProlongation(t *testing.T) {
	morts := []types.Death{mort(111, 6_000)}
	tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), morts, calageNul(),
		[]Participant{presentAvecMorts("111", 2)})
	if tracks[1].EndFrame != 110 {
		t.Errorf("EndFrame = %d, attendu 110 : l API compte une mort que le film n ecrit pas", tracks[1].EndFrame)
	}
	if n := nbRepli(fb, fallback.NomFinDeVieMortManquante); n != 1 {
		t.Errorf("repli mort manquante = %d, attendu 1", n)
	}
	tracks, fb = pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), morts, calageNul(),
		[]Participant{presentAvecMorts("111", 1)})
	if tracks[1].EndFrame != finDuFilm || nbRepli(fb, fallback.NomFinDeVieMortManquante) != 0 {
		t.Errorf("a egalite : EndFrame %d, repli %d, attendu %d et 0", tracks[1].EndFrame,
			nbRepli(fb, fallback.NomFinDeVieMortManquante), finDuFilm)
	}
}

// TestUneMortEcriteApresLOuvertureDecideSeule : une mort ecrite du joueur posterieure a l ouverture de
// sa derniere vie, dans la plage ecrite (11,5 s -> frame 105 <= dernier point 110), ne se prolonge pas
// par la mort — et la presence n y change rien : le joueur est mort, il n est pas fige vivant.
func TestUneMortEcriteApresLOuvertureDecideSeule(t *testing.T) {
	tracks, _ := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 11_500)}, calageNul(),
		[]Participant{presentAvecMorts("111", 1)})
	if tracks[1].EndFrame != 110 {
		t.Errorf("EndFrame = %d, attendu 110 : la mort de 11,5 s clot la vie", tracks[1].EndFrame)
	}
}

// TestLaProlongationNeFranchitPasLaVieSuivanteDuSlot : un autre joueur ouvre une vie sur le meme slot a
// 15 s (frame 140) : la fin est la frame 139, STRICTEMENT avant, jamais le bout du film.
//
// MUTATION JOUEE : borner a `frameDe(...)` au lieu de `frameDe(...)-1` -> EndFrame 140, ce test rouge.
func TestLaProlongationNeFranchitPasLaVieSuivanteDuSlot(t *testing.T) {
	vies := append(deuxViesDuMemeJoueur(111),
		lifeSpan{slot: 100, from: 15_000_000, to: 16_000_000, xuid: 222, cause: CauseVieFinFilm})
	tracks, _ := pisteDeFinAvecTableau(t, vies, nil, calageNul(), []Participant{presentAvecMorts("111", 0)})
	if tracks[1].EndFrame != 139 {
		t.Errorf("EndFrame = %d, attendu 139 (la frame qui precede l ouverture a 140)", tracks[1].EndFrame)
	}
}

// TestUneVieQuiToucheLeBoutDuFilmNeCompteRien : le dernier point est la derniere frame, rien a decider.
func TestUneVieQuiToucheLeBoutDuFilmNeCompteRien(t *testing.T) {
	vies := deuxViesDuMemeJoueur(111)
	fb := fallback.NouveauCompteur()
	h := replayClock{origin: finOrigineUS, step: finPasUS, frames: 111}
	tracks, _ := decimateTracks(positionsDeuxVies(), decoupeDesTraces{
		origin: finOrigineUS, step: finPasUS, minPoints: 1, vies: vies, fb: fb,
		fin: nouvelleClotureParMort(nil, calageNul(), vies, nil, h),
	})
	if tracks[1].EndFrame != 110 || nbRepli(fb, fallback.NomFinDeVieSansTableauAPI) != 0 {
		t.Errorf("EndFrame %d, repli %d, attendu 110 et 0", tracks[1].EndFrame,
			nbRepli(fb, fallback.NomFinDeVieSansTableauAPI))
	}
}

// TestSansCalageUnJoueurSansMortProlongeEtCeluiQuiEnAEcritUneNeSeDecidePas : sans calage, aucune mort
// ne se date. Le joueur dont le film n ecrit aucune mort n a rien a clore, il est prolonge ; celui qui
// en a ecrit une ne se decide pas (le repli du calage le compte deja).
func TestSansCalageUnJoueurSansMortProlongeEtCeluiQuiEnAEcritUneNeSeDecidePas(t *testing.T) {
	tracks, _ := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), nil, nil, []Participant{presentAvecMorts("111", 0)})
	if tracks[1].EndFrame != finDuFilm {
		t.Errorf("sans mort ni calage : EndFrame = %d, attendu %d", tracks[1].EndFrame, finDuFilm)
	}
	tracks, fb := pisteDeFinAvecTableau(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 6_000)}, nil,
		[]Participant{presentAvecMorts("111", 1)})
	if tracks[1].EndFrame != 110 {
		t.Errorf("une mort non datable : EndFrame = %d, attendu 110", tracks[1].EndFrame)
	}
	if n := nbRepli(fb, fallback.NomFinDeVieSansTableauAPI) + nbRepli(fb, fallback.NomFinDeVieMortManquante); n != 0 {
		t.Errorf("%d repli(s) du tableau compte(s), attendu 0 : le calage inconnu decide", n)
	}
}

// TestUnBotNeSeJointPasAuTableauParXUID : une ligne `bid(N.0)` n entre pas dans l index.
func TestUnBotNeSeJointPasAuTableauParXUID(t *testing.T) {
	m := presencesDuTableau([]Participant{{ID: "bid(7.0)", PresentAtCompletion: bp(true)}, {ID: "", Deaths: ip(1)},
		presentAvecMorts("111", 3)})
	if len(m) != 1 || m[111].deaths == nil || *m[111].deaths != 3 {
		t.Errorf("index = %+v, attendu le seul xuid 111", m)
	}
}
