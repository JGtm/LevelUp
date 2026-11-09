package config

import (
	"testing"
	"time"
)

func TestReadCheckpointInterval(t *testing.T) {
	cas := []struct {
		name    string
		content map[string]any
		want    time.Duration
	}{
		{"fichier absent", nil, DefaultDesktopCheckpointInterval},
		{"clé absente", map[string]any{"lang": "fr"}, DefaultDesktopCheckpointInterval},
		{"valeur valide", map[string]any{"desktop_checkpoint_interval": "90s"}, 90 * time.Second},
		{"minutes", map[string]any{"desktop_checkpoint_interval": "2m"}, 2 * time.Minute},
		{"illisible", map[string]any{"desktop_checkpoint_interval": "souvent"}, DefaultDesktopCheckpointInterval},
		{"mauvais type", map[string]any{"desktop_checkpoint_interval": 5}, DefaultDesktopCheckpointInterval},
		{"sous le plancher", map[string]any{"desktop_checkpoint_interval": "1s"}, MinDesktopCheckpointInterval},
		{"au-dessus du plafond", map[string]any{"desktop_checkpoint_interval": "48h"}, MaxDesktopCheckpointInterval},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			if got := readCheckpointInterval(settingsFile(t, c.content)); got != c.want {
				t.Errorf("readCheckpointInterval = %v ; attendu %v", got, c.want)
			}
		})
	}
}

func TestAdoptCheckpointInterval_OnlyUnderTheCapability(t *testing.T) {
	path := settingsFile(t, map[string]any{"desktop_checkpoint_interval": "2m"})
	off := &AppConfig{AppSettingsPath: path}
	adoptCheckpointInterval(off)
	if off.CheckpointInterval != 0 {
		t.Errorf("hors capacité : intervalle %v ; attendu 0 (tâche non lancée)", off.CheckpointInterval)
	}
	on := &AppConfig{AppSettingsPath: path}
	on.Capabilities.FrequentCheckpoint = true
	adoptCheckpointInterval(on)
	if on.CheckpointInterval != 2*time.Minute {
		t.Errorf("sous la capacité : intervalle %v ; attendu 2m", on.CheckpointInterval)
	}
}
