package replay

// filmfacts_feuilles_films_test.go — L ALLER-RETOUR FEUILLE A FEUILLE SUR LES ENTREES REELLES DES
// HUIT BUILDS (jalon J11.0-bis, 2026-09-28).
//
// # POURQUOI EN PLUS DU TEMOIN SYNTHETIQUE
//
// Le temoin pose UNE valeur par feuille et UN element par tranche : il ne voit ni les suites
// delta-codees de longueur reelle, ni les domaines que le balayage produit vraiment (quanta de
// position, drapeaux conditionnels, zero signe). Ce test prend les entrees DECODEES DU FILM — jamais
// `testdata/inputs_*.bin.gz`, qui est deja passe par le codec et ne peut donc rien perdre de plus —,
// les fait passer par le FICHIER de faits, et compare chaque feuille a l octet, SANS ACCORD. Les
// ecarts admis sont exactement ceux de [feuillesNonTransportees].
//
// # IL SAUTE EN CI, ET C EST UN GATE LOCAL OBLIGATOIRE DU CODEC
//
// Comme `TestGoldenInputsFidelite` : il exige le cache de films. Il prend le VERROU DE DECODAGE de
// ce cache (`filmproc.AcquireSoloWait`, a cote des chunks) — un decodage de film a la fois sur la
// machine.
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run CodecFeuilleAFeuilleSurLesFilms -v

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// attenteVerrouFeuilles borne l attente du verrou de decodage : au-dela, un autre decodage tient la
// machine depuis trop longtemps pour qu attendre ait un sens.
const attenteVerrouFeuilles = 30 * time.Minute

func TestCodecFeuilleAFeuilleSurLesFilms(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	if cache == "" {
		t.Skipf("aller-retour feuille a feuille sur films : %s non defini (cache de films absent, CI "+
			"comprise). CE SKIP N EST PAS UNE PERMISSION : tout lot qui touche au codec des faits le "+
			"JOUE localement sur les huit builds.", miniFilmCacheEnv)
	}
	lock, err := filmproc.AcquireSoloWait(context.Background(), filepath.Dir(cache),
		"replay-test-feuilles", "huit builds", attenteVerrouFeuilles)
	if err != nil {
		t.Fatalf("verrou de decodage : %v", err)
	}
	defer lock.Release()
	exercees := map[string][]string{}
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant()
			if err != nil {
				t.Fatalf("carte %q hors catalogue de bornes : %v", b.Map, err)
			}
			frais, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("decodage du film : %v", err)
			}
			ecarts := ecartsDeLAllerRetour(frais, allerRetourParLeFichier(t, frais, entry))
			for _, e := range ecarts.nonNommes() {
				t.Errorf("FEUILLE PERDUE PAR LE FICHIER DE FAITS sur un film reel : %s", e)
			}
			for c := range ecarts.nommes() {
				exercees[c] = append(exercees[c], b.Short8)
			}
		})
	}
	t.Log(bilanDesExceptions(exercees))
}

// bilanDesExceptions dit, pour chaque exception, sur quels films elle a ete exercee. Une exception
// jamais exercee sur un film reel n est pas une faute (le temoin synthetique la garde vivante) : le
// bilan la nomme pour que la lecture le sache.
func bilanDesExceptions(exercees map[string][]string) string {
	cles := make([]string, 0, len(feuillesNonTransportees))
	for c := range feuillesNonTransportees {
		cles = append(cles, c)
	}
	sort.Strings(cles)
	var b strings.Builder
	b.WriteString("exceptions exercees sur les films reels :\n")
	for _, c := range cles {
		films := exercees[c]
		sort.Strings(films)
		if len(films) == 0 {
			fmt.Fprintf(&b, "  %-45s jamais non nulle sur les huit films\n", c)
			continue
		}
		fmt.Fprintf(&b, "  %-45s %d film(s)\n", c, len(films))
	}
	return b.String()
}
