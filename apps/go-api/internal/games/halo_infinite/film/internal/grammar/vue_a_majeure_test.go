package grammar

// vue_a_majeure_test.go — LA GARDE DE VERSION MAJEURE DU JEU DECIDE LA CLASSE D UN FILM A TABLE
// EGALE, ET LA SUITE DES GENRES DIT OU SA NUMEROTATION DEVIENT PRESUMEE (lot VA, decisions du
// pilote du 2026-10-06).
//
// `FUN_1428e219c` ne lit un film que sous `*film == 0x29` (`va_ghidra/FUN_1428e219c.c`) : un film de
// table egale sous une autre majeure suit la regle des films PREFIXE, la fin de sa vue A ne vaut que
// prouvee paquet par paquet.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// majeureHI1120 est la version majeure des films HI_1_12_0 (0x28), lue en tete de leur `chunk_00`.
const majeureHI1120 = 0x28

// tableNative rend la table des genres que l executable porte.
func tableNative() []uint32 {
	natives := make([]uint32, GenresVueA)
	for g := range natives {
		natives[g] = versionNative(g)
	}
	return natives
}

// profilDeMajeure resout le profil d un film de table native, de version majeure `majeure` (lue
// quand `lue`), sans carte.
func profilDeMajeure(majeure int, lue bool) profile.Profile {
	return profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Majeure: majeure, MajeureLue: lue,
		Identite: profile.FilmIdentity{TypeVersions: tableNative()}, IdentiteLue: true}, nil)
}

// TestLaClasseDUnFilmSuitLaVersionMajeureQueLeJeuJoue : une table egale ne rend la classe EGALE que
// sous la majeure 0x29, LUE ; sous la majeure 0x28 des films HI_1_12_0, ou une majeure non lue, la
// classe est PREFIXE, cardinal inchange. Les classes PREFIXE et ILLISIBLE ne changent pas. MUTATION —
// la garde retiree de [classeSousLaMajeure] : ROUGE.
func TestLaClasseDUnFilmSuitLaVersionMajeureQueLeJeuJoue(t *testing.T) {
	for _, c := range []struct {
		nom     string
		classe  classeDeLaVueA
		majeure int
		lue     bool
		attendu classeDeLaVueA
	}{
		{"table egale, majeure 0x29", vueAEgale, versionMajeureJouee, true, vueAEgale},
		{"table egale, majeure 0x28", vueAEgale, majeureHI1120, true, vueAPrefixe},
		{"table egale, majeure non lue", vueAEgale, versionMajeureJouee, false, vueAPrefixe},
		{"table prefixe, majeure 0x29", vueAPrefixe, versionMajeureJouee, true, vueAPrefixe},
		{"table illisible, majeure 0x29", vueAIllisible, versionMajeureJouee, true, vueAIllisible},
	} {
		if got := classeSousLaMajeure(c.classe, c.majeure, c.lue); got != c.attendu {
			t.Errorf("%s : classe %d, attendu %d", c.nom, got, c.attendu)
		}
	}
	for _, c := range []struct {
		majeure int
		attendu classeDeLaVueA
	}{{versionMajeureJouee, vueAEgale}, {majeureHI1120, vueAPrefixe}} {
		if classe, genres := tableDesGenresDuFilm(profilDeMajeure(c.majeure, true)); classe != c.attendu ||
			genres != GenresVueA {
			t.Errorf("profil de majeure %#x : (%d, %d), attendu (%d, %d)", c.majeure, classe, genres, c.attendu, GenresVueA)
		}
	}
	for film, attendu := range map[string]classeDeLaVueA{"bcb6d393": vueAPrefixe, "fb1a1a72": vueAEgale} {
		if classe, _ := tableDesGenresDuFilm(ResolveProfile(bobineFilm(t, film), nil)); classe != attendu {
			t.Errorf("bobine %s : classe %d, attendu %d", film, classe, attendu)
		}
	}
}

