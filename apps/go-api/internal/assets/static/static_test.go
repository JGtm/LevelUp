package static

import (
	"path/filepath"
	"testing"
)

func TestKindValid(t *testing.T) {
	cases := []struct {
		k    Kind
		want bool
	}{
		{KindMap, true},
		{KindMedal, true},
		{KindCSRRank, true},
		{KindWeapon, true},
		{KindCommendation, true},
		{Kind(""), false},
		{Kind("unknown"), false},
		{Kind("MAP"), false}, // case-sensitive
	}
	for _, c := range cases {
		if got := c.k.Valid(); got != c.want {
			t.Errorf("Kind(%q).Valid() = %v, want %v", c.k, got, c.want)
		}
	}
}

func TestFolder(t *testing.T) {
	cases := []struct {
		k    Kind
		want string
	}{
		{KindMap, "maps"},
		{KindMedal, "medals/icons"},
		{KindCSRRank, "ranks"},
		{KindWeapon, "weapons-assets"},
		{KindCommendation, "commendations"},
		{Kind(""), ""},
		{Kind("unknown"), ""},
	}
	for _, c := range cases {
		if got := Folder(c.k); got != c.want {
			t.Errorf("Folder(%q) = %q, want %q", c.k, got, c.want)
		}
	}
}

func TestURL(t *testing.T) {
	cases := []struct {
		name      string
		kind      Kind
		titleSlug string
		id        string
		ext       string
		want      string
	}{
		{"map png", KindMap, "halo_infinite", "Aquarius", ".png", "/static/maps/halo_infinite/Aquarius.png"},
		{"map jpg", KindMap, "halo_infinite", "Bazaar", ".jpg", "/static/maps/halo_infinite/Bazaar.jpg"},
		{"medal numeric id", KindMedal, "halo_infinite", "12345", ".png", "/static/medals/icons/halo_infinite/12345.png"},
		{"csr rank", KindCSRRank, "halo_infinite", "120px-HINF-CSR_Onyx", ".png", "/static/ranks/halo_infinite/120px-HINF-CSR_Onyx.png"},
		{"weapon", KindWeapon, "halo_infinite", "br75", ".png", "/static/weapons-assets/halo_infinite/br75.png"},
		{"commendation", KindCommendation, "halo_5_guardians", "achilles", ".png", "/static/commendations/halo_5_guardians/achilles.png"},
		{"id without extension", KindMap, "halo_infinite", "Streets", "", "/static/maps/halo_infinite/Streets"},
		{"different title slug", KindMap, "synthetic_title_b", "frags_arena", ".png", "/static/maps/synthetic_title_b/frags_arena.png"},
		{"invalid kind → empty", Kind("unknown"), "halo_infinite", "x", ".png", ""},
		{"empty kind → empty", Kind(""), "halo_infinite", "x", ".png", ""},
		{"empty slug → empty", KindMap, "", "x", ".png", ""},
		{"empty id → empty", KindMap, "halo_infinite", "", ".png", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := URL(c.kind, c.titleSlug, c.id, c.ext); got != c.want {
				t.Errorf("URL(%q, %q, %q, %q) = %q, want %q",
					c.kind, c.titleSlug, c.id, c.ext, got, c.want)
			}
		})
	}
}

func TestAbsRoot(t *testing.T) {
	cases := []struct {
		repoRoot string
		want     string
	}{
		{".", filepath.Join(".", "static")},
		{"/repo", filepath.Join("/repo", "static")},
		{"", filepath.Join("", "static")},
	}
	for _, c := range cases {
		if got := AbsRoot(c.repoRoot); got != c.want {
			t.Errorf("AbsRoot(%q) = %q, want %q", c.repoRoot, got, c.want)
		}
	}
}

func TestAbsKindRoot(t *testing.T) {
	cases := []struct {
		name      string
		repoRoot  string
		kind      Kind
		titleSlug string
		want      string
	}{
		{"valid map HI", "/repo", KindMap, "halo_infinite", filepath.Join("/repo", "static", "maps", "halo_infinite")},
		{"valid medal", "/repo", KindMedal, "halo_infinite", filepath.Join("/repo", "static", "medals/icons", "halo_infinite")},
		{"empty repoRoot still composes", "", KindCSRRank, "halo_infinite", filepath.Join("", "static", "ranks", "halo_infinite")},
		{"invalid kind → empty", "/repo", Kind("unknown"), "halo_infinite", ""},
		{"empty slug → empty", "/repo", KindMap, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AbsKindRoot(c.repoRoot, c.kind, c.titleSlug); got != c.want {
				t.Errorf("AbsKindRoot(%q, %q, %q) = %q, want %q",
					c.repoRoot, c.kind, c.titleSlug, got, c.want)
			}
		})
	}
}
