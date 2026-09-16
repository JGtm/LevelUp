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
// UNE FIXTURE PAR BUILD (lot 0.B.7, 2026-09-13). La table est celle de `golden_builds_test.go`
// — les huit entrees, `000d5950` compris — et non une seconde liste qui deriverait d'elle : un
// build ajoute au golden d'assemblage publie desormais sa fixture au web sans autre geste.
// Chaque document est cuit depuis les `testdata/inputs_<short8>.bin.gz` figes, par le MEME
// chemin d'assemblage que le golden du build (`chargerGoldenBuild` puis `assemblerGoldenBuild`,
// avec le catalogue de libelles reel du titre et l'entree de catalogue de SA carte). ZERO OCTET
// DE FILM n'est lu.
//
// CE QUE CES DOCUMENTS DECRIVENT (decouverte D9 du plan, COMBLEE au lot 0.D.3). Ils decrivent
// la sortie de production : `TestGoldenInputsFidelite` prouve sur les HUIT builds que
// l'assemblage bati sur les entrees relues egale celui bati sur les entrees fraichement
// decodees. Avant le 2026-09-14 ce n'etait pas vrai — le codec ne portait ni le rang de grenade
// SELECTIONNE, ni les munitions des deltas, ni des coordonnees exactes — et ces documents
// decrivaient alors un sous-ensemble. Ce qu'on leur demande n'a pas change pour autant : un
// contrat de FORME (les clefs, les types, la nullabilite, ce que les logiques pures savent
// traverser), jamais un contrat de VALEURS. Les valeurs, elles, sont figees cote Go par les
// goldens d'assemblage.
//
// D'OU LA COUPE DES POINTS DE PISTE (arbitrage du pilote, 2026-09-13). Mesure du lot 0.B.7 :
// les huit documents PLEINS pesent 5,86 Mio compresses, pour un plafond de 3 Mio — et
// `tracks[].points` y fait 85 a 89 % des octets. Les sept fixtures PAR BUILD ne gardent donc
// qu'un point sur cinq (le premier et le DERNIER de chaque piste sont toujours gardes, pour que
// la fenetre de vie et la premiere position restent celles du document plein). Un point sur
// cinq garde la FORME des pistes — leur nombre, leurs champs, leurs bornes, leur nullabilite —
// et 15 % des octets, ce qui est exactement le marche que ces fixtures existent pour passer.
// `000d5950` reste INTACTE (`pointsStride` = 1) : c'est la reference historique du chantier,
// identique octet pour octet a la fixture d'avant ce lot. Le taux voyage dans le manifeste,
// fixture par fixture : un consommateur qui voudrait compter des points doit savoir qu'il en
// manque quatre sur cinq, et ne pas le deduire d'une constante ecrite ailleurs.
//
// UN SEUL JEU VIVANT (meme arbitrage). Le nom du fichier porte la version de schema : sans
// regle, chaque montee deposerait un jeu complet DE PLUS dans l'arbre. La regeneration ecrit
// donc le jeu de la version COURANTE et SUPPRIME celui de la precedente ; le manifeste ne liste
// que la version courante, et l'historique git garde le reste. C'est
// [TestContractFixturesUnSeulJeuVivant] qui le tient hors regeneration.
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
//
// ET LA REGENERATION ECHOUE TOUJOURS (lecon C1 de la ronde 2 du lot 0.A) : `go test` jette la
// sortie d'un paquet qui PASSE, donc une reecriture annoncee par `t.Logf` est invisible avec la
// commande documentee (sans `-v`). La porte termine par un `t.Fatalf` qui NOMME ce qu'elle a
// reecrit ; on relance sans les drapeaux pour verifier.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"
)

// contractFixturesEnv : la seconde condition explicite de la regeneration (cf. en-tete).
const contractFixturesEnv = "REPLAY_CONTRACT_UPDATE"

// contractFixturesStride : un point de piste sur combien, pour les fixtures PAR BUILD.
//
// LA VALEUR EST UN ARBITRAGE, PAS UN REGLAGE (pilote, 2026-09-13) : 5 est ce qui ramene le jeu
// sous le plafond avec de la marge (2,01 Mio mesures contre 3 Mio) sans descendre au point ou
// une piste n'aurait plus assez de points pour exercer les logiques du web. La changer, c'est
// republier tout le jeu : passer par la porte de regeneration, jamais par une edition.
const contractFixturesStride = 5

// contractFixture : un document cuit publie au web.
type contractFixture struct {
	// film : l'identifiant court du film dont le document est cuit.
	film string
	// build : le build du jeu, tel que la table de `golden_builds_test.go` le nomme. Il voyage
	// jusqu'au manifeste : le web doit pouvoir dire de QUELLE generation de jeu vient le
	// document qu'il vient de faire traverser sa frontiere.
	build string
	// stride : un point de piste sur combien (cf. en-tete). 1 = document intact.
	stride int
	// assemble : la cuisson, identique a celle du golden d'assemblage du meme build.
	assemble func(t *testing.T) ReplayDocument
}

