// Command openapi-gen GÉNÈRE api/openapi.yaml (chantier V72-01 / H6).
//
// Le contrat n'est plus écrit à la main : il est la fusion du document OpenAPI
// PARTAGÉ de Huma (routes + schémas dérivés des types Go) et du fragment manuel
// versionné api/openapi_manual_fragment.yaml (tout ce qui n'est pas dérivable).
//
// Usage (depuis apps/go-api, ou via `make openapi-gen` depuis la racine) :
//
//	go run ./cmd/openapi-gen                 # écrit api/openapi.yaml
//	go run ./cmd/openapi-gen -check          # vérifie sans écrire (exit 1 si drift)
//
// Le golden TestOpenAPIYAMLIsUpToDate rejoue la MÊME fonction en mémoire et
// compare byte-à-byte : toute modification du code Go qui change le contrat sans
// régénération fait échouer les tests.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/api/openapigen"
)

func main() {
	out := flag.String("out", "api/openapi.yaml", "fichier de sortie")
	fragment := flag.String("fragment", "api/openapi_manual_fragment.yaml", "fragment manuel versionné")
	check := flag.Bool("check", false, "ne rien écrire ; sortie 1 si le fichier n'est pas à jour")
	verbose := flag.Bool("v", false, "conserver les logs d'assemblage du routeur (mode démo)")
	flag.Parse()

	// L'assemblage en mode démo journalise des WARN attendus (aucune DuckDB, aucun
	// TOML de titre dans le répertoire temporaire). Ils masquent la sortie utile :
	// on ne garde que les ERROR, sauf -v.
	if !*verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	}

	root, err := os.MkdirTemp("", "openapi-gen")
	if err != nil {
		fail("répertoire temporaire: %v", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	yamlBytes, err := openapigen.Generate(context.Background(), root, *fragment)
	if err != nil {
		fail("%v", err)
	}

	if *check {
		current, err := os.ReadFile(*out)
		if err != nil {
			fail("lecture %s: %v", *out, err)
		}
		if !bytes.Equal(current, yamlBytes) {
			fmt.Fprintf(os.Stderr, "openapi-gen: %s N'EST PAS à jour (%d octets attendus, %d présents) — relancer `make openapi-gen`\n",
				*out, len(yamlBytes), len(current))
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "openapi-gen: %s est à jour (%d octets)\n", *out, len(current))
		return
	}

	if err := os.WriteFile(*out, yamlBytes, 0o644); err != nil {
		fail("écriture %s: %v", *out, err)
	}
	fmt.Fprintf(os.Stderr, "openapi-gen: %s écrit (%d octets)\n", *out, len(yamlBytes))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "openapi-gen: "+format+"\n", args...)
	os.Exit(2)
}
