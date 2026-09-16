package revision_test

// empreinte_test.go — LA PREUVE QUE L EMPREINTE MORD, ET QU ELLE NE MORD PAS SUR N IMPORTE QUOI.
//
// Meme doctrine que `TestEmpreinteSourcesGoMord` (le gate de `killsource`) : un garde-rail qui ne
// detecte jamais rien est inutile, et un garde-rail qui detecte tout est amende par reflexe. Les
// cas ci-dessous tiennent les proprietes sur lesquelles les quatre gates de couche reposeront.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// couche pose une arborescence de sources dans un dossier temporaire.
func couche(t *testing.T, fichiers map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for nom, contenu := range fichiers {
		chemin := filepath.Join(dir, filepath.FromSlash(nom))
		if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
			t.Fatalf("mkdir %s : %v", nom, err)
		}
		if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
			t.Fatalf("ecrire %s : %v", nom, err)
		}
	}
	return dir
}

// empreinteDe rend l empreinte d une ou plusieurs racines, sans exclusion ni amont.
func empreinteDe(t *testing.T, racines ...string) string {
	t.Helper()
	e, err := revision.Empreinte(racines, nil)
	if err != nil {
		t.Fatalf("empreinte de %v : %v", racines, err)
	}
	return e
}

// sourcesDeReference : la couche de reference des cas ci-dessous.
func sourcesDeReference() map[string]string {
	return map[string]string{
		"decode.go": "package p\n\nconst Largeur = 5\n",
		"scan.go":   "package p\n\nfunc Scan() {}\n",
	}
}

func TestEmpreinteMordSurLesSources(t *testing.T) {
	ref := empreinteDe(t, couche(t, sourcesDeReference()))

	// 1. UN FICHIER AJOUTE SOUS LA RACINE -> l empreinte bouge. C est le cas du lecteur neuf.
	ajout := sourcesDeReference()
	ajout["lecteur.go"] = "package p\n\nfunc Lire() {}\n"
	if empreinteDe(t, couche(t, ajout)) == ref {
		t.Error("empreinte immobile apres l ajout d une source : le mecanisme est aveugle")
	}

	// 2. UN OCTET CHANGE -> l empreinte bouge. C est le cas de la largeur corrigee.
	mute := sourcesDeReference()
	mute["decode.go"] = "package p\n\nconst Largeur = 6\n"
	if empreinteDe(t, couche(t, mute)) == ref {
		t.Error("empreinte immobile apres un changement de source")
	}

	// 3. UN FICHIER RENOMME -> l empreinte bouge : le chemin entre dans le hachage.
	renomme := map[string]string{
		"decode2.go": "package p\n\nconst Largeur = 5\n",
		"scan.go":    "package p\n\nfunc Scan() {}\n",
	}
	if empreinteDe(t, couche(t, renomme)) == ref {
		t.Error("empreinte immobile apres un renommage : le chemin n entre pas dans le hachage")
	}
}

func TestEmpreinteIgnoreCeQuiNEstPasDeLaProduction(t *testing.T) {
	ref := empreinteDe(t, couche(t, sourcesDeReference()))

	// UN TEST OU UNE FIXTURE AJOUTE -> l empreinte NE BOUGE PAS. Sans cette exclusion, tout
	// ajout de test exigerait une montee de revision, ce qui viderait la revision de son sens :
	// elle designerait des lignes produites identiques.
	avecTests := sourcesDeReference()
	avecTests["scan_test.go"] = "package p\n\nfunc TestScan() {}\n"
	avecTests["testdata/film.go"] = "package fixture\n"
	avecTests["testdata/sous/encore.go"] = "package fixture\n"
	if got := empreinteDe(t, couche(t, avecTests)); got != ref {
		t.Errorf("empreinte modifiee par un test ou une fixture : %s != %s", got, ref)
	}

	// LES MEMES SOURCES EN CRLF -> meme empreinte. C est ce qui rend le gate identique sur la CI
	// (Linux) et sur un poste Windows.
	crlf := map[string]string{
		"decode.go": "package p\r\n\r\nconst Largeur = 5\r\n",
		"scan.go":   "package p\r\n\r\nfunc Scan() {}\r\n",
	}
	if got := empreinteDe(t, couche(t, crlf)); got != ref {
		t.Errorf("empreinte sensible aux fins de ligne : %s != %s", got, ref)
	}
}

