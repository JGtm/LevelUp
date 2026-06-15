// Package mappings — validate.go : validation de la présence des mappings TOML
// requis pour un titre (PMT-12 / MT-21).
//
// Complément BOOT (fail-fast) du diagnostic runtime admin (PMT-14) : au démarrage,
// un titre actif sans ses mappings requis doit faire échouer le boot plutôt que
// servir des pages à moitié configurées.
package mappings

import (
	"fmt"
	"os"
	"path/filepath"
)

// requiredTitleTOML = mappings TOML obligatoires pour un titre exploitable :
// fields.toml (labels/units/format) + capabilities.toml (surfaces produit).
// Les autres (outcomes/assets/constants) sont optionnels (dégradation gracieuse).
var requiredTitleTOML = []string{"fields.toml", "capabilities.toml"}

// ValidateRequiredTOML vérifie que les mappings TOML requis existent pour le
// titre `slug` sous config/titles/{slug}/mappings/. Retourne une erreur par
// fichier requis manquant (slice vide si tout est présent). Read-only (os.Stat).
func ValidateRequiredTOML(repoRoot, slug string) []error {
	dir := filepath.Join(repoRoot, "config", "titles", slug, "mappings")
	var errs []error
	for _, name := range requiredTitleTOML {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			errs = append(errs, fmt.Errorf("titre %q : mapping requis manquant : %s", slug, name))
		}
	}
	return errs
}
