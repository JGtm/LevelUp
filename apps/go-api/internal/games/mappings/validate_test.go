package mappings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// repoRootForTest remonte depuis ce fichier jusqu'au répertoire contenant
// config/titles (racine du repo), pour exercer le validateur contre la VRAIE
// config (oracle de parité et de boot-safety).
func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller indisponible")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("racine repo (config/titles) introuvable depuis le test")
	return ""
}

// TestRequiredTOMLFor — le required-set est dérivé des capabilities, pas d'un slug.
func TestRequiredTOMLFor(t *testing.T) {
	base := &titlePkg.TitleDescriptor{Slug: "x"}
	if got := RequiredTOMLFor(base); len(got) != 2 || !contains(got, "fields.toml") || !contains(got, "capabilities.toml") {
		t.Errorf("sans capability : attendu {fields, capabilities}, got %v", got)
	}
	withAssets := &titlePkg.TitleDescriptor{Slug: "x", Capabilities: []titlePkg.Capability{titlePkg.CapAssetImages}}
	if !contains(RequiredTOMLFor(withAssets), "assets.toml") {
		t.Error("CapAssetImages doit requérir assets.toml")
	}
	withMM := &titlePkg.TitleDescriptor{Slug: "x", Capabilities: []titlePkg.Capability{titlePkg.CapMatchmaking}}
	if !contains(RequiredTOMLFor(withMM), "outcomes.toml") {
		t.Error("CapMatchmaking doit requérir outcomes.toml")
	}
}

// TestValidateRequiredTOML_TempDir — un descripteur sans capability asset/match
// requiert fields + capabilities : 0/1/2 fichiers présents → 2/1/0 erreurs.
func TestValidateRequiredTOML_TempDir(t *testing.T) {
	root := t.TempDir()
	slug := "test_title"
	dir := filepath.Join(root, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	desc := &titlePkg.TitleDescriptor{Slug: slug}

	if errs := ValidateRequiredTOML(root, desc); len(errs) != 2 {
		t.Fatalf("0 fichier → 2 erreurs, got %d : %v", len(errs), errs)
	}
	mustWriteFile(t, filepath.Join(dir, "fields.toml"))
	if errs := ValidateRequiredTOML(root, desc); len(errs) != 1 {
		t.Fatalf("fields seul → 1 erreur, got %d : %v", len(errs), errs)
	}
	mustWriteFile(t, filepath.Join(dir, "capabilities.toml"))
	if errs := ValidateRequiredTOML(root, desc); len(errs) != 0 {
		t.Errorf("les deux → 0 erreur, got %d : %v", len(errs), errs)
	}
}

// TestValidateRequiredTOML_RealHaloInfinite_Passes — ORACLE (a) parité/boot-safety :
// le VRAI halo_infinite (descripteur prod + config réelle) passe le validateur
// boot. Sans ce test, un required-set mal calibré ferait os.Exit(1) en prod.
func TestValidateRequiredTOML_RealHaloInfinite_Passes(t *testing.T) {
	root := repoRootForTest(t)
	hi := titlePkg.NewRegistry().Default()
	if hi == nil {
		t.Fatal("descripteur halo_infinite introuvable dans le registre")
	}
	if errs := ValidateRequiredTOML(root, hi); len(errs) != 0 {
		t.Fatalf("halo_infinite réel doit passer le boot fail-fast, got %d erreur(s) : %v", len(errs), errs)
	}
}

// TestValidateRequiredTOML_SyntheticDrift_MissingCapabilities — ORACLE (b) :
// synthetic_title_b (fixture : fields/assets/outcomes présents, capabilities.toml
// ABSENT) → exactement 1 erreur STRUCTURÉE nommant slug + capabilities.toml,
// sans panic. Preuve que le validateur détecte un titre mal configuré.
func TestValidateRequiredTOML_SyntheticDrift_MissingCapabilities(t *testing.T) {
	root := repoRootForTest(t)
	desc := &titlePkg.TitleDescriptor{
		Slug:         "synthetic_title_b",
		Status:       titlePkg.StatusComingSoon,
		Capabilities: []titlePkg.Capability{titlePkg.CapAssetImages, titlePkg.CapMatchmaking},
	}
	errs := ValidateRequiredTOML(root, desc)
	if len(errs) != 1 {
		t.Fatalf("synthetic_title_b → attendu 1 erreur (capabilities.toml), got %d : %v", len(errs), errs)
	}
	m, ok := errs[0].(MissingRequiredTOML)
	if !ok {
		t.Fatalf("erreur doit être structurée MissingRequiredTOML, got %T", errs[0])
	}
	if m.Slug != "synthetic_title_b" || m.File != "capabilities.toml" {
		t.Errorf("erreur doit nommer slug+fichier, got slug=%q file=%q", m.Slug, m.File)
	}
	if m.Path == "" {
		t.Error("erreur structurée doit porter le path (observabilité)")
	}
}
