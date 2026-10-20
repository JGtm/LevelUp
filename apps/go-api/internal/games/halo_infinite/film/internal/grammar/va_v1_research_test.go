//go:build research

package grammar

// va_v1_research_test.go — LOT VA, ETAPE V1 (2026-10-05) : LA LECTURE UNIQUE DE LA VUE A, MESUREE
// SUR LES FILMS DU CORPUS. Un instrument de recherche : aucune sortie de production ne change.
//
//	TestVAV1Tete       la marche de PRODUCTION ([FilmContext.Trames], contexte de cuisson : la
//	                   carte du match posee par [NewFilmContextForMap]) ; paquet par paquet, la tete
//	                   donnee aux canaux ([teteDe]), la route vers la localisation ([listeAnnoncee]) et
//	                   l etendue rangee de la vue A, confrontees a la lecture d AVANT le lot (le temoin
//	                   [teteDAvant], recopie de `consumeVueA` et de `teteDe` a `87cdfa761`) ; puis la
//	                   PASSE DES TETES ([distribuerLesTetes], sans canal des trames) sur un contexte
//	                   neuf, dont chaque trame doit porter la meme tete et la meme vue A que la marche.
//	TestVAV1FinDeVueA  la marche de la carte v2 ([cmMarcher]) et la localisation de production
//	                   ([localiserLaListe]) : sur chaque paquet a evenements, la vue A lue par
//	                   [lireLaVueA] — jusqu a son terminateur ou non, et pourquoi — et sa fin E
//	                   confrontee au debut S que la localisation rend, sur le monde d avant le paquet.
//	TestVAV1Extraire   le payload d un paquet (VA_PAQUET = film:chunk:index), pour un vecteur de test.
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	VA_CATALOGUE=<map_quant_bounds.json> VA_CARTES="id=Carte;..." \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestVAV1' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// lireLaTeteDeMarche lit la vue A d un payload a la position du lecteur `br` par la lecture unique
// ([lireLaVueA]) et place `br` la ou la marche partie de la tete s arrete ([lireTrameParRangs]) :
// a sa fin quand elle est vide, apres sa tete sinon. C est la forme des sondes qui recopient la
// marche depuis la tete (`mouvement_5_*_research_test.go`).
func lireLaTeteDeMarche(br *Lecteur, pay []byte, cfg FrameConfig) FluxVueA {
	a := lireLaVueA(pay, br.BitPos(), cfg.Profil, grammaireDeLaVueA{})
	br.SetBitPos(a.finDeTete())
	return a
}

// vaCarte rend l entree de catalogue de la carte d un film (VA_CATALOGUE, VA_CARTES).
func vaCarte(t *testing.T, id string) (profile.MapQuantEntry, bool) {
	t.Helper()
	cat, err := profile.LoadMapQuantCatalog(os.Getenv("VA_CATALOGUE"))
	if err != nil {
		t.Logf("%s : catalogue illisible : %v", id, err)
		return profile.MapQuantEntry{}, false
	}
	for _, kv := range strings.Split(os.Getenv("VA_CARTES"), ";") {
		if k, nom, ok := strings.Cut(kv, "="); ok && k == id {
			e, err := cat.Lookup(nom)
			return e, err == nil
		}
	}
	return profile.MapQuantEntry{}, false
}

// vaNomDeClasse nomme la classe de la table des genres d un film.
func vaNomDeClasse(c classeDeLaVueA) string {
	return [...]string{"illisible", "prefixe", "egale"}[c]
}

// vaComparerLaVueA classe la vue A rangee d un paquet contre celle d avant : identique, desormais
// portee (meme tete, la lecture va plus loin), ou ecart.
func vaComparerLaVueA(n, avant *lecture.VueA) string {
	if n.Debut == avant.Debut && n.Bits == avant.Bits && n.Etat == avant.Etat && slices.Equal(n.Genres, avant.Genres) {
		return "identique"
	}
	if n.Debut == avant.Debut && len(avant.Genres) == 1 && len(n.Genres) >= 1 && n.Genres[0] == avant.Genres[0] &&
		n.Bits >= avant.Bits {
		return "desormais portee"
	}
	return "ECART"
}

