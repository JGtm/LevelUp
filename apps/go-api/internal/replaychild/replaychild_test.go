package replaychild

// replaychild_test.go — LE VERROU DE DECODAGE, VU DU PARENT (PLAN_CUISSON_PERF item 5.7).
//
// CE QUE CES CAS TIENNENT. Le post-sync vit DANS le serveur : il ne peut ni attendre son tour
// (la synchronisation entiere attendrait derriere lui) ni decoder en parallele d'un autre
// processus (c'est le mecanisme des quatre sinistres RAM, cf. internal/filmproc/solo.go). Son
// regime est donc le REFUS IMMEDIAT, pris par le PARENT — avant meme de faire naitre l'enfant,
// pour qu'un refus ne coute pas un lancement de processus et une lecture de catalogue.
//
// AUCUN FILM N'EST DECODE ICI, ET AUCUN ENFANT N'EST LANCE : c'est precisement la propriete
// verifiee.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
)

func TestSpawn_VerrouTenuAilleurs_RefusImmediat(t *testing.T) {
	repoRoot := t.TempDir()
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()

	// UN AUTRE OUTIL TIENT LA MACHINE (une passe de backfill, un ouvrier, `replay-build`).
	autre, err := filmproc.AcquireSolo(cacheRoot, "backfill-replay", "un-autre-film")
	if err != nil {
		t.Fatalf("le verrou temoin n'a pas pu etre pris : %v", err)
	}
	defer autre.Release()

	debut := time.Now()
	_, err = Spawn(context.Background(), Request{
		RepoRoot: repoRoot, TitleSlug: title.DefaultSlug,
		MatchID: "64e8adfa-0000-0000-0000-000000000000", MapNames: []string{"Catalyst"},
	})
	if !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("err = %v, attendu un refus %v — le post-sync ne doit ni attendre ni decoder "+
			"en parallele", err, filmproc.ErrDecodeBusy)
	}
	// IMMEDIAT, PAS BORNE : un refus qui prendrait des secondes serait une attente deguisee.
	if ecoule := time.Since(debut); ecoule > 2*time.Second {
		t.Errorf("le refus a pris %v : le post-sync attend au lieu de refuser", ecoule)
	}
	// LE MESSAGE NOMME LE DETENTEUR : sans lui, l'operateur ne sait pas quoi attendre ni quoi
	// arreter (cf. filmproc.soloBusyError).
	if !strings.Contains(err.Error(), "backfill-replay") {
		t.Errorf("message = %q : il ne nomme pas le detenteur du verrou", err)
	}
}

// TestSpawn_VerrouRendu_LaVoieEstLibre : le refus ne consomme ni ne casse le verrou du
// detenteur — apres sa liberation, la machine est de nouveau disponible pour une cuisson.
func TestSpawn_VerrouRendu_LaVoieEstLibre(t *testing.T) {
	repoRoot := t.TempDir()
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()

	autre, err := filmproc.AcquireSolo(cacheRoot, "replay-worker", "un-autre-film")
	if err != nil {
		t.Fatalf("verrou temoin : %v", err)
	}
	if _, err := Spawn(context.Background(), Request{RepoRoot: repoRoot, MatchID: "m"}); !errors.Is(err, filmproc.ErrDecodeBusy) {
		t.Fatalf("err = %v, attendu un refus", err)
	}
	autre.Release()

	repris, err := filmproc.AcquireSolo(cacheRoot, "post-sync", "m")
	if err != nil {
		t.Fatalf("apres liberation, le verrou reste pris : %v", err)
	}
	repris.Release()
}
