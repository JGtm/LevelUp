package halo_infinite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testfixtures"
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

// weaponIDs de PRODUCTION, relevés dans metadata.duckdb (table weapon_labels). Ils
// portent le tag `weap` dans leurs 32 bits HAUTS — la seule chose que la résolution
// regarde. Écrits en uint64 puis convertis : un weapon_id Halo Infinite dépasse la
// borne d'int64, et int64 en porte le MOTIF BINAIRE (convention du domaine).
var (
	idBR75          = weaponIDDe(3105272441439086495)  // tag 2b1824d5
	idSidekick      = weaponIDDe(17584332298403800991) // tag f408190f
	idMA5KAvenger   = weaponIDDe(17709057395522743307) // tag f5c335df — SANS weapon_key
	idMutilator     = weaponIDDe(15533290483178104735) // tag d7915565 — HORS registre
	idGravityHammer = weaponIDDe(9519138350859642783)  // tag 841ac5e5
	idDiminisher    = weaponIDDe(9519138352544146591)  // même tag : VARIANTE
	idCindershot    = weaponIDDe(2523220517889599391)  // tag 230447b1
	idHeatwave      = weaponIDDe(3083209821504759711)  // tag 2ac9c2ff
	idFragGrenade   = weaponIDDe(13176383349356849055) // tag b6dbead8 — pas un `weap`
	idInconnu       = weaponIDDe(0x0badc0de00000000)   // tag absent de l'atlas
)

// weaponIDDe convertit à l'EXÉCUTION : une conversion de constante uint64 vers int64
// est refusée à la compilation dès qu'elle dépasse la borne, ce qui est le cas des
// deux tiers du référentiel.
func weaponIDDe(u uint64) int64 { return int64(u) } //nolint:gosec // motif binaire conservé

// Sentinelles produit : des weapon_id conventionnels, pas des objets du jeu.
const (
	idSentinelleFrag int64 = 0
	idSentinelleCaC  int64 = 1
	idSentinelleVeh  int64 = 2
)

const weaponURLBase = "/static/weapons-assets/halo_infinite/"

func TestAssetURLAdapter_WeaponImageURL_ResoutParTagWeap(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name string
		id   int64
		want string
	}{
		{"BR75", idBR75, weaponURLBase + "jeu/contour-01.png"},
		// DÉFAUT 1 CORRIGÉ. weapon_labels dit « Mk51 Sidekick », weapon_names.toml dit
		// « Mk50 Sidekick ». L'ancienne résolution était keyée sur ce nom : corriger la
		// traduction faisait disparaître l'image. Le tag, lui, ne bouge pas.
		{"Sidekick malgré la divergence Mk51/Mk50", idSidekick, weaponURLBase + "jeu/contour-03.png"},
		// Deux armes que `weapon_key` ne couvre pas — la clé essayée puis réfutée.
		{"MA5K Avenger sans weapon_key", idMA5KAvenger, weaponURLBase + "jeu/contour-36.png"},
		{"Mutilator hors registre", idMutilator, weaponURLBase + "jeu/contour-37.png"},
		// Une variante partage le tag de son arme de base : elle suit, sans entrée dédiée.
		{"Gravity Hammer", idGravityHammer, weaponURLBase + "jeu/contour-16.png"},
		{"Diminisher of Hope (variante)", idDiminisher, weaponURLBase + "jeu/contour-16.png"},
		// DÉFAUT 2 CORRIGÉ. L'ancienne table servait « Cremator.png » au Cindershot : un
		// nom de fichier n'est pas une source. Les deux armes ont maintenant chacune la
		// sienne, et le nom INTERNE de l'atlas (index 20 = « heatwave ») ne les décide pas.
		{"Cindershot", idCindershot, weaponURLBase + "jeu/contour-20.png"},
		{"Heatwave", idHeatwave, weaponURLBase + "jeu/contour-21.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.WeaponImageURL(c.id); got != c.want {
				t.Errorf("WeaponImageURL(%d) = %q, want %q", c.id, got, c.want)
			}
		})
	}
	if got, want := a.WeaponImageURL(idCindershot), a.WeaponImageURL(idHeatwave); got == want {
		t.Errorf("Cindershot et Heatwave partagent %q — c'est le défaut n°2 revenu", got)
	}
}

func TestAssetURLAdapter_WeaponImageURL_ConceptsHorsAtlas(t *testing.T) {
	t.Parallel()
	a := NewAssetURLAdapter()
	cases := []struct {
		name string
		id   int64
		want string
	}{
		// Les grenades ne sont pas des `weap` (elles vivent en eqip+proj) : l'atlas
		// extrait ne les porte pas. Elles gardent le PNG dessiné à la main, keyé par tag.
		{"sentinelle grenade", idSentinelleFrag, weaponURLBase + "Grenade.png"},
		{"Frag Grenade réelle", idFragGrenade, weaponURLBase + "Grenade.png"},
		{"sentinelle mêlée", idSentinelleCaC, weaponURLBase + "Melee.png"},
		// Trous ASSUMÉS : rien ne représente « un véhicule » ni une arme absente de
		// l'atlas. Repli sur le libellé côté produit — jamais l'icône d'une autre arme.
		{"sentinelle véhicule", idSentinelleVeh, ""},
		{"tag inconnu de l'atlas", idInconnu, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.WeaponImageURL(c.id); got != c.want {
				t.Errorf("WeaponImageURL(%d) = %q, want %q", c.id, got, c.want)
			}
		})
	}
}

// TestWeaponIconFilesExistentSurDisque est le DÉFAUT 3 : jusqu'ici aucun test ne
// vérifiait qu'un PNG servi existe. Un renommage passait la CI et cassait l'UI en
// silence. Couvre la table générée ET les deux fichiers dessinés à la main.
func TestWeaponIconFilesExistentSurDisque(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testfixtures.RepoRoot(), "static", "weapons-assets", TitleSlug)
	if len(weaponIconFileByTag) == 0 {
		t.Fatal("weaponIconFileByTag est vide : la table générée n'a pas été produite")
	}
	for tag, stem := range weaponIconFileByTag {
		p := filepath.Join(dir, "jeu", stem+".png")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("tag %08x -> %s : fichier absent (%v)", tag, stem, err)
		}
	}
	for id, stem := range weaponIconSentinelFiles {
		if _, err := os.Stat(filepath.Join(dir, stem+".png")); err != nil {
			t.Errorf("sentinelle %d -> %s : fichier absent (%v)", id, stem, err)
		}
	}
	for tag, stem := range weaponIconConceptFiles {
		if _, err := os.Stat(filepath.Join(dir, stem+".png")); err != nil {
			t.Errorf("concept %08x -> %s : fichier absent (%v)", tag, stem, err)
		}
	}
}

// TestAucunPNGOrphelin ferme l'autre sens : un PNG servi doit être référencé. Sans
// ce test, « Needler-2.png » avait survécu à la suppression de son entrée de map.
func TestAucunPNGOrphelin(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(testfixtures.RepoRoot(), "static", "weapons-assets", TitleSlug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	reference := map[string]bool{}
	for _, stem := range weaponIconSentinelFiles {
		reference[stem] = true
	}
	for _, stem := range weaponIconConceptFiles {
		reference[stem] = true
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".png" {
			continue
		}
		if stem := strings.TrimSuffix(e.Name(), ".png"); !reference[stem] {
			t.Errorf("%s n'est référencé par aucune table : PNG orphelin", e.Name())
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