// vaCle est la cle d une trame : son chunk et son index.
func vaCle(chunk, index int) uint64 { return uint64(chunk)<<32 | uint64(uint32(index)) } //nolint:gosec // cle de recherche

// vaPasseDesTetes joue la passe des tetes sur un contexte neuf du film et compte les trames dont la
// tete ou la vue A rangee differe de celle que la marche a rangee (`marche`), ou que la marche n a pas
// rendues ; rend aussi le nombre de trames de la passe.
func vaPasseDesTetes(t *testing.T, fc *FilmContext, marche map[uint64]teteVue) (trames, ecarts int) {
	t.Helper()
	passe := &canalDeTeteTemoin{}
	if err := Distribuer(fc, passe); err != nil {
		t.Errorf("passe des tetes : %v", err)
		return 0, 0
	}
	for _, v := range passe.vues {
		m, ok := marche[vaCle(v.chunk, v.index)]
		if !ok || m.tete != v.tete || !reflect.DeepEqual(m.vueA, v.vueA) {
			ecarts++
		}
	}
	return len(passe.vues), ecarts
}

// TestVAV1Tete joue la preuve (a) et (b) de la representation intermediaire sur CAMPAGNE_FILMS.
func TestVAV1Tete(t *testing.T) {
	racine, sortie, films := b2Env(t)
	out := []string{"film\tbuild\tclasse\tpaquets\tvue_a\ttete_ecarts\troute_ecarts\tlistes\tpasse_trames\tpasse_ecarts"}
	for _, id := range films {
		func() {
			garde := filmproc.Arm("campagne/va-v1", 4, func(pic uint64) {
				fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
				os.Exit(3)
			})
			defer garde.Disarm()
			fc0, _, _ := ContexteDeFilm(filepath.Join(racine, id))
			if fc0 == nil {
				t.Errorf("%s : film illisible", id)
				return
			}
			entree, ok := vaCarte(t, id)
			var e *profile.MapQuantEntry
			if ok {
				e = &entree
			}
			fc := NewFilmContextForMap(fc0.Film(), e, nil)
			classe := vaNomDeClasse(fc.grammaireDeLaVueA().classe)
			vues := map[string]int{}
			marche := map[uint64]teteVue{}
			paquets, teteEcarts, routeEcarts, listes := 0, 0, 0, 0
			for p, err := range fc.Trames(nil) {
				if err != nil {
					t.Errorf("%s : %v", id, err)
					return
				}
				paquets++
				a := p.VueA
				a.Genres = append([]uint8(nil), a.Genres...)
				marche[vaCle(p.Chunk, p.Index)] = teteVue{chunk: p.Chunk, index: p.Index, vueA: a, tete: teteDe(p)}
				avant, tete, route := teteDAvant(p.Payload)
				vues[vaComparerLaVueA(&p.VueA, &avant)]++
				if teteDe(p) != tete {
					teteEcarts++
				}
				if listeAnnoncee(&p.VueA) != route {
					routeEcarts++
				}
				if route {
					listes++
				}
			}
			var cles []string
			for k, n := range vues {
				cles = append(cles, k+"="+strconv.Itoa(n))
			}
			slices.Sort(cles)
			passeTrames, passeEcarts := vaPasseDesTetes(t, NewFilmContextForMap(fc0.Film(), e, nil), marche)
			out = append(out, rnTab(id, cmBuild(fc), classe, paquets, strings.Join(cles, ","), teteEcarts,
				routeEcarts, listes, passeTrames, passeEcarts))
			if teteEcarts+routeEcarts+vues["ECART"]+passeEcarts > 0 || passeTrames == 0 {
				t.Errorf("%s : %d tete(s), %d route(s), %d etendue(s), %d trame(s) de la passe des tetes (sur %d) en ecart",
					id, teteEcarts, routeEcarts, vues["ECART"], passeEcarts, passeTrames)
			}
			t.Logf("%s : %d paquets, pic %d Mio", id, paquets, garde.Peak()>>20)
		}()
	}
	b2Ecrire(t, sortie, "va_v1_tete.tsv", out)
}

