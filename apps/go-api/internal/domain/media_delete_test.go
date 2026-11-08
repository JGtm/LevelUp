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
		{
			name: "original déposé juste après file_path, avant miniature et hls",
			target: MediaDeletionTarget{
				FilePath: "a/hls/x/master.m3u8", SourcePath: "a/clip.mkv",
				ThumbnailPath: "a/t.webp", HLSPath: "a/hls/x/master.m3u8",
			},
			want: []string{"a/hls/x/master.m3u8", "a/clip.mkv", "a/t.webp"},
		},
		{
			name:   "source_path == file_path → une seule entrée",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4", SourcePath: "a/clip.mp4", ThumbnailPath: "a/t.webp"},
			want:   []string{"a/clip.mp4", "a/t.webp"},
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

func assertPaths(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}

// TestMediaDeletionTarget_ForgetPaths : oublier ne retire que ce que l'app a
// créé ; l'original déposé par l'utilisateur n'y figure jamais.
func TestMediaDeletionTarget_ForgetPaths(t *testing.T) {
	cases := []struct {
		name   string
		target MediaDeletionTarget
		want   []string
	}{
		{
			name:   "média direct avec miniature : l'original n'est pas rendu",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4", SourcePath: "a/clip.mp4", ThumbnailPath: "a/t.webp"},
			want:   []string{"a/t.webp"},
		},
		{
			name:   "sans miniature ni hls : rien",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4"},
			want:   []string{},
		},
		{
			name: "vidéo convertie : miniature et hls, jamais l'original",
			target: MediaDeletionTarget{
				FilePath: "a/hls/x/master.m3u8", SourcePath: "a/clip.mkv",
				ThumbnailPath: "a/t.webp", HLSPath: "a/hls/x/master.m3u8",
			},
			want: []string{"a/t.webp", "a/hls/x/master.m3u8"},
		},
		{
			name: "file_path == hls_path sans source : une seule entrée hls",
			target: MediaDeletionTarget{
				FilePath: "a/hls/x/master.m3u8", HLSPath: "a/hls/x/master.m3u8",
			},
			want: []string{"a/hls/x/master.m3u8"},
		},
		{
			name: "file_path différent du hls : file_path n'est pas rendu",
			target: MediaDeletionTarget{
				FilePath: "a/clip.mp4", HLSPath: "a/hls/x/master.m3u8", ThumbnailPath: "a/t.webp",
			},
			want: []string{"a/t.webp", "a/hls/x/master.m3u8"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertPaths(t, "ForgetPaths()", tc.target.ForgetPaths(), tc.want)
		})
	}
}

// TestMediaDeletionTarget_OriginalPaths : les fichiers déposés par l'utilisateur
// seulement, source d'abord, sans doublon ni valeur vide.
func TestMediaDeletionTarget_OriginalPaths(t *testing.T) {
	cases := []struct {
		name   string
		target MediaDeletionTarget
		want   []string
	}{
		{
			name:   "média direct sans source_path : file_path",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4", ThumbnailPath: "a/t.webp"},
			want:   []string{"a/clip.mp4"},
		},
		{
			name:   "média direct, source_path == file_path : une entrée",
			target: MediaDeletionTarget{FilePath: "a/clip.mp4", SourcePath: "a/clip.mp4"},
			want:   []string{"a/clip.mp4"},
		},
		{
			name: "vidéo convertie avec original : source_path seul",
			target: MediaDeletionTarget{
				FilePath: "a/hls/x/master.m3u8", HLSPath: "a/hls/x/master.m3u8", SourcePath: "a/clip.mkv",
			},
			want: []string{"a/clip.mkv"},
		},
		{
			name: "vidéo convertie sans original : rien",
			target: MediaDeletionTarget{
				FilePath: "a/hls/x/master.m3u8", HLSPath: "a/hls/x/master.m3u8",
			},
			want: []string{},
		},
		{
			name: "file_path différent du hls et source_path distinct : les deux, source d'abord",
			target: MediaDeletionTarget{
				FilePath: "a/clip.mp4", HLSPath: "a/hls/x/master.m3u8", SourcePath: "a/orig.mkv",
			},
			want: []string{"a/orig.mkv", "a/clip.mp4"},
		},
		{
			name:   "cible vide : rien",
			target: MediaDeletionTarget{},
			want:   []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertPaths(t, "OriginalPaths()", tc.target.OriginalPaths(), tc.want)
		})
	}
}
