package title

import (
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// TitleDescriptor
// ---------------------------------------------------------------------------

func TestTitleDescriptor_HasCapability(t *testing.T) {
	desc := &TitleDescriptor{
		Slug:         "halo_infinite",
		Capabilities: []Capability{CapMatchmaking, CapRanked},
	}
	if !desc.HasCapability(CapMatchmaking) {
		t.Error("expected HasCapability(matchmaking) = true")
	}
	if desc.HasCapability(CapFirefight) {
		t.Error("expected HasCapability(firefight) = false")
	}
}

// TestPathResolver_MediaDataDir vérifie la base média canonique {root}/data/media
// (title-agnostic, plate) — fallback de résolution quand media_captures_base_dir
// est absent/invalide (garde-fou incident prod 2026-06-13).
func TestPathResolver_MediaDataDir(t *testing.T) {
	pr := NewPathResolver(filepath.Join("srv", "app"))
	got := pr.MediaDataDir()
	want := filepath.Join("srv", "app", "data", "media")
	if got != want {
		t.Errorf("MediaDataDir() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// TitleRegistry
// ---------------------------------------------------------------------------

func TestNewRegistry_ContainsHaloInfinite(t *testing.T) {
	r := NewRegistry()
	hi := r.Get(DefaultSlug)
	if hi == nil {
		t.Fatal("expected halo_infinite in registry")
	}
	if hi.Name != "Halo Infinite" {
		t.Errorf("expected name 'Halo Infinite', got %q", hi.Name)
	}
	if !hi.IsDefault {
		t.Error("expected halo_infinite to be default")
	}
}

// TestNewRegistry_HaloInfiniteHasExpectedStats : halo_infinite déclare la
// capability expected_stats (écart au FDA attendu, kills/deaths_expected natifs).
// Un titre sans cette cap (Halo 5) masque les 3 charts côté front.
func TestNewRegistry_HaloInfiniteHasExpectedStats(t *testing.T) {
	hi := NewRegistry().Get(DefaultSlug)
	if hi == nil {
		t.Fatal("expected halo_infinite in registry")
	}
	if !hi.HasCapability(CapExpectedStats) {
		t.Error("halo_infinite doit déclarer CapExpectedStats")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{
		Slug:     "halo_mcc",
		Name:     "Halo MCC",
		Provider: "halo_mcc",
		Status:   StatusComingSoon,
	})
	if !r.Exists("halo_mcc") {
		t.Error("expected halo_mcc to exist")
	}
	if r.Exists("unknown_game") {
		t.Error("expected unknown_game to not exist")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{Slug: "halo_mcc", Name: "Halo MCC", Status: StatusComingSoon})
	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 titles, got %d", len(all))
	}
}

func TestRegistry_Default(t *testing.T) {
	r := NewRegistry()
	d := r.Default()
	if d == nil || d.Slug != DefaultSlug {
		t.Error("expected Default() to return halo_infinite")
	}
}

// TestRegistry_PublicTitles : la vue SWITCHER UTILISATEUR exclut archived ET
// internal (fixtures de test type synthetic_title_b), tout en gardant les
// coming_soon non-internes. Garde-fou contre la fuite de fixture en prod.
func TestRegistry_PublicTitles(t *testing.T) {
	r := NewRegistry() // seed halo_infinite (active, non-internal)
	r.Register(&TitleDescriptor{Slug: "soon", Name: "Soon", Status: StatusComingSoon})
	r.Register(&TitleDescriptor{Slug: "old", Name: "Old", Status: StatusArchived})
	r.Register(&TitleDescriptor{Slug: "fixture", Name: "Fixture", Status: StatusComingSoon, IsInternal: true})

	got := map[string]bool{}
	for _, d := range r.PublicTitles() {
		got[d.Slug] = true
	}
	if !got[DefaultSlug] {
		t.Errorf("PublicTitles doit inclure l'actif par défaut %q", DefaultSlug)
	}
	if !got["soon"] {
		t.Error("PublicTitles doit inclure un coming_soon NON interne")
	}
	if got["old"] {
		t.Error("PublicTitles ne doit PAS inclure un archived")
	}
	if got["fixture"] {
		t.Error("PublicTitles ne doit PAS inclure un titre interne (IsInternal)")
	}
}

// ---------------------------------------------------------------------------
// PathResolver — chemins title-aware
// ---------------------------------------------------------------------------

func TestPathResolver_TitleDataDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.TitleDataDir("halo_infinite")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite")
	if got != want {
		t.Errorf("TitleDataDir: got %q, want %q", got, want)
	}
}

func TestPathResolver_SharedDBPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.SharedDBPath("halo_infinite")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	if got != want {
		t.Errorf("SharedDBPath: got %q, want %q", got, want)
	}
}

func TestPathResolver_GlobalXuidAliasesDBPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.GlobalXuidAliasesDBPath()
	want := filepath.Join("/repo", "data", "global", "xbox_aliases.duckdb")
	if got != want {
		t.Errorf("GlobalXuidAliasesDBPath: got %q, want %q", got, want)
	}
	// La méthode ne prend aucun paramètre titre par construction (xuid global Microsoft).
}

func TestPathResolver_MetadataDBPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.MetadataDBPath("halo_infinite")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")
	if got != want {
		t.Errorf("MetadataDBPath: got %q, want %q", got, want)
	}
}

func TestPathResolver_PlayerDBPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.PlayerDBPath("halo_infinite", "Chocoboflor")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite", "players", "Chocoboflor", "stats.duckdb")
	if got != want {
		t.Errorf("PlayerDBPath: got %q, want %q", got, want)
	}
}

