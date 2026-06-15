//go:build cgo

// levelup-titles — CLI de diagnostic des titres (PMT-14 volet A).
//
// Sous-commande `diagnose` : appelle directement service.TitleDiagnosticService
// (le même que GET /admin/titles/{slug}/diagnostic) et imprime le rapport en
// table texte ou en JSON (--format=json). Read-only.
//
// Usage :
//
//	levelup-titles diagnose --slug halo_infinite [--format text|json] [--repo-root .]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "diagnose" {
		fmt.Fprintln(os.Stderr, "usage: levelup-titles diagnose --slug <slug> [--format text|json] [--repo-root <path>]")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("diagnose", flag.ExitOnError)
	slug := fs.String("slug", "halo_infinite", "slug du titre à diagnostiquer")
	format := fs.String("format", "text", "format de sortie : text | json")
	repoRoot := fs.String("repo-root", ".", "racine du repo (config/titles, data/)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	svc := service.NewTitleDiagnosticService(*repoRoot, platform_duckdb.NewTableInspector())
	rep, err := svc.Diagnose(context.Background(), *slug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erreur de diagnostic :", err)
		os.Exit(1)
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintln(os.Stderr, "erreur encodage JSON :", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprint(os.Stdout, renderDiagnosticText(rep))
}
