package killsource

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : les tris de killsource qui DECIDENT D'UNE
// SORTIE (un appariement, un « premier gagne », l'epinglage d'un bot) et dont la cle n'etait pas
// prouvee unique. Plus de douze elements et peu de cles : sous treize, `sort.Slice` passe par un
// tri par insertion, stable par accident — le hasard que DT-9 retire.

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

const (
	nExAequo      = 48
	graineExAequo = 20260927
)

// attenduStable : les rangs d'entree ranges par cle, puis par rang (l'ordre d'un tri STABLE).
func attenduStable(cles []int) []int {
	maxi := 0
	for _, c := range cles {
		maxi = max(maxi, c)
	}
	out := make([]int, 0, len(cles))
	for c := 0; c <= maxi; c++ {
		for i, k := range cles {
			if k == c {
				out = append(out, i)
			}
		}
	}
	return out
}

func TestTrierKillEvents_ExAequoDansLOrdreDuFilm(t *testing.T) {
	in := make([]killEventRec, nExAequo)
	for i := range in {
		in[i] = killEventRec{ms: i % 3, chunk: i % 2, pidx: i % 5, bit: i}
	}
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierKillEvents(in)
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		if [4]int{a.ms, a.chunk, a.pidx, a.bit} == [4]int{b.ms, b.chunk, b.pidx, b.bit} ||
			!lexInf([]int{a.ms, a.chunk, a.pidx, a.bit}, []int{b.ms, b.chunk, b.pidx, b.bit}) {
			t.Fatalf("rang %d : %+v puis %+v — ordre (instant, chunk, paquet, bit) rompu", i, a, b)
		}
	}
}

// lexInf : a < b dans l'ordre lexicographique.
func lexInf(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// Deux bots d'un meme slot gardent leur rang de DECOUVERTE (cf. [paquetsBotMeta]) : c'est lui qui
// decide lequel nomme l'indice au kill-feed.
func TestTrierBotsParSlot_ExAequoDansLOrdreDeDecouverte(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]bot, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = bot{Slot: i % 3, Name: fmt.Sprint(i)}
	}
	trierBotsParSlot(in)
	got := make([]int, len(in))
	for i, b := range in {
		_, _ = fmt.Sscan(b.Name, &got[i])
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("bots :\n got %v\nwant %v", got, want)
	}
}

func TestTrierMortsDeLaMarche_ExAequoDansLOrdreDuFilm(t *testing.T) {
	cles := make([]int, nExAequo)
	in := make([]deadRecord, nExAequo)
	for i := range in {
		cles[i] = i % 3
		in[i] = deadRecord{ms: i % 3, chunk: 1, pidx: 1, bit: -1, slot: i}
	}
	trierMortsDeLaMarche(in)
	got := make([]int, len(in))
	for i, d := range in {
		got[i] = d.slot
	}
	if want := attenduStable(cles); !reflect.DeepEqual(got, want) {
		t.Fatalf("dead-states :\n got %v\nwant %v", got, want)
	}
}

// Des candidats de la marche SANS position enregistree (`bit = -1`) d'un meme paquet ne se
// departageaient pas : leur ordre alimentait la population de la bijection.
func TestSortCandidates_BitAbsentDepartageParLeContenu(t *testing.T) {
	in := make([]candidate, nExAequo)
	for i := range in {
		in[i] = candidate{chunk: 3, pidx: 4, bit: -1, ms: 7, tag: uint32(i % 4), victim: i % 5, killer: i % 7, cat: i % 2}
	}
	rand.New(rand.NewSource(graineExAequo)).Shuffle(len(in), func(a, b int) { in[a], in[b] = in[b], in[a] })
	sortCandidates(in)
	for i := 1; i < len(in); i++ {
		a, b := in[i-1], in[i]
		ka := []int{a.chunk, a.pidx, a.bit, a.victim, a.killer, int(a.tag), a.cat, a.ms}
		kb := []int{b.chunk, b.pidx, b.bit, b.victim, b.killer, int(b.tag), b.cat, b.ms}
		if !lexInf(ka, kb) && !reflect.DeepEqual(ka, kb) {
			t.Fatalf("rang %d : %+v puis %+v — ordre des candidats rompu", i, a, b)
		}
	}
}
