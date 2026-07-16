package analysis

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestDisplayPlaylistLabel couvre le chokepoint unique de résolution du libellé
// d'affichage : strip de catégorie (Infinite) + override data-driven (Halo 5).
func TestDisplayPlaylistLabel(t *testing.T) {
	h5Overrides := map[string]string{"Super Fiesta Fête": "Super Fiesta"}
	cases := []struct {
		name          string
		raw           string
		stripCategory bool
		overrides     map[string]string
		want          string
	}{
		{"h5 override applique", "Super Fiesta Fête", false, h5Overrides, "Super Fiesta"},
		{"h5 sans override inchange", "Super Fiesta Hardcore", false, h5Overrides, "Super Fiesta Hardcore"},
		{"infinite strip categorie", "Arène delta : Héritage", true, nil, "Delta : Héritage"},
		{"infinite playlist == categorie non videe", "Fiesta", true, nil, "Fiesta"},
		{"aucune config = brut", "Grand combat en équipe", false, nil, "Grand combat en équipe"},
		{"vide -> vide", "  ", true, h5Overrides, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DisplayPlaylistLabel(tc.raw, tc.stripCategory, tc.overrides); got != tc.want {
				t.Errorf("DisplayPlaylistLabel(%q, %v) = %q, want %q", tc.raw, tc.stripCategory, got, tc.want)
			}
			// Parité méthode/fonction.
			cfg := PlaylistLabelConfig{StripCategory: tc.stripCategory, Overrides: tc.overrides}
			if got := cfg.Display(tc.raw); got != tc.want {
				t.Errorf("PlaylistLabelConfig.Display(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestNoInlinePlaylistLabelResolution est le GARDE-RAIL anti-divergence (CLAUDE.md
// règle 6). Après centralisation dans DisplayPlaylistLabel, plus AUCUNE surface ne
// doit réappliquer NormalizePlaylistLabel inline pour l'affichage : toute nouvelle
// surface passe par le chokepoint (DisplayPlaylistLabel / PlaylistLabelConfig).
// Le seul APPEL autorisé de NormalizePlaylistLabel est celui de DisplayPlaylistLabel
// (playlist_label.go). Les mentions en commentaire sont tolérées.
func TestNoInlinePlaylistLabelResolution(t *testing.T) {
	// Racine du package internal/ (le test tourne depuis internal/analysis).
	internalRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	const chokepointFile = "playlist_label.go"
	var offenders []string
	err = filepath.Walk(internalRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == chokepointFile {
			return nil // le chokepoint LUI-MÊME a le droit d'appeler NormalizePlaylistLabel
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue // mention en commentaire tolérée
			}
			if strings.Contains(line, "NormalizePlaylistLabel(") {
				rel, _ := filepath.Rel(internalRoot, path)
				offenders = append(offenders, rel+":"+strconv.Itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("appel(s) inline de NormalizePlaylistLabel hors du chokepoint DisplayPlaylistLabel "+
			"(playlist_label.go) — passer par analysis.DisplayPlaylistLabel / PlaylistLabelConfig : %v", offenders)
	}
}
