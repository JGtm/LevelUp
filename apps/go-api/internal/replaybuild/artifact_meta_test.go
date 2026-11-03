package replaybuild

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

func TestVersionPlusRecente(t *testing.T) {
	cas := []struct {
		a, b string
		want bool
	}{
		{"8.2.0", "8.1.0", true},
		{"v8.1.1", "8.1.0", true},
		{"9.0", "8.9.9", true},
		{"8.1.0", "8.1.0", false},
		{"8.0.9", "8.1.0", false},
		{"8.1.0", "8.1.0-rc1", true},
		{"8.1.0-rc1", "8.1.0", false},
		{"8.1.0+build7", "8.1.0", false},
		{"", "8.1.0", false},
		{"8.2.0", "", false},
		{"dev", "8.1.0", false},
		{"8.2.0", "dev", false},
		{"8.1.0.4", "8.1.0", false},
	}
	for _, c := range cas {
		if got := versionPlusRecente(c.a, c.b); got != c.want {
			t.Errorf("versionPlusRecente(%q, %q) = %v, attendu %v", c.a, c.b, got, c.want)
		}
	}
}

// TestWriteArtifact_EcritLEnteteEtLaVersionDeLApplication : l'écrivain d'artefact pose le fichier d'en-tete voisin,
// avec la version de l'application du processus ; ArtifactHeader le relit sans ouvrir l'artefact.
func TestWriteArtifact_EcritLEnteteEtLaVersionDeLApplication(t *testing.T) {
	SetAppVersion("8.1.0")
	t.Cleanup(func() { SetAppVersion("") })
	path := filepath.Join(t.TempDir(), "000d5950.json")
	if _, err := StoreArtifactAt(context.Background(), path, title.DefaultSlug, "000d5950", docJSON(t, "000d5950", true)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(MetaPath(path)); err != nil {
		t.Fatalf("fichier d'en-tete absent : %v", err)
	}
	avant := observability.LoadCounter(CompteurArtefactsLusEnEntier)
	h, ok := ArtifactHeader(context.Background(), path)
	if !ok || h.AppVersion != "8.1.0" || h.SchemaVersion == 0 {
		t.Fatalf("en-tete = %+v, %v", h, ok)
	}
	if observability.LoadCounter(CompteurArtefactsLusEnEntier) != avant {
		t.Fatal("l'en-tete fait foi : l'artefact n'est pas relu")
	}
	if !h.ProduitParUneVersionPlusRecente("8.0.0") || h.ProduitParUneVersionPlusRecente("8.1.0") {
		t.Fatal("comparaison de la version du producteur")
	}
}

// TestArtifactHeader_LitLArtefactQuandLEnteteNeFaitPasFoi : absent, plus ancien que l'artefact, ou de taille
// différente : l'artefact est lu une fois et l'en-tete réécrit ; la lecture suivante n'ouvre plus l'artefact.
func TestArtifactHeader_LitLArtefactQuandLEnteteNeFaitPasFoi(t *testing.T) {
	path := filepath.Join(t.TempDir(), "000d5950.json")
	blob := docJSON(t, "000d5950", true)
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatal(err)
	}
	lectures := func() int64 { return observability.LoadCounter(CompteurArtefactsLusEnEntier) }

	n := lectures()
	if _, ok := ArtifactHeader(context.Background(), path); !ok || lectures() != n+1 {
		t.Fatal("artefact d'avant l'en-tete : lu en entier une fois")
	}
	if _, err := os.Stat(MetaPath(path)); err != nil {
		t.Fatal("l'en-tete est écrit au premier décompte")
	}
	n = lectures()
	if _, ok := ArtifactHeader(context.Background(), path); !ok || lectures() != n {
		t.Fatal("second décompte : l'en-tete fait foi")
	}

	// Artefact remplacé par un outil qui n'écrit pas l'en-tete : plus récent que lui.
	futur := time.Now().Add(time.Hour)
	if err := os.WriteFile(path, docJSON(t, "000d5950", false), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, futur, futur); err != nil {
		t.Fatal(err)
	}
	if _, ok := ArtifactHeader(context.Background(), path); !ok || lectures() != n+1 {
		t.Fatal("artefact plus récent que son en-tete : relu")
	}

	// En-tete de taille différente de l'artefact.
	if err := os.WriteFile(MetaPath(path), []byte(`{"schemaVersion":1,"bytes":3}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(MetaPath(path), futur.Add(time.Hour), futur.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	n = lectures()
	h, ok := ArtifactHeader(context.Background(), path)
	if !ok || lectures() != n+1 || h.SchemaVersion == 1 {
		t.Fatalf("en-tete de mauvaise taille : relu (%+v, %v)", h, ok)
	}
}

func TestPurgeMatchFiles_EmporteLEnTete(t *testing.T) {
	pr := title.NewPathResolver(t.TempDir())
	art := pr.ReplayArtifactPath(title.DefaultSlug, "000d5950")
	ecrireFichier(t, art)
	ecrireFichier(t, MetaPath(art))
	touches, err := PurgeMatchFiles(pr, title.DefaultSlug, "000d5950", false)
	if err != nil || len(touches) != 2 {
		t.Fatalf("touches = %v, %v", touches, err)
	}
	if _, err := os.Stat(MetaPath(art)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("l'en-tete suit son artefact")
	}
}
