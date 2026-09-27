package objectives

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : le tri du pied de film. Deux evenements th=10
// a la MEME milliseconde (deux joueurs, une meme capture) : leur rang ordonne les actions publiees
// (numero `Seq` dense) et `captureScorer` garde le PREMIER du plus grand instant. Il doit tenir a
// la position de l'evenement dans le pied, pas au tri.

import (
	"math/rand"
	"testing"
)

func TestTrierPied_ExAequoDepartagesParPosition(t *testing.T) {
	const n = 48
	in := make([]piedLu, n)
	for i := range in {
		in[i] = piedLu{ev: FooterEvent{TimeMS: i % 3, Slot: i % 4}, bit: 100 * i}
	}
	rand.New(rand.NewSource(20260927)).Shuffle(n, func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierPied(in)
	for i := 1; i < n; i++ {
		a, b := in[i-1], in[i]
		if a.ev.TimeMS > b.ev.TimeMS || (a.ev.TimeMS == b.ev.TimeMS && a.bit >= b.bit) {
			t.Fatalf("rang %d : %+v puis %+v — ordre (instant, position) rompu", i, a, b)
		}
	}
}
