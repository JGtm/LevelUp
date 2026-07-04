package halo_infinite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssetURLAdapter_TitleSlug(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	if a.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", a.TitleSlug())
	}
}

func TestAssetURLAdapter_MatchWebURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	if got := a.MatchWebURL("abc-123"); got != "https://www.halowaypoint.com/halo-infinite/matches/abc-123" {
		t.Errorf("MatchWebURL = %q", got)
	}
	if got := a.MatchWebURL("  "); got != "" {
		t.Errorf("MatchWebURL(blank) = %q, want empty", got)
	}
}

func TestAssetURLAdapter_PlayerMatchWebURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	want := "https://www.halowaypoint.com/halo-infinite/players/Player1/matches/abc-123"
	if got := a.PlayerMatchWebURL("Player1", "abc-123"); got != want {
		t.Errorf("PlayerMatchWebURL = %q, want %q", got, want)
	}
	if got := a.PlayerMatchWebURL("", "m1"); got != "" {
		t.Errorf("PlayerMatchWebURL(empty gt) = %q, want empty", got)
	}
	if got := a.PlayerMatchWebURL("Player1", ""); got != "" {
		t.Errorf("PlayerMatchWebURL(empty match) = %q, want empty", got)
	}
}

func TestAssetURLAdapter_MapImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"png map known", "Aquarius", "/static/maps/halo_infinite/Aquarius.png"},
		{"jpg map (default ext)", "Argyle", "/static/maps/halo_infinite/Argyle.jpg"},
		{"png map with hyphen+space", "Streets - Ranked", "/static/maps/halo_infinite/Streets%20-%20Ranked.png"},
		{"map with multiple spaces", "Highpower Sentry Defense", "/static/maps/halo_infinite/Highpower%20Sentry%20Defense.png"},
		{"empty → empty", "", ""},
		{"whitespace only → empty", "   ", ""},
		{"UUID → empty", "12345678-abcd-1234-9876-1234567890ab", ""},
		{"UUID upper → empty", "12345678-ABCD-1234-9876-1234567890AB", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.MapImageURL(c.in); got != c.want {
				t.Errorf("MapImageURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_MedalImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name    string
		medalID uint64
		want    string
	}{
		{"numeric", 12345, "/static/medals/halo_infinite/12345.png"},
		{"zero", 0, "/static/medals/halo_infinite/0.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.MedalImageURL(c.medalID); got != c.want {
				t.Errorf("MedalImageURL(%d) = %q, want %q", c.medalID, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURL(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name    string
		tier    string
		subTier int
		want    string
	}{
		{"gold 3", "Gold", 3, "/static/ranks/halo_infinite/120px-HINF-CSR_Gold3.png"},
		{"platinum 5", "Platinum", 5, "/static/ranks/halo_infinite/120px-HINF-CSR_Platinum5.png"},
		{"diamond 1", "Diamond", 1, "/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png"},
		{"empty tier → empty", "", 1, ""},
		{"whitespace tier → empty", "  ", 1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.CSRRankImageURL(c.tier, c.subTier); got != c.want {
				t.Errorf("CSRRankImageURL(%q, %d) = %q, want %q", c.tier, c.subTier, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_CSRRankImageURLOnyx(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	if got, want := a.CSRRankImageURLOnyx(), "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"; got != want {
		t.Errorf("CSRRankImageURLOnyx() = %q, want %q", got, want)
	}
}

// ─── WeaponImageURL ────────────────────────────────────────────────────

func TestAssetURLAdapter_WeaponImageURL_KnownWeapon(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name   string
		nameEN string
		want   string
	}{
		{"BR75 direct", "BR75", "/static/weapons-assets/halo_infinite/BR75.png"},
		{"Bandit Evo via stem", "Bandit Evo", "/static/weapons-assets/halo_infinite/Bandit.png"},
		{"Needler-1 stem", "Needler", "/static/weapons-assets/halo_infinite/Needler-1.png"},
		{"frag grenade via shared stem", "Frag Grenade", "/static/weapons-assets/halo_infinite/Grenade.png"},
		{"energy sword via shared stem", "Energy Sword", "/static/weapons-assets/halo_infinite/Sword.png"},
		{"shock rifle ranked variant", "Shock Rifle (Ranked)", "/static/weapons-assets/halo_infinite/Shock-rifle.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.WeaponImageURL(c.nameEN); got != c.want {
				t.Errorf("WeaponImageURL(%q) = %q, want %q", c.nameEN, got, c.want)
			}
		})
	}
}

func TestAssetURLAdapter_WeaponImageURL_UnknownReturnsEmpty(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []string{"", "Not A Weapon", "BR42", "Random"}
	for _, in := range cases {
		if got := a.WeaponImageURL(in); got != "" {
			t.Errorf("WeaponImageURL(%q) = %q, want empty (weapon not in mapping)", in, got)
		}
	}
}

// ─── WithMapImagesDir ─────────────────────────────────────────────────

func TestAssetURLAdapter_WithMapImagesDir_FiltersUnknownMap(t *testing.T) {
	t.Parallel()
	// Crée un répertoire temporaire avec seulement Aquarius.png et Streets.jpg.
	dir := t.TempDir()
	for _, name := range []string{"Aquarius.png", "Streets.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("png"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}

	a := NewAssetURLAdapter().WithMapImagesDir(dir)

	// Aquarius présent → URL avec extension .png
	if got, want := a.MapImageURL("Aquarius"), "/static/maps/halo_infinite/Aquarius.png"; got != want {
		t.Errorf("MapImageURL(Aquarius) = %q, want %q", got, want)
	}
	// Streets en .jpg
	if got, want := a.MapImageURL("Streets"), "/static/maps/halo_infinite/Streets.jpg"; got != want {
		t.Errorf("MapImageURL(Streets) = %q, want %q", got, want)
	}
	// Map absente → "" (mode strict)
	if got := a.MapImageURL("Behemoth"); got != "" {
		t.Errorf("MapImageURL(Behemoth) = %q, want empty (mode strict)", got)
	}
}

func TestAssetURLAdapter_WithMapImagesDir_FiltersVariantSuffixes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Crée des fichiers même pour les variantes (Heavies, Firefight).
	for _, name := range []string{"Breaker.png", "Breaker Heavies.png", "Highpower Sentry Defense.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("png"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	a := NewAssetURLAdapter().WithMapImagesDir(dir)

	// Map de base OK.
	if got := a.MapImageURL("Breaker"); got == "" {
		t.Error("MapImageURL(Breaker) returned empty, want non-empty")
	}
	// Variantes Heavies/Sentry Defense filtrées même si fichier présent.
	for _, in := range []string{"Breaker Heavies", "Highpower Sentry Defense"} {
		if got := a.MapImageURL(in); got != "" {
			t.Errorf("MapImageURL(%q) = %q, want empty (variant filter)", in, got)
		}
	}
}

func TestAssetURLAdapter_WithMapImagesDir_NonexistentDirReturnsPermissive(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter().WithMapImagesDir(filepath.Join(t.TempDir(), "nonexistent"))
	// Mode permissif : Aquarius (carte connue avec extension .png hardcoded) doit fonctionner.
	if got := a.MapImageURL("Aquarius"); got == "" {
		t.Error("MapImageURL(Aquarius) returned empty, want non-empty (mode permissif après dir invalide)")
	}
}

func TestAssetURLAdapter_WithMapImagesDir_IgnoresNonImageFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Ne crée que des .txt → exts vide → reste en mode permissif.
	for _, name := range []string{"foo.txt", "bar.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	a := NewAssetURLAdapter().WithMapImagesDir(dir)
	// Reste permissif (Aquarius → .png hardcoded).
	if got := a.MapImageURL("Aquarius"); got == "" {
		t.Error("dir without images should leave adapter in permissive mode")
	}
}

func TestEncodeSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"Streets", "Streets"},
		{"Live Fire", "Live%20Fire"},
		{"a b c d", "a%20b%20c%20d"},
		{"", ""},
		{"   ", "%20%20%20"},
		{"hyphen-no-space", "hyphen-no-space"},
	}
	for _, c := range cases {
		if got := encodeSpaces(c.in); got != c.want {
			t.Errorf("encodeSpaces(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
