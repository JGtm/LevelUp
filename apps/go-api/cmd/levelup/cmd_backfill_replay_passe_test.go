package main

// cmd_backfill_replay_passe_test.go — LA PLANIFICATION (quels enfants, dans quel ordre, avec
// quels arguments) et LA VENTILATION des issues dans le recap.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
)

// ecrireArtefact pose un artefact de rejeu portant la version de schema demandee.
func ecrireArtefact(t *testing.T, repoRoot, slug, matchID string, schema int) {
	t.Helper()
	pr := titlePkg.NewPathResolver(repoRoot)
	path := pr.ReplayArtifactPath(slug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw, err := json.Marshal(map[string]int{"schemaVersion": schema})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

const (
	idPetitA = "aaaaaaaa-0000-0000-0000-000000000001"
	idPetitB = "bbbbbbbb-0000-0000-0000-000000000002"
	idMoyen  = "cccccccc-0000-0000-0000-000000000003"
	idGros   = "dddddddd-0000-0000-0000-000000000004"
)

func corpusDeTest() []replayCandidat {
	return []replayCandidat{
		{matchID: idGros, chunks: 50},
		{matchID: idPetitB, chunks: 8},
		{matchID: idMoyen, chunks: 13},
		{matchID: idPetitA, chunks: 8},
	}
}

func idsDe(cs []replayCandidat) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.matchID)
	}
	return out
}

// TestFiltrerEtTrierReplay_LesGrosEnDernier : l'ordre de la passe est reproductible, et a
// cout egal l'identifiant departage.
func TestFiltrerEtTrierReplay_LesGrosEnDernier(t *testing.T) {
	repo := t.TempDir()
	pr := titlePkg.NewPathResolver(repo)
	var r replayBackfillReport

	got := filtrerEtTrierReplay(corpusDeTest(), pr,
		replayBackfillOptions{titleSlug: titlePkg.DefaultSlug}, &r)

	veut := []string{idPetitA, idPetitB, idMoyen, idGros}
	if strings.Join(idsDe(got), ",") != strings.Join(veut, ",") {
		t.Fatalf("ordre = %v, veut %v", idsDe(got), veut)
	}
}

// TestFiltrerEtTrierReplay_LimitApresLeFiltre : un pilote `--limit 2` doit livrer 2
// constructions REELLES, pas 2 sauts.
func TestFiltrerEtTrierReplay_LimitApresLeFiltre(t *testing.T) {
	repo := t.TempDir()
	slug := titlePkg.DefaultSlug
	// Les deux moins chers sont deja a jour : le --limit 2 doit donc porter sur les SUIVANTS.
	ecrireArtefact(t, repo, slug, idPetitA, replay.SchemaVersion)
	ecrireArtefact(t, repo, slug, idPetitB, replay.SchemaVersion)

	var r replayBackfillReport
	got := filtrerEtTrierReplay(corpusDeTest(), titlePkg.NewPathResolver(repo),
		replayBackfillOptions{titleSlug: slug, limit: 2}, &r)

	if veut := []string{idMoyen, idGros}; strings.Join(idsDe(got), ",") != strings.Join(veut, ",") {
		t.Fatalf("liste d enfants = %v, veut %v", idsDe(got), veut)
	}
	if r.dejaAJour != 2 {
		t.Fatalf("dejaAJour = %d, veut 2", r.dejaAJour)
	}
}

