package main

import (
	"context"
	"errors"
	"levelup/go-api/internal/config"
	"os"
	"strings"
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
	cfg.SpnkrAutoSyncEnabled = true
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

// TestArmGameWatch_PremierReleveAvantLesTachesLourdes : l'ordre du démarrage est le contrat. La
// détection du jeu fait son premier relevé avant le lancement du cœur des tâches, des ordonnanceurs
// et des crons du catalogue ; sans quoi leur premier cycle partirait en pleine partie.
func TestArmGameWatch_PremierReleveAvantLesTachesLourdes(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	arm := strings.Index(src, "\tarmGameWatch(st)")
	if arm < 0 {
		t.Fatal("main : armGameWatch absent")
	}
	for _, later := range []string{"launchCoreTasks", "launchSchedulers", "launchCatalogCrons", "launchSpartanCron", "launchWorldLeaderboardCron"} {
		at := strings.Index(src, "\t"+later+"(st)")
		if at < 0 || at < arm {
			t.Errorf("main : %s doit venir APRES armGameWatch", later)
		}
	}
}

// TestArmGameWatch_ArmeLaPorteEtFaitLeReleve : sous la capacité, le mode du processus est relevé
// avant le retour ; sans elle, rien n'est armé.
func TestArmGameWatch_ArmeLaPorteEtFaitLeReleve(t *testing.T) {
	prev := gameproc.SetDefault(gameproc.NewMode())
	t.Cleanup(func() { gameproc.SetDefault(prev) })

	st := &bootState{cfg: &config.AppConfig{}, settingsStore: nil}
	armGameWatch(st)
	if st.gameWatcher != nil || !gameproc.Default().HeavyWorkAllowed() {
		t.Fatal("sans capacité : aucune détection armée, aucune attente")
	}
}

// TestGameWatcher_AutoSyncCoupeeNeDemandeRien : la fermeture du jeu ne demande pas de rattrapage quand
// l'auto-sync est coupée (il serait ignoré par la boucle), et le dit : issue SyncDisabled.
func TestGameWatcher_AutoSyncCoupeeNeDemandeRien(t *testing.T) {
	cfg := settings.Defaults()
	cfg.SpnkrAutoSyncEnabled = false
	for name, tc := range map[string]struct {
		enabled bool
		want    gameproc.SyncOutcome
	}{"coupée": {false, gameproc.SyncDisabled}, "active": {true, gameproc.SyncRequested}} {
		t.Run(name, func(t *testing.T) {
			c := *cfg
			c.SpnkrAutoSyncEnabled = tc.enabled
			sync := &fakeSync{}
			w := newGameWatcher(fakeSettings{cfg: &c}, lister(), gameproc.NewMode(), sync)
			if got := w.OnGameClosed(gameproc.ReasonGameClosed); got != tc.want {
				t.Fatalf("issue = %v, attendu %v", got, tc.want)
			}
			if wantCalls := map[bool]int{false: 0, true: 1}[tc.enabled]; len(sync.reasons) != wantCalls {
				t.Fatalf("demandes de rattrapage = %d, attendu %d", len(sync.reasons), wantCalls)
			}
		})
	}
}