// cuire rend le document PUBLIE : l'assemblage du build, puis la coupe des points de piste.
//
// UN SEUL CHEMIN POUR LES DEUX USAGES (comparaison et regeneration) : si la comparaison cuisait
// autrement que la regeneration, le golden ne comparerait pas ce qu'il a ecrit.
func (f contractFixture) cuire(t *testing.T) ReplayDocument {
	t.Helper()
	return amincirPistes(f.assemble(t), f.stride)
}

// amincirPistes garde un point sur `stride` par piste, PLUS LE DERNIER.
//
// LE PREMIER ET LE DERNIER NE SE PERDENT JAMAIS, et ce n'est pas de la coquetterie : le web lit
// la fin de la fenetre de vie dans `endFrame` OU, a defaut, dans le `t` du dernier point
// (`replayLogic.trackWindow`), et la premiere position a l'image de depart. Les couper
// deplacerait la fenetre des pistes que le document plein decrit — la coupe doit alleger le
// document, pas le changer de sens.
func amincirPistes(doc ReplayDocument, stride int) ReplayDocument {
	if stride <= 1 {
		return doc
	}
	tracks := make([]Track, len(doc.Tracks))
	copy(tracks, doc.Tracks)
	for i := range tracks {
		pts := tracks[i].Points
		if len(pts) <= 2 {
			continue
		}
		gardes := make([]Point, 0, len(pts)/stride+2)
		for j, p := range pts {
			if j%stride == 0 || j == len(pts)-1 {
				gardes = append(gardes, p)
			}
		}
		tracks[i].Points = gardes
	}
	doc.Tracks = tracks
	return doc
}

// file : le nom du fichier publie. La version de schema ET le film y sont, parce que ce sont
// les deux axes du jeu : le web nomme la version dans sa matrice de compatibilite (0.B.2), et
// une fixture d'un autre build est un autre document, jamais le meme ecrase.
//
// COMPRESSE, ET LA MESURE LE JUSTIFIE : le document du film de reference pese 3 318 459 octets
// indente, 401 589 une fois gzippe (mesure du 2026-09-13). Le depot versionne deja un fixture
// binaire de cette famille par la meme logique (`testdata/inputs_000d5950.bin.gz`, 1 033 495
// octets). Un diff lisible ne se perd pas pour autant : ce qui doit se relire en revue est la
// FORME du document, et elle a son propre golden en clair (`testdata/document_shape.golden`).
func (f contractFixture) file() string {
	return fmt.Sprintf("replay_schema_%d_%s.json.gz", SchemaVersion, f.film)
}

// contractFixtures : ce que Go publie au web, DERIVE de la table des builds.
//
// LA TABLE N'EST PAS RECOPIEE ICI, et c'est tout l'objet de 0.B.7 : une seconde liste de films
// se serait desynchronisee du premier build ajoute a `goldenBuilds()`. Le web recevrait alors
// une fixture de moins que ce que le depot fige, sans qu'aucun test ne le dise.
func contractFixtures() []contractFixture {
	builds := goldenBuilds()
	out := make([]contractFixture, 0, len(builds))
	for _, b := range builds {
		// `000d5950` INTACT, les autres a un point sur cinq (cf. en-tete) : la reference
		// historique du chantier ne se coupe pas, le budget se tient sur les sept autres.
		stride := contractFixturesStride
		if b.Short8 == goldenFilm {
			stride = 1
		}
		out = append(out, contractFixture{
			film:   b.Short8,
			build:  b.Build,
			stride: stride,
			assemble: func(t *testing.T) ReplayDocument {
				t.Helper()
				g, entry := chargerGoldenBuild(t, b)
				return assemblerGoldenBuild(t, b, g, entry)
			},
		})
	}
	return out
}

// contractManifestFile : le manifeste du jeu de fixtures. Il existe pour une raison precise
// et unique : `testDoc.ts` (la fixture minimale partagee par ~250 tests web) doit prendre sa
// version de schema ICI et non plus d'un litteral ecrit a la main, sans pour autant charger
// et analyser les documents complets dans chaque fichier de test. Le manifeste est la PARTIE
// LEGERE du meme fait, ecrite par le meme producteur, et sa coherence avec les documents est
// verifiee par TestContractFixturesMatchCommitted.
const contractManifestFile = "manifest.json"

// contractManifestEntry : ce que le manifeste dit d'une fixture.
//
// TROIS FAITS ET UNE MESURE. Le film et le build disent d'ou vient le document (le web itere le
// jeu entier et nomme le build dans ses messages) ; la version de schema laisse la matrice de
// compatibilite statuer sur CHAQUE fixture sans ouvrir un document de 3,3 Mio ; la taille est
// ce qui rend le budget du jeu verifiable sans relire le dossier.
type contractManifestEntry struct {
	File          string `json:"file"`
	Film          string `json:"film"`
	Build         string `json:"build"`
	SchemaVersion int    `json:"schemaVersion"`
	Bytes         int    `json:"bytes"`
	// PointsStride : un point de piste sur combien ce document porte (1 = intact). DECLARE, et
	// non deduit : un consommateur qui compterait des points doit apprendre du document
	// lui-meme qu'il en manque quatre sur cinq.
	PointsStride int `json:"pointsStride"`
}