func TestPathResolver_BackupDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.BackupDir("halo_infinite", "TestGT")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite", "backups", "TestGT")
	if got != want {
		t.Errorf("BackupDir: got %q, want %q", got, want)
	}
}

// TestPathResolver_WatcherTokensDir_Global : le store de tokens watcher est GLOBAL
// title-agnostic. Les tokens d'auth (refresh-token / XSTS) sont attachés au compte Xbox
// (xuid), pas à un jeu → un seul dossier data/auth/watcher_tokens partagé par tous les titres.
func TestPathResolver_WatcherTokensDir_Global(t *testing.T) {
	pr := NewPathResolver("/repo")

	want := filepath.Join("/repo", "data", "auth", "watcher_tokens")
	if got := pr.WatcherTokensDir(); got != want {
		t.Errorf("WatcherTokensDir() = %q, want %q", got, want)
	}
}

func TestPathResolver_DemoFixturesDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.DemoFixturesDir("halo_infinite")
	want := filepath.Join("/repo", "tests", "fixtures", "titles", "halo_infinite", "ref_player")
	if got != want {
		t.Errorf("DemoFixturesDir: got %q, want %q", got, want)
	}
}

// --- Chemins globaux ---

func TestPathResolver_SessionDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.SessionDir()
	want := filepath.Join("/repo", "data", "sessions")
	if got != want {
		t.Errorf("SessionDir: got %q, want %q", got, want)
	}
}

func TestPathResolver_JobsCachePath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.JobsCachePath()
	want := filepath.Join("/repo", "data", "cache", "jobs.json")
	if got != want {
		t.Errorf("JobsCachePath: got %q, want %q", got, want)
	}
}

// --- Validation ---

func TestPathResolver_ValidateTitle_OK(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	if err := pr.ValidateTitle("halo_infinite"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPathResolver_ValidateTitle_Unknown(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	if err := pr.ValidateTitle("unknown"); err == nil {
		t.Error("expected error for unknown title")
	}
}

func TestPathResolver_ValidateTitle_Empty(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	if err := pr.ValidateTitle(""); err == nil {
		t.Error("expected error for empty slug")
	}
}

// --- Isolement inter-titres ---

func TestPathResolver_TitleIsolation(t *testing.T) {
	r := NewRegistry()
	r.Register(&TitleDescriptor{Slug: "halo_mcc", Name: "Halo MCC", Status: StatusActive})
	pr := NewPathResolver("/repo", r)

	hiPlayer := pr.PlayerDBPath("halo_infinite", "TestGT")
	mccPlayer := pr.PlayerDBPath("halo_mcc", "TestGT")

	if hiPlayer == mccPlayer {
		t.Error("same gamertag on different titles must have different paths")
	}

	hiShared := pr.SharedDBPath("halo_infinite")
	mccShared := pr.SharedDBPath("halo_mcc")

	if hiShared == mccShared {
		t.Error("different titles must have different shared DB paths")
	}
}

func TestPathResolver_RepoRoot(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	if pr.RepoRoot() != "/repo" {
		t.Errorf("RepoRoot() = %q, want /repo", pr.RepoRoot())
	}
}

func TestPathResolver_SharedPVEDBPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.SharedPVEDBPath("halo_infinite")
	want := filepath.Join("/repo", "data", "titles", "halo_infinite", "warehouse", "shared_pve.duckdb")
	if got != want {
		t.Errorf("SharedPVEDBPath = %q, want %q", got, want)
	}
}

func TestPathResolver_PlayerArchiveDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.PlayerArchiveDir("halo_infinite", "TestGT")
	if got == "" {
		t.Error("expected non-empty path")
	}
}

func TestPathResolver_PlayerCapturesDir(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.PlayerCapturesDir("halo_infinite", "TestGT")
	if got == "" {
		t.Error("expected non-empty path")
	}
}

func TestPathResolver_ResolveCapturesDir(t *testing.T) {
	pr := NewPathResolver("/repo", NewRegistry())

	// baseDir vide → fallback PlayerCapturesDir (chemin interne au repo).
	gotFallback := pr.ResolveCapturesDir("halo_infinite", "TestGT", "")
	wantFallback := filepath.Join("/repo", "data", "titles", "halo_infinite", "players", "TestGT", "captures")
	if gotFallback != wantFallback {
		t.Errorf("fallback: got %q, want %q", gotFallback, wantFallback)
	}

	// baseDir rempli → {baseDir}/{gamertag}, indépendant du repoRoot et du titleSlug.
	gotConfigured := pr.ResolveCapturesDir("halo_infinite", "TestGT", "C:\\Videos\\Captures")
	wantConfigured := filepath.Join("C:\\Videos\\Captures", "TestGT")
	if gotConfigured != wantConfigured {
		t.Errorf("configured: got %q, want %q", gotConfigured, wantConfigured)
	}

	// Garde-fou : ResolveCapturesDir(_, _, "") doit retourner exactement
	// PlayerCapturesDir, jamais un autre nom de sous-dossier (ex. "media").
	// Empêche la régression observée sur data/titles/.../{gamertag}/media/.
	if pr.ResolveCapturesDir("halo_infinite", "Gt", "") != pr.PlayerCapturesDir("halo_infinite", "Gt") {
		t.Error("fallback doit être strictement égal à PlayerCapturesDir")
	}
}

func TestPathResolver_DBProfilesPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.DBProfilesPath()
	if got == "" {
		t.Error("expected non-empty path")
	}
}

func TestPathResolver_AppSettingsPath(t *testing.T) {
	r := NewRegistry()
	pr := NewPathResolver("/repo", r)
	got := pr.AppSettingsPath()
	if got == "" {
		t.Error("expected non-empty path")
	}
}
