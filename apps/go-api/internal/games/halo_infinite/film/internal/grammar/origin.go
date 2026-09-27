package grammar

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// origin.go — L'HORLOGE DU FILM : L HORODATAGE MOTEUR DE SON PREMIER PAQUET.
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM, DU-3 = S1) :
// c est une LECTURE du film (deux en-tetes de paquet). Deplacement pur. Ce que la publication en
// fait — l origine de la frame 0 sur l horloge du fil, son temoin par le fil des morts et son
// refus — reste en `replay` (`origin.go`, `resolveOriginMs`), avec la mesure qui la fonde.

// ScanFilmClockOrigin rend l'horodatage moteur du PREMIER PAQUET du film, c'est-a-dire le
// zero de l'horloge sur laquelle les highlight events sont dates.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requete.
// ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanClockOrigin].
func ScanFilmClockOrigin(filmDir string) (uint64, error) {
	film, err := source.LoadDir(filmDir, nil)
	if err != nil {
		return 0, err
	}
	return ScanClockOrigin(film)
}

// ScanClockOrigin rend l'horodatage moteur du PREMIER PAQUET d'un film DEJA CHARGE.
func ScanClockOrigin(film *source.Film) (uint64, error) {
	_, packets, ok := FilmChunkAt(film, 1)
	if !ok {
		return 0, fmt.Errorf("chunk 1 (origine d'horloge) : absent du film")
	}
	if len(packets) == 0 {
		return 0, fmt.Errorf("chunk 1 (origine d'horloge) : aucun paquet lisible")
	}
	return packets[0].TimestampUS, nil
}
