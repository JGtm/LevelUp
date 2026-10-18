//go:build cgo

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/dblease"
)

// TestPostSyncPlayerOpener_TientLeLeaseJoueur : la base joueur du post-sync V2 est tenue sous le
// lease writer joueur jusqu'à sa fermeture — un autre écrivain (sync du watcher, relance des
// citations) attend ; il passe dès la fermeture.
func TestPostSyncPlayerOpener_TientLeLeaseJoueur(t *testing.T) {
	pr := titlePkg.NewPathResolver(t.TempDir())
	ouvrir := postSyncPlayerOpener(pr, titlePkg.DefaultSlug)
	path := pr.PlayerDBPath(titlePkg.DefaultSlug, "Joueur")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}

	db, fermer, err := ouvrir(context.Background(), "Joueur")
	if err != nil || db == nil {
		t.Fatalf("ouverture : db=%v err=%v", db, err)
	}
	if _, err := dblease.AcquireWriter(nil, path, dblease.KindPlayer, 50*time.Millisecond); !errors.Is(err, dblease.ErrDBLocked) {
		t.Fatalf("lease pendant le post-sync : err=%v, attendu ErrDBLocked", err)
	}
	fermer()
	w, err := dblease.AcquireWriter(nil, path, dblease.KindPlayer, time.Second)
	if err != nil {
		t.Fatalf("lease après fermeture : %v", err)
	}
	w.Release()

	// Base tenue ailleurs et contexte qui expire : l'ouverture échoue proprement, sans base ouverte.
	tenu, _ := dblease.AcquireWriter(nil, path, dblease.KindPlayer, time.Second)
	defer tenu.Release()
	ctx, annuler := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer annuler()
	if _, _, err := ouvrir(ctx, "Joueur"); err == nil {
		t.Fatal("base tenue et contexte expiré : une erreur est attendue")
	}
}
