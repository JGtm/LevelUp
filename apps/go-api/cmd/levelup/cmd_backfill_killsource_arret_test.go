package main

// cmd_backfill_killsource_arret_test.go — L ARRET SE DISTINGUE D UNE PANNE (lot 5.24.4).

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSortirSur_CodeDedie — un script doit distinguer « relance-moi » de « repare-moi ».
func TestSortirSur_CodeDedie(t *testing.T) {
	interruption := &interruptionDeLaPasse{cause: "signal recu (SIGINT/SIGTERM)"}
	if got := sortirSur(interruption); got != CodeSortieInterrompue {
		t.Errorf("code = %d, attendu %d pour une passe interrompue", got, CodeSortieInterrompue)
	}
	// ENVELOPPEE, elle doit rester reconnaissable : une passe interrompue dont l erreur remonte
	// enrichie par un `%w` ne doit pas redevenir une panne.
	if got := sortirSur(fmt.Errorf("passe des films: %w", interruption)); got != CodeSortieInterrompue {
		t.Errorf("code = %d sur une interruption enveloppee, attendu %d", got, CodeSortieInterrompue)
	}
	if got := sortirSur(errors.New("base illisible")); got != 1 {
		t.Errorf("code = %d sur une vraie panne, attendu 1", got)
	}
	if CodeSortieInterrompue == 0 || CodeSortieInterrompue == 1 {
		t.Errorf("le code d interruption vaut %d : il ne se distingue plus d une passe finie "+
			"ni d une panne", CodeSortieInterrompue)
	}
}

// TestInterruption_LeMessageDitQuoiFaire — un arret muet oblige a relire le code.
func TestInterruption_LeMessageDitQuoiFaire(t *testing.T) {
	msg := (&interruptionDeLaPasse{cause: "signal recu"}).Error()
	for _, attendu := range []string{"interrompue", "relancer LA MEME commande", "decoder_rev"} {
		if !strings.Contains(msg, attendu) {
			t.Errorf("le message d interruption ne contient pas %q : %s", attendu, msg)
		}
	}
}

// TestCauseDArret_SeLitSurLeContexte — pas de chaine partagee entre goroutines.
func TestCauseDArret_SeLitSurLeContexte(t *testing.T) {
	if got := causeDArret(nil); got != "" {
		t.Errorf("cause = %q sur un contexte nil, attendu vide", got)
	}
	if got := causeDArret(context.Background()); got != "" {
		t.Errorf("cause = %q sur une passe non interrompue, attendu vide", got)
	}
	ctx, annuler := context.WithCancel(context.Background())
	annuler()
	if got := causeDArret(ctx); got == "" {
		t.Error("cause vide sur un contexte annule : la passe sortirait avec le code d une " +
			"passe finie")
	}
}
