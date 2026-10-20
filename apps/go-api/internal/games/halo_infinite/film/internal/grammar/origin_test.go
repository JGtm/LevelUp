package grammar

// origin_test.go — LA LECTURE DE L HORLOGE DU FILM (horodatage moteur de son premier paquet).
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26) avec `ScanClockOrigin`. Les tests de ce que
// la publication en fait (`resolveOriginMs`, `coverage.originResolved`) restent en `replay`.

import "testing"

func TestScanFilmClockOrigin_LitUnHorodatage(t *testing.T) {
	// LA VALEUR N'EST PAS VERROUILLEE ICI, et c'est delibere : la mini-bobine REORDONNE les
	// paquets de son chunk 1 (identites d'abord, cf. minifilm_test.go), son premier
	// horodatage n'est donc pas le zero d'un vrai film. Ce qui se verrouille est la LECTURE.
	got, err := ScanFilmClockOrigin(miniBobineChunks)
	if err != nil {
		t.Fatalf("lecture de l'origine d'horloge : %v", err)
	}
	if got == 0 {
		t.Fatalf("horodatage nul : l'en-tete de paquet n'a pas ete lu")
	}
}

func TestScanFilmClockOrigin_FilmAbsent(t *testing.T) {
	if _, err := ScanFilmClockOrigin("testdata/film_inexistant"); err == nil {
		t.Fatalf("aucune erreur sur un film absent")
	}
}
