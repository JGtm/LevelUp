package replaybuild

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

type dossierUnique struct{ dir string }

func (d dossierUnique) ReplayArtifactsDir(string) string { return d.dir }

// corpsDeSchema : un document minimal qui ne dit que sa version de schéma, comme l'en-tête que le décompte lit.
func corpsDeSchema(version int) []byte {
	return []byte(fmt.Sprintf(`{"schemaVersion":%d,"matchId":"m","titleSlug":"halo_infinite","tracks":[]}`, version))
}

// TestSchemaCensus_CompteLesTroisEtats : un rejeu au schéma courant, un à un schéma antérieur et un dont l'en-tête
// ne se lit pas se rangent chacun dans son compte ; le total est leur somme.
func TestSchemaCensus_CompteLesTroisEtats(t *testing.T) {
	dir := t.TempDir()
	for nom, version := range map[string]int{"aaaaaaaa.json.gz": replay.SchemaVersion, "bbbbbbbb.json.gz": replay.SchemaVersion - 1, "cccccccc.json.gz": 1} {
		if err := poserCompresseTest(filepath.Join(dir, nom), corpsDeSchema(version)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "dddddddd.json.gz"), []byte("pas un rejeu"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewSchemaCensus(dossierUnique{dir})
	got, err := c.CountReplaySchemas(context.Background(), "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if got.TitleSlug != "halo_infinite" || got.Total != 4 || got.Current != 1 || got.Behind != 2 || got.Unreadable != 1 {
		t.Fatalf("décompte = %+v, attendu 4 rejeux : 1 courant, 2 à l'ancien schéma, 1 illisible", got)
	}
	if c.CurrentSchema() != replay.SchemaVersion {
		t.Fatalf("schéma courant = %d, attendu %d", c.CurrentSchema(), replay.SchemaVersion)
	}
}

// TestSchemaCensus_DossierAbsent : un titre sans dossier de rejeux n'est pas une erreur, ses comptes sont à zéro.
func TestSchemaCensus_DossierAbsent(t *testing.T) {
	c := NewSchemaCensus(dossierUnique{filepath.Join(t.TempDir(), "absent")})
	got, err := c.CountReplaySchemas(context.Background(), "halo_5")
	if err != nil {
		t.Fatalf("un dossier absent n'est pas une erreur : %v", err)
	}
	if got.Total != 0 || got.Current != 0 || got.Behind != 0 || got.Unreadable != 0 {
		t.Fatalf("décompte = %+v, attendu des zéros", got)
	}
}

// TestSchemaCensus_ContexteAnnule : l'annulation interrompt le décompte au lieu de lire tous les en-têtes.
func TestSchemaCensus_ContexteAnnule(t *testing.T) {
	dir := t.TempDir()
	if err := poserCompresseTest(filepath.Join(dir, "aaaaaaaa.json.gz"), corpsDeSchema(replay.SchemaVersion)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewSchemaCensus(dossierUnique{dir}).CountReplaySchemas(ctx, "halo_infinite"); err == nil {
		t.Fatal("un contexte annulé doit interrompre le décompte")
	}
}
