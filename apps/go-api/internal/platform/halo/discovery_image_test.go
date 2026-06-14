package halo

import "testing"

func TestBuildAssetImageURL(t *testing.T) {
	const pfx = "https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/map/A/B"

	cases := []struct {
		name   string
		prefix string
		paths  []string
		want   string
	}{
		{
			name:   "thumbnail prioritaire",
			prefix: pfx + "/",
			paths:  []string{"map.mvar", "images/hero.jpg", "images/thumbnail.jpg"},
			want:   pfx + "/images/thumbnail.jpg",
		},
		{
			name:   "ajoute le slash manquant au prefix",
			prefix: pfx, // sans slash final
			paths:  []string{"images/thumbnail.png"},
			want:   pfx + "/images/thumbnail.png",
		},
		{
			name:   "hero/screenshot si pas de thumbnail",
			prefix: pfx + "/",
			paths:  []string{"data.bin", "images/screenshot1.jpg"},
			want:   pfx + "/images/screenshot1.jpg",
		},
		{
			name:   "1re image si ni thumbnail ni hero",
			prefix: pfx + "/",
			paths:  []string{"map.mvar", "images/cover.webp", "images/other.png"},
			want:   pfx + "/images/cover.webp",
		},
		{
			name:   "aucune image → vide",
			prefix: pfx + "/",
			paths:  []string{"map.mvar", "config.json"},
			want:   "",
		},
		{name: "prefix vide → vide", prefix: "", paths: []string{"images/thumbnail.jpg"}, want: ""},
		{name: "pas de paths → vide", prefix: pfx, paths: nil, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAssetImageURL(tc.prefix, tc.paths); got != tc.want {
				t.Errorf("buildAssetImageURL(%q, %v) = %q, want %q", tc.prefix, tc.paths, got, tc.want)
			}
		})
	}
}
