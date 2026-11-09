package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestUpdate_EcrituresConcurrentesSansPerte : cinquante écrivains modifient des clés différentes du même
// fichier en même temps (le PATCH /settings, les abonnements du watcher et le drapeau de la file de
// régénération sont de ces écrivains). Sans le verrou d'écriture partagé, deux lectures-modifications-écritures
// entrelacées se réécrivent l'une l'autre : une modification disparaît. Sous `-race`, le test dit aussi qu'aucun
// accès n'est non synchronisé.
func TestUpdate_EcrituresConcurrentesSansPerte(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_settings.json")
	s := NewStore(path)
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			if i%2 == 0 {
				_, err := s.Update(func(cfg *AppSettings) error {
					cfg.WatcherSubscribedPlayers = append(cfg.WatcherSubscribedPlayers, fmt.Sprintf("joueur-%d", i))
					return nil
				})
				if err != nil {
					t.Error(err)
				}
				return
			}
			if err := s.SetReplayRegenPending(i%4 == 1); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	cfg, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(cfg.WatcherSubscribedPlayers); got != n/2 {
		t.Fatalf("abonnements conservés = %d, attendu %d : une écriture concurrente en a écrasé une autre", got, n/2)
	}
}

// TestUpdate_RefusDeLaFonctionNEcritRien : une validation qui refuse laisse le fichier octet pour octet.
func TestUpdate_RefusDeLaFonctionNEcritRien(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app_settings.json")
	if err := os.WriteFile(path, []byte(`{"lang":"fr"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)
	refus := fmt.Errorf("refusé")
	if _, err := s.Update(func(cfg *AppSettings) error { cfg.Lang = "en"; return refus }); err != refus {
		t.Fatalf("erreur = %v, attendu le refus tel quel", err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != `{"lang":"fr"}` {
		t.Fatalf("fichier = %s, attendu inchangé", raw)
	}
}
