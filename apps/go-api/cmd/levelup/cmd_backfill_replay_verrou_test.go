package main

// cmd_backfill_replay_verrou_test.go — LE VERROU DE DECODAGE DE L'ENFANT DE PASSE
// (PLAN_CUISSON_PERF item 5.7).
//
// DEUX REGIMES, ET CELUI-CI EST L'ATTENTE. Le post-sync REFUSE tout de suite quand un autre
// decodage tient la machine : son match revient au cycle suivant. Une PASSE n'a pas de cycle
// suivant — un refus sur simple chevauchement (le serveur qui cuit au meme moment) ferait
// echouer un film parfaitement sain, et le recap accuserait le decodage. L'enfant attend donc
// son tour jusqu'a [attenteVerrouPasse].
//
// AUCUN FILM N'EST DECODE ICI : la racine du depot est vide, donc l'enfant s'arrete au
// catalogue de bornes manquant (code de PREPARATION) — apres avoir obtenu le verrou, ce qui est
// exactement ce que ce cas veut prouver.

import (
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/filmproc"
)

// TestEnfantDePasse_AttendSonTourPuisPasse : l'enfant ne rend pas la main tant que le verrou est
// tenu, et il repart des qu'il est rendu.
func TestEnfantDePasse_AttendSonTourPuisPasse(t *testing.T) {
	cacheRoot := t.TempDir()
	tenu, err := filmproc.AcquireSolo(cacheRoot, "post-sync", "un-autre-film")
	if err != nil {
		t.Fatalf("verrou temoin : %v", err)
	}

	fini := make(chan int, 1)
	go func() {
		fini <- runBackfillReplayUn(
			&config.AppConfig{RepoRoot: t.TempDir()},
			replayBackfillOptions{one: "64e8adfa-0000-0000-0000-000000000000", titleSlug: "halo_infinite"},
			cacheRoot)
	}()

	// TANT QUE LE VERROU EST TENU, L'ENFANT ATTEND. Sans cette attente il aurait deja rendu son
	// code — c'est la difference observable entre les deux regimes.
	select {
	case code := <-fini:
		t.Fatalf("l'enfant a rendu %d sans attendre : une passe ne doit pas echouer sur un "+
			"simple chevauchement de decodage", code)
	case <-time.After(300 * time.Millisecond):
	}

	tenu.Release()
	select {
	case code := <-fini:
		// LE VERROU OBTENU, L'ENFANT AVANCE : il echoue plus loin, sur le catalogue de bornes
		// absent de cette racine temporaire — donc APRES le verrou, ce qui prouve le passage.
		if code != filmproc.CodePreparation {
			t.Fatalf("code = %d, attendu %d (echec de preparation : racine sans catalogue)",
				code, filmproc.CodePreparation)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("le verrou a ete rendu mais l'enfant ne repart pas")
	}
}

// TestAttenteVerrouPasse_BorneEcrite : la borne est une DECISION (D7), pas un reglage qui derive.
// Dix minutes, c'est plus long que toute cuisson connue : une attente qui expire signale une
// machine vraiment occupee, jamais un chevauchement ordinaire.
func TestAttenteVerrouPasse_BorneEcrite(t *testing.T) {
	if attenteVerrouPasse != 10*time.Minute {
		t.Errorf("attente du verrou = %v, attendu 10m (PLAN_CUISSON_PERF §3 D7)", attenteVerrouPasse)
	}
}
