//go:build research

package grammar

// va_v2_research_test.go — LOT VA, ETAPE V2 (2026-10-05) : LES SAINS PERDUS, INSTRUITS PAQUET PAR
// PAQUET. Un instrument de recherche : aucune sortie de production ne change.
//
//	TestVAV2Pertes  la marche de la carte v2 ([rnMarcher], qui suit la cuisson du lot :
//	                [debutDeLaVueBDeCuisson]) ; pour chaque paquet de VA_PERDUS (lignes
//	                `film<TAB>chunk:paquet`), sur le monde d avant lui : la vue A lue (classe, portee,
//	                E, arret), le debut que la cuisson rend et comment, le debut que la regle d AVANT
//	                le lot rendrait sur ce meme monde ([vaLocaliserSansLoi] : ni la fin de la vue A,
//	                ni le bit nul devant les candidats de tete) et le bit qui le precede, les verdicts
//	                des deux marches d essai ; puis le verdict que la marche rend au paquet (controle
//	                croise avec la carte).
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	VA_PERDUS=<fichier> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestVAV2Pertes$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// vaLocaliserSansLoi est la regle de la cuisson d AVANT le lot (`87cdfa761`, [localiserLaListe]),
// recopiee : la signature stricte, puis la chaine de tete, puis la fermeture rangee, SANS le bit nul
// devant les candidats de tete. Le repli ferme au bit n y est pas compte.
func vaLocaliserSansLoi(pay []byte, w *World, cfg FrameConfig) (int, lecture.DebutDeVueB) {
	debut, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignatureStricte)
	extra := motFacultatifDEnTete(cfg)
	if debut >= 0 {
		for _, p := range candidatsDeTete(pay, debut, w) {
			if p-extra >= 0 && p < debut && chaineJusqua(pay, p-extra, debut, extra, w, cfg) {
				return p - extra, lecture.DebutParChaine
			}
		}
		return debut, lecture.DebutParSignature
	}
	auBit := -1
	for _, p := range candidatsDeTete(pay, len(pay)*8, w) {
		if p-extra < 0 {
			continue
		}
		l := lectureDEssai(pay, w, cfg, p-extra)
		if l.Fermee {
			return p - extra, lecture.DebutParFermeture
		}
		if l.FermeeAuBit && auBit < 0 {
			auBit = p - extra
		}
	}
	if auBit < 0 {
		return -1, lecture.DebutNonLocalise
	}
	return auBit, lecture.DebutParFermetureAuBit
}

// vaNomDuDebutV2 nomme un debut de vue B.
func vaNomDuDebutV2(d lecture.DebutDeVueB) string {
	if d == lecture.DebutParVueA {
		return "vue A"
	}
	return vaNomDuDebut(d)
}

// vaBitAvant rend le bit qui precede `s`, ou "-".
func vaBitAvant(pay []byte, s int) string {
	if s < 1 {
		return "-"
	}
	return strconv.Itoa(int(source.BitAt(pay, s-1)))
}

// vaSondeV2 instruit les sains perdus d un film.
type vaSondeV2 struct {
	f      *cmFilm
	g      grammaireDeLaVueA
	cibles map[uint64]bool
	avant  map[uint64]string
	lignes []string
}

func (x *vaSondeV2) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (x *vaSondeV2) finDeFilm()                                     {}

// paquet joint au detail d avant le paquet le verdict que la marche lui rend.
func (x *vaSondeV2) paquet(c int, p *cmPaquet, _ *World) {
	k := vaCle(c, p.d.Index)
	if d, ok := x.avant[k]; ok {
		x.lignes = append(x.lignes, rnTab(x.f.id, x.f.build, c, p.d.Index, d, p.d.Fermee, vaCellule(p.d.Cause)))
		delete(x.avant, k)
	}
}

// mesurer instruit un paquet cible sur le monde d avant lui.
func (x *vaSondeV2) mesurer(c int, pk FilmPacket, pay []byte, w *World) {
	k := vaCle(c, pk.Index)
	if !x.cibles[k] {
		return
	}
	cfg := x.f.cfg
	a := lireLaVueA(pay, 1, cfg.Profil, x.g)
	arret := vaArretTerminateur
	if !a.Porte {
		arret = vaArret(pay, &a, x.g)
	}
	d, comment := debutDeLaVueBDeCuisson(pay, &a, x.g.classe, w, cfg)
	s0, comment0 := vaLocaliserSansLoi(pay, w, cfg)
	essai := func(s int) string {
		if s < 0 {
			return "-"
		}
		return vaFerme(lectureDEssai(pay, w, cfg, s))
	}
	x.avant[k] = rnTab(vaNomDeClasse(x.g.classe), a.Porte, a.Fin, arret, len(a.Genres), d,
		vaNomDuDebutV2(comment), s0, vaNomDuDebut(comment0), vaBitAvant(pay, s0), essai(d), essai(s0),
		vaArretDEssai(pay, w, cfg, d))
}

