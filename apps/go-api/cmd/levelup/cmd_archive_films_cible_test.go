package main

import "testing"

// Le ciblage `--match` d archive-films : identifiant complet ou identifiant court de film,
// insensible a la casse ; une cible vide accepte tout (comportement historique).
func TestMatchCible_CompletOuCourt(t *testing.T) {
	const id = "D9781168-1F2A-4B3C-9D4E-5F6A7B8C9D0E"
	var cible listeMatchs
	if err := cible.Set(" d9781168 "); err != nil {
		t.Fatalf("Set court: %v", err)
	}
	if err := cible.Set("51EBBC0F-0000-0000-0000-000000000000"); err != nil {
		t.Fatalf("Set complet: %v", err)
	}
	if !matchCible(cible, id) {
		t.Fatalf("l identifiant court doit cibler le match complet %s", id)
	}
	if !matchCible(cible, "51ebbc0f-0000-0000-0000-000000000000") {
		t.Fatalf("l identifiant complet doit cibler, insensible a la casse")
	}
	if matchCible(cible, "bf15f7ab-3224-42fd-86e2-a24e53e3277a") {
		t.Fatalf("un match hors cible ne doit pas etre retenu")
	}
	if !matchCible(nil, id) {
		t.Fatalf("une cible vide accepte tout")
	}
	if err := cible.Set("  "); err == nil {
		t.Fatalf("une valeur vide doit etre refusee")
	}
}
