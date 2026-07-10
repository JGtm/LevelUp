// Package jobs — store_list_test.go : tests de Store.List (dashboard
// monitoring admin : jobs récents, actifs d'abord).
package jobs_test

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// TestStore_List_ActiveFirstThenRecent : les jobs actifs sortent avant les
// terminaux, et le tri est StartedAt décroissant à statut égal.
func TestStore_List_ActiveFirstThenRecent(t *testing.T) {
	store := newTestStore(t)

	oldDone := store.Create(domain.JobTypeBackfill, "p1")
	store.SetStatus(oldDone.JobID, domain.JobStatusSucceeded, nil)
	time.Sleep(2 * time.Millisecond) // StartedAt strictement croissants (le tri de List repose sur StartedAt, pas sur l'ID)
	recentDone := store.Create(domain.JobTypeScanMedia, "p2")
	store.SetStatus(recentDone.JobID, domain.JobStatusFailed, nil)
	time.Sleep(2 * time.Millisecond)
	active := store.Create(domain.JobTypeForcedSyncCycle, "_all")
	store.SetStatus(active.JobID, domain.JobStatusRunning, nil)

	got := store.List(10)
	if len(got) != 3 {
		t.Fatalf("len(List) = %d (attendu 3)", len(got))
	}
	if got[0].JobID != active.JobID {
		t.Errorf("le job actif doit sortir en premier, got %s (%s)", got[0].JobID, got[0].Status)
	}
	if got[1].JobID != recentDone.JobID || got[2].JobID != oldDone.JobID {
		t.Errorf("terminaux attendus du plus récent au plus ancien, got [%s, %s]",
			got[1].JobID, got[2].JobID)
	}
}

// TestStore_List_LimitAndDefault : limit appliqué, et limit <= 0 → défaut 20.
func TestStore_List_LimitAndDefault(t *testing.T) {
	store := newTestStore(t)
	for i := 0; i < 5; i++ {
		j := store.Create(domain.JobTypeBackfill, "p")
		store.SetStatus(j.JobID, domain.JobStatusSucceeded, nil)
		time.Sleep(time.Millisecond)
	}
	if got := store.List(2); len(got) != 2 {
		t.Fatalf("List(2) = %d éléments (attendu 2)", len(got))
	}
	if got := store.List(0); len(got) != 5 {
		t.Fatalf("List(0) = %d éléments (attendu 5 — défaut 20 englobe tout)", len(got))
	}
}

// TestStore_List_ReturnsCopies : muter un élément retourné ne corrompt pas le
// store (cohérent avec Get/FindActiveJob).
func TestStore_List_ReturnsCopies(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeBackfill, "p")

	list := store.List(5)
	if len(list) != 1 {
		t.Fatalf("len(List) = %d (attendu 1)", len(list))
	}
	list[0].PlayerSlug = "corrompu"

	if again := store.Get(job.JobID); again.PlayerSlug != "p" {
		t.Fatalf("List doit retourner des copies — store corrompu : %s", again.PlayerSlug)
	}
}