// vaSonde mesure la fin de la vue A sur la marche de la carte v2.
type vaSonde struct {
	f      *cmFilm
	g      grammaireDeLaVueA
	t      cmTables
	detail string
	lignes []string
}

func (x *vaSonde) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (x *vaSonde) finDeFilm()                                     {}

func (x *vaSonde) paquet(c int, p *cmPaquet, _ *World) {
	if x.detail != "" {
		x.lignes = append(x.lignes, rnTab(x.f.id, x.f.build, c, p.d.Index, x.detail))
		x.detail = ""
	}
}

func (x *vaSonde) tete(int) func([]byte, *World, FrameConfig) (int, bool) {
	return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
		s, comment := localiserLaListe(pay, w, cfg)
		x.mesurer(pay, w, cfg, s, comment)
		return s, comment != lecture.DebutParSignature && comment != lecture.DebutNonLocalise
	}
}

// vaArretTerminateur : la vue A s est lue jusqu a son terminateur.
const vaArretTerminateur = "lue jusqu au terminateur"

// vaArret dit pourquoi une vue A ne s est pas lue jusqu a son terminateur.
func vaArret(pay []byte, a *FluxVueA, g grammaireDeLaVueA) string {
	switch {
	case g.classe == vueAIllisible:
		return "film illisible"
	case len(pay) > 0 && pay[0]&0x80 == 0:
		return "configuration a 0"
	case len(a.Genres) == 0:
		return "fin de payload"
	}
	genre := a.Genres[len(a.Genres)-1]
	_, vide := descripteurDuGenre(min(genre, GenresVueA-1))
	switch {
	case genre >= g.genres:
		return "hors cardinal " + strconv.Itoa(genre)
	case !vide && chargeDuGenre(genre) == nil:
		return "non porte " + strconv.Itoa(genre)
	}
	return "refuse " + strconv.Itoa(genre)
}

// vaFerme nomme le verdict d une marche d essai.
func vaFerme(l LectureVueC) string {
	switch {
	case l.Fermee:
		return "ferme"
	case l.FermeeAuBit:
		return "ferme au bit"
	}
	return "non ferme"
}

// vaNomDuDebut nomme la facon dont la localisation a trouve S.
func vaNomDuDebut(d lecture.DebutDeVueB) string {
	return map[lecture.DebutDeVueB]string{lecture.DebutParSignature: "signature", lecture.DebutParChaine: "chaine",
		lecture.DebutParFermeture: "fermeture", lecture.DebutParFermetureAuBit: "fermeture au bit",
		lecture.DebutNonLocalise: "non localise"}[d]
}

func (x *vaSonde) mesurer(pay []byte, w *World, cfg FrameConfig, s int, comment lecture.DebutDeVueB) {
	g := x.g
	a := lireLaVueA(pay, 1, cfg.Profil, g)
	cle := cmJoindre(x.f.build, vaNomDeClasse(g.classe))
	x.t.un("paquets a evenements", cle)
	verdict, arret := "-", vaArretTerminateur
	switch {
	case !a.Porte:
		arret = vaArret(pay, &a, g)
		x.t.un("arret", cmJoindre(cle, arret))
	case s < 0:
		verdict = "non localise, depuis E " + vaFerme(lectureDEssai(pay, w, cfg, a.Fin))
	case a.Fin == s:
		verdict = "accord"
	case a.Fin < s && chaineJusqua(pay, a.Fin, s, motFacultatifDEnTete(cfg), w, cfg):
		verdict = "accord par chaine"
	default:
		sens := "E > S"
		if a.Fin < s {
			sens = "E < S"
		}
		verdict = cmJoindre("desaccord", sens, "depuis E "+vaFerme(lectureDEssai(pay, w, cfg, a.Fin)),
			"depuis S "+vaFerme(lectureDEssai(pay, w, cfg, s)))
	}
	if a.Porte {
		x.t.un("lues jusqu au terminateur", cle)
		x.t.un("verdict", cmJoindre(cle, verdict))
		x.t.un("verdict par localisation", cmJoindre(cle, vaNomDuDebut(comment), verdict))
	}
	x.detail = rnTab(vaNomDeClasse(g.classe), vaNomDuDebut(comment), s, a.Fin, arret, len(a.Genres), verdict)
}

