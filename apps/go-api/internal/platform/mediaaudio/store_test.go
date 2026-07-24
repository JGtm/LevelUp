package mediaaudio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

func TestStore_Load_AbsentReturnsDefaultAuto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "media_audio_config.json")
	cfg, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("Load absent: err = %v", err)
	}
	if cfg.Mode != domain.MediaAudioModeAuto {
		t.Errorf("mode par défaut = %q, want auto", cfg.Mode)
	}
}

func TestStore_SaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "media_audio_config.json")
	in := domain.PlayerMediaAudioConfig{
		Mode:       domain.MediaAudioModeManual,
		TrackRoles: []domain.AudioTrackRole{domain.AudioTrackRoleGame, domain.AudioTrackRoleVoice},
	}
	s := NewStore(path)
	if err := s.Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Le dossier parent doit avoir été créé, et aucun fichier temporaire ne doit rester.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("fichier temporaire non nettoyé après Save")
	}
	out, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Mode != in.Mode || len(out.TrackRoles) != 2 ||
		out.TrackRoles[0] != domain.AudioTrackRoleGame || out.TrackRoles[1] != domain.AudioTrackRoleVoice {
		t.Errorf("round-trip = %+v, want %+v", out, in)
	}
}

func TestStore_Load_CorruptReturnsErrorAndDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "media_audio_config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewStore(path).Load()
	if err == nil {
		t.Fatal("Load JSON corrompu: err attendu, got nil")
	}
	if cfg.Mode != domain.MediaAudioModeAuto {
		t.Errorf("défaut auto attendu malgré l'erreur, got %q", cfg.Mode)
	}
}

func TestManualRolesForPlayer(t *testing.T) {
	repo := t.TempDir()
	pr := titlePkg.NewPathResolver(repo)
	ctx := context.Background()

	// Aucun sidecar → nil (auto).
	if roles := ManualRolesForPlayer(ctx, pr, "halo_infinite", "GT"); roles != nil {
		t.Errorf("absent → roles = %v, want nil", roles)
	}

	// Mode auto persisté → nil.
	if err := NewStore(pr.PlayerMediaAudioConfigPath("halo_infinite", "GTauto")).Save(
		domain.PlayerMediaAudioConfig{Mode: domain.MediaAudioModeAuto}); err != nil {
		t.Fatal(err)
	}
	if roles := ManualRolesForPlayer(ctx, pr, "halo_infinite", "GTauto"); roles != nil {
		t.Errorf("auto → roles = %v, want nil", roles)
	}

	// Mode manuel → chaînes de rôles.
	if err := NewStore(pr.PlayerMediaAudioConfigPath("halo_infinite", "GTman")).Save(
		domain.PlayerMediaAudioConfig{
			Mode:       domain.MediaAudioModeManual,
			TrackRoles: []domain.AudioTrackRole{domain.AudioTrackRoleGame, domain.AudioTrackRoleOther},
		}); err != nil {
		t.Fatal(err)
	}
	roles := ManualRolesForPlayer(ctx, pr, "halo_infinite", "GTman")
	if len(roles) != 2 || roles[0] != "game" || roles[1] != "other" {
		t.Errorf("manuel → roles = %v, want [game other]", roles)
	}
}
