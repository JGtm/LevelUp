// Package service — career_live_fetcher_test.go : tests unitaires de
// CareerFetcherFactoryFromTokens, en particulier l'attribution du budget API au
// PORTEUR réel des tokens (finding ID3, revue 2026-07).
package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/ratebudget"
)

// TestCareerFetcherFactory_BudgetKeyedOnTokensOwner verrouille le finding ID3 :
// une page X consultée par une session Y (le PORTEUR des tokens) doit débiter le
// bucket ratebudget de Y — le porteur réel du quota Xbox — et laisser le bucket
// de X (le xuid de page, imposé par forcePageIdentityXUID) intact.
//
// Observabilité : ratebudget.ForXUID crée l'entrée du registre au premier appel.
// Donc, après construction du fetcher, CurrentRPS(porteur) est non nul (entrée
// créée) et CurrentRPS(page) reste 0 (jamais passé à ForXUID).
func TestCareerFetcherFactory_BudgetKeyedOnTokensOwner(t *testing.T) {
	const (
		// xuids uniques à ce test (le registre ratebudget est process-wide).
		ownerY = "id3-owner-y-0001"
		pageX  = "id3-page-x-0002"
		rps    = 7
	)

	// Contexte reproduisant le chemin "Home forcé" : tokens du compte connecté Y,
	// puis sujet forcé sur la page X (forcePageIdentityXUID → WithHaloXUID).
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "spartan-xxx"}, ownerY)
	ctx = ctxkeys.WithHaloXUID(ctx, pageX)

	factory := CareerFetcherFactoryFromTokens(rps)
	fetcher := factory(ctx)
	if fetcher == nil {
		t.Fatal("fetcher attendu non-nil (tokens présents)")
	}

	if got := ratebudget.CurrentRPS(ownerY); got != float64(rps) {
		t.Errorf("bucket du porteur Y : CurrentRPS = %v, attendu %v (le budget se débite au porteur)", got, float64(rps))
	}
	if got := ratebudget.CurrentRPS(pageX); got != 0 {
		t.Errorf("bucket de la page X : CurrentRPS = %v, attendu 0 (jamais throttlé sur le xuid de page)", got)
	}
}

// TestCareerFetcherFactory_NoOwner_LocalLimiter : sans porteur connu (tokens posés
// avec xuid vide), aucun bucket partagé n'est créé — limiteur local, comportement
// historique préservé.
func TestCareerFetcherFactory_NoOwner_LocalLimiter(t *testing.T) {
	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "spartan-yyy"}, "")

	factory := CareerFetcherFactoryFromTokens(5)
	if fetcher := factory(ctx); fetcher == nil {
		t.Fatal("fetcher attendu non-nil même sans porteur (tokens présents)")
	}
	// Aucune entrée de registre pour "" (ForXUID n'est pas appelé quand owner == "").
	if got := ratebudget.CurrentRPS(""); got != 0 {
		t.Errorf("aucun bucket partagé attendu pour porteur vide, CurrentRPS = %v", got)
	}
}

// TestCareerFetcherFactory_NoTokens_Nil : sans tokens exploitables, la factory
// renvoie nil (dégradation silencieuse), sans créer de bucket.
func TestCareerFetcherFactory_NoTokens_Nil(t *testing.T) {
	ctx := ctxkeys.WithHaloAuth(context.Background(), nil, "id3-notokens-0003")
	factory := CareerFetcherFactoryFromTokens(5)
	if fetcher := factory(ctx); fetcher != nil {
		t.Fatalf("fetcher attendu nil sans tokens, obtenu %T", fetcher)
	}
	if got := ratebudget.CurrentRPS("id3-notokens-0003"); got != 0 {
		t.Errorf("aucun bucket ne doit être créé sans tokens, CurrentRPS = %v", got)
	}
}
