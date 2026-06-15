// engine_title_seam_test.go — oracle PMT-3 / MT-11 : seam NewSyncEngineForTitle.
//
// (a) Parité halo_infinite : NewSyncEngineForTitle("halo_infinite") produit des
//     chemins DB byte-identiques à l'ancien NewSyncEngine + e.titleSlug correct.
// (b) Routing synthetic_test_title : chemins distincts sous data/titles/{slug}/.

package sync

import (
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func TestNewSyncEngineForTitle_HaloParity(t *testing.T) {
	const repoRoot = "/repo"
	legacy := NewSyncEngine(repoRoot, "GT", "123", nil, nil)
	seam := NewSyncEngineForTitle(repoRoot, titlePkg.DefaultSlug, "GT", "123", nil, nil)

	if seam.titleSlug != titlePkg.DefaultSlug {
		t.Errorf("titleSlug = %q, want %q", seam.titleSlug, titlePkg.DefaultSlug)
	}
	// Byte-identique sur les 4 chemins + le slug.
	cases := []struct {
		name        string
		got, expect string
	}{
		{"player", seam.playerDBPath, legacy.playerDBPath},
		{"shared", seam.sharedDBPath, legacy.sharedDBPath},
		{"metadata", seam.metadataDBPath, legacy.metadataDBPath},
		{"global", seam.globalDBPath, legacy.globalDBPath},
	}
	for _, c := range cases {
		if c.got != c.expect {
			t.Errorf("chemin %s = %q, want %q (parité)", c.name, c.got, c.expect)
		}
	}
}

func TestNewSyncEngineForTitle_SyntheticRouting(t *testing.T) {
	const repoRoot = "/repo"
	const slug = "synthetic_test_title"
	halo := NewSyncEngineForTitle(repoRoot, titlePkg.DefaultSlug, "GT", "123", nil, nil)
	syn := NewSyncEngineForTitle(repoRoot, slug, "GT", "123", nil, nil)

	if syn.titleSlug != slug {
		t.Errorf("titleSlug = %q, want %q", syn.titleSlug, slug)
	}
	// Les chemins player/shared/metadata doivent contenir le slug synthétique et
	// différer de halo (routing réel, pas cosmétique).
	for _, p := range []string{syn.playerDBPath, syn.sharedDBPath, syn.metadataDBPath} {
		if !strings.Contains(p, slug) {
			t.Errorf("chemin %q ne contient pas le slug %q", p, slug)
		}
	}
	if syn.playerDBPath == halo.playerDBPath {
		t.Errorf("playerDBPath synthetic == halo (%q) : routing non effectif", syn.playerDBPath)
	}
	if syn.sharedDBPath == halo.sharedDBPath {
		t.Errorf("sharedDBPath synthetic == halo : routing non effectif")
	}
	// globalDBPath (xuid_aliases) est volontairement global (pas title-scoped).
	if syn.globalDBPath != halo.globalDBPath {
		t.Errorf("globalDBPath devrait rester partagé : %q vs %q", syn.globalDBPath, halo.globalDBPath)
	}
}

func TestNewSyncEngineForTitle_EmptySlugFallsBackToDefault(t *testing.T) {
	seam := NewSyncEngineForTitle("/repo", "", "GT", "123", nil, nil)
	if seam.titleSlug != titlePkg.DefaultSlug {
		t.Errorf("slug vide → titleSlug = %q, want %q", seam.titleSlug, titlePkg.DefaultSlug)
	}
}
