package replay

// golden_inputs_test.go — LES ENTREES DECODEES DU FILM DE REFERENCE, FIGEES.
//
// POURQUOI CE FICHIER EXISTE. `BuildFromPositions` est deja PUR — il ne lui manquait que ses
// entrees. Tant qu elles ne vivaient que dans un film de 20,2 Mo hors depot, les chiffres du
// chantier (475/519 tirs, 90/105 vies nommees, 70 lancers, 439 projectiles, 184 etats
// d inventaire) n etaient verrouilles par AUCUN test : ils vivaient dans des Markdown et dans
// un artefact genere. Un refactor de l assemblage pouvait les deplacer en silence.
//
// CE QUE CE FIXTURE EST, ET CE QU IL N EST PAS. C est un ETAGE 1 : les entrees DEJA DECODEES,
// serialisees, rejouees dans l assemblage pur. Il verrouille l ASSEMBLAGE et rien d autre — un
// changement du DECODAGE ne le fait pas bouger, c est le travail de l etage 2 (la mini-bobine,
// minifilm_test.go). Les deux etages sont complementaires et ne se remplacent pas.
//
// ZERO OCTET DE FILM. Les tests de golden_assembly_test.go ne lisent que ce fichier fige. Le
// film n intervient qu a la REGENERATION, ci-dessous, qui est la seule porte d ecriture.
//
// REGENERATION (la seule ; un golden ne s edite JAMAIS a la main) :
//
//	REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/games/halo_infinite/film/replay/ -run GoldenInputs -update
//
// puis, la sortie figee elle-meme :
//
//	go test ./internal/games/halo_infinite/film/replay/ -run GoldenAssembly -update
//
// CE QUE LE FIXTURE PORTE, ET POURQUOI PAS PLUS. Il porte exactement les champs que
// l assemblage CONSOMME (cf. la liste par type ci-dessous). Il ne porte NI la geometrie NI la
// structure de carte : celles-ci ne viennent pas du film mais de catalogues versionnes a part,
// et les inclure ferait grossir le fixture d un ordre de grandeur pour verrouiller un chargement
// de fichier, pas un decodage.
//
// LE FORMAT EST DELTA-CODE, ET L ORDRE D ORIGINE EST PRESERVE. `BuildFromPositions` trie ses
// positions par `sort.SliceStable` et numerote les traces dans l ORDRE DE PREMIERE APPARITION
// des slots : reordonner les positions changerait la sortie. Le codec conserve donc la suite
// telle que le decodeur l a produite, et ne delta-code que les VALEURS (horodatage global,
// coordonnees par slot).

import (
	"bytes"
	"compress/gzip"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "reecrire les fichiers figes de testdata/")

// goldenFilm est le film de reference du chantier : Cliffhanger, Fiesta, 8 joueurs.
const goldenFilm = "000d5950"

// goldenDir est le repertoire des sorties figees.
const goldenDir = "testdata"

// goldenInputsPath est le fixture d entrees, versionne.
func goldenInputsPath() string {
	return filepath.Join(goldenDir, "inputs_"+goldenFilm+".bin.gz")
}

// loadGoldenInputs relit le fixture versionne. AUCUN OCTET DE FILM.
func loadGoldenInputs(t *testing.T) *FilmFacts {
	t.Helper()
	raw, err := os.ReadFile(goldenInputsPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("fixture d entrees illisible : %v — regenerer avec "+
			"REPLAY_FILM_DIR=<film> go test -run GoldenInputs -update", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture d entrees : gzip illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture d entrees : decompression : %v", err)
	}
	entry, err := goldenMapQuant()
	if err != nil {
		t.Fatalf("entree de catalogue du film de reference : %v", err)
	}
	g, err := DecodeFilmFacts(buf.Bytes(), entry)
	if err != nil {
		t.Fatalf("fixture d entrees : %v", err)
	}
	// LES ENTITES `ti=9` NE SONT PAS DANS LE BLOB : elles ont leur fichier (golden_entites_test.go).
	g.PlayerEntities = chargerEntitesDuBuild(t, goldenFilm)
	return g
}

// TestGoldenInputsRoundTrip : le codec ne perd rien de ce que l assemblage consomme.
//
// C EST LE FILET DU FIXTURE LUI-MEME. Un codec qui perdrait un champ produirait un golden
// parfaitement stable et parfaitement faux. On le verifie par la seule mesure qui compte :
// l assemblage rejoue sur les entrees RELUES doit rendre le MEME document que sur les entrees
// d origine — ici, les entrees d origine etant deja le fixture, on controle qu un second
// aller-retour est un point fixe, octet pour octet.
func TestGoldenInputsRoundTrip(t *testing.T) {
	g := loadGoldenInputs(t)
	blob := EncodeFilmFacts(g)
	again, err := DecodeFilmFacts(blob, goldenEntryPourTest(t))
	if err != nil {
		t.Fatalf("second decodage : %v", err)
	}
	again.PlayerEntities = g.PlayerEntities // hors du blob, cf. golden_entites_test.go
	if got := EncodeFilmFacts(again); !bytes.Equal(blob, got) {
		t.Fatalf("le codec n est pas un point fixe : %d octets contre %d — un champ se perd "+
			"ou se reconstruit differemment a chaque tour", len(got), len(blob))
	}
	docA := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, g.options())
	docB := BuildFromPositions(goldenFilm, "halo_infinite", again.Positions, again.Fire, again.options())
	if renderAssembly(docA) != renderAssembly(docB) {
		t.Error("l assemblage differe entre les entrees relues et leur re-serialisation : " +
			"le codec perd un champ que BuildFromPositions consomme")
	}
}

