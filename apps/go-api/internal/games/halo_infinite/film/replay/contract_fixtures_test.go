package replay

// contract_fixtures_test.go — LES FIXTURES DE CONTRAT, PRODUITES PAR GO, LUES PAR VITEST.
//
// POURQUOI CE FICHIER EXISTE (architecture §12, garde-rail 1). Les preuves du rejeu etaient
// ASYMETRIQUES : cote Go des goldens sur un film reel, cote web une fixture ECRITE A LA MAIN
// qui declarait `schemaVersion: 1` — un document que le serveur n'envoie jamais. Aucune preuve
// ne traversait la frontiere : un champ renomme cote cuisson laissait le web vert jusqu'a la
// production. Ce test ferme le trou dans le seul sens qui vaille : le PRODUCTEUR ecrit la
// fixture, le CONSOMMATEUR la relit (voir apps/web/src/features/match-replay/test/goFixtures.contract.test.ts).
//
// LA CUISSON N'EST PAS REINVENTEE ICI. Le document vient de [buildGolden] — exactement la
// meme sequence que `TestGoldenAssembly` : les entrees figees de `testdata/inputs_000d5950.bin.gz`
// (decodees du film de reference, cf. golden_inputs_test.go), le catalogue de libelles reel du
// titre et l'entree de catalogue de carte. ZERO OCTET DE FILM n'est lu, et la fixture web fige
// donc EXACTEMENT ce que le golden d'assemblage fige, dans l'autre forme.
//
// REGENERATION (jamais d'edition a la main) — DEUX conditions explicites, comme
// TestGoldenInputsRegenerate :
//
//	REPLAY_CONTRACT_UPDATE=1 go test ./internal/games/halo_infinite/film/replay/ \
//	  -run ContractFixtures -update
//
// La seconde condition n'est pas une ceinture de plus : `-update` seul est la porte du golden
// d'assemblage, que l'on rejoue souvent. Sans le second verrou, une regeneration d'assemblage
// re-benirait AUSSI, en silence, le contrat que le web consomme — c'est-a-dire la preuve.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testutil"
)

// contractFixturesEnv : la seconde condition explicite de la regeneration (cf. en-tete).
const contractFixturesEnv = "REPLAY_CONTRACT_UPDATE"

// contractFixture : un document cuit publie au web. La forme est une TABLE des le premier
// jour parce que le lot 0.B.7 y ajoutera un mini-film par build : y ajouter une ligne ne doit
// rien changer d'autre.
type contractFixture struct {
	// film : l'identifiant court du film de reference de la fixture.
	film string
	// build : la cuisson, identique a celle du golden d'assemblage.
	build func(t *testing.T) ReplayDocument
}

// file : le nom du fichier publie. La version de schema est DANS le nom parce que c'est elle
// que le web nomme dans sa matrice de compatibilite (0.B.2) : une fixture d'une autre version
// est un autre fichier, jamais le meme ecrase.
//
// COMPRESSE, ET LA MESURE LE JUSTIFIE : le document du film de reference pese 3 318 459 octets
// indente, 403 260 une fois gzippe (mesure du 2026-09-13). Le depot versionne deja un fixture
// binaire de cette famille par la meme logique (`testdata/inputs_000d5950.bin.gz`, 1 033 495
// octets). Un diff lisible ne se perd pas pour autant : ce qui doit se relire en revue est la
// FORME du document, et elle a son propre golden en clair (`testdata/document_shape.golden`).
func (contractFixture) file() string {
	return fmt.Sprintf("replay_schema_%d.json.gz", SchemaVersion)
}

// contractFixtures : ce que Go publie au web. Une seule entree au 2026-09-13 — le film de
// reference, seul mini-film existant (lot 0.A.2 en produira un par build).
var contractFixtures = []contractFixture{
	{film: goldenFilm, build: buildGolden},
}

