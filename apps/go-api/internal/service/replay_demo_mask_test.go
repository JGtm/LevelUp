package service

// replay_demo_mask_test.go — masque des joueurs réels du rejeu servi en démo (décision D-1) et
// liste blanche des rejeux figés (décision D-2).

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/domain/title"
)

const (
	maskXUIDReel = "2533274812345678"
	maskBot      = "bid(2.0)"
)

func documentAvecJoueurReel() replaydoc.ReplayDocument {
	return replaydoc.ReplayDocument{
		Tracks: []replaydoc.Track{
			{Slot: 1, XUID: maskXUIDReel, Name: "VraiNom", Points: []replaydoc.Point{}},
			{Slot: 2, Bot: maskBot, Name: "Bot Marine", Points: []replaydoc.Point{}},
		},
		Identity: &replaydoc.IdentitySection{Players: []replaydoc.IdentityPlayer{
			{FilmIndex: 0, XUID: maskXUIDReel, Name: "VraiNom"},
			{FilmIndex: 1, Bid: maskBot, Name: "Bot Marine"},
		}},
	}
}

func TestMaskReplayIdentities_RemplaceXUIDEtNomsGardeLesBots(t *testing.T) {
	masked, err := maskReplayIdentities(documentAvecJoueurReel(), []domain.DemoReplayIdentity{
		{XUID: maskXUIDReel, DemoXUID: "0000000000000002", DemoGamertag: "DemoPlayer3"},
	})
	if err != nil {
		t.Fatalf("maskReplayIdentities : %v", err)
	}
	raw, _ := json.Marshal(masked)
	corps := string(raw)
	if strings.Contains(corps, maskXUIDReel) || strings.Contains(corps, "VraiNom") {
		t.Errorf("identité réelle encore présente : %s", corps)
	}
	if masked.Tracks[0].XUID != "0000000000000002" || masked.Tracks[0].Name != "DemoPlayer3" {
		t.Errorf("piste 0 = (%q, %q), attendu l'identité démo", masked.Tracks[0].XUID, masked.Tracks[0].Name)
	}
	if masked.Identity.Players[0].Name != "DemoPlayer3" {
		t.Errorf("table d'identités : nom %q, attendu DemoPlayer3", masked.Identity.Players[0].Name)
	}
	if masked.Tracks[1].Name != "Bot Marine" || masked.Tracks[1].Bot != maskBot {
		t.Errorf("le bot doit traverser intact : %+v", masked.Tracks[1])
	}
}

func TestDemoReplayAllowlist_ServesIndexedMatchesOnly(t *testing.T) {
	demo := t.TempDir()
	layout := title.NewDemoLayout(demo)
	a := NewDemoReplayAllowlist(layout)
	ctx := context.Background()
	if a.Allows(ctx, title.DefaultSlug, "abcd1234-0000") {
		t.Error("sans index, aucun rejeu ne doit être servi")
	}
	index := domain.DemoReplayIndex{Matches: []domain.DemoReplayIndexEntry{
		{MatchID: "abcd1234-1111-2222-3333-444455556666", Mode: "ctf", SchemaVersion: 1},
	}}
	raw, _ := json.Marshal(index)
	path := layout.ReplayIndexPath(title.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"abcd1234", "abcd1234-1111-2222-3333-444455556666"} {
		if !a.Allows(ctx, title.DefaultSlug, id) {
			t.Errorf("match figé %q refusé", id)
		}
	}
	if a.Allows(ctx, title.DefaultSlug, "ffff0000") {
		t.Error("match hors index servi")
	}
	if a.Allows(ctx, "halo_5", "abcd1234") {
		t.Error("l'index d'un titre ne vaut pas pour un autre")
	}
}
