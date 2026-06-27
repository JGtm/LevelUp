package api

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/config"
)

// writeExperienceRules crée un experience_rules.toml minimal valide
// (config/titles/<slug>/catalog/) sous root pour le titre donné.
func writeExperienceRules(t *testing.T, root, titleSlug string) {
	t.Helper()
	dir := filepath.Join(root, "config", "titles", titleSlug, "catalog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	const rules = "schema_version = 1\n\n[[rule]]\nexperience = \"ranked\"\n[rule.match_any]\nis_ranked = true\n"
	if err := os.WriteFile(filepath.Join(dir, "experience_rules.toml"), []byte(rules), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}
}

// TestHasCatalogAdapter_PresentWhenRulesExist : un titre avec un experience_rules.toml
// valide a un catalog adapter résolvable (gate RÉEL → true, le drain/sweep tourne).
func TestHasCatalogAdapter_PresentWhenRulesExist(t *testing.T) {
	root := t.TempDir()
	writeExperienceRules(t, root, "halo_infinite")
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}

	if !reg.HasCatalogAdapter("halo_infinite") {
		t.Error("HasCatalogAdapter(halo_infinite) = false, want true (experience_rules.toml présent)")
	}
}

// TestHasCatalogAdapter_AbsentWhenNoRules : un titre SANS experience_rules.toml (modèle
// Halo 5, catalogue metadata-side) n'a pas d'adapter résolvable → false (skip propre).
// C'est l'équivalent fonctionnel de resolver.Catalog(slug) == ErrTitleNotResolved.
func TestHasCatalogAdapter_AbsentWhenNoRules(t *testing.T) {
	root := t.TempDir() // aucun fichier de règles écrit
	reg := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: root}}

	if reg.HasCatalogAdapter("halo_5") {
		t.Error("HasCatalogAdapter(halo_5) = true, want false (pas d'experience_rules.toml)")
	}
}

// TestHasCatalogAdapter_NilSafe : entrées dégénérées → false, jamais de panic.
func TestHasCatalogAdapter_NilSafe(t *testing.T) {
	var nilReg *ServiceRegistry
	if nilReg.HasCatalogAdapter("halo_infinite") {
		t.Error("HasCatalogAdapter sur registry nil = true, want false")
	}
	regNoCfg := &ServiceRegistry{}
	if regNoCfg.HasCatalogAdapter("halo_infinite") {
		t.Error("HasCatalogAdapter sans cfg = true, want false")
	}
	regEmptySlug := &ServiceRegistry{cfg: &config.AppConfig{RepoRoot: t.TempDir()}}
	if regEmptySlug.HasCatalogAdapter("") {
		t.Error("HasCatalogAdapter(\"\") = true, want false")
	}
}
