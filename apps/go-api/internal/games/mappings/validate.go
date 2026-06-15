// Package mappings — validate.go : validation de la présence des mappings TOML
// requis pour un titre (PMT-12 / MT-21).
//
// Complément BOOT (fail-fast) du diagnostic runtime admin (PMT-14) : au démarrage,
// un titre actif sans ses mappings requis doit faire échouer le boot plutôt que
// servir des pages à moitié configurées.
//
// Le required-set est DÉRIVÉ des capabilities déclarées du titre
// (title.Capability), jamais d'un switch sur le slug : un 2e titre déclare ses
// capabilities et hérite automatiquement du bon ensemble de fichiers requis.
package mappings

import (
	"fmt"
	"os"
	"path/filepath"

	titlePkg "levelup/go-api/internal/domain/title"
)

// RequiredTOMLFor dérive la liste des mappings TOML requis pour un titre à
// partir de ses capabilities déclarées (title.Capability).
//   - fields.toml + capabilities.toml : toujours (labels/units + surfaces produit).
//   - assets.toml   : si CapAssetImages (thumbnails maps/armes).
//   - outcomes.toml : si CapMatchmaking (résultats de match win/loss/tie/dnf).
//
// Les autres (awards/constants) restent optionnels (dégradation gracieuse).
func RequiredTOMLFor(desc *titlePkg.TitleDescriptor) []string {
	req := []string{"fields.toml", "capabilities.toml"}
	if desc.HasCapability(titlePkg.CapAssetImages) {
		req = append(req, "assets.toml")
	}
	if desc.HasCapability(titlePkg.CapMatchmaking) {
		req = append(req, "outcomes.toml")
	}
	return req
}

// requiredByLabel décrit la capability/raison qui rend un fichier requis (pour
// l'observabilité — clé de log required_by).
func requiredByLabel(file string) string {
	switch file {
	case "assets.toml":
		return string(titlePkg.CapAssetImages)
	case "outcomes.toml":
		return string(titlePkg.CapMatchmaking)
	default:
		return "always"
	}
}

// MissingRequiredTOML — erreur structurée : un mapping TOML requis est absent
// pour un titre. Porte le chemin et la raison (capability) pour le logging
// d'observabilité au boot (required_toml_missing).
type MissingRequiredTOML struct {
	Slug       string
	File       string
	Path       string
	RequiredBy string
}

func (m MissingRequiredTOML) Error() string {
	return fmt.Sprintf("titre %q : mapping requis manquant : %s (requis par %s)",
		m.Slug, m.File, m.RequiredBy)
}

// ValidateRequiredTOML vérifie que les mappings TOML requis (dérivés des
// capabilities de `desc`) existent sous config/titles/{slug}/mappings/.
// Read-only (os.Stat). Retourne une erreur structurée MissingRequiredTOML par
// fichier requis manquant (slice vide si tout est présent).
func ValidateRequiredTOML(repoRoot string, desc *titlePkg.TitleDescriptor) []error {
	dir := filepath.Join(repoRoot, "config", "titles", desc.Slug, "mappings")
	var errs []error
	for _, name := range RequiredTOMLFor(desc) {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			errs = append(errs, MissingRequiredTOML{
				Slug:       desc.Slug,
				File:       name,
				Path:       p,
				RequiredBy: requiredByLabel(name),
			})
		}
	}
	return errs
}
