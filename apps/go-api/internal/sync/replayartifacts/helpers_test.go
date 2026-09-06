package replayartifacts

// helpers_test.go — les aides partagees par les tests du paquet, unitaires ET d'integration.
//
// SANS TAG DE BUILD, ET C'EST LE POINT : `racineDepot` vivait dans un fichier
// `//go:build integration`, donc invisible des tests unitaires. Le premier test unitaire qui a
// eu besoin des capabilities LIVREES (derivations_marque_test.go) l'aurait recopiee — deuxieme
// exemplaire, qui aurait diverge au premier deplacement du paquet.

import (
	"os"
	"path/filepath"
	"testing"
)

// racineDepot remonte du package au depot (apps/go-api/internal/sync/replayartifacts -> racine)
// pour que les gardes par capability lisent les `capabilities.toml` LIVRES.
func racineDepot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := filepath.Join(wd, "..", "..", "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "config", "titles")); err != nil {
		t.Fatalf("racine du depot introuvable depuis %s: %v", wd, err)
	}
	return root
}
