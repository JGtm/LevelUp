package main

// cmd_backfill_usage_summary_test.go — LA CLE DE REPRISE de `backfill-usage-summary`, seule
// logique de la commande qui decide quoi re-ecrire (revue adversariale 2026-09-04, constat
// « la reprise n est prouvee par rien » — l inverser en silence servirait des resumes d un
// vieux schema a vie, ou re-ecrirait tout le corpus a chaque lancement).
//
// `projeterUnArtefact` se teste sans DB : la cle est (summary_rev courant, schema de
// l artefact SUR DISQUE) contre ce que la vue `_latest` a dit du match (passe en map).

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// artefactUsageDeTest ecrit un artefact minimal portant ce schema, et rend son chemin.
func artefactUsageDeTest(t *testing.T, schema int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artefact.json")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"schemaVersion":%d}`, schema)), 0o644); err != nil {
		t.Fatalf("ecrire artefact: %v", err)
	}
	return path
}

func TestProjeterUnArtefact_CleDeReprise(t *testing.T) {
	aJour := map[string]passeCouranteUsage{
		"m1": {rev: replay.UsageSummaryRev, schema: replay.SchemaVersion},
	}
	cas := []struct {
		nom    string
		path   string
		opts   usageSummaryOptions
		passes map[string]passeCouranteUsage
		want   etatUsageMatch
	}{
		{
			nom:  "deja a jour (rev + schema identiques) : saute",
			path: artefactUsageDeTest(t, replay.SchemaVersion),
			opts: usageSummaryOptions{}, passes: aJour, want: usageDejaAJour,
		},
		{
			nom:  "artefact re-cuit a un schema plus recent : re-resume",
			path: artefactUsageDeTest(t, replay.SchemaVersion+1),
			opts: usageSummaryOptions{}, passes: aJour, want: usageAProjeter,
		},
		{
			nom:  "revision de projection changee en base : re-resume",
			path: artefactUsageDeTest(t, replay.SchemaVersion),
			opts: usageSummaryOptions{},
			passes: map[string]passeCouranteUsage{
				"m1": {rev: "us0-perimee", schema: replay.SchemaVersion},
			},
			want: usageAProjeter,
		},
		{
			nom:  "--force : re-resume meme a jour",
			path: artefactUsageDeTest(t, replay.SchemaVersion),
			opts: usageSummaryOptions{force: true}, passes: aJour, want: usageAProjeter,
		},
		{
			nom:  "match jamais resume : projete",
			path: artefactUsageDeTest(t, replay.SchemaVersion),
			opts: usageSummaryOptions{}, passes: map[string]passeCouranteUsage{}, want: usageAProjeter,
		},
		{
			nom:  "pas d artefact : saute (releve de backfill-replay)",
			path: filepath.Join(t.TempDir(), "jamais-ecrit.json"),
			opts: usageSummaryOptions{}, passes: aJour, want: usageSansArtefact,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			s, etat := projeterUnArtefact(c.path, "m1", c.opts, c.passes)
			if etat != c.want {
				t.Fatalf("etat = %d, attendu %d", etat, c.want)
			}
			if c.want == usageAProjeter && s == nil {
				t.Fatal("un match a projeter doit rendre un resume non-nil")
			}
			if c.want != usageAProjeter && s != nil {
				t.Fatal("un match saute ne doit rendre aucun resume")
			}
		})
	}
}

func TestProjeterUnArtefact_ArtefactIllisible(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrompu.json")
	if err := os.WriteFile(path, []byte("{pas du json"), 0o644); err != nil {
		t.Fatalf("ecrire artefact: %v", err)
	}
	if _, etat := projeterUnArtefact(path, "m1", usageSummaryOptions{}, nil); etat != usageEchec {
		t.Fatalf("etat = %d, attendu usageEchec (un artefact corrompu degrade CE match, pas la passe)", etat)
	}
}
