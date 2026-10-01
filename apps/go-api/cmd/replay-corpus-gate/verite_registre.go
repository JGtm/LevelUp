package main

// verite_registre.go — LE REGISTRE DES REPLIS DE LA BASE, POUR LA REGLE R-1 DU BANC.
//
// R-1 excuse un repli « nouveau » quand son compteur n'etait simplement PAS BRANCHE a la base (le
// repli decidait deja, on ne le comptait pas — cas de J8). Seul le registre de la BASE le dit
// (`film/internal/facts/fallback`, champ `CompteurBranche`) : il est produit par l'outil du banc
// COMPILE A LA BASE (`cmd/replay-verite -registre`), dans le worktree de base que le gate cree
// deja, avec le GOCACHE de la base (les paquets du decodeur y sont deja compiles pour
// `replay-build`).
//
// # POURQUOI PAS PAR LE CACHE DE LA BASE (choix du 2026-09-30)
//
// Le registre ne depend QUE du code de la base, donc du SHA complet que la cle du cache porte
// deja : l'y ranger n'ajouterait aucune invalidation juste, seulement un quatrieme fichier par
// temoin (le registre est le meme pour tous) et un cas « entree ancienne sans registre » a gerer.
// Il se calcule UNE FOIS par passage, meme quand tous les artefacts de la base viennent du cache
// (le worktree de base et sa compilation de `replay-build` existent toujours) ; son cout est la
// compilation d'un petit outil sur un GOCACHE chaud.
//
// # UNE BASE SANS L'OUTIL
//
// Une base anterieure au banc ne porte pas `cmd/replay-verite` : le registre est alors INCONNU
// (nil), c'est journalise, le rapport le dit, et tout repli nouveau est un FAUX. Ce cas disparait
// des que la base contient le banc.

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"levelup/go-api/internal/replayverite"
)

// cheminOutilDuBanc : l'outil du banc, relatif a `apps/go-api`.
const cheminOutilDuBanc = "cmd/replay-verite"

// registreDeLaBase compile l'outil du banc a la base et lit son registre ; nil (journalise) sur
// tout echec — le banc juge alors avec un registre d'avant inconnu.
func registreDeLaBase(ctx context.Context, goAPIDirBase, gocache, sortie string) replayverite.RegistreReplis {
	if _, err := os.Stat(filepath.Join(goAPIDirBase, filepath.FromSlash(cheminOutilDuBanc))); err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: la base ne porte pas l'outil du banc — registre des "+
			"replis d'avant INCONNU, tout repli nouveau sera un FAUX", "outil", cheminOutilDuBanc, "err", err)
		return nil
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", sortie, "./"+cheminOutilDuBanc)
	build.Dir = goAPIDirBase
	build.Env = append(os.Environ(), "GOCACHE="+gocache, "CGO_ENABLED=0")
	var stderr bytes.Buffer
	build.Stderr = &stderr
	if err := build.Run(); err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: compilation de l'outil du banc a la base impossible — "+
			"registre des replis d'avant INCONNU", "err", err, "stderr", stderr.String())
		return nil
	}
	reg, err := lireRegistreDeLOutil(ctx, sortie)
	if err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: registre des replis de la base illisible — INCONNU",
			"err", err)
		return nil
	}
	slog.InfoContext(ctx, "replay-corpus-gate: registre des replis de la base lu", "replis", len(reg))
	return reg
}

// lireRegistreDeLOutil execute `<outil> -registre` et decode sa sortie.
func lireRegistreDeLOutil(ctx context.Context, outil string) (replayverite.RegistreReplis, error) {
	cmd := exec.CommandContext(ctx, outil, "-registre") //nolint:gosec // binaire compile par ce gate
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s -registre : %w\n%s", outil, err, stderr.String())
	}
	return replayverite.LireRegistre(stdout.Bytes())
}
