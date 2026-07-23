package adminstate

import (
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestFileStore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	s := NewFileStore(path)

	// Fichier absent au premier Load.
	var got sample
	found, err := s.Load(&got)
	if err != nil {
		t.Fatalf("Load fichier absent: err=%v (attendu nil)", err)
	}
	if found {
		t.Fatalf("Load fichier absent: found=true (attendu false)")
	}

	want := sample{Name: "cycle", Count: 3}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Le répertoire parent doit avoir été créé.
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("répertoire parent non créé: %v", err)
	}

	var back sample
	found, err = s.Load(&back)
	if err != nil || !found {
		t.Fatalf("Load après Save: found=%v err=%v", found, err)
	}
	if back != want {
		t.Fatalf("round-trip: got %+v, want %+v", back, want)
	}
}

func TestFileStore_CorruptFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("seed corrompu: %v", err)
	}
	s := NewFileStore(path)
	var got sample
	found, err := s.Load(&got)
	if !found {
		t.Fatalf("fichier corrompu présent: found=false (attendu true)")
	}
	if err == nil {
		t.Fatalf("fichier corrompu: err=nil (attendu erreur de parse pour dégradation loggée)")
	}
}

func TestFileStore_EmptyFileTreatedAsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("seed vide: %v", err)
	}
	s := NewFileStore(path)
	var got sample
	found, err := s.Load(&got)
	if err != nil || found {
		t.Fatalf("fichier vide: found=%v err=%v (attendu found=false, err=nil)", found, err)
	}
}

func TestFileStore_SaveIsAtomic_NoTempLeftOver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := NewFileStore(path)
	for i := 0; i < 3; i++ {
		if err := s.Save(sample{Name: "x", Count: i}); err != nil {
			t.Fatalf("Save #%d: %v", i, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// Seul le fichier final doit rester (aucun fichier temporaire orphelin).
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Fatalf("fichier résiduel inattendu après Save atomique: %q", e.Name())
		}
	}
}

func TestFileStore_ConcurrentSavesDoNotCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := NewFileStore(path)
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 25; j++ {
				_ = s.Save(sample{Name: "concurrent", Count: n*100 + j})
			}
		}(i)
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	// Après la tempête d'écritures, le fichier doit rester un JSON valide et
	// complet (l'écriture atomique interdit tout état à moitié écrit).
	var got sample
	found, err := s.Load(&got)
	if err != nil || !found {
		t.Fatalf("Load après écritures concurrentes: found=%v err=%v", found, err)
	}
}