// TestFiltrerEtTrierReplay_OnlyExisting : la passe d'apres un bump de schema ne retient que
// ce qui est DEJA servi — quelle que soit la version de l'artefact trouve.
func TestFiltrerEtTrierReplay_OnlyExisting(t *testing.T) {
	repo := t.TempDir()
	slug := titlePkg.DefaultSlug
	ecrireArtefact(t, repo, slug, idPetitA, replay.SchemaVersion-1) // perime : a re-cuire
	ecrireArtefact(t, repo, slug, idMoyen, replay.SchemaVersion)    // a jour : saute

	var r replayBackfillReport
	got := filtrerEtTrierReplay(corpusDeTest(), titlePkg.NewPathResolver(repo),
		replayBackfillOptions{titleSlug: slug, onlyExisting: true}, &r)

	if veut := []string{idPetitA}; strings.Join(idsDe(got), ",") != strings.Join(veut, ",") {
		t.Fatalf("liste d enfants = %v, veut %v", idsDe(got), veut)
	}
	if r.sansArtefact != 2 {
		t.Fatalf("sansArtefact = %d, veut 2 (les deux films sans artefact sur disque)", r.sansArtefact)
	}
	if r.dejaAJour != 1 {
		t.Fatalf("dejaAJour = %d, veut 1", r.dejaAJour)
	}
}

// TestFiltrerEtTrierReplay_Force : `--force` re-cuit meme ce qui est a jour.
func TestFiltrerEtTrierReplay_Force(t *testing.T) {
	repo := t.TempDir()
	slug := titlePkg.DefaultSlug
	ecrireArtefact(t, repo, slug, idPetitA, replay.SchemaVersion)

	var r replayBackfillReport
	got := filtrerEtTrierReplay(corpusDeTest(), titlePkg.NewPathResolver(repo),
		replayBackfillOptions{titleSlug: slug, force: true}, &r)

	if len(got) != 4 || r.dejaAJour != 0 {
		t.Fatalf("--force : %d enfants (veut 4), dejaAJour = %d (veut 0)", len(got), r.dejaAJour)
	}
}

// TestArgsEnfantReplay : la ligne de commande de l'enfant porte le film, le cache, le titre
// et le plafond — et AUCUN drapeau de planification.
func TestArgsEnfantReplay(t *testing.T) {
	o := replayBackfillOptions{
		titleSlug: "halo_infinite", memLimitGiB: 3,
		// Ces arbitrages sont deja rendus : ils ne doivent PAS repartir chez l'enfant.
		force: true, onlyExisting: true, dryRun: true, limit: 25,
	}
	c := replayCandidat{matchID: idMoyen, mapNames: []string{"Live Fire", "brut"}, chunks: 13}

	args := argsEnfantReplay(o, "C:/cache", c)

	if args[0] != "backfill-replay" {
		t.Fatalf("args[0] = %q, veut la sous-commande", args[0])
	}
	joint := strings.Join(args, " ")
	for _, veut := range []string{
		"--one " + idMoyen, "--cache C:/cache", "--title halo_infinite",
		"--mem-limit-gib 3", "--map-name Live Fire", "--map-name brut",
	} {
		if !strings.Contains(joint, veut) {
			t.Fatalf("args = %v — il manque %q", args, veut)
		}
	}
	for _, interdit := range []string{"--force", "--only-existing", "--dry-run", "--limit"} {
		if strings.Contains(joint, interdit) {
			t.Fatalf("args = %v — le drapeau de planification %q ne doit PAS partir chez l enfant",
				args, interdit)
		}
	}
}

// TestArgsEnfantReplay_OrdreDesCartes : l'ordre des candidats de carte est porteur de sens
// (asset EN d'abord, brut ensuite) — il doit survivre au passage parent -> enfant.
func TestArgsEnfantReplay_OrdreDesCartes(t *testing.T) {
	args := argsEnfantReplay(replayBackfillOptions{}, "C:/cache",
		replayCandidat{matchID: idMoyen, mapNames: []string{"premier", "second"}})

	var noms []string
	for i, a := range args {
		if a == "--map-name" && i+1 < len(args) {
			noms = append(noms, args[i+1])
		}
	}
	if strings.Join(noms, ",") != "premier,second" {
		t.Fatalf("ordre des cartes = %v, veut [premier second]", noms)
	}
}

