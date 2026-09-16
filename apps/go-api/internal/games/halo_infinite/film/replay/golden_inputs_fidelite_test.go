package replay

// golden_inputs_fidelite_test.go — LE CODEC DES ENTREES TRANSPORTE-T-IL TOUT CE QUE
// L ASSEMBLAGE LIT ?
//
// # LE DEFAUT QU IL FERME (decouverte D9 du PLAN_DECODEUR_FILM, lot 0.A.2)
//
// Les goldens par build decrivaient l assemblage sur le SOUS-ENSEMBLE d entrees que le codec
// transporte, pas la sortie de production : sur les sept builds, l assemblage bati sur les
// entrees fraichement decodees differait de celui bati sur les MEMES entrees relues depuis le
// fixture. Mesure du 2026-09-13 : `bcb6d393` « lecture(s) portent le rang SELECTIONNE » 36 en
// frais contre 136 en relu ; `fb1a1a72` origines de pose 20/319 en frais contre 17/322 en relu,
// et 479 lignes sur 582 decalees. Le round-trip (`TestGoldenBuildsInputsRoundTrip`) etait
// pourtant vert : il prouve que le codec ne perd rien DE CE QU IL PORTE, jamais qu il porte
// tout.
//
// # CE QUE CE TEST PROUVE, ET QUE LE ROUND-TRIP NE PEUT PAS PROUVER
//
// Il compare deux ASSEMBLAGES : celui des entrees fraichement decodees du film, et celui des
// entrees relues du fixture. Egaux, ils disent que le fixture est une representation FIDELE des
// entrees — donc que le golden d assemblage decrit bien la sortie de production. Le round-trip
// compare le fixture a lui-meme ; celui-ci le compare au FILM.
//
// # IL SAUTE EN CI, ET C EST VOULU
//
// Il exige le cache de films (`REPLAY_FILM_CACHE`), absent de la CI comme tous les oracles
// adosses au cache. Ce qui reste garde en CI : le round-trip du codec et les goldens
// d assemblage. Ce test-ci est le gate LOCAL a jouer quand on touche au codec ou au decodeur.
//
// USAGE :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenInputsFidelite -v

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenInputsFidelite : assemblage sur entrees FRAICHES == assemblage sur entrees RELUES.
func TestGoldenInputsFidelite(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	if cache == "" {
		t.Skipf("fidelite du codec : %s non defini (cache de films absent, CI comprise)", miniFilmCacheEnv)
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			fideliteUnBuild(t, b, filepath.Join(cache, b.Short8))
		})
	}
}

// fideliteUnBuild compare les deux assemblages d un build et NOMME ce qui differe.
func fideliteUnBuild(t *testing.T, b goldenBuild, dir string) {
	t.Helper()
	entry, err := b.mapQuant()
	if err != nil {
		t.Fatalf("carte %q hors catalogue de bornes : %v", b.Map, err)
	}
	frais, err := decodeFilmInputsForEntry(b.Short8, dir, entry)
	if err != nil {
		t.Fatalf("decodage du film %s : %v", dir, err)
	}
	relu, _ := chargerGoldenBuild(t, b)

	renduFrais := renderAssembly(assemblerGoldenBuild(t, b, frais, entry))
	renduRelu := renderAssembly(assemblerGoldenBuild(t, b, relu, entry))
	if renduFrais == renduRelu {
		return
	}
	t.Errorf("l assemblage sur entrees FRAICHES differe de l assemblage sur entrees RELUES —\n"+
		"le fixture %s ne transporte pas tout ce que l assemblage lit.\n%s",
		b.inputsPath(), ecartsFideliteResume(renduFrais, renduRelu))
}

// fideliteMaxLignes borne le nombre d ecarts imprimes : un inventaire, pas un diff complet.
const fideliteMaxLignes = 12

// ecartsFideliteResume rend les premieres lignes qui different, avec le compte total.
//
// Les deux rendus ont la MEME structure (meme renderer, memes calques) : comparer ligne a ligne
// suffit, et nommer la ligne dit QUEL champ manque au codec.
func ecartsFideliteResume(frais, relu string) string {
	lf, lr := strings.Split(frais, "\n"), strings.Split(relu, "\n")
	var b strings.Builder
	total, montres := 0, 0
	for i := 0; i < len(lf) || i < len(lr); i++ {
		a, c := ligneAssembly(lf, i), ligneAssembly(lr, i)
		if a == c {
			continue
		}
		total++
		if montres < fideliteMaxLignes {
			montres++
			b.WriteString("  ligne " + itoaFidelite(i+1) + " :\n")
			b.WriteString("    frais : " + a + "\n")
			b.WriteString("    relu  : " + c + "\n")
		}
	}
	b.WriteString("  " + itoaFidelite(total) + " ligne(s) differentes sur " +
		itoaFidelite(max(len(lf), len(lr))) + "\n")
	return b.String()
}

// itoaFidelite evite d importer strconv pour trois appels.
func itoaFidelite(n int) string {
	if n == 0 {
		return "0"
	}
	var chiffres []byte
	for n > 0 {
		chiffres = append([]byte{byte('0' + n%10)}, chiffres...)
		n /= 10
	}
	return string(chiffres)
}
