package service

// replay_shield_cycles_test.go — le document servi porte, par joueur mesuré, les instants des bris et des
// recharges de bouclier en millisecondes de l'axe du document, avec la définition de `analysis/shieldcycle`.

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
)

func shieldPoint(t int, sh *float32) replay.Point { return replay.Point{T: t, Sh: sh} }

func shieldVal(v float32) *float32 { return &v }

func shieldDoc(interval int) *replay.ReplayDocument {
	return &replay.ReplayDocument{
		SchemaVersion:   115,
		MatchID:         "0000cb01",
		FrameCount:      100,
		FrameIntervalMS: interval,
		Tracks: []replay.Track{
			// Mesuré : plein, bris à l'image 10, recharge à l'image 80 ; second bris à 90.
			{XUID: "111", Points: []replay.Point{
				shieldPoint(0, shieldVal(1)), shieldPoint(5, nil), shieldPoint(10, shieldVal(0)),
				shieldPoint(40, shieldVal(0.5)), shieldPoint(80, shieldVal(1)), shieldPoint(90, shieldVal(0)),
			}},
			// Mesuré, aucun événement : toujours plein.
			{XUID: "222", Points: []replay.Point{shieldPoint(0, shieldVal(1)), shieldPoint(50, shieldVal(1))}},
			// Non mesuré : aucune mesure de bouclier.
			{XUID: "333", Points: []replay.Point{shieldPoint(0, nil), shieldPoint(50, nil)}},
			// Bot : pas de xuid, servi sous `bot:<nom>` ; première mesure à 0 = bris, puis recharge.
			{Bot: "Spartan", Points: []replay.Point{shieldPoint(0, shieldVal(0)), shieldPoint(5, shieldVal(1))}},
		},
	}
}

func TestShieldCyclesOf(t *testing.T) {
	got := shieldCyclesOf(shieldDoc(50))
	want := map[string]replaydoc.ShieldCycle{
		"111":         {Breaks: []int{500, 4500}, Recharges: []int{4000}},
		"222":         {Breaks: []int{}, Recharges: []int{}},
		"bot:Spartan": {Breaks: []int{0}, Recharges: []int{250}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shieldCyclesOf = %+v, attendu %+v", got, want)
	}
	if _, ok := got["333"]; ok {
		t.Error("un joueur sans mesure n'a pas d'entrée (non mesuré)")
	}
}

func TestShieldCyclesOf_SansHorlogeOuSansMesure(t *testing.T) {
	if shieldCyclesOf(nil) != nil {
		t.Error("document nil : aucune entrée")
	}
	if shieldCyclesOf(shieldDoc(0)) != nil {
		t.Error("document sans pas d'image : aucune entrée, jamais un instant deviné")
	}
	doc := shieldDoc(50)
	doc.Tracks = doc.Tracks[2:3]

	if shieldCyclesOf(doc) != nil {
		t.Error("aucun joueur mesuré : le champ est absent, pas une table vide")
	}
}

// TestGetReplay_ServiLesCyclesDeBouclier : le champ traverse la requête et se sérialise avec des listes
// vides (`[]`, jamais `null`) pour un joueur mesuré sans événement.
func TestGetReplay_ServiLesCyclesDeBouclier(t *testing.T) {
	raw, err := json.Marshal(shieldDoc(50))
	if err != nil {
		t.Fatalf("sérialisation : %v", err)
	}
	root := artefactAt(t, title.DefaultSlug, "0000cb01", string(raw))
	got, err := NewReplayService(title.DefaultSlug, root, nil, nil).GetReplay(context.Background(), "0000cb01")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	body, err := json.Marshal(got.ReplayDocument.ShieldCycles)
	if err != nil {
		t.Fatalf("sérialisation du champ : %v", err)
	}
	const want = `{"111":{"breaks":[500,4500],"recharges":[4000]},"222":{"breaks":[],"recharges":[]},"bot:Spartan":{"breaks":[0],"recharges":[250]}}`
	if string(body) != want {
		t.Fatalf("shieldCycles = %s, attendu %s", body, want)
	}
}
