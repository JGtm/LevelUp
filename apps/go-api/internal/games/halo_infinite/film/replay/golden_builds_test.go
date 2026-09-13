package replay

// golden_builds_test.go — LES GOLDENS, UN PAR BUILD (lot 0.A.2).
//
// # CE QUE CE FICHIER AJOUTE
//
// `TestGoldenAssembly` et `TestGoldenInputsRoundTrip` verrouillent l assemblage sur UN film,
// `000d5950` (build HI_1_13_0). Ce fichier en fait une TABLE : un fixture d entrees et un golden
// d assemblage PAR BUILD present au cache. `000d5950` reste une entree de la table — la premiere,
// et la seule dont les chemins ne changent pas.
//
// # POURQUOI LES ENTREES VIENNENT DU FILM REEL ET NON DE LA MINI-BOBINE
//
// Les mini-bobines du lot 0.A.2 portent des paquets d image-cle concatenes HORS de leur
// continuite : leurs POSITIONS de biped, qui s accumulent par deltas, n ont aucun sens (chaque
// PROVENANCE.txt le dit). Un golden d assemblage construit dessus figerait du bruit. Les fixtures
// d entrees se decodent donc depuis le FILM ENTIER, exactement comme celui de `000d5950`, et les
// bobines gardent leur role : verrouiller les DECODEURS d evenements et la fermeture d image-cle
// (lot 0.A.3), qui n ont pas besoin de continuite.
//
// # LA CARTE CHANGE AVEC LE BUILD
//
// Un fixture par build, c est une carte par build : les bornes de quantification viennent du
// catalogue versionne du titre, par `NormalizeMapName` comme en production. C est ce que
// `decodeFilmInputsForEntry` rend possible sans dupliquer la sequence de decodage.
//
// REGENERATION (jamais d edition a la main) :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenBuildsRegenerate -update

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// goldenBuild est une entree de la table : le film, sa carte, son build.
type goldenBuild struct {
	Short8 string
	Map    string
	Build  string
}

// goldenBuilds : la table. `000d5950` EN FAIT PARTIE — c est l entree historique, et la garder
// dans la table est ce qui empeche les deux chemins de diverger.
func goldenBuilds() []goldenBuild {
	return []goldenBuild{
		{goldenFilm, "Cliffhanger", "HI_1_13_0"},
		{"a521164d", "Fragmentation Heavies", "HI_1_4_1"},
		{"60ae07c4", "Live Fire - Ranked", "HI_1_8_0"},
		{"11de8353", "Thunderhead", "HI_1_9_0"},
		{"111fa685", "Command", "HI_1_10_0"},
		{"e5adf7b2", "Fragmentation", "HI_1_11_0"},
		{"bcb6d393", "Cliffhanger", "HI_1_12_0"},
		{"fb1a1a72", "Banished Narrows", "HI_1_13_0"},
	}
}

// inputsPath / assemblyPath : les deux references d une entree de la table.
func (b goldenBuild) inputsPath() string {
	return filepath.Join(goldenDir, "inputs_"+b.Short8+".bin.gz")
}

func (b goldenBuild) assemblyPath() string {
	return filepath.Join(goldenDir, "assembly_"+b.Short8+".golden")
}

// mapQuant rend l entree de catalogue de la carte du film, par le MEME chemin que la production
// (`NormalizeMapName` via `Lookup`).
func (b goldenBuild) mapQuant() (filmdec.MapQuantEntry, error) {
	path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "data", "titles",
		"halo_infinite", "reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		return filmdec.MapQuantEntry{}, fmt.Errorf("catalogue de bornes %s : %w", path, err)
	}
	return cat.Lookup(b.Map)
}

// TestGoldenBuildsRegenerate : LA SEULE PORTE D ECRITURE des fixtures par build.
//
// `000d5950` EST ECARTE : son fixture est celui de `TestGoldenInputsRegenerate`, avec sa propre
// porte et sa propre magie de version. Deux portes d ecriture sur le meme fichier, c est une
// regeneration accidentelle qui attend son tour.
func TestGoldenBuildsRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des goldens par build : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des goldens par build : " + miniFilmCacheEnv + " non defini")
	}
	for _, b := range goldenBuilds() {
		if b.Short8 == goldenFilm {
			continue
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			regenererGoldenBuild(t, b, filepath.Join(cache, b.Short8))
		})
	}
}