// TestUnFilmDUneAutreMajeureEstProuvePaquetParPaquet : le paquet de
// [TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite] — la marche depuis E bute, une signature suit.
// Sous la grammaire derivee d un film de table native et de majeure 0x29, la marche part de E ; sous
// la meme table et la majeure 0x28, E n est pas prouve, et le paquet suit le localisateur a
// l identique. MUTATION — la garde retiree de [classeSousLaMajeure] : ROUGE.
func TestUnFilmDUneAutreMajeureEstProuvePaquetParPaquet(t *testing.T) {
	pay, e := paquetVueA(1, zoomCourt, func(w *bitWriter) {
		w.deltaMasque13(124, 5)
		w.bit(0)
		w.signature123()
		w.finDeVueB()
		w.vueCUneEntree()
	})
	w := mondeDeCarte(compHighFrequency)
	cfg := cadreDeCarte()
	if l := lectureDEssai(pay, w, cfg, e); l.Fermee {
		t.Fatalf("la marche depuis E ferme le paquet : le vecteur doit la faire buter")
	}
	jouee := grammaireDeLaVueASousFilm(profilDeMajeure(versionMajeureJouee, true))
	if d, comment := debutDuPaquet(t, pay, w, jouee); d != e || comment != lecture.DebutParVueA {
		t.Errorf("majeure 0x29 : debut (%d, %d) ; attendu %d par la vue A", d, comment, e)
	}
	s, commentS := localiserLaListe(pay, w, cfg)
	autre := grammaireDeLaVueASousFilm(profilDeMajeure(majeureHI1120, true))
	if d, comment := debutDuPaquet(t, pay, w, autre); d == e || d != s || comment != commentS {
		t.Errorf("majeure 0x28 : debut (%d, %d) ; attendu le localisateur (%d, %d), et pas E = %d",
			d, comment, s, commentS, e)
	}
}

// TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee : la vue A d un zoom (genre 21), puis d un genre
// au-dela du 107 que la lecture ne porte pas (elle s arrete apres lui). Sous un film PREFIXE, le
// second genre est presume (rang 1) ; sous un film EGALE, aucun (rang = 2). Le rang est range dans
// `lecture.Paquet.VueA`. Le dernier genre a version distinctive est bien le 107 de la table native.
// MUTATIONS — [premierGenrePresume] rend toujours len(genres) ; le rang non range : ROUGES.
func TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee(t *testing.T) {
	dernier := -1
	for g := range GenresVueA {
		if versionNative(g) != 1 {
			dernier = g
		}
	}
	if dernier != dernierGenreAVersionDistinctive {
		t.Fatalf("dernier genre a version distinctive %d, constante %d", dernier, dernierGenreAVersionDistinctive)
	}
	pay, _ := paquetVueA(1, func(w *bitWriter) {
		zoomCourt(w)
		w.ecrireEnTeteDeMessage(dernierGenreAVersionDistinctive + 3)
	}, vueBFermee)
	for _, c := range []struct {
		nom     string
		g       grammaireDeLaVueA
		attendu uint16
	}{
		{"film PREFIXE", grammaireDeTest(vueAPrefixe, GenresVueA), 1},
		{"film EGALE", grammaireRecente(), 2},
	} {
		p := lecture.Paquet{Payload: pay}
		rangerLaTete(&p, ProfilDeBalayageParDefaut(), c.g)
		if len(p.VueA.Genres) != 2 || p.VueA.Genres[1] != dernierGenreAVersionDistinctive+3 {
			t.Fatalf("%s : genres %v, attendu [21 %d]", c.nom, p.VueA.Genres, dernierGenreAVersionDistinctive+3)
		}
		if p.VueA.PremierPresume != c.attendu {
			t.Errorf("%s : premier genre presume au rang %d, attendu %d", c.nom, p.VueA.PremierPresume, c.attendu)
		}
	}
}
