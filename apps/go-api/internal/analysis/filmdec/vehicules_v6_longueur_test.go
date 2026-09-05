package filmdec

// vehicules_v6_longueur_test.go — INSTRUMENT (lot V6) : LA LONGUEUR D'UN EVENEMENT PAR TYPE.
//
// D'OU VIENT CETTE MESURE. `TestV6Ancrage` etablit que le score de PROFONDEUR de marche ECS ne
// designe le vrai debut de trame que dans 23,3 % des cas PAR PAQUET — trop faible pour ancrer un
// paquet isole. Mais il est SANS BIAIS : agrege sur des centaines de paquets du MEME type de
// tete, son maximum doit tomber sur le vrai debut de trame, si celui-ci est CONSTANT.
//
// L'HYPOTHESE TESTEE : pour un type d'evenement donne, la liste (refs gardees comprises) a une
// longueur FIXE. Elle n'est pas gratuite — `fire_events.go` lit le record de type 36 a des
// offsets de bit ABSOLUS (attaquant a 36, arme a 44/76, drapeaux a 108), ce qui suppose deja que
// la section de refs de ce type ne varie pas.
//
// LE CONTROLE EST INTERNE : les types 8 (embarquement) et 22 (sortie) ont une longueur CONNUE au
// bit pres. Le profil agrege doit y culminer AU BON ENDROIT — sinon la methode est refutee, et
// c'est ce chiffre-la qui est publie.
//
// Garde d'environnement V6_ROOT / V6_FILMS : sans elle, tout SKIP.

import (
	"os"
	"sort"
	"strconv"
	"testing"
)

// v6ModeKey rend la cle la plus frequente d'un histogramme (0 si vide).
func v6ModeKey(h map[int]int) int {
	best, bestN := 0, -1
	for k, v := range h {
		if v > bestN || (v == bestN && k < best) {
			best, bestN = k, v
		}
	}
	return best
}

// v6ProfileMax : borne du balayage de candidats de debut de trame (bits depuis le debut du
// payload). Les evenements vehicule finissent vers 40-60 bits ; le record de tir lit jusqu'au
// bit 143. 512 couvre largement.
const v6ProfileMax = 512

// v6ProfileSamples : nombre de paquets echantillonnes par type (le balayage coute
// v6ProfileMax decodages par paquet).
const v6ProfileSamples = 250

// v6TypeProfile accumule, pour un type de tete, la somme des profondeurs par candidat S.
type v6TypeProfile struct {
	sum   []int
	vote  []int // PLURALITE : +1 a chaque candidat de profondeur maximale du paquet
	n     int
	trueS map[int]int // pour 8/22 : distribution du vrai S (controle de constance)
}

func newV6TypeProfile() *v6TypeProfile {
	return &v6TypeProfile{sum: make([]int, v6ProfileMax+1), vote: make([]int, v6ProfileMax+1),
		trueS: map[int]int{}}
}

// add balaie un paquet, cumule la profondeur et vote pour ses maxima.
func (p *v6TypeProfile) add(pay []byte, w *World, cfg FrameConfig) {
	p.n++
	d := make([]int, v6ProfileMax+1)
	best := 0
	for S := 2; S <= v6ProfileMax && S < len(pay)*8; S++ {
		d[S] = v6Depth(pay, S, w, cfg)
		if d[S] > 0 {
			p.sum[S] += d[S]
		}
		if d[S] > best {
			best = d[S]
		}
	}
	if best <= 0 {
		return
	}
	for S := range d {
		if d[S] == best {
			p.vote[S]++
		}
	}
}

// v6Rank rend les 5 meilleurs indices d'un vecteur de score (egalites tranchees par l'indice).
func v6Rank(v []int) []int {
	idx := make([]int, 0, len(v))
	for S := range v {
		idx = append(idx, S)
	}
	sort.Slice(idx, func(i, j int) bool {
		if v[idx[i]] != v[idx[j]] {
			return v[idx[i]] > v[idx[j]]
		}
		return idx[i] < idx[j]
	})
	if len(idx) > 5 {
		idx = idx[:5]
	}
	return idx
}

// v6ProfileTypes : les types de tete profiles (les plus frequents du corpus + les deux
// evenements vehicule de controle).
var v6ProfileTypes = []int{8, 22, 36, 0, 82, 5, 15, 21, 38, 7, 9, 75, 76, 1}

// TestV6Longueur : profil agrege de profondeur par type de tete.
func TestV6Longueur(t *testing.T) {
	dirs := v6FilmDirs(t)
	release := LockProcessDecode()
	defer release()
	cfg := DefaultFrameConfig()
	types := v6ProfileTypes
	if l := os.Getenv("V6_TYPES"); l != "" {
		types = nil
		for _, s := range splitComma(l) {
			v, err := strconv.Atoi(s)
			if err != nil {
				t.Fatalf("V6_TYPES: %q n'est pas un entier", s)
			}
			types = append(types, v)
		}
	}
	want := map[int]bool{}
	for _, ty := range types {
		want[ty] = true
	}
	prof := map[int]*v6TypeProfile{}
	for _, d := range dirs {
		v6ProfileFilm(d, want, prof, cfg)
	}
	t.Logf("== V6 LONGUEUR — profil agrege de profondeur ECS par type de tete ==")
	for _, ty := range types {
		p := prof[ty]
		if p == nil || p.n == 0 {
			continue
		}
		sline, vline := "", ""
		for _, s := range v6Rank(p.sum) {
			sline += " " + itoa(s) + "(" + itoa(p.sum[s]) + ")"
		}
		for _, s := range v6Rank(p.vote) {
			vline += " " + itoa(s) + "(" + itoa(p.vote[s]) + ")"
		}
		t.Logf("type %3d · n=%3d · SOMME top5 :%s", ty, p.n, sline)
		t.Logf("             VOTE  top5 :%s", vline)
		if len(p.trueS) > 0 {
			t.Logf("        CONTROLE vrai S mesure :%s · somme(vrai)=%d vote(vrai)=%d",
				v6TopHist(p.trueS, 8), p.sum[v6ModeKey(p.trueS)], p.vote[v6ModeKey(p.trueS)])
		}
	}
}

// v6ProfileFilm alimente les profils depuis un film.
func v6ProfileFilm(dir string, want map[int]bool, prof map[int]*v6TypeProfile, cfg FrameConfig) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return
	}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		pks := WalkPackets(data)
		w := v6ChunkWorld(reg, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 8 {
				continue
			}
			pay := pk.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present || !want[ty] {
				continue
			}
			p := prof[ty]
			if p == nil {
				p = newV6TypeProfile()
				prof[ty] = p
			}
			if end, ok := v6EventEnd(pay, 1, ty); ok {
				p.trueS[end+1]++
			}
			if p.n >= v6ProfileSamples {
				continue
			}
			p.add(pay, w, cfg)
		}
	}
}
