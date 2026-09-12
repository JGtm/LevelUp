package replay

// deaths_source_version_test.go — LE FIL DES MORTS SUIT LA VERSION DU FILM, ET UN TEST LE PROUVE.
//
// # LE DEFAUT QU IL FERME
//
// Revue adversariale du 2026-09-12, constat P1-2 : repasser 0 au lieu de la version lue dans
// [ScanDeaths] laissait toute la CI verte. La seule bobine versionnee du depot
// (`replay/testdata/minifilm_000d5950`) vient d un film de version 41 — et sur une version >= 41
// le decoupage « gamertag a l octet 0 » et le decoupage versionne SE CONFONDENT. Le correctif du
// meme jour n avait donc aucun temoin : il ne pouvait pas etre defait bruyamment.
//
// # CE QUE CE TEST UTILISE
//
// La bobine de VERSION 40 posee a cote du decodeur de source de degat
// (`killsource/testdata/minibobine_e5adf7b2`, 888 Kio, provenance versionnee avec elle). Elle est
// designee par un chemin relatif, comme `filmdec` designe deja `replay/testdata/minifilm_000d5950`
// dans l autre sens : un second exemplaire des memes octets serait de la dette.
//
// Mesure du 2026-09-12 sur cette bobine : 26 gamertags distincts sous la version lue, 2 sous la
// version 0 — pour 199 morts dans les deux cas. Le compte de morts ne bouge PAS (le xuid et
// l instant se lisent hors du bloc) ; ce sont les NOMS qui s effondrent, et ce sont eux qui
// nomment `roster[]` dans l artefact de rejeu.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// miniBobineV40 : la bobine de version 40, relative a CE paquet.
const miniBobineV40 = "../../games/halo_infinite/film/killsource/testdata/minibobine_e5adf7b2"

// miniBobineV40Version : la version que son registre declare.
const miniBobineV40Version = 40

// miniBobineV40NomsPlancher : le plancher de noms distincts. 26 sous la version lue, 2 sous la
// version 0 : il n y a pas de zone grise, et 20 laisse de la marge sans figer un compte exact.
const miniBobineV40NomsPlancher = 20

// TestScanDeathsSuitLaVersionDuFilm — LE GARDE. Sans variable d environnement et sans fixture hors
// depot : il tourne en CI.
func TestScanDeathsSuitLaVersionDuFilm(t *testing.T) {
	film, err := filmsource.LoadDir(miniBobineV40, nil)
	if err != nil {
		t.Fatalf("bobine v40 illisible sous %s : %v — elle est VERSIONNEE, son absence est une "+
			"erreur, pas une raison d ignorer le test", miniBobineV40, err)
	}
	version, lue := filmdec.FilmMajorVersion(film)
	if !lue || version != miniBobineV40Version {
		t.Fatalf("la bobine v40 declare la version %d (lue=%v), %d attendue : son registre "+
			"(`chunk_00.bin`) a change ou manque", version, lue, miniBobineV40Version)
	}
	deaths, err := ScanDeaths(film)
	if err != nil {
		t.Fatalf("ScanDeaths sur la bobine v40 : %v", err)
	}
	noms := map[string]bool{}
	for _, d := range deaths {
		if strings.TrimSpace(d.Gamertag) != "" {
			noms[d.Gamertag] = true
		}
	}
	if len(noms) < miniBobineV40NomsPlancher {
		t.Fatalf(`LE FIL DES MORTS NE SUIT PLUS LA VERSION DU FILM.

  bobine    : %s (version %d)
  morts     : %d, portant %d nom(s) distinct(s) — plancher %d
  attendu   : 199 morts, 26 noms (mesure du 2026-09-12)

Sur un film de version 39-40 le gamertag vit a l OCTET 12 du bloc d event. Passer 0 au parseur
rend ici 2 noms pour 199 morts : l artefact de rejeu nomme alors ses vies avec du rembourrage
(.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md).

Verifier que ScanDeaths passe la version lue par filmdec.FilmMajorVersion a
analysis.ParseHighlightEvents.`,
			miniBobineV40, version, len(deaths), len(noms), miniBobineV40NomsPlancher)
	}
}
