//go:build research

package grammar

// mpp_resolution_corpus_research_test.go — LA MESURE DU CONSTAT 2 DE LA REVUE M1 (lentille D13).
//
// # LA QUESTION, ET UNE SEULE
//
// Les deux sites qui installent le decoupage du bloc `object-multiplayer-properties` resolvaient
// le profil par [BuildProfileFromFilm], donc par la cle BUILD (table de sept builds en dur dans
// `personnalisationOctets`). Le registre des replis, lui, declare la condition
// `format_sans_profil_relu` : la cle est la VERSION DE FORMAT (`chunk_00+4`).
//
// LES DEUX CLES NE COINCIDENT PAS. Un film dont le FORMAT porte sa largeur relue (27 -> 9/5)
// mais dont le BUILD est absent de la table — un patch du jeu qui ne change pas le format —
// tombait sur les largeurs CALIBREES alors que la grammaire etait disponible. Cet instrument
// COMPTE cette population sur le cache, film par film, AVANT que la correction ne soit ecrite.
//
// LECTURE SEULE, `chunk_00` uniquement (aucun film entier n est lu).
//
//	CGO_ENABLED=0 CHUNK00_CORPUS="C:/.../data/cache/film_chunks" \
//	  go test -tags research ./internal/games/halo_infinite/film/filmdec/ \
//	  -run TestMPPResolutionCorpus -v -timeout 30m

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// bilanMPP : ce que la passe compte.
type bilanMPP struct {
	lus              int
	sansChunk00      int
	sansSection      int
	parCle           map[cleMPP]int
	buildsHorsTable  map[string]int
	octetsChangeants int
}

// cleMPP : le couple (build, format) et ce que chacune des deux cles en dit.
type cleMPP struct {
	build         string
	format        int
	buildConnu    bool
	formatRelu    bool
	formatConnu   bool
	profilComplet bool
}

// TestMPPResolutionCorpus colle la table (build x format) du cache et le compte des films dont
// les octets cuits changeraient si la resolution passait du BUILD au FORMAT.
func TestMPPResolutionCorpus(t *testing.T) {
	dirs := corpusFilms(t)
	b := &bilanMPP{parCle: map[cleMPP]int{}, buildsHorsTable: map[string]int{}}
	for _, dir := range dirs {
		mesurerMPPUnFilm(t, dir, b)
	}
	publierBilanMPP(t, b)
}

// mesurerMPPUnFilm lit le seul `chunk_00` d un film et range son couple de cles au bilan.
func mesurerMPPUnFilm(t *testing.T, dir string, b *bilanMPP) {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin")) //nolint:gosec // corpus local
	if err != nil {
		b.sansChunk00++
		return
	}
	d := filmsource.Inflate(brut)
	b.lus++
	format, okF := FilmFormatVersionFromHeader(d)
	build := ""
	if id, errID := ReadFilmIdentity(d); errID == nil {
		build = id.Build
	} else if errors.Is(errID, ErrNoFilmIdentity) {
		b.sansSection++
	}
	_, buildConnu := personnalisationOctets(build)
	largeurs, formatConnu := mppWidthsPourFormat(format)
	_, errProfil := BuildProfileFor(build, format)
	c := cleMPP{
		build: build, format: format,
		buildConnu:    buildConnu,
		formatRelu:    okF && formatConnu && largeurs.Valid(),
		formatConnu:   okF && formatConnu,
		profilComplet: errProfil == nil && largeurs.Valid(),
	}
	b.parCle[c]++
	if !buildConnu {
		b.buildsHorsTable[build]++
	}
	// LES OCTETS CHANGENT EXACTEMENT LA : le format porte sa largeur RELUE et le profil complet
	// la refusait (build hors table). Partout ailleurs, les deux resolutions rendent la meme
	// chose — rien n est installe, ou la meme largeur l est.
	if c.formatRelu && !c.profilComplet {
		b.octetsChangeants++
	}
}

// publierBilanMPP colle la table du bilan. Il n ASSERTE RIEN : c est une mesure.
func publierBilanMPP(t *testing.T, b *bilanMPP) {
	t.Helper()
	t.Logf("######## RESOLUTION DU DECOUPAGE MPP — CACHE (%d films lus, %d sans chunk_00) ########",
		b.lus, b.sansChunk00)
	t.Logf("  %-14s %7s %7s %8s %8s %7s", "build", "format", "buildOK", "formatOK", "largeur", "films")
	cles := make([]cleMPP, 0, len(b.parCle))
	for c := range b.parCle {
		cles = append(cles, c)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].build != cles[j].build {
			return cles[i].build < cles[j].build
		}
		return cles[i].format < cles[j].format
	})
	for _, c := range cles {
		nom := c.build
		if nom == "" {
			nom = "(sans section)"
		}
		largeur := "indeterminee"
		if c.formatRelu {
			largeur = "RELUE"
		} else if !c.formatConnu {
			largeur = "FORMAT INCONNU"
		}
		t.Logf("  %-14s %7d %7v %8v %8s %7d", nom, c.format, c.buildConnu, c.formatConnu,
			largeur, b.parCle[c])
	}
	t.Logf("")
	t.Logf("  films sans section d identification : %d", b.sansSection)
	t.Logf("  builds HORS TABLE (%d distincts) :", len(b.buildsHorsTable))
	horsTable := make([]string, 0, len(b.buildsHorsTable))
	for nom := range b.buildsHorsTable {
		horsTable = append(horsTable, nom)
	}
	sort.Strings(horsTable)
	total := 0
	for _, nom := range horsTable {
		affiche := nom
		if affiche == "" {
			affiche = "(sans section)"
		}
		t.Logf("    %-14s %d film(s)", affiche, b.buildsHorsTable[nom])
		total += b.buildsHorsTable[nom]
	}
	t.Logf("    TOTAL films au build hors table : %d sur %d", total, b.lus)
	t.Logf("")
	t.Logf("  OCTETS CUITS QUI CHANGERAIENT (format a largeur RELUE, profil complet refuse) : %d",
		b.octetsChangeants)
}