// TestCandidatsCarte : l'ordre des identites de carte (asset EN d'abord, brut ensuite) et le
// rejet des vides sont LE point de verite partage parent (masse) / enfant (--one a la main).
func TestCandidatsCarte(t *testing.T) {
	cas := []struct {
		nom      string
		en, brut string
		veut     []string
	}{
		{"les deux, EN d abord", "Dredge", "e4bb06db-brut", []string{"Dredge", "e4bb06db-brut"}},
		{"EN vide -> brut seul", "", "Dredge", []string{"Dredge"}},
		{"brut vide -> EN seul", "Live Fire", "", []string{"Live Fire"}},
		{"les deux vides -> aucun", "  ", "", nil},
		{"espaces resserres au bord", "  Dredge  ", "  brut  ", []string{"Dredge", "brut"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := candidatsCarte(c.en, c.brut)
			if strings.Join(got, "|") != strings.Join(c.veut, "|") {
				t.Fatalf("candidatsCarte(%q,%q) = %v, veut %v", c.en, c.brut, got, c.veut)
			}
		})
	}
}

// TestTraiterResultatEnfant_Ventilation : CHAQUE issue tombe dans SA ligne de recap, et une
// mort d'enfant n'est jamais comptee comme une construction.
func TestTraiterResultatEnfant_Ventilation(t *testing.T) {
	cas := []struct {
		nom  string
		code int
		lire func(replayBackfillReport) int
	}{
		{"construit", filmproc.CodeOK, func(r replayBackfillReport) int { return r.construits }},
		{"hors catalogue", filmproc.CodeSkipped, func(r replayBackfillReport) int { return r.horsCatalogue }},
		{"erreur de decodage", filmproc.CodeFailed, func(r replayBackfillReport) int { return r.erreurs }},
		{"preparation", filmproc.CodePreparation, func(r replayBackfillReport) int { return r.preparation }},
		{"mort memoire", filmproc.CodeMemory, func(r replayBackfillReport) int { return r.mortsMemoire }},
		{"mort subite (fatal error)", 2, func(r replayBackfillReport) int { return r.mortsSubites }},
		{"mort subite (tue par l OS)", -1, func(r replayBackfillReport) int { return r.mortsSubites }},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var r replayBackfillReport
			res := filmproc.Result{Code: c.code, Issue: filmproc.IssueForCode(c.code)}
			traiterResultatEnfant(&r, res, idMoyen, 1, 1)

			if got := c.lire(r); got != 1 {
				t.Fatalf("code %d : la ligne de recap attendue vaut %d, veut 1", c.code, got)
			}
			if c.code != filmproc.CodeOK && r.construits != 0 {
				t.Fatalf("code %d compte a tort une construction", c.code)
			}
		})
	}
}

// TestLibellePlafond et TestLibelleOctets — deplaces depuis backfill_memlimit_test.go (lot
// v2 G.1, 2026-09-05) : la sentinelle memoire elle-meme est desormais internal/filmproc.Arm,
// mais ces deux fonctions d'AFFICHAGE du recap restent locales a cmd/levelup.

func TestLibellePlafond(t *testing.T) {
	if got := libellePlafond(0); got != "DESARME" {
		t.Fatalf("libellePlafond(0) = %q", got)
	}
	if got := libellePlafond(3); got != "3 GiB" {
		t.Fatalf("libellePlafond(3) = %q", got)
	}
}

func TestLibelleOctets(t *testing.T) {
	if got := libelleOctets(0); got != "inconnu" {
		t.Fatalf("libelleOctets(0) = %q — 0 veut dire NON MESURE, pas zero octet", got)
	}
	if got := libelleOctets(2 * octetsParGiB); got != "2.00 GiB" {
		t.Fatalf("libelleOctets(2 GiB) = %q", got)
	}
	if got := libelleOctets(512 * 1024 * 1024); got != "512 MiB" {
		t.Fatalf("libelleOctets(512 MiB) = %q", got)
	}
}
