// Package archlint — no_replay_firefight_literal_test.go : ratchet « UNE SEULE FORMULATION DU
// MODE NON GERE PAR LE REJEU » (decision du 2026-10-15 : le Firefight n'a pas de rejeu).
//
// # CE QU'IL EMPECHE
//
// Tout ce qui FABRIQUE un rejeu (file post-sync, `backfill-replay`, action admin, mise en file,
// choix des rejeux de la demo) ou le SERT (service de rejeu) doit poser la meme question au
// registre : `analysis.SQLModeNonGere`. Une formulation recopiee (`is_firefight` ecrit a la main)
// diverge au premier ajustement — un NULL oublie, un mode ajoute d'un cote seulement — et le
// rejeu se remet a fabriquer ou a servir ce qu'il ne gere pas. Le litteral n'est legitime que
// dans le fragment proprietaire.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fichiersDuRejeu : les motifs (relatifs a apps/go-api) des fichiers qui fabriquent ou servent un
// rejeu. Les tests (`_test.go`) sont exclus : ils fabriquent leurs tables avec la colonne.
var fichiersDuRejeu = []string{
	"internal/sync/replayartifacts/*.go",
	"internal/replaybuild/*.go",
	"cmd/levelup/cmd_backfill_replay*.go",
	"internal/api/wire/registry_replay_build.go",
	"internal/api/wire/registry_build_queue.go",
	"internal/service/replay_service.go",
	"internal/platform/duckdb/replay_mode_gate.go",
	"internal/platform/duckdb/replay_map_repo.go",
	"internal/ops/seed_demo_replay*.go",
}

// consommateursDuFragment : les fichiers qui DOIVENT employer le fragment (self-check positif :
// si l'un d'eux cesse de l'employer, le garde ne protege plus rien).
var consommateursDuFragment = []string{
	"internal/sync/replayartifacts/backlog.go",
	"internal/sync/replayartifacts/zones_cartes.go",
	"cmd/levelup/cmd_backfill_replay.go",
	"cmd/levelup/cmd_backfill_replay_nongeres.go",
	"internal/api/wire/registry_replay_build.go",
	"internal/platform/duckdb/replay_mode_gate.go",
	"internal/ops/seed_demo_replay_picks.go",
}

const (
	fragmentProprietaire = "internal/analysis/sql_fragments.go"
	litteralFirefight    = "is_firefight"
	nomDuFragment        = "SQLModeNonGere"
)

func TestUneSeuleFormulationDuModeNonGereDuRejeu(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	proprio, err := os.ReadFile(filepath.Join(apiRoot, fragmentProprietaire))
	if err != nil || !strings.Contains(string(proprio), litteralFirefight) {
		t.Fatalf("self-check : %s doit porter le litteral %q (%v)", fragmentProprietaire, litteralFirefight, err)
	}

	scannes := 0
	for _, motif := range fichiersDuRejeu {
		chemins, err := filepath.Glob(filepath.Join(apiRoot, filepath.FromSlash(motif)))
		if err != nil {
			t.Fatal(err)
		}
		for _, chemin := range chemins {
			if strings.HasSuffix(chemin, "_test.go") {
				continue
			}
			scannes++
			raw, err := os.ReadFile(chemin)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), litteralFirefight) {
				rel, _ := filepath.Rel(apiRoot, chemin)
				t.Errorf("%s : litteral %q interdit ici — employer analysis.%s (une seule formulation)",
					filepath.ToSlash(rel), litteralFirefight, nomDuFragment)
			}
		}
	}
	if scannes < 10 {
		t.Fatalf("self-check : %d fichiers scannes seulement — les motifs ne couvrent plus le rejeu", scannes)
	}

	for _, rel := range consommateursDuFragment {
		raw, err := os.ReadFile(filepath.Join(apiRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("self-check : %s illisible : %v", rel, err)
		}
		if !strings.Contains(string(raw), nomDuFragment) {
			t.Errorf("%s doit employer analysis.%s : le rejeu ne gere pas le mode non gere", rel, nomDuFragment)
		}
	}
}
