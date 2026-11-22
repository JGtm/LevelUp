package duckdb

import (
	"context"
	"errors"
	"testing"
	"time"
)

// historyRepoFixture : un dépôt dont la lecture réelle est remplacée par une couture comptée, et
// dont l'horloge est pilotée.
type historyRepoFixture struct {
	repo   *SyncHistoryRepo
	reads  int
	result bool
	err    error
	clock  time.Time
}

func newHistoryRepoFixture() *historyRepoFixture {
	f := &historyRepoFixture{clock: time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)}
	f.repo = NewSyncHistoryRepo(func(player, title string) (string, error) { return title + "/" + player + ".duckdb", nil }, "history_complete_at")
	f.repo.now = func() time.Time { return f.clock }
	f.repo.read = func(context.Context, string) (bool, error) {
		f.reads++
		return f.result, f.err
	}
	return f
}

func (f *historyRepoFixture) get(t *testing.T) bool {
	t.Helper()
	got, err := f.repo.HistoryComplete(context.Background(), "jgtm", "halo_5")
	if err != nil {
		t.Fatalf("HistoryComplete: %v", err)
	}
	return got
}

// Deux lectures rapprochées d'un état non complet : une seule ouverture.
func TestSyncHistoryRepo_FauxNEstRelQuApresLeDelai(t *testing.T) {
	f := newHistoryRepoFixture()
	if f.get(t) || f.get(t) {
		t.Fatal("non complet attendu")
	}
	if f.reads != 1 {
		t.Fatalf("lectures = %d, want 1 pour deux appels rapprochés", f.reads)
	}
	f.clock = f.clock.Add(syncHistoryRecheckEvery - time.Second)
	f.get(t)
	if f.reads != 1 {
		t.Fatalf("lectures = %d, want 1 avant le délai", f.reads)
	}
	f.clock = f.clock.Add(2 * time.Second)
	f.result = true
	if !f.get(t) || f.reads != 2 {
		t.Fatalf("après le délai : relu une fois et complet attendu (lectures=%d)", f.reads)
	}
}

// Un état complet n'est plus jamais relu, quelle que soit l'horloge.
func TestSyncHistoryRepo_VraiJamaisRelu(t *testing.T) {
	f := newHistoryRepoFixture()
	f.result = true
	if !f.get(t) {
		t.Fatal("complet attendu")
	}
	f.clock = f.clock.Add(100 * time.Hour)
	f.result = false // une relecture dirait faux : elle ne doit pas avoir lieu
	if !f.get(t) || f.reads != 1 {
		t.Fatalf("complet attendu sans relecture (lectures=%d)", f.reads)
	}
}

// Une erreur de lecture n'est pas mémorisée : l'appel suivant relit.
func TestSyncHistoryRepo_ErreurNonMemorisee(t *testing.T) {
	f := newHistoryRepoFixture()
	f.err = errors.New("base verrouillée")
	if _, err := f.repo.HistoryComplete(context.Background(), "jgtm", "halo_5"); !errors.Is(err, f.err) {
		t.Fatalf("err = %v, want l'erreur de lecture enveloppée", err)
	}
	f.err = nil
	f.result = true
	if !f.get(t) || f.reads != 2 {
		t.Fatalf("relecture attendue après l'erreur (lectures=%d)", f.reads)
	}
}

// Forget oublie un état complet : la lecture suivante rouvre la base (purge des données du titre).
func TestSyncHistoryRepo_ForgetOubliePourUnChemin(t *testing.T) {
	f := newHistoryRepoFixture()
	f.result = true
	f.get(t)
	f.repo.Forget("halo_5/jgtm.duckdb")
	f.result = false
	if f.get(t) || f.reads != 2 {
		t.Fatalf("après Forget : relu et non complet attendu (lectures=%d)", f.reads)
	}
}

// Les états sont tenus par joueur et par titre.
func TestSyncHistoryRepo_UnEtatParJoueurEtTitre(t *testing.T) {
	f := newHistoryRepoFixture()
	f.result = true
	f.get(t)
	f.result = false
	if got, _ := f.repo.HistoryComplete(context.Background(), "jgtm", "halo_infinite"); got || f.reads != 2 {
		t.Fatalf("un autre titre se lit séparément (got=%v lectures=%d)", got, f.reads)
	}
}