// TestGoldenInputsVersionGuard : un fixture d une AUTRE version est refuse par la MAGIE, et le
// message dit quoi faire.
//
// C EST LE VERROU DU CONSTAT DE REVUE DU 2026-08-25. La section `InventoryDeltas` a ete inseree
// au milieu du flux sans que la magie bouge : un fixture de la version precedente passait la
// garde, puis mourait plus loin sur « uvarint illisible a l offset N » — un message qui parle
// d octets alors que le probleme est une version. Le test relit le corps COURANT precede de la
// magie PRECEDENTE : la seule reponse acceptable est le refus de version.
func TestGoldenInputsVersionGuard(t *testing.T) {
	const previousMagic = "REPLAYINPUTS22\n"
	if previousMagic == filmFactsMagic {
		t.Fatal("la magie precedente et la courante sont identiques : le test ne prouve plus rien")
	}
	body := EncodeFilmFacts(loadGoldenInputs(t))[len(filmFactsMagic):]
	stale := append([]byte(previousMagic), body...)
	_, err := DecodeFilmFacts(stale, goldenEntryPourTest(t))
	if err == nil {
		t.Fatal("un fixture d une autre version a ete accepte : la garde de version ne sert a rien")
	}
	if !strings.Contains(err.Error(), "version inconnue") {
		t.Fatalf("la version doit etre refusee POUR CE QU ELLE EST ; message obtenu : %v", err)
	}
}

// TestGoldenInputsRegenerate : LA SEULE PORTE D ECRITURE DU FIXTURE.
//
// Elle exige DEUX conditions explicites — `-update` et `REPLAY_FILM_DIR` — parce qu une
// regeneration accidentelle transformerait n importe quelle regression en « nouvelle
// reference ». C est la meme discipline que le paquet killsource.
func TestGoldenInputsRegenerate(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	switch {
	case !*updateGolden:
		t.Skip("regeneration du fixture : passer -update (et REPLAY_FILM_DIR)")
	case dir == "":
		t.Skip("regeneration du fixture : REPLAY_FILM_DIR non defini")
	}
	g, err := decodeFilmInputs(goldenFilm, dir)
	if err != nil {
		t.Fatalf("decodage du film %s : %v", dir, err)
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
	if err := os.WriteFile(goldenInputsPath(), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture du fixture : %v", err)
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R1, constat R1-8).
	t.Fatalf("fixture reecrit : %s (%d octets brut, %d compresse) — %d positions, %d tirs, "+
		"%d loadouts, %d lancers, %d projectiles, %d inventaires, %d lectures grappin, "+
		"%d impulsions de capacite, %d lectures de charge, %d morts, %d index",
		goldenInputsPath(), len(blob), buf.Len(), len(g.Positions), len(g.Fire), len(g.Loadouts),
		len(g.Grenades), len(g.Projectiles), len(g.Inventory), len(g.GrappleReads),
		len(g.AbilityImpulses), len(g.AbilityCharges), len(g.Deaths), len(g.PlayerIndices.ByXUID))
}

// decodeFilmInputs rejoue EXACTEMENT la sequence de decodage de BuildFromFilm — c est ce qui
// garantit que le fixture porte les memes entrees que la production. Les bornes de carte sont
// celles de Cliffhanger, lues dans le catalogue versionne du titre.
