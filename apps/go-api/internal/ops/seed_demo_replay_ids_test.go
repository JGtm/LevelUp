package ops

import (
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// poserManifesteDemoDeTest écrit un manifeste brut sous <root>/config/demo/<gamertag>/<nom>.json.
func poserManifesteDemoDeTest(t *testing.T, root, gamertag, nom, contenu string) {
	t.Helper()
	dir := filepath.Join(root, "config", "demo", gamertag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, nom+".json"), []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDemoReplayMatchIDs_FiltreParTitreEtDedoublonne(t *testing.T) {
	root := t.TempDir()
	poserManifesteDemoDeTest(t, root, "JGtm", "halo_infinite", `{"version":"1","title_slug":"halo_infinite",
		"corpus":{"solo_match_ids":["s1"],"replay_matches":[{"match_id":"AAAA","mode":"slayer"},{"match_id":"bbbb","mode":"ctf"}]}}`)
	poserManifesteDemoDeTest(t, root, "Autre", "halo_infinite", `{"version":"1","title_slug":"halo_infinite",
		"corpus":{"solo_match_ids":["s2"],"replay_matches":[{"match_id":"aaaa","mode":"slayer"},{"match_id":"cccc","mode":"oddball"}]}}`)
	poserManifesteDemoDeTest(t, root, "JGtm", "halo_5", `{"version":"1","title_slug":"halo_5",
		"corpus":{"solo_match_ids":["s3"],"replay_matches":[{"match_id":"h5h5","mode":"slayer"}]}}`)

	pr := titlePkg.NewPathResolver(root)
	got, err := DemoReplayMatchIDs(pr, "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	// Ordre : chemins triés (Autre avant JGtm), doublon AAAA/aaaa retenu une fois, le titre
	// halo_5 absent.
	if len(got) != 3 || got[0] != "aaaa" || got[1] != "cccc" || got[2] != "bbbb" {
		t.Fatalf("DemoReplayMatchIDs = %v, attendu [aaaa cccc bbbb]", got)
	}
	h5, err := DemoReplayMatchIDs(pr, "halo_5")
	if err != nil || len(h5) != 1 || h5[0] != "h5h5" {
		t.Fatalf("halo_5 : %v, %v", h5, err)
	}
}

func TestDemoReplayMatchIDs_RefusDesCasVides(t *testing.T) {
	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	if _, err := DemoReplayMatchIDs(pr, "halo_infinite"); err == nil {
		t.Error("aucun manifeste : attendu un refus")
	}
	poserManifesteDemoDeTest(t, root, "JGtm", "halo_infinite",
		`{"version":"1","title_slug":"halo_infinite","corpus":{"solo_match_ids":["s1"]}}`)
	if _, err := DemoReplayMatchIDs(pr, "halo_infinite"); err == nil {
		t.Error("manifeste sans replay_matches : attendu un refus (une liste vide ne doit jamais filtrer)")
	}
	if _, err := DemoReplayMatchIDs(pr, "halo_5"); err == nil {
		t.Error("titre sans manifeste : attendu un refus")
	}
	poserManifesteDemoDeTest(t, root, "JGtm", "casse", `{pas du json`)
	if _, err := DemoReplayMatchIDs(pr, "halo_infinite"); err == nil {
		t.Error("manifeste invalide : attendu une erreur dure")
	}
}
