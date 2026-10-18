package replayartifacts

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// TestMarquerDerivations_NotifieLesSeulsMatchsEcrits : le notifieur branché apprend les matchs
// dont les dérivations sont écrites, pas ceux qu'une famille a ratés ; writer indisponible = rien.
func TestMarquerDerivations_NotifieLesSeulsMatchsEcrits(t *testing.T) {
	var titre string
	var recus []string
	SetNotifieurDerivations(func(_ context.Context, slug string, ids []string) {
		titre = slug
		recus = append(recus, ids...)
	})
	t.Cleanup(func() { SetNotifieurDerivations(nil) })

	dir := t.TempDir()
	lus := []artefactLu{
		{matchID: "ecrit", path: filepath.Join(dir, "ecrit.json"), doc: &replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}},
		{matchID: "rate", path: filepath.Join(dir, "rate.json"), doc: &replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}},
	}
	b := &bilanDerivations{}
	b.echec("rate")
	marquerDerivations(context.Background(), "halo_infinite", b, lus)
	if titre != "halo_infinite" || len(recus) != 1 || recus[0] != "ecrit" {
		t.Fatalf("notifiés : %q %v, attendu halo_infinite [ecrit]", titre, recus)
	}

	recus = nil
	sansWriter := &bilanDerivations{}
	sansWriter.writerIndisponible()
	marquerDerivations(context.Background(), "halo_infinite", sansWriter, lus)
	if len(recus) != 0 {
		t.Fatalf("writer indisponible : %v notifiés, attendu aucun", recus)
	}
}

func TestNotifierDerivations_SansNotifieurNeFaitRien(t *testing.T) {
	SetNotifieurDerivations(nil)
	notifierDerivations(context.Background(), "halo_infinite", []string{"m1"}) // ne panique pas
}
