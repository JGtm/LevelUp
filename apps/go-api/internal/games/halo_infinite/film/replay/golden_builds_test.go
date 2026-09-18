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
// # CE QUE CES GOLDENS DECRIVENT — LA SORTIE DE PRODUCTION (decouverte D9, COMBLEE au lot 0.D.3)
//
// Ils la decrivent depuis le 2026-09-14, et ils ne la decrivaient PAS avant : le fixture ne
// portait pas tout ce que l assemblage lit, si bien que le golden figeait un document que la
// production ne produit pas. `TestGoldenInputsFidelite` compare desormais, sur les HUIT builds,
// l assemblage bati sur les entrees FRAICHEMENT decodees a celui bati sur les memes entrees
// RELUES — et il est vert.
//
// NOTE HISTORIQUE, DATEE (mesures du 2026-09-13, avant correction) : `bcb6d393` publiait
// « 136 lecture(s) portent le rang SELECTIONNE » quand la production en produit 36 ; `fb1a1a72`
// publiait « 17 deployee(s) / 322 lachee(s) » quand la production rend 20 / 319, et 479 lignes
// sur 582 etaient decalees. Trois causes, toutes comblees au lot 0.D.3 :
// `KeyframeInventory.SelectedGrenadeRank` non serialise (-1 relu 0, donc une selection inventee),
// `InventoryDelta.Ammo` non serialise, et des coordonnees ARRONDIES au centimetre alors
// qu `equipmentOwner` choisit le poseur a la plus courte distance. Les positions portent depuis
// les QUANTA du film (lot 0.D.3 bis) et le decoupage d i0 vient du CATALOGUE comme en production
// (lot 0.D.7).
//
// CE QU ILS NE DECRIVENT TOUJOURS PAS, et c est ecrit au plan (lot 1.0) : le chemin du fixture
// est une COPIE de la sequence de balayages de `BuildFromFilm`, et cinq canaux y manquent des
// deux cotes — d ou des calques VIDES dans tous les goldens (« prises et lachers d arme
// decodes=0 », « vehicules balaye=false »). Le golden verrouille donc la non-regression du
// constructeur sur les canaux qu il porte, pas sur ceux qu il ignore.
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
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
func (b goldenBuild) mapQuant() (profile.MapQuantEntry, error) {
	path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "data", "titles",
		"halo_infinite", "reference", "map_quant_bounds.json")
	cat, err := profile.LoadMapQuantCatalog(path)
	if err != nil {
		return profile.MapQuantEntry{}, fmt.Errorf("catalogue de bornes %s : %w", path, err)
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
	var ecrits []string
	for _, b := range goldenBuilds() {
		if b.Short8 == goldenFilm {
			continue
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			regenererGoldenBuild(t, b, filepath.Join(cache, b.Short8))
		})
		ecrits = append(ecrits, b.inputsPath())
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R2, C1) : `go test` jette la sortie d un
	// paquet qui passe, donc une reecriture annoncee par `t.Logf` ou sur stderr est INVISIBLE avec
	// la commande documentee (sans `-v`).
	t.Fatalf("%d reference(s) reecrite(s) : %s ; relancer sans -update pour verifier",
		len(ecrits), strings.Join(ecrits, ", "))
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
	blob := EncodeFilmFacts(g)
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
	var ecrits []string
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
			t.Logf("%s (%s) : golden %d lignes", b.Short8, b.Build,
				bytes.Count([]byte(rendu), []byte("\n")))
		})
		ecrits = append(ecrits, b.assemblyPath())
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R2, C1) : `go test` jette la sortie d un
	// paquet qui passe, donc une reecriture annoncee par `t.Logf` ou sur stderr est INVISIBLE avec
	// la commande documentee (sans `-v`).
	t.Fatalf("%d reference(s) reecrite(s) : %s ; "+
		"relancer sans -update-golden-builds-assembly pour verifier",
		len(ecrits), strings.Join(ecrits, ", "))
}

// assemblerGoldenBuild rejoue l assemblage d une entree, avec SON catalogue de carte.
func assemblerGoldenBuild(t *testing.T, b goldenBuild, g *FilmFacts,
	entry profile.MapQuantEntry,
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
				t.Fatalf("golden d assemblage absent (%s) : %v — regenerer avec "+
					"-update-golden-builds-assembly (depuis les fixtures figes, sans film)",
					b.assemblyPath(), err)
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
			blob := EncodeFilmFacts(g)
			again, err := DecodeFilmFacts(blob, entry)
			if err != nil {
				t.Fatalf("second decodage : %v", err)
			}
			if got := EncodeFilmFacts(again); !bytes.Equal(blob, got) {
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
func chargerGoldenBuild(t *testing.T, b goldenBuild) (*FilmFacts, profile.MapQuantEntry) {
	t.Helper()
	blob, err := os.ReadFile(b.inputsPath()) //nolint:gosec // chemin construit depuis la table
	if err != nil {
		t.Fatalf("fixture d entrees absent (%s) : %v — regenerer avec -update et "+
			miniFilmCacheEnv, b.inputsPath(), err)
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
	entry, err := b.mapQuant()
	if err != nil {
		t.Fatalf("carte %q hors catalogue : %v", b.Map, err)
	}
	g, err := DecodeFilmFacts(buf.Bytes(), entry)
	if err != nil {
		t.Fatalf("decodage %s : %v", b.inputsPath(), err)
	}
	return g, entry
}
