// resources-manifest fabrique le manifeste d'un paquet de ressources (ADR 0045, amendement L24) :
// l'arbre de fichiers livré avec le programme qui change rarement. Il est le SEUL générateur : les
// scripts de construction de l'installeur, le workflow de publication et le Dockerfile l'appellent,
// le serveur lit ce qu'il écrit (internal/resourcepack).
//
// Usage :
//
//	resources-manifest -dir <paquet> [-version X.Y.Z] [-zip <archive.zip>]
//
// Écrit <paquet>/resources.manifest.json et imprime l'empreinte (64 caractères hexadécimaux) sur la
// sortie standard, seule. Avec -zip, écrit aussi l'archive du paquet. Code de sortie 1 en cas d'échec.
package main

import (
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/resourcepack"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("resources-manifest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "dossier du paquet")
	version := fs.String("version", "", "version documentaire du moteur qui produit le paquet")
	zipOut := fs.String("zip", "", "archive à écrire (facultatif)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprintln(stderr, "resources-manifest: -dir est obligatoire")
		return 2
	}
	m, err := resourcepack.Build(*dir, *version)
	if err == nil {
		err = resourcepack.WriteManifest(*dir, m)
	}
	if err == nil && *zipOut != "" {
		err = resourcepack.WriteZip(*dir, *zipOut)
	}
	if err != nil {
		fmt.Fprintln(stderr, "resources-manifest:", err)
		return 1
	}
	fmt.Fprintln(stdout, m.Fingerprint)
	return 0
}
