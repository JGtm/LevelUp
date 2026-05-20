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