// vaArretDEssai rend le bit ou s arrete la marche d essai partie de `debut` (monde restaure), -1 sans
// debut.
func vaArretDEssai(pay []byte, w *World, cfg FrameConfig, debut int) int {
	if debut < 0 {
		return -1
	}
	essai := cfg
	essai.Obs = NouvelleObservation()
	snap := w.Snapshot()
	defer w.Restore(snap)
	_, _, cur := DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, debut)
	return cur
}

// vaCellule rend `s`, ou "-" s il est vide.
func vaCellule(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// vaLirePerdus lit VA_PERDUS : par film, les cles des paquets perdus.
func vaLirePerdus(t *testing.T) map[string]map[uint64]bool {
	t.Helper()
	fh, err := os.Open(os.Getenv("VA_PERDUS"))
	if err != nil {
		t.Skipf("VA_PERDUS illisible : %v", err)
	}
	defer fh.Close()
	out := map[string]map[uint64]bool{}
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		film, cp, ok := strings.Cut(sc.Text(), "\t")
		cs, ps, ok2 := strings.Cut(cp, ":")
		c, e1 := strconv.Atoi(cs)
		p, e2 := strconv.Atoi(ps)
		if !ok || !ok2 || e1 != nil || e2 != nil {
			t.Fatalf("VA_PERDUS : ligne %q", sc.Text())
		}
		if out[film] == nil {
			out[film] = map[uint64]bool{}
		}
		out[film][vaCle(c, p)] = true
	}
	return out
}

// TestVAV2Pertes instruit les sains perdus de VA_PERDUS sur CAMPAGNE_FILMS.
func TestVAV2Pertes(t *testing.T) {
	racine, sortie, films := b2Env(t)
	perdus := vaLirePerdus(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tchunk\tpaquet\tclasse\tporte\tE\tarret\tgenres\tdebut\tcomment\tS0\tcomment0" +
		"\tbit_avant_S0\tessai_debut\tessai_S0\tarret_depuis_debut\tferme\tcause"}
	for _, id := range films {
		if len(perdus[id]) == 0 {
			continue
		}
		func() {
			garde := filmproc.Arm("campagne/va-v2", 4, func(pic uint64) {
				fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
				os.Exit(3)
			})
			defer garde.Disarm()
			debut := time.Now()
			f, ok := cmOuvrir(t, racine, id, utiles)
			if !ok {
				return
			}
			x := &vaSondeV2{f: f, g: f.fc.grammaireDeLaVueA(), cibles: perdus[id], avant: map[uint64]string{}}
			rnMarcher(f, cmVariante{}, x, func(c, _ int, pk FilmPacket, pay []byte, w *World) {
				x.mesurer(c, pk, pay, w)
			})
			lignes = append(lignes, x.lignes...)
			t.Logf("%s %s : %d paquet(s) instruit(s) sur %d, pic %d Mio, %s", id, f.build, len(x.lignes),
				len(perdus[id]), garde.Peak()>>20, time.Since(debut).Round(time.Second))
		}()
	}
	b2Ecrire(t, sortie, "va_v2_pertes.tsv", lignes)
}

// vaMorts marche `fc` avec le canal des morts ([ScanMarcheDesTramesAvec]). Sans la vue A
// (`sansVueA`), la grammaire de vue A du film est retiree du contexte avant la marche : la vue A ne se
// lit pas au-dela de sa tete, la cuisson suit le localisateur (la regle d AVANT le lot).
func vaMorts(fc *FilmContext, sansVueA bool) (MarcheDesTrames, error) {
	if sansVueA {
		fc.grammaireDuFilmDerivee().vueA = grammaireDeLaVueA{}
	}
	return ScanMarcheDesTramesAvec(fc, LecturesDeLaMarche{Morts: true})
}

// vaDiffDesLignes rend les lignes de `a` absentes de `b`.
func vaDiffDesLignes(a, b []string) []string {
	vues := map[string]int{}
	for _, x := range b {
		vues[x]++
	}
	var out []string
	for _, x := range a {
		if vues[x] > 0 {
			vues[x]--
			continue
		}
		out = append(out, x)
	}
	return out
}

