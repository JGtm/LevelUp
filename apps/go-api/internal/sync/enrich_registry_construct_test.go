package sync

import "testing"

// Tests purs (sans DB) du fallback de construction de pair_name. Couvre la
// règle : on ne fabrique "{mode} on {map}" que si les deux parts sont des noms
// réellement résolus (≠ leur asset_id brut).

func TestConstructPairName(t *testing.T) {
	gvName, gvID := "Slayer", "gv-guid"
	mpName, mpID := "Chasm", "map-guid"

	cases := []struct {
		desc         string
		gvName, gvID *string
		mpName, mpID *string
		want         string
		wantOK       bool
	}{
		{
			desc:   "mode + map résolus → construit",
			gvName: &gvName, gvID: &gvID, mpName: &mpName, mpID: &mpID,
			want: "Slayer on Chasm", wantOK: true,
		},
		{
			desc:   "game_variant encore un GUID (== id) → refuse",
			gvName: &gvID, gvID: &gvID, mpName: &mpName, mpID: &mpID,
			want: "", wantOK: false,
		},
		{
			desc:   "map encore un GUID (== id) → refuse",
			gvName: &gvName, gvID: &gvID, mpName: &mpID, mpID: &mpID,
			want: "", wantOK: false,
		},
		{
			desc:   "game_variant nil → refuse",
			gvName: nil, gvID: &gvID, mpName: &mpName, mpID: &mpID,
			want: "", wantOK: false,
		},
		{
			desc:   "map nil → refuse",
			gvName: &gvName, gvID: &gvID, mpName: nil, mpID: &mpID,
			want: "", wantOK: false,
		},
	}
	for _, c := range cases {
		got, ok := constructPairName(c.gvName, c.gvID, c.mpName, c.mpID)
		if got != c.want || ok != c.wantOK {
			t.Errorf("%s: constructPairName = (%q, %v), want (%q, %v)",
				c.desc, got, ok, c.want, c.wantOK)
		}
	}
}

func TestResolvedRegistryName(t *testing.T) {
	id := "abc-guid"
	name := "Aquarius"
	empty := "   "
	if got := resolvedRegistryName(&name, &id); got != "Aquarius" {
		t.Errorf("vrai nom: got %q, want Aquarius", got)
	}
	if got := resolvedRegistryName(&id, &id); got != "" {
		t.Errorf("name == id: got %q, want \"\"", got)
	}
	if got := resolvedRegistryName(nil, &id); got != "" {
		t.Errorf("nil: got %q, want \"\"", got)
	}
	if got := resolvedRegistryName(&empty, &id); got != "" {
		t.Errorf("blank: got %q, want \"\"", got)
	}
}