func TestEmpreinteSuitLesValeursAmont(t *testing.T) {
	dir := couche(t, sourcesDeReference())
	sansAmont, err := revision.Empreinte([]string{dir}, nil)
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}

	// UNE VALEUR AMONT DIFFERENTE -> l empreinte bouge. C est la decision V15 (12) : une
	// correction de grammaire qui change la sortie des faits sans toucher un octet de `facts/`
	// doit faire monter `facts.Rev`.
	a, err := revision.Empreinte([]string{dir}, nil, "grammar-2026-09-17")
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}
	b, err := revision.Empreinte([]string{dir}, nil, "grammar-2026-09-18")
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}
	if a == b {
		t.Error("empreinte immobile alors que la revision amont a change : une correction de " +
			"grammaire ne ferait pas monter la revision des faits (V15 (12))")
	}

	// AUCUNE VALEUR AMONT -> AUCUN OCTET ECRIT. C est la propriete dont depend l heritage sans
	// renumerotation : une couche sans amont rend la meme empreinte qu un mecanisme qui ne
	// connait pas les amonts (cf. `equivalence_test.go`).
	if a == sansAmont || b == sansAmont {
		t.Error("une valeur amont ne change pas l empreinte : elle n entre pas dans le hachage")
	}
	vide, err := revision.Empreinte([]string{dir}, nil, []string{}...)
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}
	if vide != sansAmont {
		t.Errorf("une liste d amonts VIDE ecrit des octets : %s != %s — l equivalence avec les "+
			"deux mecanismes existants tombe", vide, sansAmont)
	}
}

func TestEmpreinteExclutCeQueLAppelantDemande(t *testing.T) {
	avec := sourcesDeReference()
	avec["rev.go"] = "package p\n\nconst Rev = \"couche-2026-09-17\"\n"
	dir := couche(t, avec)
	sans := empreinteDe(t, couche(t, sourcesDeReference()))

	exclu, err := revision.Empreinte([]string{dir}, func(rel string) bool { return rel == "rev.go" })
	if err != nil {
		t.Fatalf("empreinte : %v", err)
	}
	// LE FICHIER QUI PORTE LA REVISION N EST PAS DE LA COUCHE. L exclure rend atteignable la
	// branche « la revision a change sans que la couche bouge » (correctif R1 / P2-3) : sans
	// elle, faire monter la revision changerait aussi l empreinte, et ce message serait mort.
	if exclu != sans {
		t.Errorf("le fichier exclu entre quand meme dans le hachage : %s != %s", exclu, sans)
	}
	if garde := empreinteDe(t, dir); garde == exclu {
		t.Error("le predicat d exclusion n a aucun effet mesurable")
	}
}

func TestEmpreinteRefuseUneRacineVide(t *testing.T) {
	vide := t.TempDir()
	_, err := revision.Empreinte([]string{vide}, nil)
	if !errors.Is(err, revision.ErrRacineSansSource) {
		t.Errorf("racine sans source : err = %v, attendu ErrRacineSansSource — une couche dont "+
			"l arborescence a bouge doit echouer bruyamment, pas hacher du vide", err)
	}
	if _, err := revision.Empreinte([]string{filepath.Join(vide, "absent")}, nil); err == nil {
		t.Error("racine inexistante acceptee")
	}
	// UNE RACINE QUI NE PORTE QUE DES TESTS EST VIDE AU SENS DU MECANISME.
	quEuxDesTests := couche(t, map[string]string{"scan_test.go": "package p\n"})
	if _, err := revision.Empreinte([]string{quEuxDesTests}, nil); !errors.Is(err, revision.ErrRacineSansSource) {
		t.Errorf("racine sans source de production : err = %v, attendu ErrRacineSansSource", err)
	}
}

func TestEmpreinteDependDeLOrdreDesRacines(t *testing.T) {
	a := couche(t, map[string]string{"a.go": "package p\n\nconst A = 1\n"})
	b := couche(t, map[string]string{"b.go": "package p\n\nconst B = 2\n"})
	// L ORDRE DES RACINES EST UNE DONNEE DU CONTRAT, pas un detail : les couches le declarent
	// une fois et ne le changent pas sans monter leur revision.
	if empreinteDe(t, a, b) == empreinteDe(t, b, a) {
		t.Error("l ordre des racines ne change pas l empreinte : deux couches ordonnees " +
			"differemment rendraient la meme valeur")
	}
}
