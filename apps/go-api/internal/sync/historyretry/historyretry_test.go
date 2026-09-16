package historyretry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/platform/auth/pool"
	"levelup/go-api/internal/sync/haloclient"
)

// historyretry_test.go — rejeu borné d'une page d'historique (D1, plan robustesse
// 2026-09-16). Aucun test ne dort : le seam `Sleep` est remplacé par un enregistreur de
// durées, et les durées DEMANDÉES sont assertées.
//
// CHACUN de ces tests rougit si la correction est retirée : sans Page, un 429 sur une page
// remonte tel quel et la passe se déclarait `success` sans rien avoir inséré.

// captureSleeps remplace le seam d'attente et rend le pointeur vers les durées demandées.
func captureSleeps(t *testing.T) *[]time.Duration {
	t.Helper()
	waits := make([]time.Duration, 0, 4)
	original := Sleep
	Sleep = func(d time.Duration) { waits = append(waits, d) }
	t.Cleanup(func() { Sleep = original })
	return &waits
}

// reponse : une réponse scriptée du fetch.
type reponse struct {
	entrees []string
	err     error
}

// fetchScripte rend les réponses dans l'ordre du script et compte les appels.
func fetchScripte(appels *int, script ...reponse) func() ([]string, error) {
	return func() ([]string, error) {
		*appels++
		if *appels > len(script) {
			return nil, fmt.Errorf("script épuisé après %d appels", len(script))
		}
		r := script[*appels-1]
		return r.entrees, r.err
	}
}

func httpErr(status int) error {
	return &haloclient.HTTPError{
		StatusCode: status,
		URL:        "https://halostats/matches",
		Err:        fmt.Errorf("HTTP %d", status),
	}
}

// (a) 429 puis 200 : la page est obtenue au rejeu IMMÉDIAT (le pool sert un autre slot),
// sans aucune attente.
func TestPage_429PuisSucces_RejoueSansAttendre(t *testing.T) {
	waits := captureSleeps(t)
	appels := 0
	entrees, err := Page(context.Background(), "Nuzzles", 225,
		fetchScripte(&appels, reponse{err: httpErr(429)}, reponse{entrees: []string{"m1"}}))
	if err != nil {
		t.Fatalf("attendu succès au rejeu, obtenu %v", err)
	}
	if len(entrees) != 1 || entrees[0] != "m1" {
		t.Fatalf("page inattendue : %+v", entrees)
	}
	if appels != 2 {
		t.Errorf("appels = %d, attendu 2", appels)
	}
	if len(*waits) != 0 {
		t.Errorf("un 429 ne doit RIEN attendre, durées demandées : %v", *waits)
	}
}

// (b) pool sans slot sain puis 200 : UNE attente, égale à min(cooldown, 60 s).
func TestPage_PoolSansSlotSain_AttendPuisRejoue(t *testing.T) {
	waits := captureSleeps(t)
	appels := 0
	entrees, err := Page(context.Background(), "Nuzzles", 0, fetchScripte(&appels,
		reponse{err: fmt.Errorf("pooled: Acquire failed: %w", pool.ErrNoHealthySlot)},
		reponse{entrees: []string{"m2"}}))
	if err != nil {
		t.Fatalf("attendu succès au rejeu, obtenu %v", err)
	}
	if len(entrees) != 1 {
		t.Fatalf("page inattendue : %+v", entrees)
	}
	want := min(NoSlotCooldown, WaitCap)
	if len(*waits) != 1 || (*waits)[0] != want {
		t.Errorf("attentes = %v, attendu exactement [%v]", *waits, want)
	}
}

// (c) 429 sur les trois tentatives : l'erreur remonte, bornée à Attempts appels.
func TestPage_429Persistant_RemonteApresTroisTentatives(t *testing.T) {
	waits := captureSleeps(t)
	appels := 0
	_, err := Page(context.Background(), "Nuzzles", 0, fetchScripte(&appels,
		reponse{err: httpErr(429)}, reponse{err: httpErr(429)}, reponse{err: httpErr(429)}))
	if err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	if appels != Attempts {
		t.Errorf("appels = %d, attendu %d", appels, Attempts)
	}
	if len(*waits) != 0 {
		t.Errorf("trois 429 n'attendent pas, durées : %v", *waits)
	}
}

// (d) 500 : erreur remontée immédiatement, aucun rejeu, aucune attente.
func TestPage_Erreur500_RemonteSansRejeu(t *testing.T) {
	waits := captureSleeps(t)
	appels := 0
	if _, err := Page(context.Background(), "Nuzzles", 50, fetchScripte(&appels,
		reponse{err: httpErr(500)}, reponse{entrees: []string{"jamais"}})); err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	if appels != 1 {
		t.Errorf("appels = %d, attendu 1 (aucun rejeu sur 500)", appels)
	}
	if len(*waits) != 0 {
		t.Errorf("aucune attente attendue, durées : %v", *waits)
	}
}

// (e) 503 : DÉJÀ retenté par le client HTTP (doGet) — pas de second étage de rejeu ici.
func TestPage_Erreur503_NonRejoueeIci(t *testing.T) {
	waits := captureSleeps(t)
	appels := 0
	_, err := Page(context.Background(), "Nuzzles", 75, fetchScripte(&appels,
		reponse{err: httpErr(503)}, reponse{entrees: []string{"jamais"}}))
	var he *haloclient.HTTPError
	if !errors.As(err, &he) || he.StatusCode != 503 {
		t.Fatalf("attendu le 503 tel quel, obtenu %v", err)
	}
	if appels != 1 {
		t.Errorf("appels = %d, attendu 1 (le 503 est retenté dans doGet, pas ici)", appels)
	}
	if len(*waits) != 0 {
		t.Errorf("aucune attente attendue, durées : %v", *waits)
	}
}
