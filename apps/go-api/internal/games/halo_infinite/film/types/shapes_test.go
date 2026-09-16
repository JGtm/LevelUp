package types_test

// shapes_test.go — LA FORME DE CHAQUE TYPE DE CONTRAT, FIGEE A COTE DES REVISIONS QUI LA DATENT
// (item 2.6.2 du PLAN_DECODEUR_FILM_2026-09-13, decision V15 (13)).
//
// # UN SEUL GOLDEN POUR TOUS LES TYPES, ET C EST MESURE
//
// V15 (13) : un golden par type ferait autant de fichiers que de types pour UNE seule question.
// Le modele est `film/replay/document_shape_test.go`, qui fige un arbre entier de 989 lignes dans
// UN fichier — et c est de lui que viennent les quatre proprietes reprises ici : la forme figee
// EN CLAIR dans un golden versionne, une porte de regeneration UNIQUE et bruyante, le golden qui
// doit porter la version COURANTE, et l exigence d une entree de chronique du cote de cette
// version.
//
// # CE QUE LE GOLDEN GARDE, ET CE QU IL NE GARDE PAS
//
// Il garde la FORME PUBLIQUE : par type, le nom et l ordre des champs, leur type Go tel que le
// systeme de types le nomme, et leur tag JSON. C est exactement ce qu un consommateur d une
// autre couche voit — donc ce qui casse chez lui quand cela bouge.
//
// Il ne garde NI le comportement NI les commentaires : un octet de godoc reformule ne fait pas
// rougir ce gate (il fait rougir celui de la couche, qui hache les octets). Les deux gates sont
// complementaires et ce n est pas une redondance : l empreinte dit « quelque chose a bouge dans
// la couche », ce golden dit « voila QUOI, et c est visible depuis l exterieur ».
//
// # POURQUOI LES REVISIONS SONT DANS LE MEME FICHIER
//
// Une forme de contrat qui change sans que la revision de sa couche bouge est une rupture
// SILENCIEUSE pour le parc deja decode. Le golden porte donc, sur sa premiere ligne de donnees,
// les valeurs COURANTES de `source.Rev` et de `facts.Rev` — les deux couches qui produisent ces
// types. Le message d echec pose la question dans l ordre ou elle se decide : la forme a change,
// la sortie peut-elle avoir change ?

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/facts"
	"levelup/go-api/internal/games/halo_infinite/film/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// updateTypesShapes : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE.
var updateTypesShapes = flag.Bool("update-types-shapes", false,
	"reecrire testdata/shapes.golden (formes ET revisions) — CE golden seulement")

const (
	// cheminGoldenFormes : le golden, relatif au paquet.
	cheminGoldenFormes = "testdata/shapes.golden"
	// variableGoldenFormes : le SECOND verrou de la porte. Un drapeau seul se tape par reflexe
	// quand un gate rougit (modele : `replay/document_shape_test.go`, qui exige `-update` ET
	// `LEVELUP_CONTRACT_FIXTURES`).
	variableGoldenFormes = "LEVELUP_UPDATE_TYPES_SHAPES"
	// commandeRegenerationFormes : la commande complete, telle qu on la tape.
	commandeRegenerationFormes = variableGoldenFormes + "=1 go test " +
		"./internal/games/halo_infinite/film/types/ -run TestFormesDesTypesEgalentLeGolden " +
		"-update-types-shapes"
)

// typeDeContrat : un type fige, avec la COUCHE qui le produit — le golden la cite, pour qu une
// forme rouge designe la revision a rouvrir sans avoir a chercher.
type typeDeContrat struct {
	couche string
	valeur any
}

