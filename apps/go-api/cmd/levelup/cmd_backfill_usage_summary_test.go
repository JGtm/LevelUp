package main

// cmd_backfill_usage_summary_test.go — LA CLE DE REPRISE de `backfill-usage-summary`, seule
// logique de la commande qui decide quoi re-ecrire (revue adversariale 2026-09-04, constat
// « la reprise n est prouvee par rien » — l inverser en silence servirait des resumes d un
// vieux schema a vie, ou re-ecrirait tout le corpus a chaque lancement).
//
// `projeterUnArtefact` se teste sans DB : la cle est (summary_rev courant, schema de
// l artefact SUR DISQUE) contre ce que la vue `_latest` a dit du match (passe en map).

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// artefactUsageDeTest ecrit un artefact minimal portant ce schema, et rend son chemin.
func artefactUsageDeTest(t *testing.T, schema int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artefact.json")
	if err := poserCompresse(path, []byte(fmt.Sprintf(`{"schemaVersion":%d}`, schema)), 0o644); err != nil {
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
		{
			// LA FIGURE DE LA REVUE DE VAGUE 4 (constat P2, 2026-09-10) : un artefact encore au
			// schema d avant le bump courant, JAMAIS resume (absent de `dejaResumes`).
			// L ANCIEN CALCUL ne comparait que (rev, schema) face a la reprise : ce cas serait
			// tombe dans `usageAProjeter`, aurait ecrit un bilan VIDE (les champs du bump
			// n existent pas a ce schema), et l aurait marque a jour pour toujours.
			nom:  "artefact a un schema perime (< courant) : refuse, jamais projete",
			path: artefactUsageDeTest(t, replay.SchemaVersion-1),
			opts: usageSummaryOptions{}, passes: map[string]passeCouranteUsage{},
			want: usageSchemaPerime,
		},
		{
			// MEME REFUS SOUS --force : forcer ne fabrique aucun champ manquant, le schema reste
			// incompatible.
			nom:  "artefact a un schema perime, meme sous --force : refuse",
			path: artefactUsageDeTest(t, replay.SchemaVersion-1),
			opts: usageSummaryOptions{force: true}, passes: aJour,
			want: usageSchemaPerime,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			s, etat := projeterUnArtefact(context.Background(), c.path, "m1", c.opts, c.passes)
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
	if err := poserCompresse(path, []byte("{pas du json"), 0o644); err != nil {
		t.Fatalf("ecrire artefact: %v", err)
	}
	if _, etat := projeterUnArtefact(context.Background(), path, "m1", usageSummaryOptions{}, nil); etat != usageEchec {
		t.Fatalf("etat = %d, attendu usageEchec (un artefact corrompu degrade CE match, pas la passe)", etat)
	}
}

// TestProjeterUnArtefact_LePasEstUneMarqueDeFraicheur : à schéma et révision égaux, un artefact régénéré
// à un autre pas est résumé de nouveau ; au même pas il est sauté. Mutation : retirer `SameCadence` de la
// clé de reprise rend le premier cas « déjà à jour » et ce test rouge.
func TestProjeterUnArtefact_LePasEstUneMarqueDeFraicheur(t *testing.T) {
	artefactAuPas := func(pas int) string {
		path := filepath.Join(t.TempDir(), "artefact.json")
		corps := fmt.Sprintf(`{"schemaVersion":%d,"frameIntervalMs":%d}`, replay.SchemaVersion, pas)
		if err := poserCompresse(path, []byte(corps), 0o644); err != nil {
			t.Fatalf("ecrire artefact: %v", err)
		}
		return path
	}
	resumeAuPas := func(pas int) map[string]passeCouranteUsage {
		return map[string]passeCouranteUsage{
			"m1": {rev: replay.UsageSummaryRev, schema: replay.SchemaVersion, frameIntervalMs: pas},
		}
	}
	cas := []struct {
		nom    string
		path   string
		passes map[string]passeCouranteUsage
		want   etatUsageMatch
	}{
		{"artefact regenere a un autre pas : resume de nouveau", artefactAuPas(replaybuild.CookFrameIntervalMS()),
			resumeAuPas(replaybuild.HistoricalFrameIntervalMS), usageAProjeter},
		{"meme pas : saute", artefactAuPas(replaybuild.CookFrameIntervalMS()),
			resumeAuPas(replaybuild.CookFrameIntervalMS()), usageDejaAJour},
		{"artefact encore a l'ancien pas, resume a l'ancien pas : saute (le remede est la regeneration de l'artefact)",
			artefactAuPas(replaybuild.HistoricalFrameIntervalMS), resumeAuPas(replaybuild.HistoricalFrameIntervalMS), usageDejaAJour},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, etat := projeterUnArtefact(context.Background(), c.path, "m1", usageSummaryOptions{}, c.passes); etat != c.want {
				t.Fatalf("etat = %d, attendu %d", etat, c.want)
			}
		})
	}
}
