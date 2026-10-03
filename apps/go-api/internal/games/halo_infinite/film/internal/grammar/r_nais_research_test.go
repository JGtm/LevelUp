//go:build research

package grammar

// r_nais_research_test.go — CHANTIER « NAIS » DE LA CAMPAGNE DE GRAMMAIRE (2026-10-02) : LE PILOTE.
// R-L1 (a), (b), (d) et R-P6 du plan (`.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.1, §6.2). Un
// instrument de recherche : aucun fichier de production n est touche, aucune sortie ne change.
//
// Par film (un a la fois, sentinelle `filmproc` 4 Gio) :
//
//	passe 1    marche de reference + [rnP1] (occurrences, paquets a rejet, pieds, scores) + juge ;
//	passe 2    la meme marche, et, AVANT chaque paquet qui porte une occurrence en region non lue,
//	           le paquet relu depuis l occurrence sur une copie du monde : ferme ? sain ? ;
//	(a)        oracle (iii') apres rejet, entier puis scinde par « Q se ferme, sain, depuis
//	           l occurrence » ;
//	R-P6       oracles des naissances a plus de 3 paquets sous des filtres d OCCURRENCE, et le
//	           controle `propre+alloc+suivant` des mesures bis 3 ; oracle (i)+(ii) seul et avec P6 ;
//	(d)        le localisateur `tete-bloc+inv` (mesures bis 1), et trois conditions qui ne
//	           l activent que la ou l allocateur predit les NEW du film.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 240m -run '^TestRNaisMesures$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// rnSeuilScore : part des NEW propres (paquets sans liste d evenements) que l allocateur a cinq
// pools predit (rang < 64, tete predite), au-dessus de laquelle le localisateur elargi s active.
// Mesure (`r_nais_cadre.tsv`) : 54,7 % au plus bas sur HI_1_12_0+, 37,7 % au plus haut ailleurs.
const rnSeuilScore = 0.5

// TestRNaisMesures joue le chantier sur les films de CAMPAGNE_FILMS.
func TestRNaisMesures(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	out := map[string][]string{
		"variantes": {rnTeteVariante},
		"tables":    {"film\tbuild\ttable\tcle\tn\tpaquets\thors_cadre\tfermes"},
		"pied":      {rnTetePied},
		"scores":    {"film\tbuild\tchunk\tneufs_propres\tpredits\tactif_chunk\tactif_cumul\tactif_film"},
	}
	for _, id := range films {
		rnUnFilm(t, racine, id, utiles, out)
	}
	for nom, l := range out {
		b2Ecrire(t, sortie, "r_nais_"+nom+".tsv", l)
	}
}

// rnVar : une marche jugee.
type rnVar struct {
	nom string
	v   cmVariante
	nl  int
}

