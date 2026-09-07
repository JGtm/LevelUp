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
//
// # EXPORT PAR TEMOIN, PAS EN UN SEUL LOT (CORPUS-R1 C4, 2026-09-07)
//
// Une invocation UNIQUE portant tous les ids faisait echouer TOUT le sous-processus (exit 2)
// au premier id inconnu du registre — AVANT toute cuisson, y compris pour les temoins valides
// du meme manifeste. Chaque id est desormais exporte par une invocation SEPAREE : un id
// inconnu ne fait echouer QUE lui (`slog.Warn`, jamais fatal), son `<short8>.facts.json` reste
// simplement absent — `orchestrate.go` le detecte alors comme un temoin ABSENT (jamais une
// ERREUR), et `verifierCouverture` (report.go, C3) en fait un plancher de couverture explicite
// plutot qu'un silence.

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// exporterUnFait exporte les faits d'UN match — extrait en type nommé pour permettre un test
// unitaire de la boucle de continuation (exportFactsAvec) SANS vrai sous-processus CGO.
type exporterUnFait func(id string) error

// exportParams regroupe les chemins fixes d'un export — un struct plutot qu'une signature a
// plus de 5 parametres une fois `ctx` ajoute (CLAUDE.md n°5).
type exportParams struct {
	GoAPIDir  string // depuis ou lancer `go run` (celui qui porte cmd/levelup)
	ParcRoot  string
	TitleSlug string
	FactsDir  string
}

// exportFacts ecrit un `<short8>.facts.json` PAR id dans `p.FactsDir`, en lecture seule sur la
// base partagee de `p.ParcRoot`.
func exportFacts(ctx context.Context, p exportParams, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := os.MkdirAll(p.FactsDir, 0o750); err != nil {
		return fmt.Errorf("dossier des faits : %w", err)
	}
	exportFactsAvec(func(id string) error {
		return exportUnFait(ctx, p, id)
	}, ids)
	return nil
}

// exportFactsAvec applique `exporter` a chaque id, EN CONTINUANT apres un echec — le coeur du
// correctif C4, teste independamment du sous-processus reel (facts_test.go).
func exportFactsAvec(exporter exporterUnFait, ids []string) {
	for _, id := range ids {
		if err := exporter(id); err != nil {
			slog.Warn("replay-corpus-gate: export des faits impossible pour ce temoin — ignore, "+
				"les autres temoins du manifeste continuent",
				"temoin", id, "err", err)
		}
	}
}

// exportUnFait invoque `levelup replay-facts-export` pour UN SEUL id.
func exportUnFait(ctx context.Context, p exportParams, id string) error {
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/levelup", "replay-facts-export", //nolint:gosec // id du manifeste + chemins internes
		"--out", p.FactsDir, "--title", p.TitleSlug, id)
	cmd.Dir = p.GoAPIDir
	cmd.Env = append(os.Environ(),
		"LEVELUP_REPO_ROOT="+p.ParcRoot,
		"CGO_ENABLED=1",
	)
	var stderr, stdout bytes.Buffer
	cmd.Stderr, cmd.Stdout = &stderr, &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("export des faits (levelup replay-facts-export %s) : %w\nstdout:\n%s\nstderr:\n%s",
			id, err, stdout.String(), stderr.String())
	}
	return nil
}
