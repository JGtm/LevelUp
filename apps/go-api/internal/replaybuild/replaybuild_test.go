package replaybuild

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// TestArtifactUpToDate — la clé de reprise du backfill : seule la version de schéma
// COURANTE vaut « à jour ». Absent, illisible ou antérieur = à re-cuire.
func TestArtifactUpToDate(t *testing.T) {
	dir := t.TempDir()
	cas := map[string]struct {
		contenu string
		attendu bool
	}{
		"version courante":   {fmt.Sprintf(`{"schemaVersion":%d,"matchId":"m"}`, replay.SchemaVersion), true},
		"version anterieure": {`{"schemaVersion":2,"matchId":"m"}`, false},
		"json illisible":     {`{pas du json`, false},
		"sans version":       {`{"matchId":"m"}`, false},
	}
	for nom, c := range cas {
		p := filepath.Join(dir, nom+".json")
		if err := os.WriteFile(p, []byte(c.contenu), 0o644); err != nil {
			t.Fatalf("écriture fixture %s: %v", nom, err)
		}
		if got := ArtifactUpToDate(p); got != c.attendu {
			t.Errorf("%s : ArtifactUpToDate = %v, attendu %v", nom, got, c.attendu)
		}
	}
	if ArtifactUpToDate(filepath.Join(dir, "absent.json")) {
		t.Error("artefact absent : attendu « à re-cuire », obtenu « à jour »")
	}
}

// TestResolveMapEntry_SurLeCatalogueLivre — le builder résout les candidats DANS L'ORDRE
// sur le catalogue de bornes VERSIONNÉ, et rend l'échec voulu quand aucun ne résout.
// Oracle réel : Cliffhanger -> module ridgeline (la référence du POC).
func TestResolveMapEntry_SurLeCatalogueLivre(t *testing.T) {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine repo introuvable: %v", err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	entry, err := b.ResolveMapEntry([]string{"", "Cliffhanger"})
	if err != nil {
		t.Fatalf("ResolveMapEntry(Cliffhanger): %v", err)
	}
	if entry.Module != "ridgeline" {
		t.Errorf("module de Cliffhanger = %q, attendu ridgeline", entry.Module)
	}
	if _, err := b.ResolveMapEntry([]string{"CarteInexistante-v75"}); !errors.Is(err, ErrMapNotInCatalog) {
		t.Errorf("carte inconnue : attendu ErrMapNotInCatalog, obtenu %v", err)
	}
	if _, err := b.ResolveMapEntry(nil); !errors.Is(err, ErrMapNotInCatalog) {
		t.Errorf("aucun candidat : attendu ErrMapNotInCatalog, obtenu %v", err)
	}
}
