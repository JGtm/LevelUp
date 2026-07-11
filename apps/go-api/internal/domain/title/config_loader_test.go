package title

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTitleFile(t *testing.T, repoRoot, slug, name string, content string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "config", "titles", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

const validSyntheticManifest = `
[meta]
title_slug = "synthetic_title_b"
schema_version = 1

[title]
name = "Synthetic Title B"
provider = "synthetic"
status = "active"
xbox_title_id = "999000111"
capabilities = ["matchmaking", "career", "media"]
`

func TestLoadTitleManifestFromBytes_Valid(t *testing.T) {
	desc, err := LoadTitleManifestFromBytes("title.toml", "synthetic_title_b", []byte(validSyntheticManifest))
	if err != nil {
		t.Fatalf("expected valid manifest, got err: %v", err)
	}
	if desc.Slug != "synthetic_title_b" {
		t.Errorf("slug = %q, want synthetic_title_b", desc.Slug)
	}
	if desc.Name != "Synthetic Title B" {
		t.Errorf("name = %q", desc.Name)
	}
	if desc.Provider != "synthetic" {
		t.Errorf("provider = %q", desc.Provider)
	}
	if desc.Status != StatusActive {
		t.Errorf("status = %q, want active", desc.Status)
	}
	if desc.XboxTitleID != "999000111" {
		t.Errorf("xbox_title_id = %q", desc.XboxTitleID)
	}
	if got := len(desc.Capabilities); got != 3 {
		t.Fatalf("capabilities len = %d, want 3", got)
	}
	if !desc.HasCapability(CapCareer) || desc.HasCapability(CapLUSR) {
		t.Errorf("capabilities mismatch: %v", desc.Capabilities)
	}
}

// TestLoadTitleManifestFromBytes_IsInternal : le flag [title].is_internal est parsé
// (true) et vaut false par défaut quand absent (vrai titre). Garantit que
// synthetic_title_b (is_internal=true) sera exclu du switcher via PublicTitles.
func TestLoadTitleManifestFromBytes_IsInternal(t *testing.T) {
	internalManifest := `
[meta]
title_slug = "fixture_x"
schema_version = 1

[title]
name = "Fixture X"
provider = "synthetic"
status = "coming_soon"
is_internal = true
`
	desc, err := LoadTitleManifestFromBytes("title.toml", "fixture_x", []byte(internalManifest))
	if err != nil {
		t.Fatalf("manifest interne valide attendu, err: %v", err)
	}
	if !desc.IsInternal {
		t.Error("is_internal=true doit être parsé en IsInternal=true")
	}

	// Absent du TOML → défaut false (vrai titre, ex le fixture partagé sans is_internal).
	descDefault, err := LoadTitleManifestFromBytes("title.toml", "synthetic_title_b", []byte(validSyntheticManifest))
	if err != nil {
		t.Fatalf("manifest valide attendu, err: %v", err)
	}
	if descDefault.IsInternal {
		t.Error("is_internal absent doit valoir false (défaut vrai titre)")
	}
}

func TestLoadTitleManifestFromBytes_Invalid(t *testing.T) {
	cases := map[string]string{
		"status inconnu": `
[meta]
title_slug = "x"
schema_version = 1
[title]
name = "X"
status = "live"
`,
		"capability inconnue": `
[meta]
title_slug = "x"
schema_version = 1
[title]
name = "X"
status = "active"
capabilities = ["matchmaking", "telepathy"]
`,
		"schema_version manquant": `
[title]
name = "X"
status = "active"
`,
		"name manquant": `
[meta]
title_slug = "x"
schema_version = 1
[title]
status = "active"
`,
		"slug discordant": `
[meta]
title_slug = "autre"
schema_version = 1
[title]
name = "X"
status = "active"
`,
		"is_default usurpé": `
[meta]
title_slug = "x"
schema_version = 1
[title]
name = "X"
status = "active"
is_default = true
`,
	}
	for label, content := range cases {
		t.Run(label, func(t *testing.T) {
			if _, err := LoadTitleManifestFromBytes("title.toml", "x", []byte(content)); err == nil {
				t.Errorf("expected validation error for %q, got nil", label)
			}
		})
	}
}

func TestLoadTitlesIntoRegistry_DiscoversAdditional(t *testing.T) {
	repoRoot := t.TempDir()
	// Titre additionnel valide.
	writeTitleFile(t, repoRoot, "synthetic_title_b", "title.toml", validSyntheticManifest)
	// halo_infinite : doit être ignoré (built-in fait foi) même avec un title.toml.
	writeTitleFile(t, repoRoot, "halo_infinite", "title.toml", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[title]
name = "OVERRIDE NE DOIT PAS S'APPLIQUER"
status = "archived"
`)
	// Dossier mappings-only (pas de title.toml) : ignoré silencieusement.
	writeTitleFile(t, repoRoot, "fixtures_only", "fields.toml", "schema_version = 1\n")

	reg := NewRegistry() // built-in halo_infinite
	errs := LoadTitlesIntoRegistry(reg, repoRoot, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected load errors: %v", errs)
	}

	// halo_infinite reste le built-in (Name non écrasé, Status active).
	hi := reg.Get(DefaultSlug)
	if hi == nil || hi.Name != "Halo Infinite" || !hi.IsActive() {
		t.Errorf("halo_infinite built-in altéré: %+v", hi)
	}
	// Le titre additionnel est enregistré.
	b := reg.Get("synthetic_title_b")
	if b == nil {
		t.Fatal("synthetic_title_b non enregistré")
	}
	if !b.HasCapability(CapMatchmaking) || b.HasCapability(CapForge) {
		t.Errorf("capabilities synthetic_title_b inattendues: %v", b.Capabilities)
	}
	// fixtures_only n'est pas un titre.
	if reg.Exists("fixtures_only") {
		t.Error("fixtures_only ne doit pas être enregistré (pas de title.toml)")
	}
}

// F11 : un title.toml pour le titre built-in doit émettre un WARN explicite
// (sinon un dev croit ses edits pris en compte alors qu'ils sont ignorés).
func TestLoadTitlesIntoRegistry_BuiltinTOMLWarns(t *testing.T) {
	repoRoot := t.TempDir()
	writeTitleFile(t, repoRoot, "halo_infinite", "title.toml", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[title]
name = "OVERRIDE"
status = "archived"
`)
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if errs := LoadTitlesIntoRegistry(NewRegistry(), repoRoot, logger); len(errs) != 0 {
		t.Fatalf("unexpected load errors: %v", errs)
	}
	if !strings.Contains(buf.String(), "title_builtin_toml_ignored") {
		t.Errorf("WARN title_builtin_toml_ignored attendu, logs: %s", buf.String())
	}
}

func TestLoadTitlesIntoRegistry_InvalidDoesNotBlockOthers(t *testing.T) {
	repoRoot := t.TempDir()
	writeTitleFile(t, repoRoot, "good_title", "title.toml", `
[meta]
title_slug = "good_title"
schema_version = 1
[title]
name = "Good"
status = "active"
capabilities = ["matchmaking"]
`)
	writeTitleFile(t, repoRoot, "bad_title", "title.toml", `
[meta]
schema_version = 1
[title]
name = "Bad"
status = "POUET"
`)
	reg := NewRegistry()
	errs := LoadTitlesIntoRegistry(reg, repoRoot, nil)
	if len(errs) == 0 {
		t.Fatal("expected an error for bad_title")
	}
	if !reg.Exists("good_title") {
		t.Error("good_title doit être enregistré malgré l'échec de bad_title")
	}
	if reg.Exists("bad_title") {
		t.Error("bad_title ne doit pas être enregistré (config invalide)")
	}
}

func TestLoadTitlesIntoRegistry_NoConfigDir(t *testing.T) {
	reg := NewRegistry()
	errs := LoadTitlesIntoRegistry(reg, t.TempDir(), nil) // pas de config/titles
	if len(errs) != 0 {
		t.Fatalf("expected no-op (mono-titre), got: %v", errs)
	}
	if len(reg.All()) != 1 || !reg.Exists(DefaultSlug) {
		t.Errorf("registre devrait rester mono-titre built-in, got %d titres", len(reg.All()))
	}
}

func TestNewRegistryFromConfig_BuiltinPlusDiscovered(t *testing.T) {
	repoRoot := t.TempDir()
	writeTitleFile(t, repoRoot, "synthetic_title_b", "title.toml", validSyntheticManifest)
	reg := NewRegistryFromConfig(repoRoot, nil)
	if !reg.Exists(DefaultSlug) {
		t.Error("halo_infinite built-in absent")
	}
	if !reg.Exists("synthetic_title_b") {
		t.Error("synthetic_title_b découvert absent")
	}
	if got := len(reg.All()); got != 2 {
		t.Errorf("registre = %d titres, want 2", got)
	}
}

func TestLoadTitleManifest_AbsentSentinel(t *testing.T) {
	_, err := LoadTitleManifest(t.TempDir(), "nope")
	if !errors.Is(err, ErrTitleManifestAbsent) {
		t.Errorf("expected ErrTitleManifestAbsent, got %v", err)
	}
}

func TestSetDefaultRegistry_SharedOverride(t *testing.T) {
	// Restaure un built-in frais après le test (état global partagé).
	t.Cleanup(func() { SetDefaultRegistry(NewRegistry()) })

	// Avant override : DefaultRegistry() retombe sur le built-in (mono-titre sûr).
	if DefaultRegistry().Exists("synthetic_title_b") {
		t.Fatal("précondition: le built-in ne devrait pas connaître synthetic_title_b")
	}

	repoRoot := t.TempDir()
	writeTitleFile(t, repoRoot, "synthetic_title_b", "title.toml", validSyntheticManifest)
	SetDefaultRegistry(NewRegistryFromConfig(repoRoot, nil))

	if !DefaultRegistry().Exists("synthetic_title_b") {
		t.Error("après SetDefaultRegistry, le registre partagé doit connaître le titre config")
	}
	if !DefaultRegistry().Exists(DefaultSlug) {
		t.Error("le built-in halo_infinite doit rester présent")
	}
	// XboxTitleIDFor (helper package-level via le registre partagé) voit le titre.
	if XboxTitleIDFor("synthetic_title_b") != "999000111" {
		t.Errorf("XboxTitleIDFor partagé = %q, want 999000111", XboxTitleIDFor("synthetic_title_b"))
	}

	// nil est ignoré (garde défensive : on ne vide jamais le partagé).
	SetDefaultRegistry(nil)
	if !DefaultRegistry().Exists("synthetic_title_b") {
		t.Error("SetDefaultRegistry(nil) ne doit pas vider le registre partagé")
	}
}

// TestMediaFilenamePrefixes_ParsedAndForeign : (F2, résidus H5 / DEC-8) le champ
// [title].media_filename_prefixes est parsé depuis title.toml et
// Registry.ForeignMediaFilenamePrefixes retourne les préfixes des AUTRES titres
// (routage de l'indexeur média : un clip Halo_5_Guardians-* n'est plus indexé
// sous halo_infinite).
func TestMediaFilenamePrefixes_ParsedAndForeign(t *testing.T) {
	manifest := strings.Replace(validSyntheticManifest,
		`capabilities = ["matchmaking", "career", "media"]`,
		"capabilities = [\"matchmaking\", \"career\", \"media\"]\nmedia_filename_prefixes = [\"Synthetic_Game-\", \"  \", \"SG_Capture-\"]",
		1)
	desc, err := LoadTitleManifestFromBytes("title.toml", "synthetic_title_b", []byte(manifest))
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	// Entrées vides/espaces filtrées au parse.
	if len(desc.MediaFilenamePrefixes) != 2 ||
		desc.MediaFilenamePrefixes[0] != "Synthetic_Game-" ||
		desc.MediaFilenamePrefixes[1] != "SG_Capture-" {
		t.Fatalf("MediaFilenamePrefixes = %v, want [Synthetic_Game- SG_Capture-]", desc.MediaFilenamePrefixes)
	}

	reg := NewRegistry()
	reg.Register(desc)

	// Depuis le titre par défaut : les préfixes du titre synthétique sont ÉTRANGERS.
	foreign := reg.ForeignMediaFilenamePrefixes(DefaultSlug)
	if len(foreign) != 2 {
		t.Fatalf("ForeignMediaFilenamePrefixes(%s) = %v, want les 2 préfixes synthétiques", DefaultSlug, foreign)
	}
	// Depuis le titre synthétique lui-même : ses propres préfixes sont EXCLUS.
	for _, p := range reg.ForeignMediaFilenamePrefixes("synthetic_title_b") {
		if p == "Synthetic_Game-" || p == "SG_Capture-" {
			t.Errorf("préfixe propre %q ne doit pas être étranger pour son propre titre", p)
		}
	}
}
