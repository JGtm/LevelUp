package title

import (
	"path/filepath"
	"testing"
)

func TestDefaultRoots_DispositionDuCheckout(t *testing.T) {
	r := DefaultRoots("/depot")
	if r.Resources != "/depot" || r.Data != filepath.Join("/depot", "data") || r.Logs != filepath.Join("/depot", "logs") {
		t.Fatalf("racines par défaut inattendues : %+v", r)
	}
	if r.DataSet {
		t.Fatal("DataSet doit rester faux par défaut")
	}
	if got := r.StateFilesDir(); got != "/depot" {
		t.Errorf("StateFilesDir par défaut = %q, attendu la tête de Resources", got)
	}
	r.DataSet = true
	r.Data = "/donnees"
	if got := r.StateFilesDir(); got != "/donnees" {
		t.Errorf("StateFilesDir avec Data explicite = %q, attendu la tête de Data", got)
	}
}

func TestNewPathResolverRoots_CompleteLesRacinesVides(t *testing.T) {
	pr := NewPathResolverRoots(Roots{Resources: "/res"})
	if pr.DataRoot() != filepath.Join("/res", "data") || pr.LogsRoot() != filepath.Join("/res", "logs") {
		t.Fatalf("racines non complétées : %+v", pr.Roots())
	}
	if pr.RepoRoot() != "/res" || pr.ResourcesRoot() != "/res" {
		t.Fatalf("RepoRoot et ResourcesRoot doivent désigner la racine de ressources : %q %q", pr.RepoRoot(), pr.ResourcesRoot())
	}
}

func TestNewPathResolver_PontDesRacinesDuProcessus(t *testing.T) {
	t.Cleanup(ResetProcessRoots)
	ResetProcessRoots()
	if got := NewPathResolver("/res").DataRoot(); got != filepath.Join("/res", "data") {
		t.Fatalf("sans pont, la racine de données doit être celle du checkout : %q", got)
	}
	InstallProcessRoots(Roots{Resources: "/res", Data: "/donnees", Logs: "/journaux", DataSet: true})
	if got := NewPathResolver("/res").DataRoot(); got != "/donnees" {
		t.Errorf("le pont doit servir la racine de données du processus : %q", got)
	}
	if got := NewPathResolver(filepath.Join("/res", "..", "res")).LogsRoot(); got != "/journaux" {
		t.Errorf("le pont compare les racines nettoyées : %q", got)
	}
	if got := NewPathResolver("/autre").DataRoot(); got != filepath.Join("/autre", "data") {
		t.Errorf("une autre racine de ressources garde la disposition par défaut : %q", got)
	}
}
