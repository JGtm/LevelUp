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
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoldenInputsFidelite : assemblage sur entrees FRAICHES == assemblage sur entrees RELUES.
//
// # C EST UN GATE LOCAL OBLIGATOIRE, ET SON SKIP N EST PAS UNE PERMISSION (2026-09-18, lot 4.1.3)
//
// Ce test est LE SEUL qui compare ce que l assemblage rend sur des entrees fraichement decodees a
// ce qu il rend sur les MEMES entrees passees par le codec. C est donc le seul qui puisse voir un
// codec QUI PERD — et il n a rien vu pendant tout le chantier, parce qu il SKIPPE sans le cache de
// films et que personne ne posait la variable.
//
// CE QUE CE SILENCE A COUTE, MESURE : le codec arrondissait les pistes d objets du monde au
// centimetre et ne portait que sept champs sur vingt d un record de creation. Le defaut n a ete vu
// qu au gate S8 du lot 4.1.3, au prix d un decodage de dix films, et il aurait ete publie en
// production — `groundWeapons[].x` a la decimale pres et `groundWeapons[].ammo` absent.
//
// IL RESTE SKIPPE SANS CACHE (la CI n a pas de films, et un test qui echoue faute de donnees
// n apprend rien), mais TOUT LOT QUI TOUCHE AU CODEC DOIT LE JOUER, localement, sur les huit
// builds :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenInputsFidelite -v
func TestGoldenInputsFidelite(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	if cache == "" {
		t.Skipf("fidelite du codec : %s non defini (cache de films absent, CI comprise). "+
			"CE SKIP N EST PAS UNE PERMISSION : ce test est le SEUL qui voit un codec qui PERD, "+
			"et son silence a laisse passer l arrondi des pistes d objets du monde jusqu au gate "+
			"S8 du lot 4.1.3. Tout lot qui touche au codec des faits le JOUE localement sur les "+
			"huit builds.", miniFilmCacheEnv)
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

	docFrais := assemblerGoldenBuild(t, b, frais, entry)
	docRelu := assemblerGoldenBuild(t, b, relu, entry)

	// L ORACLE EST L ARTEFACT SERIALISE, ET PLUS LE RENDU TEXTE (2026-09-18, lot 4.1.3).
	//
	// CE QUE LE RENDU TEXTE LAISSAIT PASSER, MESURE : `renderAssembly` est un resume LISIBLE, pas
	// le document. Il ne porte ni les munitions d une arme au sol, ni la cause de fin de vie d un
	// vehicule, ni les compteurs de `coverage.vehicles`. Ce test etait donc VERT pendant que
	// l artefact rejoue perdait jusqu a 223 064 octets sur un BTB, et c est le gate S8 — au prix
	// d un decodage de dix films — qui l a trouve. Ce qu on compare desormais est ce qu on PUBLIE.
	octetsFrais, errF := json.Marshal(docFrais)
	if errF != nil {
		t.Fatalf("serialisation du document sur entrees fraiches : %v", errF)
	}
	octetsRelu, errR := json.Marshal(docRelu)
	if errR != nil {
		t.Fatalf("serialisation du document sur entrees relues : %v", errR)
	}
	if bytes.Equal(octetsFrais, octetsRelu) {
		return
	}
	renduFrais, renduRelu := renderAssembly(docFrais), renderAssembly(docRelu)
	if renduFrais == renduRelu {
		t.Errorf("l ARTEFACT sur entrees FRAICHES differe de l ARTEFACT sur entrees RELUES "+
			"(%d octets contre %d), et le rendu texte ne montre AUCUN ecart : la perte n est "+
			"visible que dans le document publie. Fixture %s.",
			len(octetsFrais), len(octetsRelu), b.inputsPath())
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