// contractManifestFile : le manifeste du jeu de fixtures. Il existe pour une raison precise
// et unique : `testDoc.ts` (la fixture minimale partagee par ~250 tests web) doit prendre sa
// version de schema ICI et non plus d'un litteral ecrit a la main, sans pour autant charger
// et analyser le document complet dans chaque fichier de test. Le manifeste est la PARTIE
// LEGERE du meme fait, ecrite par le meme producteur, et sa coherence avec les documents est
// verifiee par TestContractFixturesMatchCommitted.
const contractManifestFile = "manifest.json"

// contractManifest : la forme du manifeste, cote Go comme cote web.
type contractManifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	Files         []string `json:"files"`
}

// contractFixturesDir : le dossier PARTAGE par les deux applications. Il vit cote web parce
// que c'est le consommateur qui doit pouvoir l'importer sans configuration ; il est ecrit par
// Go parce que c'est le producteur qui sait ce que le document contient.
func contractFixturesDir(t *testing.T) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	return filepath.Join(root, "apps", "web", "src", "features", "match-replay",
		"test", "fixtures", "go")
}

// contractFixtureBytes serialise le document tel que le web le lira.
//
// INDENTE, ET C'EST UN CHOIX : la fixture est un fichier VERSIONNE que l'on relit en revue ;
// un diff d'une seule ligne de 800 Kio ne dit rien. Le cout est en octets sur disque, pas a
// l'execution (vitest analyse le meme arbre dans les deux cas).
func contractFixtureBytes(t *testing.T, doc ReplayDocument) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatalf("serialisation du document de contrat : %v", err)
	}
	return append(raw, '\n')
}

// contractFixtureLire relit une fixture commise et rend son JSON DECOMPRESSE.
//
// LA COMPARAISON PORTE SUR LE CONTENU, PAS SUR LES OCTETS COMPRESSES : deux versions de Go
// peuvent gzipper la meme entree differemment, et un golden qui rougirait a la mise a jour du
// compilateur ne prouverait plus rien.
func contractFixtureLire(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit depuis la racine du depot
	if err != nil {
		t.Fatalf("fixture de contrat absente : %v — regenerer avec "+
			"%s=1 go test -run ContractFixtures -update", err, contractFixturesEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture de contrat : gzip illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture de contrat : decompression : %v", err)
	}
	return buf.Bytes()
}

// contractFixtureEcrire ecrit une fixture compressee.
func contractFixtureEcrire(t *testing.T, path string, blob []byte) int {
	t.Helper()
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
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", path, err)
	}
	return buf.Len()
}

// TestContractFixturesMatchCommitted : la fixture commise est-elle toujours celle que la
// cuisson produit ?
//
// C'EST UN GOLDEN, ET IL ECHOUE POUR DEUX RAISONS DIFFERENTES. Soit le document a change
// (un calque, un champ, une valeur) : il faut alors monter `SchemaVersion` si le CONTENU cuit
// change, ecrire l'entree de chronique, et regenerer. Soit la fixture a ete editee a la main :
// il faut la regenerer. Dans les deux cas le message dit quoi faire.
func TestContractFixturesMatchCommitted(t *testing.T) {
	dir := contractFixturesDir(t)
	var publies []string
	for _, f := range contractFixtures {
		t.Run(f.film, func(t *testing.T) {
			want := contractFixtureLire(t, filepath.Join(dir, f.file()))
			got := contractFixtureBytes(t, f.build(t))
			if string(want) != string(got) {
				t.Errorf("la fixture de contrat %s a change (%d octets commis, %d produits).\n%s",
					f.file(), len(want), len(got), premierEcartAssembly(string(want), string(got)))
			}
		})
		publies = append(publies, f.file())
	}
	verifierManifesteContrat(t, dir, publies)
}

