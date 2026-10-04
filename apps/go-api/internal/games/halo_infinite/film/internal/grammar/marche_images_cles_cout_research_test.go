//go:build research

package grammar

// marche_images_cles_cout_research_test.go — INSTRUMENT : LE COUT DE LA PHASE DES IMAGES-CLES SUR
// DES FILMS ENTIERS (lot 2.2 de la representation intermediaire).
//
// La question : faire consommer la phase des images-cles par les canaux d image-cle ajoute-t-il un
// cout a la cuisson ? Ces canaux lisent les ANCRES de la marche memorisee
// ([FilmContext.MarcheDImageCle]) et quelques fenetres de bits ; la phase COMPLETE, celle de
// l iterateur [FilmContext.ImagesCles], traverse en plus l etat complet de CHAQUE ancre. L instrument
// mesure, sur un contexte neuf, la marche des ancres de tous les paquets d image-cle, puis la phase
// complete (memoire chaude), a comparer a la duree de la phase « decodage » de la cuisson du meme
// film (journal de `replay-equiv`). Mesure du 2026-10-03 sur cinq films : 18 a 65 ms pour la phase
// complete, contre 14 a 99 s de decodage ; une distribution ([Distribuer]) ne parcourt de toute facon
// que les corps qu un canal lit (decision 3 du lot 2.2 de l etape 2).
//
//	FILM_CACHE_ROOT=<depot>/data/cache COUT_IMAGES_CLES_FILMS=000d5950,084a804d \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/grammar/ -run CoutDesImagesCles -v

import (
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// TestCoutDesImagesCles imprime, film par film, la duree de la marche des ancres et celle de la
// phase complete des images-cles.
func TestCoutDesImagesCles(t *testing.T) {
	racine, liste := os.Getenv("FILM_CACHE_ROOT"), os.Getenv("COUT_IMAGES_CLES_FILMS")
	if racine == "" || liste == "" {
		t.Skip("FILM_CACHE_ROOT ou COUT_IMAGES_CLES_FILMS absent : instrument saute")
	}
	for _, id := range strings.Split(liste, ",") {
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		fc := NewFilmContext(film)
		marche := fc.MarcheDImageCle()
		debut := time.Now()
		paquets, ancres := 0, 0
		for _, num := range fc.ChunkNumbers() {
			data, pks, ok := fc.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type == PacketTypeKeyframe {
					paquets++
					ancres += len(marche.Records(pk.Payload(data)))
				}
			}
		}
		dAncres := time.Since(debut)
		debut = time.Now()
		occurrences := 0
		for p, err := range fc.ImagesCles() {
			if err != nil {
				t.Fatalf("%s : %v", id, err)
			}
			occurrences += len(p.Comps)
		}
		dPhase := time.Since(debut)
		t.Logf("%s : %d paquet(s) d image-cle, %d ancre(s) ; marche des ancres %v ; phase complete %v "+
			"(%d occurrence(s) rangees)", id, paquets, ancres, dAncres.Round(time.Millisecond),
			dPhase.Round(time.Millisecond), occurrences)
	}
}
