// Package settings — store_replay_sound_test.go : les deux réglages « Sons du rejeu 2D ».
//
// POURQUOI UN FICHIER À PART. store_test.go dépasse déjà 500 lignes ; et ces deux réglages
// ont un piège qui leur est propre — un défaut de 100 sur un entier dont le zéro-value (0)
// est une valeur LÉGITIME. Sans réapplication « clé absente → 100 », un fichier écrit avant
// l'arrivée du réglage serait lu comme « variation coupée », c'est-à-dire l'inverse de
// l'intention (le défaut, c'est la variation du jeu telle quelle).
package settings_test

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/settings"
)

func TestDefaults_ReplaySound(t *testing.T) {
	d := settings.Defaults()
	if d.ReplaySoundVariationPercent != 100 {
		t.Errorf("variation par défaut = %d, attendu 100 (fourchettes du jeu telles quelles)",
			d.ReplaySoundVariationPercent)
	}
	if d.ReplaySoundDistancePercent != 0 {
		t.Errorf("distance par défaut = %d, attendu 0 (son pur)", d.ReplaySoundDistancePercent)
	}
}

func TestStore_Load_ReplaySoundVariationDefaultsWhenAbsent(t *testing.T) {
	// Fichier antérieur au réglage : la variation doit valoir 100, pas 0.
	store := newTestStore(t, map[string]interface{}{"lang": "fr"})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ReplaySoundVariationPercent != 100 {
		t.Errorf("clé absente → %d, attendu 100", cfg.ReplaySoundVariationPercent)
	}
}

func TestStore_Load_ReplaySoundVariationZeroRespected(t *testing.T) {
	// 0 explicite = l'opérateur a coupé la variation. Le confondre avec « absent »
	// rallumerait un effet qu'il a délibérément éteint.
	store := newTestStore(t, map[string]interface{}{
		"lang":                           "fr",
		"replay_sound_variation_percent": 0,
	})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ReplaySoundVariationPercent != 0 {
		t.Errorf("0 explicite → %d, attendu 0", cfg.ReplaySoundVariationPercent)
	}
}

func TestApply_ReplaySound(t *testing.T) {
	cfg := settings.Defaults()
	variation, distance := 40, 70
	settings.Apply(cfg, &domain.UpdateSettingsRequest{
		ReplaySoundVariationPercent: &variation,
		ReplaySoundDistancePercent:  &distance,
	})
	if cfg.ReplaySoundVariationPercent != 40 || cfg.ReplaySoundDistancePercent != 70 {
		t.Errorf("après Apply : variation=%d distance=%d, attendu 40 et 70",
			cfg.ReplaySoundVariationPercent, cfg.ReplaySoundDistancePercent)
	}

	// PATCH partiel : un champ absent ne doit pas écraser l'autre.
	autre := 10
	settings.Apply(cfg, &domain.UpdateSettingsRequest{ReplaySoundDistancePercent: &autre})
	if cfg.ReplaySoundVariationPercent != 40 {
		t.Errorf("variation écrasée par un PATCH qui ne la mentionne pas : %d",
			cfg.ReplaySoundVariationPercent)
	}
}

func TestToResponse_ReplaySoundMapped(t *testing.T) {
	cfg := settings.Defaults()
	cfg.ReplaySoundVariationPercent = 25
	cfg.ReplaySoundDistancePercent = 60
	resp := settings.ToResponse(cfg)
	if resp.ReplaySoundVariationPercent != 25 || resp.ReplaySoundDistancePercent != 60 {
		t.Errorf("ToResponse : variation=%d distance=%d, attendu 25 et 60",
			resp.ReplaySoundVariationPercent, resp.ReplaySoundDistancePercent)
	}
}

func TestStore_SaveLoadRoundTrip_ReplaySound(t *testing.T) {
	store := newTestStore(t, map[string]interface{}{"lang": "fr"})
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.ReplaySoundVariationPercent = 0
	cfg.ReplaySoundDistancePercent = 100
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	relu, err := store.Load()
	if err != nil {
		t.Fatalf("Load2: %v", err)
	}
	if relu.ReplaySoundVariationPercent != 0 || relu.ReplaySoundDistancePercent != 100 {
		t.Errorf("après aller-retour : variation=%d distance=%d, attendu 0 et 100",
			relu.ReplaySoundVariationPercent, relu.ReplaySoundDistancePercent)
	}
}