// formesFigees — LES TYPES DE CONTRAT DES COUCHES `source` ET `facts`, dans l ordre du golden.
//
// Un type ajoute a `film/types` sans entree ici serait fige par personne : c est ce que
// `TestTousLesTypesDuPaquetSontFiges` interdit.
var formesFigees = []typeDeContrat{
	{"source", types.ChunkMeta{}},
	{"source", types.Packet{}},
	{"facts/killsource", types.ApparStats{}},
	{"facts/killsource", types.Assist{}},
	{"facts/killsource", types.CoupleStats{}},
	{"facts/objectives", types.DeathInstant{}},
	{"facts/objectives", types.FlagGrabsNetPlayer{}},
	{"facts/objectives", types.FlagSpan{}},
	{"facts/objectives", types.FlagTrack{}},
	{"facts/objectives", types.PlayerLine{}},
	{"facts/objectives", types.ScorePoint{}},
	{"facts/objectives", types.StatRecord{}},
	{"facts/objectives", types.StatValue{}},
}

// ligneDesRevisions : la premiere ligne de donnees du golden.
func ligneDesRevisions() string {
	return fmt.Sprintf("revisions\tsource=%s\tfacts=%s", source.Rev, facts.Rev)
}

// corpsAttendu rend le contenu de donnees du golden : la ligne des revisions, puis une section
// par type.
func corpsAttendu() string {
	var b strings.Builder
	b.WriteString(ligneDesRevisions() + "\n")
	for _, t := range formesFigees {
		rt := reflect.TypeOf(t.valeur)
		fmt.Fprintf(&b, "\n## %s (%s)\n", rt.Name(), t.couche)
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			tag, ok := f.Tag.Lookup("json")
			if !ok {
				tag = "-"
			}
			fmt.Fprintf(&b, "%s\t%s\t%s\n", f.Name, f.Type.String(), tag)
		}
	}
	return b.String()
}

// TestFormesDesTypesEgalentLeGolden — LE GATE.
func TestFormesDesTypesEgalentLeGolden(t *testing.T) {
	attendu := corpsAttendu()
	if *updateTypesShapes {
		regenererGoldenFormes(t, attendu)
		return
	}
	_, donnees := lireGoldenFormes(t)
	if donnees == attendu {
		return
	}
	if premiereLigne(donnees) != ligneDesRevisions() {
		t.Fatalf(`LE GOLDEN DES FORMES NE PORTE PAS LES REVISIONS COURANTES.

  figees   : %s
  courantes: %s

Les formes des types de contrat sont figees A COTE des revisions des couches qui les
produisent : un golden qui date d avant ne dit plus de quelle sortie ces formes sont le
contrat. Regenerer :

  %s`, premiereLigne(donnees), ligneDesRevisions(), commandeRegenerationFormes)
	}
	t.Fatalf(`LA FORME D UN TYPE DE CONTRAT A CHANGE.

%s

DEUX GESTES, ET LES DEUX SONT OBLIGATOIRES :

  1. DECIDER si la SORTIE de la couche change, et faire monter sa revision si oui —
     `+"`facts.Rev`"+` rouvre le backlog killsource (D6, signal utilisateur), `+"`source.Rev`"+`
     veut dire que tout re-decode. Un champ ajoute, retire, renomme ou retype est visible
     depuis les autres couches : c est une rupture de contrat, pas un detail interne.
  2. REGENERER le golden, qui fige les formes ET les revisions :

       %s`, ecartDesFormes(donnees, attendu), commandeRegenerationFormes)
}

// TestTousLesTypesDuPaquetSontFiges : aucun type exporte de `film/types` n echappe au golden.
//
// Sans lui, ajouter un type de contrat SANS l inscrire dans `formesFigees` le laisserait hors de
// toute forme figee — le gate resterait vert en ne gardant rien, qui est le defaut que tous les
// ratchets de ce chantier ferment.
func TestTousLesTypesDuPaquetSontFiges(t *testing.T) {
	declares := typesExportesDuPaquet(t)
	figes := map[string]bool{}
	for _, f := range formesFigees {
		figes[reflect.TypeOf(f.valeur).Name()] = true
	}
	var manquants []string
	for _, nom := range declares {
		if !figes[nom] {
			manquants = append(manquants, nom)
		}
	}
	if len(manquants) > 0 {
		t.Fatalf("types exportes par `film/types` et figes par personne : %s\n"+
			"Les inscrire dans `formesFigees` — un type de contrat dont la forme n est pas figee "+
			"peut changer sans que rien ne le dise aux couches qui le nomment.",
			strings.Join(manquants, ", "))
	}
	if len(declares) != len(formesFigees) {
		t.Errorf("`formesFigees` porte %d entrees pour %d types declares : une entree en trop "+
			"designe un type disparu, et le golden garde alors une forme que personne ne sert.",
			len(formesFigees), len(declares))
	}
}

