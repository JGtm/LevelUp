package main

// facts.go — LES FAITS DU MATCH, PAR LA MEME PORTE QUE LE BALAYAGE DU PARC.
//
// # POURQUOI UN SOUS-PROCESSUS, ET PAS UN IMPORT DIRECT
//
// `levelup replay-facts-export` ouvre la base partagee (`OpenReadForQuery`, CGO/DuckDB) et
// resout le registre des cartes candidates (`registreParShort`, non exporte : package `main`
// de `cmd/levelup`). Une troisieme copie de cette resolution DIVERGERAIT au premier champ
// ajoute (regle du depot, "une seule ecriture des memes requetes") ; ce fichier invoque donc
// L'OUTIL CANONIQUE tel quel, exactement comme la methode du balayage
// (.ai/V7.5/V2/BALAYAGE_PARC_2026-09-06.md §9). Ce gate reste lui-meme compilable SANS CGO
// (`go build ./cmd/replay-corpus-gate` marche a vide) : seule CETTE etape, en sous-processus,
// exige CGO/gcc — comme `go-api-test-gamefiles` exige le jeu pour SA seule etape.
//
// # LECTURE SEULE SUR LE PARC
//
// `LEVELUP_REPO_ROOT=parcRoot` : l'export lit `parcRoot/data/titles/{slug}/warehouse/...` en
// RO (`OpenReadForQuery`, jamais `OpenReadOnly` force) et n'ecrit que les `<short8>.facts.json`
// demandes, dans `factsDir` (une racine temporaire, jamais le parc).

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// exportFacts ecrit un `<short8>.facts.json` par id dans `factsDir`, en lecture seule sur la
// base partagee de `parcRoot`. `goAPIDir` est le repertoire depuis lequel lancer `go run`
// (celui qui porte `cmd/levelup` — cf. resolveSourceRoot).
func exportFacts(goAPIDir, parcRoot, titleSlug, factsDir string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := os.MkdirAll(factsDir, 0o750); err != nil {
		return fmt.Errorf("dossier des faits : %w", err)
	}
	args := append([]string{"run", "./cmd/levelup", "replay-facts-export",
		"--out", factsDir, "--title", titleSlug}, ids...)
	cmd := exec.Command("go", args...) //nolint:gosec // args = ids du manifeste + chemins internes
	cmd.Dir = goAPIDir
	cmd.Env = append(os.Environ(),
		"LEVELUP_REPO_ROOT="+parcRoot,
		"CGO_ENABLED=1",
	)
	var stderr, stdout bytes.Buffer
	cmd.Stderr, cmd.Stdout = &stderr, &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("export des faits (levelup replay-facts-export) : %w\nstdout:\n%s\nstderr:\n%s",
			err, stdout.String(), stderr.String())
	}
	return nil
}
