// Package watcher — daemon_remove_title_test.go : retrait d'UN couple
// (joueur, titre) du suivi live (revue adversariale du 2026-09-16, P1).
package watcher

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// Seul le titre visé part ; l'autre titre du même xuid et l'homonyme restent.
func TestRemovePlayerTitle_NeRetireQueLeTitreVise(t *testing.T) {
	d := newRemovalDaemon(t)
	for _, p := range []domain.PlayerSummary{
		{Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite"},
		{Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_5"},
		{Gamertag: "Autre", XUID: "222", TitleSlug: "halo_infinite"},
	} {
		if err := d.AddPlayer(context.Background(), p); err != nil {
			t.Fatalf("AddPlayer %+v: %v", p, err)
		}
	}

	if !d.RemovePlayerTitle(context.Background(), "111", "halo_5") {
		t.Fatal("le couple (111, halo_5) devait etre retire")
	}
	watched := d.WatchedPlayers()
	if len(watched) != 2 {
		t.Fatalf("suivi restant = %+v, attendu 2 couples", watched)
	}
	for _, w := range watched {
		if w.XUID == "111" && w.TitleSlug != "halo_infinite" {
			t.Fatalf("le titre halo_infinite de 111 devait rester : %+v", watched)
		}
	}
}

// Idempotent : un couple absent rend false sans toucher au reste ; nil-safe.
func TestRemovePlayerTitle_IdempotentEtNilSafe(t *testing.T) {
	var nilDaemon *Daemon
	if nilDaemon.RemovePlayerTitle(context.Background(), "111", "halo_infinite") {
		t.Fatal("un daemon nil ne retire rien")
	}
	d := newRemovalDaemon(t)
	if err := d.AddPlayer(context.Background(), domain.PlayerSummary{
		Gamertag: "Spartan", XUID: "111", TitleSlug: "halo_infinite",
	}); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if d.RemovePlayerTitle(context.Background(), "111", "halo_5") {
		t.Fatal("halo_5 n'etait pas suivi : rien a retirer")
	}
	if d.RemovePlayerTitle(context.Background(), "999", "halo_infinite") {
		t.Fatal("xuid inconnu : rien a retirer")
	}
	if len(d.WatchedPlayers()) != 1 {
		t.Fatalf("le suivi ne doit pas avoir bouge : %+v", d.WatchedPlayers())
	}
	if !d.RemovePlayerTitle(context.Background(), "111", "") {
		t.Fatal("titre vide = titre par defaut (halo_infinite) : devait etre retire")
	}
	if len(d.WatchedPlayers()) != 0 {
		t.Fatalf("plus aucun suivi attendu : %+v", d.WatchedPlayers())
	}
}
