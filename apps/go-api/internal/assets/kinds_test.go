package assets

import "testing"

func TestKind_Valid_KnownKinds(t *testing.T) {
	known := []Kind{
		KindMedalImage, KindMapImage, KindChallengeBadge,
		KindBPTrackImage, KindBPBackground,
		KindSpartanEmblem, KindSpartanBanner, KindSpartanBackdrop, KindCareerRankImage,
		KindMedalMetadata, KindChallengeDefinition,
		KindRewardTrackDefinition, KindAssetTranslation,
	}
	for _, k := range known {
		if !k.Valid() {
			t.Errorf("Kind %q devrait être valide", k)
		}
	}
}

func TestKind_Valid_UnknownKind(t *testing.T) {
	if Kind("unknown-kind").Valid() {
		t.Error("Kind 'unknown-kind' ne devrait pas être valide")
	}
}

func TestKind_IsBinary_ImageKinds(t *testing.T) {
	binary := []Kind{
		KindMedalImage,
		KindMapImage,
		KindChallengeBadge,
		KindBPTrackImage,
		KindBPBackground,
		KindSpartanEmblem,
		KindSpartanBanner,
		KindSpartanBackdrop,
		KindCareerRankImage,
	}
	for _, k := range binary {
		if !k.IsBinary() {
			t.Errorf("Kind %q devrait être binaire", k)
		}
	}
}

func TestKind_IsBinary_JSONKinds(t *testing.T) {
	json := []Kind{KindMedalMetadata, KindChallengeDefinition, KindRewardTrackDefinition, KindAssetTranslation}
	for _, k := range json {
		if k.IsBinary() {
			t.Errorf("Kind %q ne devrait pas être binaire", k)
		}
	}
}

func TestSource_String(t *testing.T) {
	cases := []struct {
		s    Source
		want string
	}{
		{SourceLocalFile, "local_file"},
		{SourceLocalDB, "local_db"},
		{SourceRemote, "remote"},
		{Source(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("Source(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}
