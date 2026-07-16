// Package analysis — playlist_label.go : normalisation du libellé de playlist Halo
// Infinite pour l'affichage. Spécifique Halo Infinite (catégories matchmaking).
// Aucun accès DB — fonction pure.
package analysis

import (
	"strings"
	"unicode"
)

// playlistCategoryPrefixes : locutions de CATÉGORIE matchmaking pouvant préfixer un
// nom de playlist (EN + libellés FR standards), à retirer pour l'affichage — le
// joueur veut "Delta : Héritage", pas "Arène delta : Héritage". Locutions multi-mots
// en premier (on retire le match le plus long). Aligné sur les catégories du skill
// halo-modes + leurs libellés FR usuels.
//
// EXCEPTION VOLONTAIRE : "Classé" / "Classée" / "Ranked" NE sont PAS dans la liste.
// Ce préfixe porte le signal CLASSÉ, lu par la détection ranked — le retirer du
// libellé risquerait de casser cette détection. On le CONSERVE (demande user).
var playlistCategoryPrefixes = []string{
	"super husky raid", "super fiesta", "husky raid", "castle wars",
	"arène", "arena",
	"btb", "fiesta", "firefight", "gruntpocalypse",
	"tactique", "tactical", "assaut", "assault", "communauté", "community",
}

// NormalizePlaylistLabel retire un préfixe de catégorie matchmaking en tête du nom
// de playlist (ex. "Arène delta : Héritage" → "Delta : Héritage") puis capitalise la
// 1re lettre du reste. Sans préfixe connu en tête, retourne le nom inchangé. Ne vide
// jamais entièrement (si la playlist EST la catégorie, ex. "Fiesta", on la garde).
func NormalizePlaylistLabel(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	low := strings.ToLower(trimmed)
	for _, p := range playlistCategoryPrefixes {
		if strings.HasPrefix(low, p+" ") {
			rest := strings.TrimSpace(trimmed[len(p):])
			// Retire un séparateur ": " résiduel EN TÊTE (cas "Classé : X" → "X"),
			// sans toucher au ": " interne (cas "Arène delta : Héritage" → "Delta : Héritage").
			rest = strings.TrimSpace(strings.TrimLeft(rest, ": "))
			if rest == "" {
				return trimmed // playlist == catégorie : ne pas vider
			}
			return capitalizeFirstRune(rest)
		}
	}
	return trimmed
}

// PlaylistLabelConfig porte la configuration d'affichage du libellé de playlist
// d'un titre : retrait du préfixe de catégorie (capability, Infinite) + overrides
// data-driven (playlist_labels.toml, Halo 5). Zéro-valeur = no-op (rend le brut).
// Câblée par titre depuis wire/ServiceRegistry.
type PlaylistLabelConfig struct {
	// StripCategory : le titre déclare CapPlaylistCategoryStrip (retire "Arène ",
	// "Fiesta ", … en tête). Infinite = true, Halo 5 = false.
	StripCategory bool
	// Overrides : nom FR brut → libellé court (Halo 5 « Super Fiesta Fête » →
	// « Super Fiesta »). Appliqué APRÈS le strip. nil/vide = no-op.
	Overrides map[string]string
}

// Display résout le libellé d'affichage d'un nom FR brut via ce config.
func (c PlaylistLabelConfig) Display(rawFR string) string {
	return DisplayPlaylistLabel(rawFR, c.StripCategory, c.Overrides)
}

// DisplayPlaylistLabel est le CHOKEPOINT UNIQUE de résolution du libellé
// d'affichage d'une playlist depuis son nom FR brut (asset_translations) : retrait
// du préfixe de catégorie (si stripCategory) PUIS override explicite data-driven.
//
// TOUTES les surfaces (Match View, tuiles home, sessions, playlists récentes,
// Explorer, historique) DOIVENT passer par ici pour afficher le même libellé —
// ne JAMAIS ré-appliquer NormalizePlaylistLabel + un lookup d'override inline
// ailleurs (garde-rail : no_inline_playlist_label_test.go). rawFR vide → "".
func DisplayPlaylistLabel(rawFR string, stripCategory bool, overrides map[string]string) string {
	s := strings.TrimSpace(rawFR)
	if s == "" {
		return ""
	}
	if stripCategory {
		s = NormalizePlaylistLabel(s)
	}
	if len(overrides) > 0 {
		if disp, ok := overrides[s]; ok {
			return disp
		}
	}
	return s
}

// capitalizeFirstRune met la 1re rune en majuscule (gère les accents : "delta" →
// "Delta", "élite" → "Élite").
func capitalizeFirstRune(s string) string {
	for i, r := range s {
		if i == 0 {
			return string(unicode.ToUpper(r)) + s[len(string(r)):]
		}
	}
	return s
}
