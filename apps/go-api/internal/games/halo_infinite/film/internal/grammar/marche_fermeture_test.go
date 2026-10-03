package grammar

// marche_fermeture_test.go — T1 DE LA SPECIFICATION DE LA REPRESENTATION INTERMEDIAIRE (ADR 0037
// IR-4) : pour chaque paquet de la structure, les bits consommes et l etat de fermeture sont
// coherents ; et la fermeture que la structure porte sur les bobines du depot ne descend jamais.
//
// LA COHERENCE SE JUGE AVEC LE PREDICAT DE LA GRAMMAIRE ([vueCFermee]) appliquee aux bits que la
// structure dit consommes : la structure porte le verdict, jamais une seconde definition de la
// fermeture.

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// updateStructureFermeture est la porte de regeneration du golden de [TestLaFermetureDeLaStructureNeDescendPas].
var updateStructureFermeture = flag.Bool("update-structure-fermeture", false,
	"reecrire testdata/structure_fermeture.golden (fermeture portee par la structure de lecture)")

// structureFermetureGoldenPath : le golden, a cote de ceux des cartes de fermeture.
const structureFermetureGoldenPath = "testdata/structure_fermeture.golden"

// enteteStructureFermeture : la prose du golden.
const enteteStructureFermeture = `# FERMETURE PORTEE PAR LA STRUCTURE DE LECTURE (ADR 0037, T1 de la specification).
# Une ligne par bobine et par mesure : bobine, mesure, fermes (ou prouves), total.
#   trames fermees               paquets delta au verdict ferme / paquets delta
#   records de trame prouves     records de trame prouves / records de trame
#   records d image-cle prouves  records d image-cle prouves / records d image-cle
# Le ratchet rougit sur une BAISSE ou une ligne qui disparait ; une hausse se fige par :
#   go test ./internal/games/halo_infinite/film/internal/grammar/ -run TestLaFermetureDeLaStructureNeDescendPas -update-structure-fermeture
`

// TestLaFermetureDeLaStructureNeDescendPas : T1. Chaque paquet des bobines tient la coherence de
// sa fermeture ([verifierLaFermeture]) et les comptes de la structure ne descendent pas.
// MUTATION — un verdict ferme rendu sans la regle de l ecrivain (`case l.verdict.FermeeAuBit:` dans
// [rangerLaFermeture]) : la coherence rougit sur les paquets refuses.
func TestLaFermetureDeLaStructureNeDescendPas(t *testing.T) {
	got := mesurerLaFermetureDeLaStructure(t)
	if *updateStructureFermeture {
		if err := os.WriteFile(structureFermetureGoldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture du golden : %v", err)
		}
		// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` : cf. `TestKeyframeClosureRatchet`.
		t.Fatalf("1 reference(s) reecrite(s) : %s (%d octets) ; relancer sans -update-structure-fermeture pour verifier",
			structureFermetureGoldenPath, len(got))
	}
	brut, err := os.ReadFile(structureFermetureGoldenPath) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent (%s) : %v — regenerer avec -update-structure-fermeture", structureFermetureGoldenPath, err)
	}
	comparerFermeture(t, string(brut), got, "-update-structure-fermeture")
}

// comptesDeLaStructure : ce que la structure dit d une bobine.
type comptesDeLaStructure struct {
	paquets, fermes                 int
	recordsDeTrame, recordsProuves  int
	recordsDImageCle, imagesProuves int
}

// mesurerLaFermetureDeLaStructure rend le rendu textuel des comptes de la structure sur les
// bobines du depot qui portent un registre, en verifiant chaque paquet au passage.
func mesurerLaFermetureDeLaStructure(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(enteteStructureFermeture)
	for _, bo := range frameClosureBobines() {
		film, err := source.LoadDir(bo.dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", bo.dir, err)
		}
		if _, ok := FilmRegistryChunk(film); !ok {
			continue
		}
		c := compterLaStructure(t, bo.nom, contexteDeBobine(film))
		fmt.Fprintf(&b, "%s\ttrames fermees\t%d\t%d\n", bo.nom, c.fermes, c.paquets)
		fmt.Fprintf(&b, "%s\trecords de trame prouves\t%d\t%d\n", bo.nom, c.recordsProuves, c.recordsDeTrame)
		fmt.Fprintf(&b, "%s\trecords d image-cle prouves\t%d\t%d\n", bo.nom, c.imagesProuves, c.recordsDImageCle)
	}
	return b.String()
}

// compterLaStructure marche les deux phases d un film et en compte les fermetures et les preuves.
func compterLaStructure(t *testing.T, nom string, fc *FilmContext) comptesDeLaStructure {
	t.Helper()
	var c comptesDeLaStructure
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("%s : images-cles : %v", nom, err)
		}
		c.recordsDImageCle += len(p.Records)
		c.imagesProuves += recordsProuves(p)
	}
	for p, err := range fc.Trames(nil) {
		if err != nil {
			t.Fatalf("%s : trames : %v", nom, err)
		}
		verifierLaFermeture(t, nom, p)
		c.paquets++
		if p.Fermeture.Verdict == lecture.VerdictFerme {
			c.fermes++
		}
		c.recordsDeTrame += len(p.Records)
		c.recordsProuves += recordsProuves(p)
	}
	return c
}

// recordsProuves compte les records prouves d un paquet.
func recordsProuves(p *lecture.Paquet) int {
	n := 0
	for _, r := range p.Records {
		if r.Preuve == lecture.PreuveFerme {
			n++
		}
	}
	return n
}

// verifierLaFermeture : la fermeture au bit pres que la structure porte est le predicat de la
// grammaire applique aux bits qu elle dit consommes ; un paquet ferme l est au bit pres sans regle
// contredite, un paquet refuse ne l est pas ; une queue opaque commence avant le curseur.
func verifierLaFermeture(t *testing.T, nom string, p *lecture.Paquet) {
	t.Helper()
	f := p.Fermeture
	ou := fmt.Sprintf("%s chunk %d paquet %d", nom, p.Chunk, p.Index)
	if p.Debut == lecture.DebutNonLocalise {
		return
	}
	auBit := p.VueC.Etat == lecture.VueTerminee && vueCFermee(p.Payload, int(f.Consommes))
	if f.AuBit != auBit {
		t.Errorf("%s : au bit %t, mais la vue C %d et %d bits consommes sur %d disent %t", ou, f.AuBit,
			p.VueC.Etat, f.Consommes, f.Longueur, auBit)
	}
	ecrivable := f.AuBit && f.Regle == lecture.AucuneRegle
	switch f.Verdict {
	case lecture.VerdictFerme:
		if !ecrivable {
			t.Errorf("%s : ferme sans la fermeture au bit pres ou avec la regle %d contredite", ou, f.Regle)
		}
	case lecture.VerdictRefuse:
		if ecrivable {
			t.Errorf("%s : refuse alors que la lecture ferme au bit pres sans regle contredite", ou)
		}
	case lecture.VerdictQueueOpaque:
		if f.Queue.Debut > f.Consommes {
			t.Errorf("%s : queue opaque au bit %d, apres le curseur %d", ou, f.Queue.Debut, f.Consommes)
		}
	}
}
