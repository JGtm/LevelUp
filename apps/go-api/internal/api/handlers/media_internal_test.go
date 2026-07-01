// Package handlers — tests internes (package handlers) pour exercer les
// helpers non exportés (filePathToURL, relIfWithin, etc.).
package handlers

import (
	"runtime"
	"testing"
)

// TestFilePathToURL_SinglePlayerCapturesBase couvre le cas Bug #8 :
// capturesBase pointe directement sur le dossier captures du joueur (pas de
// sous-folder slug). Cas typique single-player.
func TestFilePathToURL_SinglePlayerCapturesBase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test Windows-specific paths")
	}
	h := &MediaHandler{}
	slug := "JGtm"
	capturesBase := `C:\Users\Guillaume\Videos\Captures`
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"screenshot",
			`C:\Users\Guillaume\Videos\Captures\Halo Infinite 2025-12-18.png`,
			`/api/v1/players/JGtm/media/files/Halo Infinite 2025-12-18.png`,
		},
		{
			"clip thumb",
			`C:\Users\Guillaume\Videos\Captures\thumbs\foo_hash.webp`,
			`/api/v1/players/JGtm/media/files/thumbs/foo_hash.webp`,
		},
		{
			"clip",
			`C:\Users\Guillaume\Videos\Captures\Halo Infinite 2025-12-18.mp4`,
			`/api/v1/players/JGtm/media/files/Halo Infinite 2025-12-18.mp4`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.filePathToURL(slug, tc.in, capturesBase)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFilePathToURL_MultiPlayerCapturesBaseWithSlug couvre le cas legacy
// multi-player : capturesBase + sous-folder par slug. L'URL inclut le slug
// en préfixe pour matcher la convention canonique {owner_slug}/{rel} utilisée
// par les paths relatifs post-migration.
func TestFilePathToURL_MultiPlayerCapturesBaseWithSlug(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test Windows-specific paths")
	}
	h := &MediaHandler{}
	slug := "JGtm"
	capturesBase := `C:\Captures`
	in := `C:\Captures\JGtm\foo.png`
	want := `/api/v1/players/JGtm/media/files/JGtm/foo.png`
	got := h.filePathToURL(slug, in, capturesBase)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFilePathToURL_RelativePath_PostMigration : un path stocké au format
// relatif {owner_slug}/{rel} est passé tel quel dans l'URL sans transformation.
// C'est le format canonique post-migration.
func TestFilePathToURL_RelativePath_PostMigration(t *testing.T) {
	h := &MediaHandler{}
	in := "Madina97294/thumbs/Halo Infinite 2026-04-19.webp"
	want := "/api/v1/players/JGtm/media/files/Madina97294/thumbs/Halo Infinite 2026-04-19.webp"
	got := h.filePathToURL("JGtm", in, `C:\Captures`)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFilePathToURL_OutsideAnyBase retourne le chemin original quand aucune
// transformation n'aboutit (warning loggué). Cas : path absolu legacy mais
// hors de tout layout connu.
func TestFilePathToURL_OutsideAnyBase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test Windows-specific paths")
	}
	h := &MediaHandler{}
	in := `D:\unrelated\foo.png`
	got := h.filePathToURL("slug", in, `C:\captures`)
	if got != in {
		t.Errorf("expected pass-through for unrelated path, got %q", got)
	}
}

// TestRelFromSlugMarker vérifie le filet portable d'extraction {slug}/{rel} d'un
// path absolu legacy. Portable (pas de t.Skip Windows) : inputs en slash avant
// pour tourner identiquement sur le CI Linux et en local Windows.
func TestRelFromSlugMarker(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		slug    string
		wantRel string
		wantOK  bool
	}{
		{"posix absolu avec marqueur", "/srv/levelup/data/media/JGtm/clip.mp4", "JGtm", "JGtm/clip.mp4", true},
		{"marqueur en milieu profond", "/x/y/data/media/JGtm/thumbs/foo.webp", "JGtm", "JGtm/thumbs/foo.webp", true},
		{"deja prefixe slug", "JGtm/clip.mp4", "JGtm", "JGtm/clip.mp4", true},
		{"pas de marqueur", "/x/y/other/clip.mp4", "JGtm", "", false},
		{"slug vide", "/x/y/JGtm/clip.mp4", "", "", false},
		{"slug sous-chaine non segmentee", "/x/JGtmXX/clip.mp4", "JGtm", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rel, ok := relFromSlugMarker(tc.in, tc.slug)
			if ok != tc.wantOK || rel != tc.wantRel {
				t.Errorf("relFromSlugMarker(%q, %q) = (%q, %v), want (%q, %v)",
					tc.in, tc.slug, rel, ok, tc.wantRel, tc.wantOK)
			}
		})
	}
}

// TestFilePathToURL_AbsoluteLegacyPosix_UsesSlugMarker : sur un OS POSIX, un path
// absolu legacy hors des bases connues est servi via l'heuristique {slug}/{rel}
// (VPS Debian avant migration). Gardé POSIX-only : filepath.IsAbs d'un chemin
// « /… » diffère sur Windows (déjà couvert par les tests Windows ci-dessus).
func TestFilePathToURL_AbsoluteLegacyPosix_UsesSlugMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chemins absolus POSIX — couvert par les tests Windows dédiés")
	}
	h := &MediaHandler{}
	in := "/mnt/old/data/media/JGtm/Halo_5_Guardians-2018.mp4"
	want := "/api/v1/players/JGtm/media/files/JGtm/Halo_5_Guardians-2018.mp4"
	got := h.filePathToURL("JGtm", in, "/mnt/captures")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRelIfWithin vérifie le helper utilisé par filePathToURL.
func TestRelIfWithin(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test Windows-specific paths")
	}
	cases := []struct {
		base, target string
		wantRel      string
		wantOK       bool
	}{
		{`C:\foo`, `C:\foo\bar.txt`, "bar.txt", true},
		{`C:\foo`, `C:\foo\sub\bar.txt`, `sub\bar.txt`, true},
		{`C:\foo\bar`, `C:\foo\bar.txt`, "", false}, // outside (sibling)
		{`C:\foo`, `C:\foo`, "", false},             // base itself (rel = ".")
	}
	for _, tc := range cases {
		rel, ok := relIfWithin(tc.base, tc.target)
		if ok != tc.wantOK || rel != tc.wantRel {
			t.Errorf("relIfWithin(%q, %q) = (%q, %v), want (%q, %v)",
				tc.base, tc.target, rel, ok, tc.wantRel, tc.wantOK)
		}
	}
}