// regenererGoldenFormes : la porte, avec ses DEUX verrous — et elle ne rend JAMAIS `ok`.
func regenererGoldenFormes(t *testing.T, corps string) {
	t.Helper()
	if os.Getenv(variableGoldenFormes) == "" {
		t.Skipf("regeneration du golden : %s non definie (le drapeau -update-types-shapes seul "+
			"ne suffit pas)", variableGoldenFormes)
	}
	prose, _ := lireGoldenFormes(t)
	if err := os.WriteFile(cheminGoldenFormes, []byte(prose+"\n"+corps), 0o600); err != nil {
		t.Fatalf("ecriture du golden %s : %v", cheminGoldenFormes, err)
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` : `go test` jette la sortie d un paquet qui
	// PASSE, donc une reecriture annoncee par `t.Logf` serait invisible.
	t.Fatalf("golden reecrit : %s (%d types, %s) ; relancer sans -update-types-shapes pour "+
		"verifier", cheminGoldenFormes, len(formesFigees), ligneDesRevisions())
}

// lireGoldenFormes separe la prose (lignes `#` de tete) des donnees.
func lireGoldenFormes(t *testing.T) (prose, donnees string) {
	t.Helper()
	blob, err := os.ReadFile(cheminGoldenFormes)
	if err != nil {
		t.Fatalf("golden %s illisible : %v — il est VERSIONNE, son absence est une erreur",
			cheminGoldenFormes, err)
	}
	lignes := strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n")
	i := 0
	for i < len(lignes) && (strings.HasPrefix(lignes[i], "#") || lignes[i] == "") {
		i++
	}
	return strings.TrimRight(strings.Join(lignes[:i], "\n"), "\n"), strings.Join(lignes[i:], "\n")
}

// premiereLigne rend la premiere ligne d un bloc.
func premiereLigne(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// ecartDesFormes rend les lignes qui different, en citant la section ou elles vivent.
func ecartDesFormes(fige, attendu string) string {
	a, b := strings.Split(fige, "\n"), strings.Split(attendu, "\n")
	var out []string
	section := ""
	for i := 0; i < len(a) || i < len(b); i++ {
		ligneA, ligneB := ligneOuVide(a, i), ligneOuVide(b, i)
		if strings.HasPrefix(ligneB, "## ") {
			section = ligneB
		}
		if ligneA == ligneB {
			continue
		}
		out = append(out, fmt.Sprintf("  %s\n    fige   : %q\n    mesure : %q", section, ligneA, ligneB))
		if len(out) == 10 {
			out = append(out, "  (... ecarts suivants non listes)")
			break
		}
	}
	return strings.Join(out, "\n")
}

// ligneOuVide rend la ligne i, ou la chaine vide au-dela.
func ligneOuVide(lignes []string, i int) string {
	if i < len(lignes) {
		return lignes[i]
	}
	return ""
}

// typesExportesDuPaquet rend les noms des types EXPORTES declares par les fichiers de production
// du paquet, tries.
//
// Par `go/parser` et pas par `reflect` : la reflexion ne voit que ce qu on lui nomme — c est-a-dire
// exactement la liste qu on cherche a controler. Seule la SOURCE dit ce que le paquet declare.
func typesExportesDuPaquet(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	paquets, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("analyse du paquet : %v", err)
	}
	var noms []string
	for _, p := range paquets {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				gen, ok := d.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if ok && ts.Name.IsExported() {
						noms = append(noms, ts.Name.Name)
					}
				}
			}
		}
	}
	if len(noms) == 0 {
		t.Fatal("aucun type exporte trouve : le balayage n a pas lu le paquet, ce test ne " +
			"garderait rien")
	}
	sort.Strings(noms)
	return noms
}
