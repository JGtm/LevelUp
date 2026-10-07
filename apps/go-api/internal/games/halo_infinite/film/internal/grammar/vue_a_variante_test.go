package grammar

// vue_a_variante_test.go — les vecteurs du lot VA, etape V3, et de la lecture du kill sans sa queue :
// PlayerKilledEvent (85), lu sans sa queue, et teleport_effects (116), que la variante de partie du
// film decide. Ecrits d apres leurs ECRIVAINS (`FUN_142f18fd0`, `FUN_142efa2a8`) et avec les immediats
// de leurs lecteurs.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// grammaireDeVariante est la grammaire d un film recent dont la variante de partie declare `v`,
// derivee par le code de production ([grammaireDeLaVueASousFilm]) sous l entree de catalogue `e`.
func grammaireDeVariante(v profile.VarianteDePartie, e *profile.MapQuantEntry) grammaireDeLaVueA {
	g := grammaireDeLaVueASousFilm(profilDIdentite(profile.FilmIdentity{Variante: v}, e))
	g.classe, g.genres = vueAEgale, GenresVueA
	return g
}

// varianteLue est une variante presente et lue, de moteur `moteur`.
func varianteLue(moteur int32, killcam, potg bool) profile.VarianteDePartie {
	return profile.VarianteDePartie{Lue: true, Presente: true, TypeDeMoteur: moteur, KillcamEnabled: killcam,
		PlayOfTheGameEnabled: potg}
}

// TestLeJoueurTueSeLitSansSaQueue : `FUN_142f18fd0` ecrit la victime, le tueur (`FUN_142b549c0` :
// W(1), W(5) s il vaut 0), W(32), W(1), l assistant, W(32), puis la queue si
// `(kill_playback_enabled && moteur != 1 && killcamEnabled) || (play_of_the_game_enabled &&
// playOfTheGameEnabled)`. Les deux reglages a leur defaut de l executable (faux), la garde est fausse :
// le message se lit, partie fixe seule, quelle que soit la variante du film, et ses champs sont
// ranges sur le lecteur. MUTATIONS — la queue lue sous playOfTheGameEnabled ; l assistant lu avant la
// part du tueur : ROUGES.
func TestLeJoueurTueSeLitSansSaQueue(t *testing.T) {
	attendu := lecture.MessageDeKill{Victime: 3, Tueur: lecture.RefAbsente, PartDuTueur: 0xdeadbeef, Drapeau: 1,
		Assistant: 9, PartDeLAssistant: 7}
	for _, c := range []struct {
		nom string
		v   profile.VarianteDePartie
	}{
		{"drapeaux nuls", varianteLue(2, false, false)},
		{"killcamEnabled, moteur 1", varianteLue(1, true, false)},
		{"killcamEnabled, moteur 2", varianteLue(2, true, false)},
		{"playOfTheGameEnabled", varianteLue(2, false, true)},
		{"playOfTheGameEnabled, moteur 1", varianteLue(1, false, true)},
		{"variante absente du film", profile.VarianteDePartie{Lue: true}},
		{"variante non lue", profile.VarianteDePartie{}},
	} {
		var w bitWriter
		w.bit(0)
		w.bits(3, 5) // victime
		w.bit(1)     // tueur absent
		w.bits(0xdeadbeef, 32)
		w.bit(1)
		w.bit(0)
		w.bits(9, 5) // assistant
		w.bits(0x7, 32)
		fin := w.n
		w.bits(0xffff, 16)
		br := LecteurSur(w.buf)
		br.vueA = grammaireDeVariante(c.v, nil)
		if lue := chargeJoueurTue(br); !lue || br.BitPos() != fin || br.killLu != attendu {
			t.Errorf("%s : lue %v (fin %d, champs %+v), attendu lue (fin %d, champs %+v)", c.nom, lue,
				br.BitPos(), br.killLu, fin, attendu)
		}
	}
}

// TestLesEffetsDeTeleportationSuiventLeMoteurDuFilm : `FUN_142efa2a8` ecrit W(1) ; s il vaut 1,
// `FUN_141f86118(mode 0)` — `FUN_142e2d8cc` quand le moteur n est pas 1 (R(1) ; s il vaut 0 : R(19) ;
// R(8) a la lecture, `FUN_140c5fa84`) ; W(1) ; `FUN_1407edb6c` (W(1), W(32) s il vaut 1) ; s il vaut
// 1, deux positions de niveau 0x10. Le moteur 1 (`FUN_142e29bac`, non porte) et un film sans
// variante lue arretent la lecture au premier bit. MUTATIONS — la branche du moteur 1 lue comme
// l autre ; les positions lues au niveau 0xf : ROUGE.
func TestLesEffetsDeTeleportationSuiventLeMoteurDuFilm(t *testing.T) {
	e := carteDeTest()
	for _, c := range []struct {
		nom       string
		v         profile.VarianteDePartie
		orient    bool
		positions bool
		porte     bool
		lue       bool
	}{
		{"moteur 2, orientation et positions par defaut", varianteLue(2, false, true), true, true, true, true},
		{"moteur 2, positions de la region jouee", varianteLue(2, false, true), true, true, false, true},
		{"moteur 2, sans positions", varianteLue(2, false, true), true, false, false, true},
		{"sans orientation, variante non lue", profile.VarianteDePartie{}, false, true, true, true},
		{"moteur 1, orientation", varianteLue(1, false, false), true, false, false, false},
		{"variante non lue, orientation", profile.VarianteDePartie{}, true, false, false, false},
	} {
		var w bitWriter
		if c.orient {
			w.bit(1)
			w.bit(0)        // FUN_140c5fa84 : porte a 0
			w.bits(5, 0x13) // direction
			w.bits(0x80, 8) // roulis
		} else {
			w.bit(0)
		}
		w.bit(1)
		if c.positions {
			w.bit(1)
			w.bits(0xcafe, 32)
			larg := profile.LargeursAxeParDefautDuBuild(0x10)
			if !c.porte {
				larg = e.AxisWidths
			}
			w.ecrirePositionDeNiveau(c.porte, uint64(e.Region), larg)
			w.ecrirePositionDeNiveau(c.porte, uint64(e.Region), larg)
		} else {
			w.bit(0)
		}
		fin := w.n
		w.bits(0xffff, 16)
		br := LecteurSur(w.buf)
		br.vueA = grammaireDeVariante(c.v, &e)
		if lue := chargeEffetsDeTeleportation(br); lue != c.lue || lue && br.BitPos() != fin {
			t.Errorf("%s : lue %v (fin %d), attendu %v (fin %d)", c.nom, lue, br.BitPos(), c.lue, fin)
		}
	}
}

// TestLesBobinesRecentesDecidentLeur116 : sous ce que le `chunk_00` de `fb1a1a72` (HI_1_13_0) et de
// `bcb6d393` (HI_1_12_0) declare, derive par le code de production, la variante est lue et son moteur
// n est pas le 1 : le 116 se lit.
func TestLesBobinesRecentesDecidentLeur116(t *testing.T) {
	for _, film := range []string{"fb1a1a72", "bcb6d393"} {
		id, err := ReadFilmIdentity(bobineChunk00(t, film))
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		g := grammaireDeLaVueASousFilm(profilDIdentite(id, nil))
		if g.variante != (varianteDeLaVueA{lue: true}) {
			t.Errorf("%s : %+v, attendu lue, moteur autre que 1", film, g.variante)
		}
	}
}
