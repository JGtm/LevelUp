package analysis

// map_variant_corpus_test.go — le fichier map_variants.toml de Halo Infinite face aux noms de cartes
// relevés au registre des matchs (2026-10-23) : les variantes se rattachent à leur base, et aucun
// nom qui n'en est pas une n'est pris pour telle. Les suffixes viennent du fichier versionné, pas
// d'une copie : un suffixe retiré ou ajouté du TOML fait échouer ce test.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

func suffixesDuTitre(t *testing.T) []string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "config", "titles", "halo_infinite", "mappings", "map_variants.toml")
	set, err := mappings.LoadMapVariantsFromFile(path)
	if err != nil {
		t.Fatalf("map_variants.toml de halo_infinite : %v", err)
	}
	return set.Suffixes()
}

func TestBaseMapName_CorpusDuRegistre(t *testing.T) {
	suffixes := suffixesDuTitre(t)

	variantes := map[string]string{
		"Fragmentation Heavies":     "Fragmentation",
		"Oasis Heavies":             "Oasis",
		"Breaker Heavies":           "Breaker",
		"Highpower Heavies":         "Highpower",
		"Deadlock Heavies":          "Deadlock",
		"Thunderhead Heavies":       "Thunderhead",
		"Fortitude Heavies":         "Fortitude",
		"Obituary Heavies":          "Obituary",
		"Insolence Heavies":         "Insolence",
		"Scarr Heavies":             "Scarr",
		"Refuge Heavies":            "Refuge",
		"Beaver Canyon - Ranked":    "Beaver Canyon",
		"Kusini Bay - Ranked":       "Kusini Bay",
		"Live Fire - Ranked":        "Live Fire",
		"Showdown Arena - Ranked":   "Showdown Arena",
		"Banished Narrows - Ranked": "Banished Narrows",
		"The Pit - Ranked":          "The Pit",
	}
	for nom, base := range variantes {
		got, ok := BaseMapName(nom, suffixes)
		if !ok || got != base {
			t.Errorf("BaseMapName(%q) = (%q, %v), attendu (%q, true)", nom, got, ok, base)
		}
	}

	// Noms longs du registre qui ne sont PAS des variantes : cartes de base ou d'autres modes.
	autres := []string{
		"Thunderhead", "Live Fire", "The Pit", "Banished Narrows", "Aquarius",
		"House of Reckoning 2F", "House of Reckoning Firefight", "Breaker One Flag CTF",
		"TFF | Night Of The Undead", "TFF | Countdown Of The Dead", "TFF | Installation Of The Dead",
	}
	for _, nom := range autres {
		if got, ok := BaseMapName(nom, suffixes); ok {
			t.Errorf("BaseMapName(%q) = (%q, true) : ce nom n'est pas une variante", nom, got)
		}
	}
}
