package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrestigeEnabled_Defaults(t *testing.T) {
	if !loadPrestigeEnabled("") {
		t.Error("attendu actif sans fichier de réglages (défaut actif)")
	}
	dir := t.TempDir()
	for nom, c := range map[string]struct {
		contenu string
		want    bool
	}{
		"clé absente":           {`{"autre": 1}`, true},
		"JSON invalide":         {`{`, true},
		"explicitement inactif": {`{"prestige_enabled": false}`, false},
		"explicitement actif":   {`{"prestige_enabled": true}`, true},
	} {
		p := filepath.Join(dir, nom+".json")
		if err := os.WriteFile(p, []byte(c.contenu), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := loadPrestigeEnabled(p); got != c.want {
			t.Errorf("%s : %v, attendu %v", nom, got, c.want)
		}
	}
}
