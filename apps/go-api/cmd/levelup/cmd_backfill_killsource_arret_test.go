package main

// cmd_backfill_killsource_arret_test.go — L ARRET SE DISTINGUE D UNE PANNE (lot 5.24.4).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
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

// TestContexteDArret_UnePasseNORMALENAnnonceAucunArret — LE TEST DE NON-REGRESSION du defaut
// trouve le 2026-09-22 en lancant simplement la commande.
//
// La premiere version employait `signal.NotifyContext` et surveillait `<-ctx.Done()`. Ce
// contexte est annule PAR LES DEUX CHEMINS — le signal ET l appel a `stop` —, donc le
// `defer stop()` de la fin d une passe NORMALE reveillait le goroutine d annonce : TOUTE
// execution reussie finissait par ecrire « arret demande » sur la sortie d erreur, `--status`
// compris. Le message le plus alarmant de la commande s affichait quand tout allait bien.
func TestContexteDArret_UnePasseNORMALENAnnonceAucunArret(t *testing.T) {
	vraiStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = vraiStderr }()

	ctx, stop := contexteDArret()
	if causeDArret(ctx) != "" {
		t.Error("une passe qui demarre est deja annoncee comme interrompue")
	}
	stop()
	// Le goroutine d annonce doit avoir choisi la branche `fini`. On lui laisse le temps de se
	// tromper : sans ce delai, le test passerait meme avec le defaut.
	time.Sleep(50 * time.Millisecond)

	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	sortie, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(sortie) != 0 {
		t.Errorf("une passe NORMALE a ecrit sur la sortie d erreur : %q — le message d arret "+
			"s affiche quand tout va bien", string(sortie))
	}
	// L idempotence de `stop` : un second appel ne doit ni paniquer (canal deja ferme) ni rien
	// annoncer.
	stop()
}
