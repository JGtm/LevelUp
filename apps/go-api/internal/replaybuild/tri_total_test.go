package replaybuild

// tri_total_test.go — LOT J10.1 (2026-09-27, DT-9) : deux bots qui rejoignent a la MEME
// milliseconde. L'ordre des relais est publie ; il ne doit pas tenir au tri.

import (
	"fmt"
	"math/rand"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func TestTrierSuccessions_ExAequoDepartagesParIndexEtNom(t *testing.T) {
	const n = 48
	in := make([]replay.Succession, n)
	for i := range in {
		in[i] = replay.Succession{SwitchMatchMS: int64(i % 3), FilmIndex: i % 5, BotName: fmt.Sprintf("b%02d", i)}
	}
	rand.New(rand.NewSource(20260927)).Shuffle(n, func(a, b int) { in[a], in[b] = in[b], in[a] })
	trierSuccessions(in)
	for i := 1; i < n; i++ {
		a, b := in[i-1], in[i]
		ka := fmt.Sprintf("%03d|%03d|%s", a.SwitchMatchMS, a.FilmIndex, a.BotName)
		kb := fmt.Sprintf("%03d|%03d|%s", b.SwitchMatchMS, b.FilmIndex, b.BotName)
		if ka >= kb {
			t.Fatalf("rang %d : %+v puis %+v — ordre (bascule, index, nom) rompu", i, a, b)
		}
	}
}
