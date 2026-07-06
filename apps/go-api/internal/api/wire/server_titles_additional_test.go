package wire

import (
	"path/filepath"
	"runtime"
	"testing"

	"log/slog"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/mappings"
)

// worktreeRoot remonte du fichier de test à la racine du repo (où vit config/).
// apps/go-api/internal/api/<file> → 5 Dir() jusqu'à la racine.
func worktreeRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	root := thisFile
	for i := 0; i < 6; i++ { // K3d: +1 niveau (wire/ plus profond)
		root = filepath.Dir(root)
	}
	return root
}

// TestRegisterAdditionalTitles_NoopSansTitreActif est l'oracle BYTE-IDENTIQUE :
// quand seul Halo Infinite (défaut) est actif, la boucle n'enregistre RIEN —
// l'unique chemin reste Halo Infinite.
func TestRegisterAdditionalTitles_NoopSansTitreActif(t *testing.T) {
	resolver := games.NewStaticResolver(titlePkg.DefaultSlug)
	reg := NewServiceRegistry(&config.AppConfig{}, nil)
	fm := mappings.NewRegistry()

	RegisterAdditionalTitles(titlePkg.NewRegistry(), resolver, reg, fm)

	if got := resolver.Slugs(); len(got) != 0 {
		t.Fatalf("aucun adapter additionnel attendu (Halo Infinite câblé ailleurs), got %v", got)
	}
	if _, err := resolver.Semantic(halo5.TitleSlug); err == nil {
		t.Fatalf("halo_5 ne doit pas être enregistré tant qu'il n'est pas actif")
	}
}

// TestRegisterAdditionalTitles_CableHalo5Actif prouve que la boucle câble bien
// Halo 5 (semantic + data) quand son descripteur est actif et ses mappings
// chargés — sans serveur ni tokens (registration pure).
func TestRegisterAdditionalTitles_CableHalo5Actif(t *testing.T) {
	root := worktreeRoot(t)
	fm := mappings.NewRegistry()
	for _, err := range fm.LoadFromConfigDir(root, []string{halo5.TitleSlug}, slog.Default()) {
		t.Fatalf("chargement mappings halo_5: %v", err)
	}
	if _, ok := fm.Get(halo5.TitleSlug); !ok {
		t.Skipf("fields.toml halo_5 introuvable sous %s — skip (config absente)", root)
	}

	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:             halo5.TitleSlug,
		Name:             "Halo 5",
		Status:           titlePkg.StatusActive,
		PlacementMatches: 10,
	})

	resolver := games.NewStaticResolver(titlePkg.DefaultSlug)
	RegisterAdditionalTitles(reg, resolver, NewServiceRegistry(&config.AppConfig{RepoRoot: root}, nil), fm)

	sem, err := resolver.Semantic(halo5.TitleSlug)
	if err != nil || sem == nil {
		t.Fatalf("semantic halo_5 doit être enregistré: err=%v sem=%v", err, sem)
	}
	if sem.TitleSlug() != halo5.TitleSlug {
		t.Fatalf("semantic.TitleSlug()=%q, attendu %q", sem.TitleSlug(), halo5.TitleSlug)
	}
	data, err := resolver.Data(halo5.TitleSlug)
	if err != nil || data == nil {
		t.Fatalf("data halo_5 doit être enregistré: err=%v data=%v", err, data)
	}
	if data.TitleSlug() != halo5.TitleSlug {
		t.Fatalf("data.TitleSlug()=%q, attendu %q", data.TitleSlug(), halo5.TitleSlug)
	}
}

// TestRegisterAdditionalTitles_SkipTitreSansRegistrar : un titre actif ≠ défaut
// SANS registrar est sauté proprement (log warn, pas de panic, rien enregistré).
func TestRegisterAdditionalTitles_SkipTitreSansRegistrar(t *testing.T) {
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: "synthetic_unwired", Name: "Synthetic", Status: titlePkg.StatusActive,
	})

	resolver := games.NewStaticResolver(titlePkg.DefaultSlug)
	RegisterAdditionalTitles(reg, resolver, NewServiceRegistry(&config.AppConfig{}, nil), mappings.NewRegistry())

	if got := resolver.Slugs(); len(got) != 0 {
		t.Fatalf("titre sans registrar ne doit rien enregistrer, got %v", got)
	}
}
