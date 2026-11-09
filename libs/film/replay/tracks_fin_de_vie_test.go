package replay

// tracks_fin_de_vie_test.go — LES PROPRIETES DE LA FIN DE VIE A LA MORT ECRITE, et la mutation qui
// fait rougir chacune.
//
// LA FIGURE : un slot replique de 1 s a 4 s (frames 0 a 30, un point toutes les 500 ms), se tait,
// puis reprend de 10 s a 12 s (frames 90 a 110). Le calage du fil des morts est nul : l'instant
// d'une mort en millisecondes du match est son instant sur l'horloge du film.

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/facts/fallback"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/types"
)

const (
	finOrigineUS = 1_000_000
	finPasUS     = 100_000
	finFrames    = 200
)

func positionsDeuxVies() []grammar.BipedPosition {
	var out []grammar.BipedPosition
	for t := uint64(1_000_000); t <= 4_000_000; t += 500_000 {
		out = append(out, posAt(100, t, 1, 1, 0))
	}
	for t := uint64(10_000_000); t <= 12_000_000; t += 500_000 {
		out = append(out, posAt(100, t, 2, 2, 0))
	}
	return out
}

func deuxViesDuMemeJoueur(xuid uint64) []lifeSpan {
	return []lifeSpan{
		{slot: 100, from: 1_000_000, to: 4_000_000, xuid: xuid, cause: CauseVieMort},
		{slot: 100, from: 10_000_000, to: 12_000_000, xuid: xuid, cause: CauseVieFinFilm},
	}
}

// pisteDeFin publie les deux vies avec le fil de morts donne et rend les pistes et le compteur de
// replis. `calage` nil = calage non resolu.
func pisteDeFin(t *testing.T, vies []lifeSpan, morts []types.Death, calage *int64) ([]Track, *fallback.Compteur) {
	t.Helper()
	return pisteDeFinAvecTableau(t, vies, morts, calage, nil)
}

// pisteDeFinAvecTableau est pisteDeFin avec le tableau de l API donne a la cloture.
func pisteDeFinAvecTableau(t *testing.T, vies []lifeSpan, morts []types.Death, calage *int64,
	tableau []Participant) ([]Track, *fallback.Compteur) {
	t.Helper()
	fb := fallback.NouveauCompteur()
	h := replayClock{origin: finOrigineUS, step: finPasUS, frames: finFrames}
	tracks, _ := decimateTracks(positionsDeuxVies(), decoupeDesTraces{
		origin: finOrigineUS, step: finPasUS, minPoints: 1, vies: vies, fb: fb,
		fin: nouvelleClotureParMort(morts, calage, vies, tableau, h),
	})
	if len(tracks) != 2 {
		t.Fatalf("%d piste(s), attendu 2", len(tracks))
	}
	return tracks, fb
}

func calageNul() *int64 { z := int64(0); return &z }

func mort(xuid uint64, ms int64) types.Death { return types.Death{XUID: xuid, TimeMS: ms} }

// TestUneVieFinitALaMortEcriteQuiSuitSonDernierPoint — LA REGLE. La mort de 6 s tombe dans le
// silence : la fin est la frame 50, le dernier point reste la frame 30, aucun point n'est ajoute.
//
// MUTATION JOUEE : `fin` rend `derniere` quoi qu'il arrive -> EndFrame 30, ce test rouge.
func TestUneVieFinitALaMortEcriteQuiSuitSonDernierPoint(t *testing.T) {
	tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 6_000)}, calageNul())
	if tracks[0].EndFrame != 50 {
		t.Errorf("EndFrame = %d, attendu 50 (la frame de la mort ecrite)", tracks[0].EndFrame)
	}
	pts := tracks[0].Points
	if len(pts) != 7 || pts[len(pts)-1].T != 30 {
		t.Errorf("%d points, dernier a T=%d : aucun point ne se fabrique entre le dernier point et la mort",
			len(pts), pts[len(pts)-1].T)
	}
	if tracks[1].EndFrame != 110 {
		t.Errorf("la seconde vie n'a pas de mort ecrite : EndFrame = %d, attendu son dernier point 110",
			tracks[1].EndFrame)
	}
}

// TestUneMortAvantLeDernierPointNeChangeRien : la mort tombe dans la derniere frame ecrite.
func TestUneMortAvantLeDernierPointNeChangeRien(t *testing.T) {
	tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 3_950)}, calageNul())
	if tracks[0].EndFrame != 30 {
		t.Errorf("EndFrame = %d, attendu 30 : la mort est deja dans la plage ecrite", tracks[0].EndFrame)
	}
}

// TestLaMortDUnAutreJoueurNeCloPasLaVie : seule une mort du MEME joueur clot la vie.
func TestLaMortDUnAutreJoueurNeCloPasLaVie(t *testing.T) {
	tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(222, 6_000)}, calageNul())
	if tracks[0].EndFrame != 30 {
		t.Errorf("EndFrame = %d, attendu 30 : la mort est celle de 222", tracks[0].EndFrame)
	}
}

// TestLaPremiereMortApresLOuvertureClotLaVie : deux morts du joueur dans le silence, la premiere
// est celle de cette vie.
func TestLaPremiereMortApresLOuvertureClotLaVie(t *testing.T) {
	morts := []types.Death{mort(111, 7_000), mort(111, 5_000)}
	tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), morts, calageNul())
	if tracks[0].EndFrame != 40 {
		t.Errorf("EndFrame = %d, attendu 40 (la mort de 5 s, la premiere)", tracks[0].EndFrame)
	}
}