// verifierManifesteContrat : le manifeste dit-il la verite sur ce que le dossier porte ?
//
// SANS CE CONTROLE le manifeste serait une seconde source de verite qui derive : `testDoc.ts`
// lirait une version que plus aucun document ne porte, et tous les tests web qui en dependent
// affirmeraient un contrat perime — verts.
func verifierManifesteContrat(t *testing.T, dir string, publies []string) {
	t.Helper()
	path := filepath.Join(dir, contractManifestFile)
	raw, err := os.ReadFile(path) //nolint:gosec // chemin construit depuis la racine du depot
	if err != nil {
		t.Fatalf("manifeste de contrat absent : %v — regenerer avec "+
			"%s=1 go test -run ContractFixtures -update", err, contractFixturesEnv)
	}
	var m contractManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifeste de contrat illisible : %v", err)
	}
	if m.SchemaVersion != SchemaVersion {
		t.Errorf("le manifeste declare le schema %d, le producteur ecrit %d — `testDoc.ts` "+
			"batirait des documents d'une version que plus rien ne cuit", m.SchemaVersion, SchemaVersion)
	}
	if fmt.Sprint(m.Files) != fmt.Sprint(publies) {
		t.Errorf("le manifeste liste %v, le producteur publie %v", m.Files, publies)
	}
}

// TestContractFixturesCarryCurrentSchema : le document publie porte-t-il bien la version
// COURANTE dans son corps, et le nom du fichier dit-il la meme chose ?
//
// LE NOM ET LE CORPS SONT DEUX AFFIRMATIONS, et le web lit les DEUX (le nom pour choisir le
// fichier, le corps pour le badge). Les laisser diverger donnerait une matrice de
// compatibilite qui teste une version en croyant en tester une autre.
func TestContractFixturesCarryCurrentSchema(t *testing.T) {
	dir := contractFixturesDir(t)
	for _, f := range contractFixtures {
		t.Run(f.film, func(t *testing.T) {
			raw := contractFixtureLire(t, filepath.Join(dir, f.file()))
			var corps struct {
				SchemaVersion int    `json:"schemaVersion"`
				MatchID       string `json:"matchId"`
				TitleSlug     string `json:"titleSlug"`
			}
			if err := json.Unmarshal(raw, &corps); err != nil {
				t.Fatalf("fixture de contrat illisible : %v", err)
			}
			if corps.SchemaVersion != SchemaVersion {
				t.Errorf("%s porte le schema %d dans son corps, le producteur ecrit %d",
					f.file(), corps.SchemaVersion, SchemaVersion)
			}
			if corps.MatchID != f.film {
				t.Errorf("%s porte le match %q, attendu %q", f.file(), corps.MatchID, f.film)
			}
			if corps.TitleSlug == "" {
				t.Errorf("%s ne porte aucun titre : le web ne saurait pas de quel jeu il parle", f.file())
			}
		})
	}
}

// TestContractFixturesRegenerate : LA SEULE PORTE D'ECRITURE des fixtures de contrat.
//
// Deux conditions explicites (cf. en-tete du fichier). Elle ecrit les documents ET le
// manifeste dans le meme geste : les laisser se regenerer separement rendrait possible un
// manifeste en avance sur les documents, ce que TestContractFixturesMatchCommitted rattraperait
// au coup suivant — mais apres avoir laisse passer une fixture incoherente dans un commit.
func TestContractFixturesRegenerate(t *testing.T) {
	switch {
	case !*updateGolden:
		t.Skip("regeneration des fixtures de contrat : passer -update (et " + contractFixturesEnv + "=1)")
	case os.Getenv(contractFixturesEnv) == "":
		t.Skip("regeneration des fixtures de contrat : " + contractFixturesEnv + " non defini")
	}
	dir := contractFixturesDir(t)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", dir, err)
	}
	var publies []string
	for _, f := range contractFixtures {
		blob := contractFixtureBytes(t, f.build(t))
		path := filepath.Join(dir, f.file())
		compresse := contractFixtureEcrire(t, path, blob)
		t.Logf("fixture de contrat reecrite : %s (%d octets JSON, %d compresses)",
			path, len(blob), compresse)
		publies = append(publies, f.file())
	}
	m, err := json.MarshalIndent(contractManifest{SchemaVersion: SchemaVersion, Files: publies}, "", " ")
	if err != nil {
		t.Fatalf("serialisation du manifeste : %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, contractManifestFile), append(m, '\n'), 0o600); err != nil {
		t.Fatalf("ecriture du manifeste : %v", err)
	}
}
