package filmdec

// vehicules_v7_correl_test.go — INSTRUMENT (lot V7) : LE TEMOIN NATUREL DU CORPUS. Un type
// d'evenement qui date la destruction d'un VEHICULE ne peut pas exister dans un film SANS
// vehicule, et son effectif doit suivre le nombre de vehicules qui disparaissent.
//
// POURQUOI CE TROISIEME ANGLE. Les deux premiers cherchent le signal DANS un film :
// `TestV7Refs` dans les bits de l'evenement, `TestV7Temps` dans son instant. Celui-ci le cherche
// ENTRE les films, et c'est le seul qui ne suppose RIEN de la grammaire de charge : il ne lit que
// le type de tete, dont le cadrage est prouve bit-exact (garde-fou `fire_events == head type36`).
//
// L'ORACLE, ECRIT AVANT LA MESURE. Soit T le type cherche. Alors :
//
//	(a) dans un film a bande `ti=40` VIDE (arene sans vehicule), l'effectif de T doit etre NUL ;
//	(b) sur les films a vehicules, l'effectif de T doit CROITRE avec le nombre de vies de
//	    vehicule dont la disparition est attestee (`fins bornees`) ;
//	(c) l'ordre de grandeur doit tenir : quelques unites a quelques dizaines par film, pas des
//	    milliers — une destruction est rare.
//
// LE TEMOIN EST DANS LA TABLE ELLE-MEME : les types de tete des evenements qui n'ont RIEN a voir
// avec les vehicules (le tir, la recharge, la lunette) doivent, eux, etre PRESENTS et nombreux
// dans les films sans vehicule. Une colonne « films sans vehicule » a zero pour TOUS les types
// dirait que le corpus n'a pas de vrai temoin negatif, et la mesure serait sans valeur.
//
// Garde d'environnement V7_ROOT (+ V7_FILMS ou V7_MAX) : sans elle, tout SKIP.

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// v7CorrelMax borne le nombre de films balayes quand V7_FILMS n'est pas donne. Le balayage lit
// les images-cles ET tous les paquets delta de chaque film : c'est la mesure la plus lourde du
// lot par film, mais la seule qui ait besoin de BEAUCOUP de films.
const v7CorrelMax = 60

// v7SansVehiculeMax : au-dela de ce nombre de vies `ti=40` recensees, un film est dit A
// VEHICULES. Le seuil est a 2 et non a 0 parce que le cadrage du 2026-08-31 a releve des cartes
// d'arene qui portent quelques entites `ti=40` statiques (Illusion, 7 slots, 17 records delta) —
// exiger une bande vide ecarterait ces temoins-la, qui sont precisement les plus utiles.
const v7SansVehiculeMax = 2

// v7FilmStat est ce qu'un film rend a cet instrument.
type v7FilmStat struct {
	id            string
	bande         int
	vies, finsBor int
	hist          map[int]int
}

// v7ScanCorrel balaie un film : recensement `ti=40` et histogramme des types de tete.
func v7ScanCorrel(dir string) v7FilmStat {
	st := v7FilmStat{id: filepath.Base(filepath.Clean(dir)), hist: map[int]int{}}
	k := ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI)
	st.bande, st.vies = len(k.Band), len(k.SeenUS)
	for _, seen := range k.SeenUS {
		if len(seen) > 0 && v7FirstAfter(k.TimesUS, seen[len(seen)-1]) > 0 {
			st.finsBor++
		}
	}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta || p.Size < 1 {
				continue
			}
			if ty, present := PacketHeadEventType(p.Payload(data)); present {
				st.hist[ty]++
			}
		}
	}
	return st
}

// v7Pearson rend le coefficient de correlation lineaire de deux series de meme longueur.
func v7Pearson(x, y []float64) float64 {
	n := float64(len(x))
	if n < 2 {
		return 0
	}
	var sx, sy float64
	for i := range x {
		sx, sy = sx+x[i], sy+y[i]
	}
	mx, my := sx/n, sy/n
	var num, dx, dy float64
	for i := range x {
		a, b := x[i]-mx, y[i]-my
		num, dx, dy = num+a*b, dx+a*a, dy+b*b
	}
	if dx == 0 || dy == 0 {
		return 0
	}
	return num / math.Sqrt(dx*dy)
}

// v7CorrelDirs rend le corpus : V7_FILMS s'il est donne, sinon les premiers repertoires de
// V7_ROOT (V7_MAX en borne).
func v7CorrelDirs(t *testing.T) []string {
	t.Helper()
	dirs := v7FilmDirs(t)
	maxN := v7CorrelMax
	if s := os.Getenv("V7_MAX"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("V7_MAX : %q n'est pas un entier", s)
		}
		maxN = v
	}
	if os.Getenv("V7_FILMS") == "" && len(dirs) > maxN {
		dirs = dirs[:maxN]
	}
	return dirs
}

// TestV7Correlation — LE TEMOIN NATUREL. Une ligne par type de tete.
func TestV7Correlation(t *testing.T) {
	dirs := v7CorrelDirs(t)
	var stats []v7FilmStat
	for _, d := range dirs {
		if CountFilmChunks(d) == 0 {
			continue
		}
		stats = append(stats, v7ScanCorrel(d))
	}
	if len(stats) == 0 {
		t.Skip("aucun film lisible")
	}
	v7PublieCorrel(t, stats)
}

// v7PublieCorrel ecrit la table de correlation.
func v7PublieCorrel(t *testing.T, stats []v7FilmStat) {
	t.Helper()
	sans, avec := 0, 0
	var fins []float64
	for _, s := range stats {
		if s.vies <= v7SansVehiculeMax {
			sans++
		} else {
			avec++
		}
		fins = append(fins, float64(s.finsBor))
	}
	types := map[int]bool{}
	for _, s := range stats {
		for ty := range s.hist {
			types[ty] = true
		}
	}
	var tys []int
	for ty := range types {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 CORRELATION — %d films (%d SANS vehicule au sens <= %d vies, %d avec) ==",
		len(stats), sans, v7SansVehiculeMax, avec)
	t.Logf("%-5s %-9s %-7s %-9s %-11s %-11s %-8s",
		"type", "total", "films", "moy/film", "moy SANS veh", "moy AVEC veh", "r(fins)")
	for _, ty := range tys {
		var serie []float64
		total, present, sumSans, sumAvec := 0, 0, 0, 0
		for _, s := range stats {
			n := s.hist[ty]
			serie = append(serie, float64(n))
			total += n
			if n > 0 {
				present++
			}
			if s.vies <= v7SansVehiculeMax {
				sumSans += n
			} else {
				sumAvec += n
			}
		}
		ms, ma := 0.0, 0.0
		if sans > 0 {
			ms = float64(sumSans) / float64(sans)
		}
		if avec > 0 {
			ma = float64(sumAvec) / float64(avec)
		}
		t.Logf("%-5d %-9d %-7d %9.1f %11.2f %11.2f %8.3f",
			ty, total, present, float64(total)/float64(len(stats)), ms, ma,
			v7Pearson(serie, fins))
	}
	t.Logf("-- detail par film (id · bande · vies · fins bornees · types de tete distincts) --")
	for _, s := range stats {
		t.Logf("%-10s bande %4d · vies %4d · fins %4d · %2d types", s.id, s.bande, s.vies,
			s.finsBor, len(s.hist))
	}
}
