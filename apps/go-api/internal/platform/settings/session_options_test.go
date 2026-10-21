package settings

// session_options_test.go — la règle unique de découpage en sessions (SessionOptions) :
// défauts d'analysis surchargés par le réglage résolu du titre.

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TestSessionOptions_OverlayPerTitle : sans overlay déclaré, le titre lit le global ; un
// titre avec overlay surcharge écart, coupure classé et mode de coéquipiers sans fuiter sur
// les autres titres.
func TestSessionOptions_OverlayPerTitle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	global := filepath.Join(root, "app_settings.json")
	writeSettingsJSON(t, global,
		`{"session_gap_minutes":90,"session_split_on_ranked_change":false,"session_team_change_mode":"friends"}`)
	store := NewStore(global)
	pr := titlePkg.NewPathResolver(root)

	halo := SessionOptions(ctx, store, pr.TitleSettingsPath("halo_infinite"))
	if halo.GapMinutes != 90 || halo.SplitOnRankedChange || halo.TeamChangeMode != domain.TeamChangeModeFriends {
		t.Fatalf("halo (overlay absent) attendu gap=90 split=false friends, obtenu %+v", halo)
	}

	overlay := pr.TitleSettingsPath("synthetic_title_b")
	if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSettingsJSON(t, overlay,
		`{"session_gap_minutes":30,"session_split_on_ranked_change":true,"session_team_change_mode":"group"}`)
	synth := SessionOptions(ctx, store, overlay)
	if synth.GapMinutes != 30 || !synth.SplitOnRankedChange || synth.TeamChangeMode != domain.TeamChangeModeGroup {
		t.Fatalf("synthetic_title_b (overlay) attendu gap=30 split=true group, obtenu %+v", synth)
	}

	if again := SessionOptions(ctx, store, pr.TitleSettingsPath("halo_infinite")); !reflect.DeepEqual(again, halo) {
		t.Fatalf("halo après overlay synthetic : %+v, attendu %+v (isolation)", again, halo)
	}
}

// TestSessionOptions_KeepsDefaultsOutsideSetting : les champs que le réglage ne porte pas
// (heure de coupure, algorithme) restent ceux d'analysis.DefaultSessionOptions.
func TestSessionOptions_KeepsDefaultsOutsideSetting(t *testing.T) {
	global := filepath.Join(t.TempDir(), "app_settings.json")
	writeSettingsJSON(t, global, `{"session_team_change_mode":"group"}`)
	got := SessionOptions(context.Background(), NewStore(global), "")

	want := analysis.DefaultSessionOptions()
	want.TeamChangeMode = domain.TeamChangeModeGroup
	if got.CutoffHour != want.CutoffHour || got.Mode != want.Mode || got.GapMinutes != want.GapMinutes {
		t.Fatalf("champs hors réglage : obtenu %+v, attendu %+v", got, want)
	}
	if got.TeamChangeMode != domain.TeamChangeModeGroup {
		t.Fatalf("mode de coéquipiers : obtenu %q, attendu group", got.TeamChangeMode)
	}
}

// TestSessionOptions_FallbacksToDefaults : écart <= 0, mode inconnu, store nil, fichier
// illisible et configuration absente rendent la règle par défaut.
func TestSessionOptions_FallbacksToDefaults(t *testing.T) {
	ctx := context.Background()
	defaults := analysis.DefaultSessionOptions()

	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.json")
	writeSettingsJSON(t, invalid, `{"session_gap_minutes":0,"session_team_change_mode":"cluster"}`)
	if got := SessionOptions(ctx, NewStore(invalid), ""); got.GapMinutes != defaults.GapMinutes || got.TeamChangeMode != defaults.TeamChangeMode {
		t.Fatalf("écart 0 / mode inconnu : obtenu %+v, attendu les défauts %+v", got, defaults)
	}

	if got := SessionOptions(ctx, nil, ""); got.GapMinutes != defaults.GapMinutes || got.TeamChangeMode != defaults.TeamChangeMode {
		t.Fatalf("store nil : obtenu %+v, attendu %+v", got, defaults)
	}

	broken := filepath.Join(dir, "broken.json")
	writeSettingsJSON(t, broken, `{not json`)
	if got := SessionOptions(ctx, NewStore(broken), ""); got.GapMinutes != defaults.GapMinutes || got.TeamChangeMode != defaults.TeamChangeMode {
		t.Fatalf("réglage illisible : obtenu %+v, attendu %+v", got, defaults)
	}

	if got := SessionOptionsForConfig(ctx, nil, "halo_infinite"); got.GapMinutes != defaults.GapMinutes {
		t.Fatalf("configuration absente : obtenu %+v, attendu %+v", got, defaults)
	}
}

// TestSessionOptionsForConfig_ReadsConfigFiles : la règle se lit dans le fichier de
// réglages de la configuration (le même que le serveur), relu à chaque appel.
func TestSessionOptionsForConfig_ReadsConfigFiles(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "app_settings.json")
	writeSettingsJSON(t, global, `{"session_team_change_mode":"group"}`)
	cfg := &config.AppConfig{RepoRoot: root, AppSettingsPath: global}
	load := func(ctx context.Context) domain.SessionComputeOptions {
		return SessionOptionsForConfig(ctx, cfg, "halo_infinite")
	}

	if got := load(context.Background()); got.TeamChangeMode != domain.TeamChangeModeGroup {
		t.Fatalf("premier appel : %q, attendu group", got.TeamChangeMode)
	}
	writeSettingsJSON(t, global, `{"session_team_change_mode":"ignore"}`)
	if got := load(context.Background()); got.TeamChangeMode != domain.TeamChangeModeIgnore {
		t.Fatalf("après modification du réglage : %q, attendu ignore (relu à chaque appel)", got.TeamChangeMode)
	}
}