// TestUneMortNeFranchitPasLaVieSuivante : la fin est dans une frame ANTERIEURE a l'ouverture de la
// vie suivante du slot — jamais le meme instant, jamais au-dela. La vie suivante s'ouvre a la
// frame 90 : une mort a 9 899 ms (frame 89) prolonge, une mort a 9 900 ms (frame 89,9 -> 89) aussi,
// une mort a 10 000 ms (frame 90) laisse la fin au dernier point.
func TestUneMortNeFranchitPasLaVieSuivante(t *testing.T) {
	for _, c := range []struct {
		ms   int64
		want int
	}{{9_899, 88}, {9_999, 89}, {10_000, 30}, {10_500, 30}} {
		tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, c.ms)}, calageNul())
		if tracks[0].EndFrame != c.want {
			t.Errorf("mort a %d ms : EndFrame = %d, attendu %d", c.ms, tracks[0].EndFrame, c.want)
		}
		if tracks[0].EndFrame >= tracks[1].StartFrame {
			t.Errorf("mort a %d ms : la vie finit a %d, la suivante s'ouvre a %d — recouvrement",
				c.ms, tracks[0].EndFrame, tracks[1].StartFrame)
		}
	}
}

// TestUneMortAuDeLaDuFilmNeClotRien : la mort est hors de la fenetre de frames du document.
func TestUneMortAuDeLaDuFilmNeClotRien(t *testing.T) {
	vies := []lifeSpan{{slot: 100, from: 1_000_000, to: 4_000_000, xuid: 111}}
	fb := fallback.NouveauCompteur()
	h := replayClock{origin: finOrigineUS, step: finPasUS, frames: 60}
	tracks, _ := decimateTracks(positionsDeuxVies()[:7], decoupeDesTraces{
		origin: finOrigineUS, step: finPasUS, minPoints: 1, vies: vies, fb: fb,
		fin: nouvelleClotureParMort([]types.Death{mort(111, 6_000)}, calageNul(), vies, nil, h),
	})
	if tracks[0].EndFrame != 50 {
		t.Fatalf("EndFrame = %d, attendu 50 : la frame 50 est dans un film de 60 frames", tracks[0].EndFrame)
	}
	h.frames = 50
	tracks, _ = decimateTracks(positionsDeuxVies()[:7], decoupeDesTraces{
		origin: finOrigineUS, step: finPasUS, minPoints: 1, vies: vies, fb: fb,
		fin: nouvelleClotureParMort([]types.Death{mort(111, 6_000)}, calageNul(), vies, nil, h),
	})
	if tracks[0].EndFrame != 30 {
		t.Errorf("EndFrame = %d, attendu 30 : la frame 50 est hors d'un film de 50 frames", tracks[0].EndFrame)
	}
}

// TestUneVieSansIdentiteResteAuDernierPoint : sans xuid, aucune mort ne s'apparie.
func TestUneVieSansIdentiteResteAuDernierPoint(t *testing.T) {
	tracks, fb := pisteDeFin(t, deuxViesDuMemeJoueur(0), []types.Death{mort(111, 6_000)}, calageNul())
	if tracks[0].EndFrame != 30 {
		t.Errorf("EndFrame = %d, attendu 30", tracks[0].EndFrame)
	}
	if n := fb.Compte(fallback.NomFinDeVieAuDernierPoint); n != 0 {
		t.Errorf("repli compte %d fois, attendu 0 : le calage est resolu", n)
	}
}

// TestSansCalageLaFinResteAuDernierPointEtLeRepliSeCompte : le fil porte des morts mais le calage
// n'est pas resolu — chaque vie nommee reste au dernier point et le repli la compte.
//
// MUTATION JOUEE : retirer `compterLeSilenceDeCalage` -> compte 0, ce test rouge.
func TestSansCalageLaFinResteAuDernierPointEtLeRepliSeCompte(t *testing.T) {
	tracks, fb := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 6_000)}, nil)
	if tracks[0].EndFrame != 30 || tracks[1].EndFrame != 110 {
		t.Errorf("fins %d et %d, attendu 30 et 110 (derniers points)", tracks[0].EndFrame, tracks[1].EndFrame)
	}
	if n := fb.Compte(fallback.NomFinDeVieAuDernierPoint); n != 2 {
		t.Errorf("repli compte %d fois, attendu 2 (une par vie nommee)", n)
	}
	_, fb = pisteDeFin(t, deuxViesDuMemeJoueur(111), nil, nil)
	if n := fb.Compte(fallback.NomFinDeVieAuDernierPoint); n != 0 {
		t.Errorf("repli compte %d fois, attendu 0 : le fil ne porte aucune mort, il n'y a rien a placer", n)
	}
}

// TestLeCalageDecaleLesMorts : le calage est celui du registre, `horlogeFilm = horlogeMatch + calage`.
func TestLeCalageDecaleLesMorts(t *testing.T) {
	calage := int64(1_000)
	tracks, _ := pisteDeFin(t, deuxViesDuMemeJoueur(111), []types.Death{mort(111, 5_000)}, &calage)
	if tracks[0].EndFrame != 50 {
		t.Errorf("EndFrame = %d, attendu 50 (5 000 + 1 000 ms sur l'horloge du film)", tracks[0].EndFrame)
	}
}