// vaLignesDeVehicules rend les morts de vehicule et les lectures d occupation d une marche, une
// ligne par lecture.
func vaLignesDeVehicules(f MarcheDesTrames) (morts, occupations []string) {
	for _, d := range f.ObjectDeaths {
		if d.TypeIndex == uint32(VehicleTypeIndex) {
			morts = append(morts, fmt.Sprintf("mort\t%d\t%d\t%d\t%+v", d.TimestampUS, d.Slot, d.Gen, d.Dead))
		}
	}
	for _, o := range f.Occupancy {
		occupations = append(occupations, fmt.Sprintf("occupation\t%d\t%d\t%d\t%+v", o.TimestampUS, o.Slot, o.Gen, o))
	}
	return morts, occupations
}

// vaPoserLeProfilDeLaCuisson pose sur `fc`, comme `replay.poserProfilPuisCarte`, le profil que
// `killsource` a calibre sur le film (VA_PROFILS/<id>.profil.json, ecrit par
// `killsource/va_v2_profil_research_test.go`), puis les largeurs d objet du monde de la carte ; le
// decoupage MPP, la marche des images-cles le pose elle-meme. Sans VA_PROFILS, le contexte garde son
// profil.
func vaPoserLeProfilDeLaCuisson(t *testing.T, id string, fc *FilmContext) {
	rep := os.Getenv("VA_PROFILS")
	if rep == "" {
		return
	}
	blob, err := os.ReadFile(filepath.Join(rep, id+".profil.json"))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	var p ProfilDeBalayage
	if err := json.Unmarshal(blob, &p); err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	fc.PoserProfilDeBalayage(p)
	bal := fc.ProfilDeBalayage()
	fc.NoterReplis(bal.PoserLargeursObjetDuMondeDepuisDecoupage(fc.Profile().Map().Layout()))
	fc.PoserProfilDeBalayage(bal)
}

// TestVAV2Vehicules : par film de CAMPAGNE_FILMS (carte par VA_CATALOGUE / VA_CARTES, contexte de
// cuisson [NewFilmContextForMap], profil de la cuisson par VA_PROFILS), le canal des morts de la
// marche des trames sous la grammaire de vue A du film contre le meme canal sans elle ; ecrit les
// morts de vehicule et les lectures d occupation qui n appartiennent qu a l une des deux marches, et
// journalise les paquets localises et le repli `repli_localisation_largeur_libre` des deux cotes
// (verification (d) de la representation intermediaire, sur le canal des morts).
func TestVAV2Vehicules(t *testing.T) {
	racine, sortie, films := b2Env(t)
	lignes := []string{"film\tcote\tlecture\tinstant_us\tslot\tgen\tdetail"}
	for _, id := range films {
		e, ok := vaCarte(t, id)
		if !ok {
			t.Logf("%s : carte inconnue, film saute", id)
			continue
		}
		film, err := source.LoadDir(filepath.Join(racine, id), nil)
		if err != nil {
			t.Logf("%s : %v", id, err)
			continue
		}
		var marches [2]MarcheDesTrames
		var replis [2]int
		for i, sansVueA := range []bool{false, true} {
			fc := NewFilmContextForMap(film, &e, DefaultScanFilmOptions().Layout)
			vaPoserLeProfilDeLaCuisson(t, id, fc)
			avant := fc.ComptesDesReplis().LocalisationsALargeurLibre
			if marches[i], err = vaMorts(fc, sansVueA); err != nil {
				t.Fatalf("%s : %v", id, err)
			}
			replis[i] = fc.ComptesDesReplis().LocalisationsALargeurLibre - avant
		}
		avec, sans := marches[0], marches[1]
		ma, oa := vaLignesDeVehicules(avec)
		ms, osans := vaLignesDeVehicules(sans)
		for _, c := range []struct {
			cote string
			l    []string
		}{
			{"seulement V2", vaDiffDesLignes(ma, ms)}, {"seulement base", vaDiffDesLignes(ms, ma)},
			{"seulement V2", vaDiffDesLignes(oa, osans)}, {"seulement base", vaDiffDesLignes(osans, oa)},
		} {
			for _, x := range c.l {
				lignes = append(lignes, rnTab(id, c.cote)+"\t"+x)
			}
		}
		t.Logf("%s : morts de vehicule %d -> %d, occupations %d -> %d, paquets localises %d -> %d, "+
			"repli a largeur libre %d -> %d", id, len(ms), len(ma), len(osans), len(oa),
			sans.ObjectDeathStats.LocatedPackets, avec.ObjectDeathStats.LocatedPackets, replis[1], replis[0])
	}
	b2Ecrire(t, sortie, "va_v2_vehicules.tsv", lignes)
}
