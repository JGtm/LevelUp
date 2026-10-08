package scheduler

// spartan_customization_bearer_internal_test.go — ordre de choix du porteur et
// contexte de lecture posé pour le refresher du titre.

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// TestOrderBearerCandidates_AdminPuisOrdreAlphabetique : l'admin désigné passe en
// tête, les autres suivent par gamertag sans tenir compte de la casse, un gamertag
// déclaré pour deux titres n'apparaît qu'une fois et un profil sans xuid est écarté.
// L'ordre ne dépend pas de l'ordre de lecture du fichier.
func TestOrderBearerCandidates_AdminPuisOrdreAlphabetique(t *testing.T) {
	players := []domain.PlayerSummary{
		{Gamertag: "Trimbutton", XUID: "5", TitleSlug: "halo_infinite", AuthOnly: true},
		{Gamertag: "JGtm", XUID: "2", TitleSlug: "halo_infinite"},
		{Gamertag: "chocoboflor", XUID: "1", TitleSlug: "halo_infinite"},
		{Gamertag: "DankerGlue", XUID: "3", TitleSlug: "halo_infinite", AuthOnly: true},
		{Gamertag: "JGtm", XUID: "2", TitleSlug: "halo_5"},
		{Gamertag: "SansXuid", XUID: "", TitleSlug: "halo_infinite"},
	}

	got := orderBearerCandidates(players, "jgtm")
	want := []bearerCandidate{
		{gamertag: "JGtm", xuid: "2"},
		{gamertag: "chocoboflor", xuid: "1"},
		{gamertag: "DankerGlue", xuid: "3"},
		{gamertag: "Trimbutton", xuid: "5"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordre des porteurs avec admin :\n got %+v\nwant %+v", got, want)
	}

	reversed := make([]domain.PlayerSummary, len(players))
	for i, p := range players {
		reversed[len(players)-1-i] = p
	}
	if again := orderBearerCandidates(reversed, "jgtm"); !reflect.DeepEqual(again, want) {
		t.Fatalf("l'ordre dépend de l'ordre de lecture :\n got %+v\nwant %+v", again, want)
	}

	sansAdmin := orderBearerCandidates(players, "")
	if sansAdmin[0].gamertag != "chocoboflor" || sansAdmin[1].gamertag != "DankerGlue" {
		t.Fatalf("sans admin, l'ordre doit être alphabétique : %+v", sansAdmin)
	}
}

// TestReaderContext_TokenPropreEtPorteur : avec le token du joueur, sujet et porteur
// sont le joueur ; avec un porteur, le sujet reste le joueur lu et le budget de
// débit est imputé au porteur.
func TestReaderContext_TokenPropreEtPorteur(t *testing.T) {
	tokens := &domain.HaloTokens{SpartanToken: "spartan"}

	own := readerContext(context.Background(), tokens, "sujet", "sujet")
	if ctxkeys.HaloXUID(own) != "sujet" || ctxkeys.TokensOwnerXUID(own) != "sujet" {
		t.Fatalf("token propre : sujet=%q porteur=%q, attendu sujet/sujet",
			ctxkeys.HaloXUID(own), ctxkeys.TokensOwnerXUID(own))
	}

	borne := readerContext(context.Background(), tokens, "porteur", "sujet")
	if ctxkeys.HaloXUID(borne) != "sujet" {
		t.Errorf("porteur : sujet=%q, attendu le joueur lu", ctxkeys.HaloXUID(borne))
	}
	if ctxkeys.TokensOwnerXUID(borne) != "porteur" {
		t.Errorf("porteur : budget imputé à %q, attendu le porteur", ctxkeys.TokensOwnerXUID(borne))
	}
	if ctxkeys.HaloTokens(borne) != tokens {
		t.Error("porteur : les tokens posés ne sont pas ceux du lease")
	}
}