// TestVAV1FinDeVueA joue la mesure de la fin de la vue A sur CAMPAGNE_FILMS.
func TestVAV1FinDeVueA(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := []string{"film\tbuild\ttable\tcle\tn"}
	lignes := []string{"film\tbuild\tchunk\tpaquet\tclasse\tlocalisation\tS\tE\tarret\tgenres\tverdict"}
	for _, id := range films {
		func() {
			garde := filmproc.Arm("campagne/va-v1", 4, func(pic uint64) {
				fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
				os.Exit(3)
			})
			defer garde.Disarm()
			debut := time.Now()
			f, ok := cmOuvrir(t, racine, id, utiles)
			if !ok {
				return
			}
			x := &vaSonde{f: f, g: f.fc.grammaireDeLaVueA(), t: cmTables{}}
			if e, ok := vaCarte(t, id); ok {
				x.g.positions = tablesDeLaRegionJouee(e) // la carte que la cuisson porterait
			}
			cmMarcher(f, cmVariante{tete: x.tete}, x)
			lignes = append(lignes, x.lignes...)
			for nom, m := range x.t {
				for _, k := range cmCles(m) {
					out = append(out, rnTab(id, f.build, nom, k, m[k].n))
				}
			}
			t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		}()
	}
	b2Ecrire(t, sortie, "va_v1.tsv", out)
	b2Ecrire(t, sortie, "va_v1_paquets.tsv", lignes)
}

// TestVAV1Extraire ecrit le payload du paquet VA_PAQUET (film:chunk:index) sous CAMPAGNE_SORTIE.
func TestVAV1Extraire(t *testing.T) {
	racine, sortie := os.Getenv("CAMPAGNE_RACINE"), os.Getenv("CAMPAGNE_SORTIE")
	champs := strings.Split(os.Getenv("VA_PAQUET"), ":")
	if racine == "" || sortie == "" || len(champs) != 3 {
		t.Skip("CAMPAGNE_RACINE, CAMPAGNE_SORTIE et VA_PAQUET=film:chunk:index requis")
	}
	chunk, err1 := strconv.Atoi(champs[1])
	index, err2 := strconv.Atoi(champs[2])
	if err1 != nil || err2 != nil {
		t.Fatalf("VA_PAQUET %q", os.Getenv("VA_PAQUET"))
	}
	fc, _, _ := ContexteDeFilm(filepath.Join(racine, champs[0]))
	if fc == nil {
		t.Fatalf("%s : film illisible", champs[0])
	}
	data, pks, ok := fc.ChunkAt(chunk)
	if !ok {
		t.Fatalf("chunk %d absent", chunk)
	}
	for _, pk := range pks {
		if pk.Index == index {
			nom := filepath.Join(sortie, fmt.Sprintf("%s_%d_%d.bin", champs[0], chunk, index))
			if err := os.WriteFile(nom, pk.Payload(data), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s : type %d, %d octets", nom, pk.Type, len(pk.Payload(data)))
			return
		}
	}
	t.Fatalf("paquet %d absent du chunk %d", index, chunk)
}