// regenererGoldenBuild decode le film, ecrit le fixture d entrees puis le golden d assemblage.
func regenererGoldenBuild(t *testing.T, b goldenBuild, dir string) {
	t.Helper()
	entry, err := b.mapQuant()
	if err != nil {
		t.Fatalf("carte %q hors catalogue de bornes : %v", b.Map, err)
	}
	g, err := decodeFilmInputsForEntry(b.Short8, dir, entry)
	if err != nil {
		t.Fatalf("decodage de %s : %v", dir, err)
	}
	blob := encodeGoldenInputs(g)
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(blob); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", goldenDir, err)
	}
	if err := os.WriteFile(b.inputsPath(), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture du fixture : %v", err)
	}
	// LE GOLDEN DECRIT CE QUE LE FIXTURE REPRODUIT, PAS CE QUE LE DECODAGE A RENDU.
	// Mesure du lot 0.A.2 : sur six des sept builds, l assemblage bati sur `g` frais differe de
	// celui bati sur `g` RELU (ex. bcb6d393 : « 136 lecture(s) portent le rang SELECTIONNE »
	// contre 36). Le codec est pourtant un point fixe (TestGoldenBuildsInputsRoundTrip vert) :
	// il ne PERD donc rien de ce qu il porte — il ne porte simplement pas tout ce que le
	// decodage rend. Figer le rendu du `g` frais poserait un golden que la comparaison ne peut
	// pas atteindre. Consigne en decouverte du lot ; non corrige ici (regle 7).
	relu, err := decodeGoldenInputs(blob)
	if err != nil {
		t.Fatalf("relecture des entrees : %v", err)
	}
	rendu := renderAssembly(assemblerGoldenBuild(t, b, relu, entry))
	if err := os.WriteFile(b.assemblyPath(), []byte(rendu), 0o600); err != nil {
		t.Fatalf("ecriture du golden d assemblage : %v", err)
	}
	t.Logf("%s (%s) : fixture %d octets compresses, %d positions, %d tirs, %d morts ; golden %d lignes",
		b.Short8, b.Build, buf.Len(), len(g.Positions), len(g.Fire), len(g.Deaths),
		bytes.Count([]byte(rendu), []byte("\n")))
}

// assemblerGoldenBuild rejoue l assemblage d une entree, avec SON catalogue de carte.
func assemblerGoldenBuild(t *testing.T, b goldenBuild, g *goldenInputs,
	entry filmdec.MapQuantEntry,
) ReplayDocument {
	t.Helper()
	opt := g.options()
	opt.Labels = goldenCatalog(t)
	opt.MapQuant = &entry
	return BuildFromPositions(b.Short8, "halo_infinite", g.Positions, g.Fire, opt)
}

// TestGoldenBuildsAssembly : l assemblage rejoue rend-il toujours la meme chose, build par build ?
func TestGoldenBuildsAssembly(t *testing.T) {
	for _, b := range goldenBuilds() {
		if b.Short8 == goldenFilm {
			continue // couvert par TestGoldenAssembly, avec ses tests nommes
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			got := renderAssembly(assemblerGoldenBuild(t, b, g, entry))
			want, err := os.ReadFile(b.assemblyPath()) //nolint:gosec // chemin construit depuis la table
			if err != nil {
				t.Fatalf("golden absent : %v — regenerer avec -update", err)
			}
			if string(want) != got {
				t.Errorf("l assemblage a change par rapport a %s.\n%s",
					b.assemblyPath(), premierEcartAssembly(string(want), got))
			}
		})
	}
}

// TestGoldenBuildsInputsRoundTrip : le codec des entrees est-il un point fixe, build par build ?
func TestGoldenBuildsInputsRoundTrip(t *testing.T) {
	for _, b := range goldenBuilds() {
		if b.Short8 == goldenFilm {
			continue // couvert par TestGoldenInputsRoundTrip
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			blob := encodeGoldenInputs(g)
			again, err := decodeGoldenInputs(blob)
			if err != nil {
				t.Fatalf("second decodage : %v", err)
			}
			if got := encodeGoldenInputs(again); !bytes.Equal(blob, got) {
				t.Fatalf("le codec n est pas un point fixe : %d octets contre %d", len(got), len(blob))
			}
			a := renderAssembly(assemblerGoldenBuild(t, b, g, entry))
			c := renderAssembly(assemblerGoldenBuild(t, b, again, entry))
			if a != c {
				t.Error("l assemblage differe entre les entrees relues et leur re-serialisation : " +
					"le codec perd un champ que BuildFromPositions consomme")
			}
		})
	}
}

// chargerGoldenBuild relit le fixture d entrees d une entree et son entree de catalogue.
func chargerGoldenBuild(t *testing.T, b goldenBuild) (*goldenInputs, filmdec.MapQuantEntry) {
	t.Helper()
	blob, err := os.ReadFile(b.inputsPath()) //nolint:gosec // chemin construit depuis la table
	if err != nil {
		t.Fatalf("fixture absent (%s) : %v — regenerer avec -update", b.inputsPath(), err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		t.Fatalf("gzip %s : %v", b.inputsPath(), err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("decompression %s : %v", b.inputsPath(), err)
	}
	g, err := decodeGoldenInputs(buf.Bytes())
	if err != nil {
		t.Fatalf("decodage %s : %v", b.inputsPath(), err)
	}
	entry, err := b.mapQuant()
	if err != nil {
		t.Fatalf("carte %q hors catalogue : %v", b.Map, err)
	}
	return g, entry
}
