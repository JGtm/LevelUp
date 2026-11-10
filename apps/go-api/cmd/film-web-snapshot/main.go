// cmd/film-web-snapshot — regenere l'instantane des fichiers de la brique du decodeur de film que les
// tests web relisent (cf. snapshot.go).
//
// Usage : go run ./cmd/film-web-snapshot [--check]
//
// `--check` n'ecrit rien et sort en 1 si l'instantane commis n'est pas celui que la brique produirait ;
// `TestInstantaneEstFrais` tient la meme verification a chaque `go test`. L'outil lit la brique par le
// resolveur des briques (dossier de l'espace de travail ou du cache de modules) et n'ecrit que dans le depot.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
)

func main() {
	check := flag.Bool("check", false, "ne rien ecrire ; sortir en 1 si l'instantane commis differe de la brique")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintf(os.Stderr, "erreur: %v\n", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	racine, err := title.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("racine du depot : %w", err)
	}
	attendu, err := generer()
	if err != nil {
		return err
	}
	if check {
		diff, err := ecarts(racine, attendu)
		if err != nil {
			return err
		}
		if len(diff) > 0 {
			return fmt.Errorf("instantane perime :\n  %s\nregenerer par `go run ./cmd/film-web-snapshot`", strings.Join(diff, "\n  "))
		}
		fmt.Println("instantane a jour")
		return nil
	}
	if err := ecrire(racine, attendu); err != nil {
		return err
	}
	fmt.Printf("instantane ecrit : %d fichiers sous %s\n", len(attendu), dossierDuWeb)
	return nil
}
