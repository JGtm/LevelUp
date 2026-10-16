package jobs

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
)

// CreateIfNoActive crée le job seulement si aucun job non terminal du même type n'existe
// pour la clé, atomiquement : N demandes simultanées en créent exactement un.
func TestStore_CreateIfNoActive_Atomic(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	var created atomic.Int32
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := s.CreateIfNoActive(domain.JobTypeOpenSpartanImport, "xuid-1"); ok {
				created.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := created.Load(); got != 1 {
		t.Fatalf("jobs créés = %d, attendu 1", got)
	}
}

// Un job terminé, ou d'une autre clé ou d'un autre type, ne bloque pas.
func TestStore_CreateIfNoActive_IgnoresTerminalAndOthers(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	first, ok := s.CreateIfNoActive(domain.JobTypeOpenSpartanImport, "xuid-1")
	if !ok {
		t.Fatal("premier job refusé")
	}
	if _, ok := s.CreateIfNoActive(domain.JobTypeOpenSpartanImport, "xuid-2"); !ok {
		t.Fatal("la clé d'un autre compte est bloquée")
	}
	if _, ok := s.CreateIfNoActive(domain.JobTypeBackfill, "xuid-1"); !ok {
		t.Fatal("un autre type de job est bloqué")
	}
	if _, ok := s.CreateIfNoActive(domain.JobTypeOpenSpartanImport, "xuid-1"); ok {
		t.Fatal("un second job actif a été créé pour la même clé")
	}
	s.SetStatus(first.JobID, domain.JobStatusSucceeded, nil)
	if _, ok := s.CreateIfNoActive(domain.JobTypeOpenSpartanImport, "xuid-1"); !ok {
		t.Fatal("un job terminé continue de bloquer")
	}
}
