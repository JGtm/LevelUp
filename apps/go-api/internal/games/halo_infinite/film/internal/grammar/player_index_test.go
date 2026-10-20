package grammar

// player_index_test.go — LA LECTURE DE LA TABLE D INDEX DE JOUEUR, SANS ROSTER NI FILM.
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26) avec `ScanPlayerIndices`. Le test sur le
// binaire reel (`TestScanFilmPlayerIndicesReadsTheFilm`) reste en `replay` : il compose la
// lecture avec les deux decisions de publication (`rosterFromDeaths`, `injectiveOrEmpty`).

import "testing"

// TestScanFilmPlayerIndicesRefusesWithoutARoster : sans roster, il n y a rien a resoudre.
func TestScanFilmPlayerIndicesRefusesWithoutARoster(t *testing.T) {
	if _, err := ScanFilmPlayerIndices(miniBobineChunks, nil); err == nil {
		t.Error("aucune erreur sur un roster vide : le resolveur balayerait le film pour rien")
	}
	if _, err := ScanFilmPlayerIndices("testdata/film-qui-n-existe-pas", []uint64{1}); err == nil {
		t.Error("aucune erreur sur un repertoire sans chunk")
	}
}
