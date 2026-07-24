package domain

import "testing"

func TestPlayerMediaAudioConfig_Validate(t *testing.T) {
	roles := func(rs ...AudioTrackRole) []AudioTrackRole { return rs }

	tooMany := make([]AudioTrackRole, MaxAudioTrackRoles+1)
	for i := range tooMany {
		tooMany[i] = AudioTrackRoleVoice
	}

	cases := []struct {
		name    string
		cfg     PlayerMediaAudioConfig
		wantErr bool
	}{
		{"auto sans pistes", PlayerMediaAudioConfig{Mode: MediaAudioModeAuto}, false},
		{"auto ignore track_roles", PlayerMediaAudioConfig{Mode: MediaAudioModeAuto, TrackRoles: roles(AudioTrackRole("zzz"))}, false},
		{"mode vide", PlayerMediaAudioConfig{Mode: ""}, true},
		{"mode inconnu", PlayerMediaAudioConfig{Mode: MediaAudioMode("magic")}, true},
		{"manuel valide", PlayerMediaAudioConfig{Mode: MediaAudioModeManual, TrackRoles: roles(AudioTrackRoleGame, AudioTrackRoleVoice, AudioTrackRoleOther)}, false},
		{"manuel sans pistes", PlayerMediaAudioConfig{Mode: MediaAudioModeManual}, true},
		{"manuel rôle invalide", PlayerMediaAudioConfig{Mode: MediaAudioModeManual, TrackRoles: roles(AudioTrackRoleGame, AudioTrackRole("mic"))}, true},
		{"manuel trop de pistes", PlayerMediaAudioConfig{Mode: MediaAudioModeManual, TrackRoles: tooMany}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestAudioTrackRole_Valid(t *testing.T) {
	for _, r := range []AudioTrackRole{AudioTrackRoleGame, AudioTrackRoleVoice, AudioTrackRoleOther} {
		if !r.Valid() {
			t.Errorf("%q devrait être valide", r)
		}
	}
	for _, r := range []AudioTrackRole{"", "mic", "GAME"} {
		if r.Valid() {
			t.Errorf("%q ne devrait pas être valide", r)
		}
	}
}

func TestDefaultPlayerMediaAudioConfig(t *testing.T) {
	if got := DefaultPlayerMediaAudioConfig(); got.Mode != MediaAudioModeAuto {
		t.Errorf("défaut mode = %q, want %q", got.Mode, MediaAudioModeAuto)
	}
}
