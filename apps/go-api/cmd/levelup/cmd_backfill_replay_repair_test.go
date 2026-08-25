package main

// cmd_backfill_replay_repair_test.go — LE MODE `--repair-impoverished` : sa DECISION et sa
// VENTILATION, sans jamais decoder un film (la selection est pure, le resolveur de faits est injecte).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// ecrireArtefactAvecJoueurs pose un artefact portant une version de schema ET un nombre donne de
// compteurs de joueur (`scoreTimeline.players`). nbJoueurs = 0 produit un artefact APPAUVRI (a jour
// mais sans compteurs) — le cas meme que le mode reparation cible.
func ecrireArtefactAvecJoueurs(t *testing.T, repoRoot, slug, matchID string, schema, nbJoueurs int) {
	t.Helper()
	pr := titlePkg.NewPathResolver(repoRoot)
	path := pr.ReplayArtifactPath(slug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	doc := map[string]any{
		"schemaVersion": schema,
		"matchId":       matchID,
		"tracks":        []map[string]any{{}}, // au moins une trajectoire : artefact non « vide »
		"scoreTimeline": map[string]any{"players": make([]map[string]any, nbJoueurs)},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestClasserReparation : les 5 etats, ET la lecture PARESSEUSE de la base (le resolveur n'est
// interroge que pour un artefact a jour ET appauvri).
func TestClasserReparation(t *testing.T) {
	slug := titlePkg.DefaultSlug
	cas := []struct {
		nom        string
		schema     int
		joueursDoc int // compteurs ecrits sur disque ; -1 = ne rien ecrire (artefact absent)
		joueursDB  int
		appelDB    bool
		veut       etatReparation
	}{
		{"appauvri + base a des joueurs", replay.SchemaVersion, 0, 8, true, reparationACuire},
		{"appauvri + base sans joueur", replay.SchemaVersion, 0, 0, true, reparationVacuiteLegitime},
		{"deja riche", replay.SchemaVersion, 5, 0, false, reparationDejaComplet},
		{"hors schema courant", replay.SchemaVersion - 1, 0, 0, false, reparationHorsSchema},
		{"absent", 0, -1, 0, false, reparationSansArtefact},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo := t.TempDir()
			if c.joueursDoc >= 0 {
				ecrireArtefactAvecJoueurs(t, repo, slug, idPetitA, c.schema, c.joueursDoc)
			}
			pr := titlePkg.NewPathResolver(repo)
			appele := false
			got := classerReparation(pr.ReplayArtifactPath(slug, idPetitA), func() int {
				appele = true
				return c.joueursDB
			})
			if got != c.veut {
				t.Fatalf("etat = %d, veut %d", got, c.veut)
			}
			if appele != c.appelDB {
				t.Fatalf("resolveur appele = %v, veut %v (lecture DB paresseuse)", appele, c.appelDB)
			}
		})
	}
}

// TestSelectionnerReparables : les trois cas imposes ENSEMBLE — un appauvri reparable RETENU, un
// legitimement vide SAUTE, un riche INTACT (non re-cuit) — plus un hors-schema, et la preuve que le
// resolveur n'est interroge QUE pour les artefacts a jour + appauvris.
func TestSelectionnerReparables(t *testing.T) {
	repo := t.TempDir()
	slug := titlePkg.DefaultSlug
	ecrireArtefactAvecJoueurs(t, repo, slug, idPetitA, replay.SchemaVersion, 0) // appauvri -> reparable
	ecrireArtefactAvecJoueurs(t, repo, slug, idPetitB, replay.SchemaVersion, 8) // riche -> INTACT
	ecrireArtefactAvecJoueurs(t, repo, slug, idMoyen, replay.SchemaVersion, 0)  // appauvri, base vide -> SAUTE
	ecrireArtefactAvecJoueurs(t, repo, slug, idGros, replay.SchemaVersion-1, 0) // hors schema

	demandes := map[string]int{}
	joueursDe := func(matchID string) int {
		demandes[matchID]++
		switch matchID {
		case idPetitA:
			return 8
		case idMoyen:
			return 0
		default:
			t.Fatalf("resolveur interroge pour %s — riche/hors-schema doit court-circuiter", matchID)
			return 0
		}
	}

	var r replayBackfillReport
	got := selectionnerReparables(corpusDeTest(), titlePkg.NewPathResolver(repo),
		replayBackfillOptions{titleSlug: slug}, &r, joueursDe)

	if veut := []string{idPetitA}; strings.Join(idsDe(got), ",") != strings.Join(veut, ",") {
		t.Fatalf("reparables = %v, veut %v (seul l appauvri avec faits)", idsDe(got), veut)
	}
	if r.dejaComplets != 1 || r.vacuitesLegitimes != 1 || r.horsSchemaCourant != 1 {
		t.Fatalf("ventilation = complets %d, vacuites %d, horsSchema %d ; veut 1,1,1",
			r.dejaComplets, r.vacuitesLegitimes, r.horsSchemaCourant)
	}
	if demandes[idPetitB] != 0 || demandes[idGros] != 0 {
		t.Fatalf("resolveur interroge a tort : %v", demandes)
	}
}

// TestSelectionnerReparables_OrdreEtLimite : les reparables sortent tries par cout croissant (id
// departage a cout egal), puis `--limit` s'applique — meme contrat que la passe ordinaire.
func TestSelectionnerReparables_OrdreEtLimite(t *testing.T) {
	repo := t.TempDir()
	slug := titlePkg.DefaultSlug
	for _, id := range []string{idPetitA, idPetitB, idMoyen} {
		ecrireArtefactAvecJoueurs(t, repo, slug, id, replay.SchemaVersion, 0)
	}
	joueursDe := func(string) int { return 8 } // tous reparables

	var r replayBackfillReport
	got := selectionnerReparables(
		[]replayCandidat{{matchID: idMoyen, chunks: 13}, {matchID: idPetitB, chunks: 8}, {matchID: idPetitA, chunks: 8}},
		titlePkg.NewPathResolver(repo), replayBackfillOptions{titleSlug: slug, limit: 2}, &r, joueursDe)

	if veut := []string{idPetitA, idPetitB}; strings.Join(idsDe(got), ",") != strings.Join(veut, ",") {
		t.Fatalf("ordre+limite = %v, veut %v (les 2 moins chers, id departage)", idsDe(got), veut)
	}
}

// TestRunBackfillReplay_RepairEtForceSExcluent : le refus tombe AVANT tout acces cache/DB (D5).
func TestRunBackfillReplay_RepairEtForceSExcluent(t *testing.T) {
	err := runBackfillReplay(&config.AppConfig{}, []string{"--repair-impoverished", "--force"})
	if err == nil || !strings.Contains(err.Error(), "s'excluent") {
		t.Fatalf("erreur = %v, veut un refus d exclusion --repair-impoverished/--force", err)
	}
}