func rnUnFilm(t *testing.T, racine, id string, utiles UsagesProduit, out map[string][]string) {
	garde := filmproc.Arm("r_nais/mesures", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	f, ok := cmOuvrir(t, racine, id, utiles)
	if !ok {
		return
	}
	b := cmLireBlocs(f)
	p1 := rnNouveauP1(f, b)
	jref := rnNouveauJuge(f, b, nil)
	rep, _ := rnMarcher(f, cmVariante{}, cmMux{p1, jref}, nil)
	out["variantes"] = append(out["variantes"], rnLigneVariante(f, "reference", 0, rep, jref))
	mode := os.Getenv("RNAIS_SEULEMENT") // "" : tout ; "d" : passe 1 et (d) ; "a" : passes 1 et 2, tables
	var rep2 FrameClosureReport
	if mode != "d" {
		rep2, _ = rnMarcher(f, cmVariante{}, nil, rnEvaluateur(f, b, p1))
	} else {
		rep2 = rep
	}
	if rep2.PaquetsFermes != rep.PaquetsFermes || rep2.Utiles.RecordsFermes != rep.Utiles.RecordsFermes {
		t.Errorf("%s : la passe 2 ne rend pas la reference (%d/%d contre %d/%d)", id, rep2.PaquetsFermes,
			rep2.Utiles.RecordsFermes, rep.PaquetsFermes, rep.Utiles.RecordsFermes)
	}
	p1.aDescripteurs()
	p1.tablesDesRejets()
	p1.temoinsP6()
	on := rnConditions(f, p1, out)
	var vs []rnVar
	if mode == "" {
		vs = append(rnOraclesA(p1), rnOraclesP6(p1)...)
	}
	inv := func(actif func(int) bool) func(c int) func([]byte, *World, FrameConfig) (int, bool) {
		ti := cmTeteInv(f, b, true, false)
		return func(c int) func([]byte, *World, FrameConfig) (int, bool) {
			if actif != nil && !actif(c) {
				return nil
			}
			return ti(c)
		}
	}
	if mode != "a" {
		vs = append(vs, rnVar{nom: "tete-bloc+inv", v: cmVariante{tete: inv(nil)}},
			rnVar{nom: "tete-bloc+inv|film", v: cmVariante{tete: inv(on[0])}},
			rnVar{nom: "tete-bloc+inv|chunk", v: cmVariante{tete: inv(on[1])}},
			rnVar{nom: "tete-bloc+inv|cumul", v: cmVariante{tete: inv(on[2])}})
	}
	for _, x := range vs {
		j := rnNouveauJuge(f, b, jref)
		r, _ := rnMarcher(f, x.v, j, nil)
		out["variantes"] = append(out["variantes"], rnLigneVariante(f, x.nom, x.nl, r, j))
	}
	for nom, m := range p1.t {
		for _, k := range cmCles(m) {
			x := m[k]
			out["tables"] = append(out["tables"], rnTab(id, f.build, nom, k, x.n, x.paquets, x.horsCadre, x.fermes))
		}
	}
	out["pied"] = append(out["pied"], p1.pied...)
	t.Logf("%s %s : %d marches, %d occurrences ; pic %d Mio, %s", id, f.build, len(vs)+2, len(p1.occs),
		garde.Peak()>>20, time.Since(debut).Round(time.Second))
}

// rnEvaluateur : la passe 2. Chaque occurrence en region non lue est jugee sur le monde d AVANT
// son paquet (copie, restauree) : le paquet relu depuis elle se ferme-t-il, et sain ?
func rnEvaluateur(f *cmFilm, b *cmBlocs, p1 *rnP1) rnAvant {
	extra := motFacultatifDEnTete(f.cfg)
	chk := nouveauCollecteur(f, b, nil)
	return func(c, j int, _ FilmPacket, pay []byte, w *World) {
		for _, k := range p1.parPaquet[[2]int{c, j}] {
			o := &p1.occs[k]
			if o.pos-extra < 0 {
				continue
			}
			snap := w.Snapshot()
			q := cmEssaiDepuis(f, pay, w, o.pos-extra)
			w.Restore(snap)
			o.evaluee, o.ferme = true, q.d.Fermee
			if q.d.Fermee {
				chk.chunk = c
				o.sain = len(cmContredit(chk, q)) == 0
			}
		}
	}
}

// rnLier construit un oracle : pour chaque cible, l occurrence que `sel` retient (-1 : aucune) est
// liee APRES son paquet, avec son `R(6)`.
func rnLier(p1 *rnP1, sel func(r *rnCible) int) (map[[2]int][]cmLiaison, int) {
	m := map[[2]int][]cmLiaison{}
	n := 0
	for i := range p1.cibles {
		r := &p1.cibles[i]
		k := sel(r)
		if k < 0 {
			continue
		}
		o := p1.occs[k]
		cle := [2]int{o.chunk, o.pkIndex}
		m[cle] = append(m[cle], cmLiaison{eid: r.eid, ti: o.ti})
		n++
	}
	return m, n
}

// rnOraclesA : R-L1 (a).
func rnOraclesA(p1 *rnP1) []rnVar {
	iii := func(garde func(o rnOcc) bool) func(r *rnCible) int {
		return func(r *rnCible) int {
			if !r.lie || p1.occs[r.meilleure].region != rnRegionRejet || !garde(p1.occs[r.meilleure]) {
				return -1
			}
			return r.meilleure
		}
	}
	var vs []rnVar
	for _, x := range []struct {
		nom string
		g   func(o rnOcc) bool
	}{
		{"a-(iii')-rejet", func(rnOcc) bool { return true }},
		{"a-(iii')-rejet-Q-ferme-sain", func(o rnOcc) bool { return o.sain }},
		{"a-(iii')-rejet-Q-non-ferme-sain", func(o rnOcc) bool { return !o.sain }},
	} {
		m, n := rnLier(p1, iii(x.g))
		vs = append(vs, rnVar{nom: x.nom, v: cmVariante{oracle: m}, nl: n})
	}
	return vs
}

// rnFiltresP6 : les filtres de R-P6 ; `propre+alloc+suivant` est le controle des mesures bis 3.
var rnFiltresP6 = []struct {
	nom string
	f   func(o rnOcc) bool
}{
	{"P6-propre+alloc+suivant", func(o rnOcc) bool { return o.ok && o.propre && o.alloc && o.suivant }},
	{"P6-ferme", func(o rnOcc) bool { return o.ok && o.sain }},
	{"P6-ferme+suivant", func(o rnOcc) bool { return o.ok && o.sain && o.suivant }},
	{"P6-ferme+alloc", func(o rnOcc) bool { return o.ok && o.sain && o.alloc }},
	{"P6-ferme-(i)(ii)", func(o rnOcc) bool { return o.ok && o.sain && !strings.HasPrefix(o.region, "(iii") }},
	{"P6-ferme-(iii')", func(o rnOcc) bool { return o.ok && o.sain && strings.HasPrefix(o.region, "(iii')") }},
}

// rnP6 retient, pour une cible NON liee par l oracle-NEW, l occurrence la plus proche du rejet, a
// plus de 3 paquets, qui passe le filtre (regle de [b3Plus]).
func rnP6(p1 *rnP1, ids []int, f func(o rnOcc) bool) int {
	best := -1
	for _, k := range ids {
		o := p1.occs[k]
		if o.dist <= 3 || !f(o) {
			continue
		}
		if best < 0 || o.j > p1.occs[best].j {
			best = k
		}
	}
	return best
}

// rnOraclesP6 : R-P6, et l oracle (i)+(ii) seul et avec `P6-ferme`.
func rnOraclesP6(p1 *rnP1) []rnVar {
	var vs []rnVar
	for _, x := range rnFiltresP6 {
		m, n := rnLier(p1, func(r *rnCible) int {
			if r.lie {
				return -1
			}
			return rnP6(p1, r.occs, x.f)
		})
		vs = append(vs, rnVar{nom: x.nom, v: cmVariante{oracle: m}, nl: n})
	}
	i12 := func(r *rnCible) int {
		if !r.lie {
			return -1
		}
		reg := p1.occs[r.meilleure].region
		if strings.HasPrefix(reg, "(i) ") || strings.HasPrefix(reg, "(ii) ") {
			return r.meilleure
		}
		return -1
	}
	m, n := rnLier(p1, i12)
	vs = append(vs, rnVar{nom: "oracle-(i)+(ii)", v: cmVariante{oracle: m}, nl: n})
	m2, n2 := rnLier(p1, func(r *rnCible) int {
		if k := i12(r); k >= 0 {
			return k
		}
		if r.lie {
			return -1
		}
		return rnP6(p1, r.occs, rnFiltresP6[1].f)
	})
	vs = append(vs, rnVar{nom: "oracle-(i)+(ii)+P6-ferme", v: cmVariante{oracle: m2}, nl: n2})
	return vs
}

// temoinsP6 : le temoin de chaque filtre — eid rejetes NON lies et leurs temoins qui ont au moins
// une occurrence passant le filtre, par classe de distance (regle de `p6_filtres`).
func (a *rnP1) temoinsP6() {
	for i := range a.cibles {
		r := &a.cibles[i]
		if r.lie {
			continue
		}
		for _, x := range rnFiltresP6 {
			for _, cl := range []string{"4-10 paquets avant", "plus de 10 paquets avant"} {
				if rnUneQuiPasse(a, r.occs, x.f, cl) {
					a.t.add("p6_temoin", cmJoindre(x.nom, cl, "rejete"), cmCompte{n: 1, paquets: r.nPaq, horsCadre: r.horsCadre})
				}
				if r.aTemoin && rnUneQuiPasse(a, r.occsT, x.f, cl) {
					a.t.un("p6_temoin", cmJoindre(x.nom, cl, "temoin"))
				}
			}
		}
		if r.aTemoin {
			a.t.un("p6_temoin", "cibles non liees avec temoin")
		}
	}
}

func rnUneQuiPasse(a *rnP1, ids []int, f func(o rnOcc) bool, cl string) bool {
	for _, k := range ids {
		if o := a.occs[k]; o.dist > 3 && classeDeDistance(o.dist) == cl && f(o) {
			return true
		}
	}
	return false
}

// rnConditions : R-L1 (d) — trois conditions d activation du localisateur elargi, calculees sur
// la marche de reference (tous les NEW propres qu elle lit ; la variante « paquets sans liste
// d evenements » a ete mesuree et ne separe pas les films) : film entier, chunk, cumul des chunks
// ANTERIEURS (causal). Rend [film, chunk, cumul].
func rnConditions(f *cmFilm, p1 *rnP1, out map[string][]string) [3]func(int) bool {
	tot := [2]int{}
	for _, s := range p1.score {
		tot[0] += s[0]
		tot[1] += s[1]
	}
	film := tot[0] >= 30 && float64(tot[1]) >= rnSeuilScore*float64(tot[0])
	chunk := map[int]bool{}
	cumul := map[int]bool{}
	acc := [2]int{}
	for _, c := range f.fc.ChunkNumbers() {
		s := p1.score[c]
		chunk[c] = s[0] >= 5 && float64(s[1]) >= rnSeuilScore*float64(s[0])
		cumul[c] = acc[0] >= 30 && float64(acc[1]) >= rnSeuilScore*float64(acc[0])
		out["scores"] = append(out["scores"], rnTab(f.id, f.build, c, s[0], s[1], s[2], s[3], chunk[c], cumul[c], film))
		acc[0] += s[0]
		acc[1] += s[1]
	}
	return [3]func(int) bool{func(int) bool { return film }, func(c int) bool { return chunk[c] },
		func(c int) bool { return cumul[c] }}
}