// contractManifest : la forme du manifeste, cote Go comme cote web.
type contractManifest struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Fixtures      []contractManifestEntry `json:"fixtures"`
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

// contractFixtureEcrire ecrit une fixture compressee et rend sa taille sur disque.
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

// contractFixtureTaille : la taille SUR DISQUE d'une fixture commise.
func contractFixtureTaille(t *testing.T, path string) int {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("fixture de contrat absente : %v — regenerer avec "+
			"%s=1 go test -run ContractFixtures -update", err, contractFixturesEnv)
	}
	return int(st.Size())
}

// TestContractFixturesMatchCommitted : les fixtures commises sont-elles toujours celles que la
// cuisson produit ?
//
// C'EST UN GOLDEN, ET IL ECHOUE POUR DEUX RAISONS DIFFERENTES. Soit le document a change
// (un calque, un champ, une valeur) : il faut alors monter `SchemaVersion` si le CONTENU cuit
// change, ecrire l'entree de chronique, et regenerer. Soit la fixture a ete editee a la main :
// il faut la regenerer. Dans les deux cas le message dit quoi faire.
func TestContractFixturesMatchCommitted(t *testing.T) {
	dir := contractFixturesDir(t)
	var attendus []contractManifestEntry
	for _, f := range contractFixtures() {
		t.Run(f.build+"/"+f.film, func(t *testing.T) {
			want := contractFixtureLire(t, filepath.Join(dir, f.file()))
			got := contractFixtureBytes(t, f.cuire(t))
			if string(want) != string(got) {
				t.Errorf("la fixture de contrat %s a change (%d octets commis, %d produits).\n%s",
					f.file(), len(want), len(got), premierEcartAssembly(string(want), string(got)))
			}
		})
		attendus = append(attendus, contractManifestEntry{
			File: f.file(), Film: f.film, Build: f.build, SchemaVersion: SchemaVersion,
			PointsStride: f.stride,
			Bytes:        contractFixtureTaille(t, filepath.Join(dir, f.file())),
		})
	}
	verifierManifesteContrat(t, dir, attendus)
}

// verifierManifesteContrat : le manifeste dit-il la verite sur ce que le dossier porte ?
//
// SANS CE CONTROLE le manifeste serait une seconde source de verite qui derive : `testDoc.ts`
// lirait une version que plus aucun document ne porte, la matrice de compatibilite statuerait
// sur des builds absents du dossier, et tous les tests web qui en dependent affirmeraient un
// contrat perime — verts. La TAILLE y est verifiee comme le reste : un manifeste qui annonce
// des octets qu'aucun fichier ne pese laisserait le budget se mesurer sur une fiction.
func verifierManifesteContrat(t *testing.T, dir string, attendus []contractManifestEntry) {
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
	if fmt.Sprint(m.Fixtures) != fmt.Sprint(attendus) {
		t.Errorf("le manifeste liste\n  %v\nle producteur publie\n  %v", m.Fixtures, attendus)
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
	for _, f := range contractFixtures() {
		t.Run(f.build+"/"+f.film, func(t *testing.T) {
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
	var entrees []contractManifestEntry
	var ecrits []string
	total := 0
	for _, f := range contractFixtures() {
		blob := contractFixtureBytes(t, f.cuire(t))
		path := filepath.Join(dir, f.file())
		compresse := contractFixtureEcrire(t, path, blob)
		total += compresse
		t.Logf("fixture de contrat reecrite : %s (%d octets JSON, %d compresses)",
			path, len(blob), compresse)
		entrees = append(entrees, contractManifestEntry{
			File: f.file(), Film: f.film, Build: f.build, SchemaVersion: SchemaVersion,
			PointsStride: f.stride,
			Bytes:        compresse,
		})
		ecrits = append(ecrits, f.file())
	}
	m, err := json.MarshalIndent(contractManifest{SchemaVersion: SchemaVersion, Fixtures: entrees}, "", " ")
	if err != nil {
		t.Fatalf("serialisation du manifeste : %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, contractManifestFile), append(m, '\n'), 0o600); err != nil {
		t.Fatalf("ecriture du manifeste : %v", err)
	}
	supprimes := purgerJeuxPerimes(t, dir, ecrits)
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (lecon C1 de la ronde 2 du lot 0.A) :
	// `go test` jette la sortie d'un paquet qui passe, donc une reecriture annoncee par
	// `t.Logf` est INVISIBLE avec la commande documentee (sans `-v`).
	t.Fatalf("%d fixture(s) et le manifeste reecrits : %s ; %d octets au total (plafond %d) ; "+
		"%d fixture(s) d'une version perimee supprimee(s)%s ; relancer sans -update pour verifier",
		len(ecrits), strings.Join(ecrits, ", "), total, contractFixturesBudget,
		len(supprimes), listeCourte(supprimes))
}
