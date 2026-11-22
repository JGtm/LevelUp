package replayshield

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/analysis/shieldcycle"
)

func f32(v float32) *float32 { return &v }

func TestLives(t *testing.T) {
	if Lives(nil) != nil {
		t.Fatal("un document nil ne rend aucune vie")
	}
	doc := &replay.ReplayDocument{Tracks: []replay.Track{
		{XUID: "a", Points: []replay.Point{{T: 1, Sh: f32(1)}, {T: 2}, {T: 3, Sh: f32(0)}}},
		{XUID: "b", Points: []replay.Point{{T: 4}}},
	}}
	got := Lives(doc)
	want := []shieldcycle.Life{
		{XUID: "a", Measures: []shieldcycle.Measure{{Frame: 1, Shield: 1}, {Frame: 3, Shield: 0}}},
		{XUID: "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lives = %+v, attendu %+v", got, want)
	}
	ev := shieldcycle.Cycles(got)
	if !reflect.DeepEqual(ev, map[string][]shieldcycle.Event{"a": {{Frame: 3, Kind: shieldcycle.Break}}}) {
		t.Fatalf("Cycles = %v", ev)
	}
}

func TestFrameMS(t *testing.T) {
	if FrameMS(10, 50) != 500 {
		t.Fatal("FrameMS(10, 50) != 500")
	}
}
