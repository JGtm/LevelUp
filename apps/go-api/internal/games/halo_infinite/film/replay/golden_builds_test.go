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
// # CE QUE CES GOLDENS DECRIVENT, ET CE QU ILS NE DECRIVENT PAS (decouverte D9, corrigee en R1)
//
// SUR LES SEPT BUILDS, l assemblage bati sur les entrees FRAICHEMENT decodees differe de celui
// bati sur les memes entrees RELUES depuis le fixture. Mesures :
//
//	bcb6d393  « lecture(s) portent le rang SELECTIONNE » : 36 en frais, 136 en relu
//	fb1a1a72  origine mesuree des poses : 20 deployee(s) / 319 lachee(s) en frais,
//	          17 / 322 en relu — TROIS poses changent d ORIGINE, et 479 lignes sur 582 sont
//	          decalees
//
// Le codec est pourtant un POINT FIXE (`TestGoldenBuildsInputsRoundTrip` vert sur les sept) : il
// ne PERD rien de ce qu il porte. Il ne porte simplement pas tout ce que le decodage rend — ni
// les rangs de capacite, ni les origines de pose.
//
// CONCLUSION HONNETE : le golden par build decrit l assemblage sur le SOUS-ENSEMBLE d entrees que
// le codec transporte, PAS la sortie de production. Sur `fb1a1a72` il publie des origines
// d equipement que la production ne produit pas. Il verrouille donc la non-regression du
// CONSTRUCTEUR a entrees constantes, ce qui est deja beaucoup, et rien de plus.
//
// Reprise : completer le codec (`inputs_*.bin.gz`) pour qu il porte rangs de capacite et origines
// de pose, puis re-figer les sept goldens. Candidat au lot 0.D, decision utilisateur. NON corrige
// ici (regle 7). `000d5950` ne montrait pas l ecart — encore un cas ou le film de reference est
// le seul sur lequel un defaut ne se voit pas.
//
// REGENERATION — DEUX PORTES SEPAREES, jamais d edition a la main (revue R1, P2-7) :
//
//	# les fixtures d entrees : re-decode les films, EXIGE le cache
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenBuildsRegenerate -update
//
//	# les goldens d assemblage : DEPUIS les fixtures figes, aucun film lu
//	go test ./internal/games/halo_infinite/film/replay/ \
//	  -run GoldenBuildsAssemblyRegenerate -update-golden-builds-assembly

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// updateGoldenBuildsAssembly : LA PORTE DES GOLDENS D ASSEMBLAGE PAR BUILD, et d eux seuls.
//
// Nommee, et SEPAREE de celle des fixtures d entrees (revue R1, P2-7). Le paquet a deja un
// `-update` (`golden_inputs_test.go`) qui sert `000d5950` : un drapeau par reference est la seule
// forme qui empeche un geste de refiger ce qu on ne voulait pas refiger.
var updateGoldenBuildsAssembly = flag.Bool("update-golden-builds-assembly", false,
	"reecrire les testdata/assembly_<short8>.golden par build, DEPUIS les fixtures figes")

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

// regenererGoldenBuild decode le film et ecrit LE SEUL fixture d entrees. Il n ecrit PLUS le
// golden d assemblage : celui-ci a sa propre porte (cf. TestGoldenBuildsAssemblyRegenerate).
//
// POURQUOI LES DEUX PORTES SONT SEPAREES (revue R1, P2-7). Tant qu un seul geste re-decodait les
// films ET refigeait les deux references, une derive du DECODEUR entrait en reference en meme
// temps qu un changement voulu de l ASSEMBLAGE : le golden d assemblage ne pouvait plus
// contredire le fixture, puisqu il etait refait a partir de lui dans la meme commande. C est
// exactement ce que `000d5950` ne fait pas — son golden se refige DEPUIS le fixture fige, sans
// jamais toucher un film.
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
	fmt.Fprintf(os.Stderr, "REECRITURE: %s (%d octets)\n", b.inputsPath(), buf.Len())
	t.Logf("%s (%s) : fixture %d octets compresses, %d positions, %d tirs, %d morts",
		b.Short8, b.Build, buf.Len(), len(g.Positions), len(g.Fire), len(g.Deaths))
}

// TestGoldenBuildsAssemblyRegenerate : LA PORTE DES GOLDENS D ASSEMBLAGE, et d eux seuls.
//
// ELLE NE LIT AUCUN FILM — seulement les `inputs_*.bin.gz` deja figes. C est ce qui rend les deux
// references independantes : refiger un assemblage ne peut plus faire entrer une derive du
// decodeur, et re-decoder un film ne peut plus refiger un assemblage. Meme forme que
// `TestGoldenAssembly` pour `000d5950`.
func TestGoldenBuildsAssemblyRegenerate(t *testing.T) {
	if !*updateGoldenBuildsAssembly {
		t.Skip("regeneration des goldens d assemblage : passer -update-golden-builds-assembly")
	}
	for _, b := range goldenBuilds() {
		if b.Short8 == goldenFilm {
			continue // couvert par TestGoldenAssembly
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			rendu := renderAssembly(assemblerGoldenBuild(t, b, g, entry))
			if err := os.WriteFile(b.assemblyPath(), []byte(rendu), 0o600); err != nil {
				t.Fatalf("ecriture du golden d assemblage : %v", err)
			}
			fmt.Fprintf(os.Stderr, "REECRITURE: %s (%d octets)\n", b.assemblyPath(), len(rendu))
			t.Logf("%s (%s) : golden %d lignes", b.Short8, b.Build,
				bytes.Count([]byte(rendu), []byte("\n")))
		})
	}
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
