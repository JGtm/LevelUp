package domain

import "testing"

// TestCanDeleteMedia couvre la MATRICE COMPLÈTE de la décision utilisateur
// verrouillée (v7.3 lot 2) : propriétaire + admin, et eux seuls.
//
// Le cas discriminant est `co-membre de groupe` : il passe le middleware
// RequirePlayerOwnership (authz.CanAccessPlayer autorise la famille) et doit
// pourtant être refusé ici. Sans cette règle, consulter une galerie partagée
// donnerait le droit d'en détruire le contenu.
func TestCanDeleteMedia(t *testing.T) {
	cases := []struct {
		name      string
		ownerSlug string
		req       MediaDeleteRequest
		want      bool
	}{
		{
			name:      "propriétaire",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "JGtm", AuthEnforced: true},
			want:      true,
		},
		{
			name:      "propriétaire, casse différente",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "jgtm", AuthEnforced: true},
			want:      true,
		},
		{
			name:      "admin sur le média d'un autre",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "Chocoboflor", RequesterIsAdmin: true, AuthEnforced: true},
			want:      true,
		},
		{
			name:      "co-membre de groupe (passe l'ownership middleware) — REFUSÉ",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "Madina97294", AuthEnforced: true},
			want:      false,
		},
		{
			name:      "aucune session en multi-utilisateur",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "", AuthEnforced: true},
			want:      false,
		},
		{
			name:      "propriétaire inconnu en base (player_slug NULL)",
			ownerSlug: "",
			req:       MediaDeleteRequest{RequesterSlug: "JGtm", AuthEnforced: true},
			want:      false,
		},
		{
			name:      "admin même sans propriétaire connu",
			ownerSlug: "",
			req:       MediaDeleteRequest{RequesterSlug: "JGtm", RequesterIsAdmin: true, AuthEnforced: true},
			want:      true,
		},
		{
			name:      "mono-utilisateur / démo : auth non appliquée",
			ownerSlug: "JGtm",
			req:       MediaDeleteRequest{RequesterSlug: "", AuthEnforced: false},
			want:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanDeleteMedia(tc.ownerSlug, tc.req); got != tc.want {
				t.Errorf("CanDeleteMedia(%q, %+v) = %v, want %v",
					tc.ownerSlug, tc.req, got, tc.want)
			}
		})
	}
}

// TestMediaDeletionTarget_StoredPaths : la source vient toujours en premier
// (c'est elle qui porte la garantie « plus servi »), et les doublons ou vides
// ne produisent pas d'appels de suppression parasites.
func TestMediaDeletionTarget_StoredPaths(t *testing.T) {
	cases := []struct {
		name   string
		target MediaDeletionTarget
		want   []string
	}{
		{
			name:   "source + miniature + hls",
			target: MediaDeletionTarget{FilePath: "a/clip.m3u8", ThumbnailPath: "a/t.webp", HLSPath: "a/hls/x/master.m3u8"},
			want:   []string{"a/clip.m3u8", "a/t.webp", "a/hls/x/master.m3u8"},
		},
		{
			name:   "sans miniature ni hls",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4"},
			want:   []string{"a/clip.mp4"},
		},
		{
			name:   "file_path == hls_path (clip transcodé) → une seule entrée",
			target: MediaDeletionTarget{FilePath: "a/hls/x/master.m3u8", HLSPath: "a/hls/x/master.m3u8"},
			want:   []string{"a/hls/x/master.m3u8"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.target.StoredPaths()
			if len(got) != len(tc.want) {
				t.Fatalf("StoredPaths() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("StoredPaths()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
