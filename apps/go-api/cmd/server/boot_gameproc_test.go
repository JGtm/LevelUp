package main

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/gameproc"
	"levelup/go-api/internal/platform/settings"
)

type fakeSettings struct {
	cfg *settings.AppSettings
	err error
}

func (f fakeSettings) Load() (*settings.AppSettings, error) { return f.cfg, f.err }

type fakeSync struct{ reasons []string }

func (f *fakeSync) RequestCatchUp(reason string) bool {
	f.reasons = append(f.reasons, reason)
	return true
}

func lister(names ...string) gameproc.Lister {
	return gameproc.ListerFunc(func() ([]string, error) { return names, nil })
}

// TestGameWatcher_NomsDesTitresEtSynchroALaFermeture : le sondeur de production cherche les
// exécutables que déclarent les titres, et la fermeture du jeu demande un rattrapage de synchro.
func TestGameWatcher_NomsDesTitresEtSynchroALaFermeture(t *testing.T) {
	cfg := settings.Defaults()
	sync := &fakeSync{}
	mode := gameproc.NewMode()
	// L'exécutable du titre built-in vient du registre, pas d'une constante du serveur.
	names := title.DefaultRegistry().GameProcessNames()
	if len(names) == 0 {
		t.Fatal("le registre ne déclare aucun exécutable de jeu")
	}
	running := true
	l := gameproc.ListerFunc(func() ([]string, error) {
		if running {
			return []string{"explorer.exe", names[0]}, nil
		}
		return []string{"explorer.exe"}, nil
	})
	w := newGameWatcher(fakeSettings{cfg: cfg}, l, mode, sync)

	w.Poll(context.Background())
	if !mode.Running() {
		t.Fatal("le jeu du registre tourne : mode jeu attendu")
	}
	running = false
	w.Poll(context.Background())
	if mode.Running() {
		t.Fatal("jeu fermé : le mode jeu retombe")
	}
	if len(sync.reasons) != 1 || sync.reasons[0] != gameproc.ReasonGameClosed {
		t.Fatalf("demandes de synchronisation = %v, attendu une, %q", sync.reasons, gameproc.ReasonGameClosed)
	}
}

// TestGameWatcher_ReglageCoupeOuIllisible : réglage coupé = jamais de mode jeu ; réglage illisible =
// le défaut (actif), pas un jeu perdu.
func TestGameWatcher_ReglageCoupeOuIllisible(t *testing.T) {
	name := title.DefaultRegistry().GameProcessNames()[0]
	off := settings.Defaults()
	off.GameProcessWatch = false
	mode := gameproc.NewMode()
	newGameWatcher(fakeSettings{cfg: off}, lister(name), mode, nil).Poll(context.Background())
	if mode.Running() {
		t.Fatal("réglage coupé : pas de mode jeu")
	}
	mode = gameproc.NewMode()
	newGameWatcher(fakeSettings{err: errors.New("illisible")}, lister(name), mode, nil).Poll(context.Background())
	if !mode.Running() {
		t.Fatal("réglage illisible : la détection reste active")
	}
}

// TestGameWatcher_SansOrdonnanceur : pas d'ordonnanceur, pas de demande (aucun appel sur nil).
func TestGameWatcher_SansOrdonnanceur(t *testing.T) {
	w := newGameWatcher(fakeSettings{cfg: settings.Defaults()}, lister(), gameproc.NewMode(), nil)
	if w.OnGameClosed != nil {
		t.Fatal("sans ordonnanceur, aucune demande de synchronisation n'est câblée")
	}
}
