package source_test

// rev_test.go — LE GATE DE [source.Rev], SUR LE MECANISME CENTRAL (`film/revision`, lot 2.6.0).
//
// # CE QU IL TIENT, ET LES DEUX GESTES QU IL EXIGE
//
// Il hache les sources non-test de la couche `source` et compare AU GOLDEN, qui porte le couple
// (revision, empreinte). Toucher la couche le fait rougir ; le remettre au vert demande de
// rouvrir la ligne de la revision — donc de DECIDER si la facon d atteindre les octets a change.
// Les deux derives sont distinguees, comme cote `killsource` : « les sources ont change » et
// « la revision a change sans la couche ».
//
// # POURQUOI IL N Y A PAS UNE LIGNE D EMPREINTE ICI
//
// CLAUDE.md regle 6 : le calcul, la chronique, la porte et les messages sont CENTRAUX depuis le
// lot 2.6.0, et le garde-rail `archlint/no_ad_hoc_source_fingerprint_test.go` interdit la copie.
// Ce fichier ne fait que DECLARER ce qui est propre a la couche : sa racine, son golden, sa
// porte, et la QUESTION que son echec pose.

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updateSourceRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE. Elle est NOMMEE
// (correctif de la revue R1, P1-1) : accrochee a un `-update` generique, elle laisserait
// `go test ./...source/ -update` refiger l empreinte d une couche CASSEE en repondant `ok`.
var updateSourceRev = flag.Bool("update-source-rev", false,
	"reecrire testdata/source_rev.golden (revision ET empreinte) — CE golden seulement")

const (
	// cheminGoldenSourceRev : le golden, relatif au paquet.
	cheminGoldenSourceRev = "testdata/source_rev.golden"
	// fichierPorteurDeRevisionSource : le fichier qui PORTE la revision, exclu du hachage.
	fichierPorteurDeRevisionSource = "rev.go"
	// prefixeRevisionSource : le prefixe de la serie, pour la chronique.
	prefixeRevisionSource = "source"
	// commandeRegenerationSourceRev : la commande complete, telle qu on la tape.
	commandeRegenerationSourceRev = "LEVELUP_UPDATE_SOURCE_REV=1 go test " +
		"./internal/games/halo_infinite/film/internal/source/ -run TestSourceRevSuitLaCoucheSource " +
		"-update-source-rev"
)

// porteSourceRev / messagesSourceRev : ce que la couche declare au mecanisme central.
func porteSourceRev() revision.Porte { return revision.Porte{Nom: "source-rev"} }

func messagesSourceRev() revision.Messages {
	return revision.Messages{
		Couche:    prefixeRevisionSource,
		Constante: "source.Rev",
		Forme:     "source-AAAA-MM-JJ[.N]",
		Porte:     porteSourceRev(),
		Commande:  commandeRegenerationSourceRev,
		Question: "LA LECTURE DES OCTETS A CHANGE : TOUT RE-DECODE. La couche `source` est la " +
			"porte aux octets (ADR 0034 D-2) — aucun fait, aucun document cuit, aucune ligne de " +
			"kill n est hors de portee. `facts.Rev` hache la VALEUR de cette revision : la " +
			"monter fait monter les faits mecaniquement, et le backlog killsource avec eux (D6).",
	}
}

// racineDeLaCoucheSource : le paquet lui-meme, resolu par `runtime.Caller` — pas un chemin
// relatif au repertoire courant : le jour ou la couche demenage, ce test doit suivre le paquet
// et non hacher un dossier vide.
func racineDeLaCoucheSource(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(ici)
}

// empreinteDeLaCoucheSource rend l empreinte et le nombre de fichiers haches.
func empreinteDeLaCoucheSource(t *testing.T) (string, int) {
	t.Helper()
	res, err := revision.Calculer(
		[]string{racineDeLaCoucheSource(t)},
		func(rel string) bool { return rel == fichierPorteurDeRevisionSource })
	if err != nil {
		t.Fatalf("empreinte de la couche source : %v", err)
	}
	return res.Empreinte, res.Fichiers
}

// TestSourceRevSuitLaCoucheSource — LE GATE.
func TestSourceRevSuitLaCoucheSource(t *testing.T) {
	empreinte, fichiers := empreinteDeLaCoucheSource(t)
	msg := messagesSourceRev()
	if *updateSourceRev {
		regenererGoldenSourceRev(t, empreinte)
		return
	}
	c, err := revision.LireChronique(prefixeRevisionSource, []string{fichierPorteurDeRevisionSource},
		cheminGoldenSourceRev)
	if err != nil {
		t.Fatalf("chronique de `source` : %v", err)
	}
	courante := c.Courante()
	switch {
	case courante.Revision == source.Rev && courante.Empreinte != empreinte:
		t.Fatal(msg.SourcesOntChange(courante.Revision, courante.Empreinte, empreinte, fichiers))
	case courante.Revision != source.Rev && courante.Empreinte == empreinte:
		t.Fatal(msg.RevisionSeuleAChange(source.Rev, courante.Revision, empreinte))
	case courante.Revision != source.Rev:
		t.Fatal(msg.GoldenPerime(source.Rev, empreinte, fichiers))
	}
}

// regenererGoldenSourceRev : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenSourceRev(t *testing.T, empreinte string) {
	t.Helper()
	porte := porteSourceRev()
	ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable()))
	if !ouverte {
		t.Skip(raison)
	}
	ecrit, err := porte.Reecrire(cheminGoldenSourceRev, source.Rev, empreinte)
	if err != nil {
		t.Fatalf("regeneration du golden : %v", err)
	}
	// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf`
	// serait INVISIBLE avec la commande documentee, et se lirait `ok`.
	t.Fatal(ecrit)
}

// TestChroniqueDeSourceCouvreLaRevisionCourante : la revision courante a son ENTREE dans le
// godoc de `rev.go` ET la derniere ligne du golden, et les rangs se suivent sans trou.
//
// Le defaut qu il ferme est mesure cote grammaire (constat F5 de la revue de jalon M1) : la
// chronique s y etait arretee a `.12` pendant que la constante valait `.14`, et rien ne
// rougissait — les deux changements de comportement intermediaires n avaient aucune entree.
func TestChroniqueDeSourceCouvreLaRevisionCourante(t *testing.T) {
	c, err := revision.LireChronique(prefixeRevisionSource, []string{fichierPorteurDeRevisionSource},
		cheminGoldenSourceRev)
	if err != nil {
		t.Fatalf("chronique de `source` : %v", err)
	}
	if err := c.VerifierCouverture(source.Rev); err != nil {
		t.Fatal(messagesSourceRev().SansEntreeDeChronique(err.Error()))
	}
	// Plancher VIDE : la chronique de `source` nait avec ce lot, elle tient la regle depuis son
	// premier rang — il n y a aucun passe a amnistier.
	if err := c.VerifierRangs(""); err != nil {
		t.Fatal(err)
	}
}
