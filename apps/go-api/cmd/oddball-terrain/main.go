// cmd/oddball-terrain — CONFRONTE le canal de score du film a la VERITE TERRAIN Oddball.
//
// Ce binaire repond a UNE question : le score decode du film (score personnel a la ms +
// emplacements du statborg) reproduit-il QUI PORTE le crane ? La verite terrain (d9781168,
// relevee image par image en Theater) calibre ; l'oracle fige (`D10_oracle_objective_stats.json`)
// generalise. Protocole : `.ai/V7.5/replay2d/registre_film/TERRAIN_PROTOCOLE.md`.
//
// # Deux processus, une seule sentinelle (patron cmd/statnames-sweep + cmd/zone-attribution)
//
//	default              PARENT : lit xuid->gamertag (OpenReadForQuery), charge l'oracle fige,
//	                     orchestre un enfant filmproc par film, aligne, confronte, ecrit les logs.
//	-child -match <id>   ENFANT : sentinelle memoire armee (2 Gio), decode UN film, emet le dump
//	                     tague. AUCUNE base ouverte.
//
// Usage (depuis apps/go-api, binaire compile — jamais `go run` en boucle) :
//
//	LEVELUP_REPO_ROOT=<principal> go build -o /tmp/oddball-terrain.exe ./cmd/oddball-terrain
//	LEVELUP_REPO_ROOT=<principal> /tmp/oddball-terrain.exe -out <registre_film>
package main

import (
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/domain/title"
)

func main() {
	cache := flag.String("cache", "", "racine du cache film (defaut : <repo>/data/cache)")
	child := flag.Bool("child", false, "ENFANT d'une passe (pose par le parent, pas par l'operateur)")
	match := flag.String("match", "", "ENFANT : l'unique film decode par ce processus")
	outDir := flag.String("out", "", "PARENT : dossier ou ecrire les logs (defaut : cwd)")
	oracle := flag.String("oracle", "", "PARENT : oracle JSON (defaut : <out>/D10_oracle_objective_stats.json)")
	flag.Parse()

	code, err := run(*child, *cache, *match, *outDir, *oracle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oddball-terrain: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(child bool, cache, match, outDir, oracle string) (int, error) {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return 0, err
	}
	if cache == "" {
		cache = repoRoot + "/data/cache"
	}
	if child {
		return runChild(cache, match), nil
	}
	if outDir == "" {
		outDir = "."
	}
	if oracle == "" {
		oracle = outDir + "/D10_oracle_objective_stats.json"
	}
	return 0, runParent(repoRoot, cache, outDir, oracle)
}
