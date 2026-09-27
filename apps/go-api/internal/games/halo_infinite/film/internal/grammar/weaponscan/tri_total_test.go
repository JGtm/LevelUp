package weaponscan

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : les deux tris de [ScanFireEventsB5]. Le
// premier decide de la DEDUPLICATION (le premier evenement d'un amas gagne, et son arme est
// comptee dans `match_weapon_shots`) ; le second rend la liste publiee. Plus de douze elements,
// peu de cles : `sort.Slice` y melangeait les ex aequo.

import (
	"math/rand"
	"testing"
)

func TestTrierParPositionDOctet_ExAequoDansLOrdreDuBalayage(t *testing.T) {
	const n = 48
	in := make([]FireEvent, n)
	for i := range in {
		in[i] = FireEvent{BytePos: i % 4, FireSeq: i}
	}
	trierParPositionDOctet(in)
	for i := 1; i < n; i++ {
		a, b := in[i-1], in[i]
		if a.BytePos > b.BytePos || (a.BytePos == b.BytePos && a.FireSeq >= b.FireSeq) {
			t.Fatalf("rang %d : %+v puis %+v — l'ordre du balayage doit departager", i, a, b)
		}
	}
}

func TestTrierParInstant_ExAequoDepartagesParOctet(t *testing.T) {
	const n = 48
	in := make([]FireEvent, n)
	for i := range in {
		in[i] = FireEvent{TimestampMS: float64(i % 3), BytePos: 10 * i}
	}
	rand.New(rand.NewSource(20260927)).Shuffle(n, func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierParInstant(in)
	for i := 1; i < n; i++ {
		a, b := in[i-1], in[i]
		if a.TimestampMS > b.TimestampMS || (a.TimestampMS == b.TimestampMS && a.BytePos >= b.BytePos) {
			t.Fatalf("rang %d : %+v puis %+v — (instant, octet) rompu", i, a, b)
		}
	}
}
