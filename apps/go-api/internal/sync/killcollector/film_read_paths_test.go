package killcollector

// film_read_paths_test.go — LE VERROU D EGALITE entre le decodeur et le vocabulaire partage.
//
// Les voies de lecture d un decodage de film ont DEUX domiciles, et c est structurel :
//
//	games/halo_infinite/film/killsource   `Path` — le TYPE, chez le decodeur qui les produit.
//	                                      Paquet title-specific : ni `persist` ni `migration`
//	                                      ne peuvent l importer.
//	domain/killscope                      les CHAINES, dans une feuille sans import, lisibles
//	                                      par les quatre ecrivains et par la migration.
//
// Ce paquet est le SEUL du depot qui importe les deux. Le verrou vit donc ici — c est exactement
// la promesse que le constat J4R-3 avait trouvee non tenue ailleurs (un commentaire renvoyait a un
// « test d egalite » qui n existait pas), et la dette H3 la reclamait pour ces deux valeurs-la.
//
// CE QUE LA DERIVE COUTERAIT : c est sur `read_path` que se detecte la presence d un
// ENRICHISSEMENT de film. Une copie qui diverge d un caractere ne leve aucune erreur — le
// detecteur cesse simplement de voir les passes de film, et un producteur credit reecrit
// par-dessus sans les relire. Le nombre de lignes ne bouge pas, aucun nom ne change : seule la
// source du degat fatal disparait de la lecture.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/persist"
)

func TestFilmReadPathsEgalesAuDecodeur(t *testing.T) {
	if got, want := killscope.ReadPathFilmWalk, string(killsource.PathWalk); got != want {
		t.Errorf("killscope.ReadPathFilmWalk = %q, decodeur = %q — la detection d un "+
			"enrichissement de film cesserait de voir la voie MARCHE, sans erreur ni compteur",
			got, want)
	}
	if got, want := killscope.ReadPathFilmScan, string(killsource.PathScan); got != want {
		t.Errorf("killscope.ReadPathFilmScan = %q, decodeur = %q — la detection d un "+
			"enrichissement de film cesserait de voir la voie SCAN, sans erreur ni compteur",
			got, want)
	}
}

// TestFilmReadPathsCouvrentToutesLesVoiesDuDecodeur — le verrou d EXHAUSTIVITE.
//
// L egalite deux a deux ne suffit pas : le jour ou le decodeur ajoute une TROISIEME voie, les
// deux valeurs existantes resteraient justes et la liste, elle, serait incomplete — la nouvelle
// voie passerait pour du credit et se ferait ecraser.
//
// CE TEST A ETE TAUTOLOGIQUE, ET C EST L AUDIT DU 2026-08-06 QUI L A ETABLI. Il comparait
// `killscope.FilmReadPaths()` a un littéral `duDecodeur` tenu a la main dans ce fichier :
// DEUX listes, aucune des deux derivee du decodeur. Ajouter `PathHybrid` dans
// `killsource/kill.go` laissait tout vert (cardinaux 2 == 2), et le ratchet archlint voisin
// ne rattrapait rien — sa propre liste est fermee et il EXEMPTE `killsource/`
// (`killScopeOwners`). Le fichier documentait pourtant lui-meme ce que la derive coute.
// Ironie relevee par l audit : ce test se presentait comme la reparation du constat J4R-3 et
// reproduisait le defaut qu il pretendait fermer.
//
// LA LISTE ATTENDUE VIENT DESORMAIS D UNE SOURCE INDEPENDANTE : le TEXTE de `killsource`, lu
// par l analyseur syntaxique de Go. Ce n est pas un detail de mise en oeuvre — un verrou dont
// les deux cotes sont ecrits par la meme main ne verrouille rien. La reflexion Go ne convient
// pas ici : les constantes n existent plus a l execution, seul le source les enumere.
func TestFilmReadPathsCouvrentToutesLesVoiesDuDecodeur(t *testing.T) {
	duDecodeur := voiesDeclareesParLeDecodeur(t)
	if len(duDecodeur) == 0 {
		t.Fatal("aucune constante de type `Path` trouvee dans le source de killsource — le " +
			"verrou ne verrouille plus rien : le type a ete renomme, ou les voies ont change de " +
			"forme de declaration")
	}
	// Temoin d auto-verification : les deux voies historiques DOIVENT etre retrouvees par le
	// scan. Sans ce controle, un scan qui ne trouverait plus rien de pertinent passerait pour
	// un decodeur sans voie.
	for _, attendue := range []killsource.Path{killsource.PathWalk, killsource.PathScan} {
		if _, ok := duDecodeur[string(attendue)]; !ok {
			t.Fatalf("la voie %q existe dans le decodeur mais le scan du source ne la voit pas — "+
				"c est le SCAN qui est casse, pas le decodeur", attendue)
		}
	}

	partagees := map[string]bool{}
	for _, p := range killscope.FilmReadPaths() {
		partagees[p] = true
	}
	for valeur, nom := range duDecodeur {
		if !partagees[valeur] {
			t.Errorf("voie %s = %q declaree par le decodeur, ABSENTE de killscope.FilmReadPaths() "+
				"— une voie hors de cette liste passe pour du credit et se fait ecraser par le "+
				"producteur suivant, sans erreur ni compteur", nom, valeur)
		}
	}
	if len(partagees) != len(duDecodeur) {
		t.Errorf("%d voie(s) partagees pour %d declaree(s) par le decodeur — killscope porte une "+
			"voie que le decodeur ne produit plus, ou l inverse", len(partagees), len(duDecodeur))
	}
	// `persist.FilmReadPaths` n est plus une copie : il LIT la liste partagee. Le verifier ici
	// interdit qu il redevienne un litteral.
	if len(persist.FilmReadPaths) != len(partagees) {
		t.Errorf("persist.FilmReadPaths porte %d voie(s) pour %d partagees — la copie est revenue",
			len(persist.FilmReadPaths), len(partagees))
	}
}

// voiesDeclareesParLeDecodeur lit le SOURCE de `killsource` et rend, par valeur, le nom de
// chaque constante de type `Path`. C est la source independante du verrou ci-dessus.
func voiesDeclareesParLeDecodeur(t *testing.T) map[string]string {
	t.Helper()
	dir := sourceDuDecodeur(t)
	fset := token.NewFileSet()
	paquets, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("analyse du source de killsource (%s) : %v", dir, err)
	}
	voies := map[string]string{}
	for _, paquet := range paquets {
		for _, fichier := range paquet.Files {
			collecterVoies(t, fset, fichier, voies)
		}
	}
	return voies
}

// collecterVoies parcourt les blocs `const` d un fichier et retient les constantes de type
// `Path`. Une constante chaine declaree dans un bloc qui manipule `Path` mais SANS type
// explicite fait echouer le test : c est le seul cas ou ce scan pourrait rater une voie, et
// il vaut mieux exiger une declaration explicite que rater silencieusement.
func collecterVoies(t *testing.T, fset *token.FileSet, fichier *ast.File, voies map[string]string) {
	t.Helper()
	for _, decl := range fichier.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		blocTypePath := false
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if id, ok := vs.Type.(*ast.Ident); ok && id.Name == nomDuTypeVoie {
				blocTypePath = true
				ajouterVoies(t, fset, vs, voies)
				continue
			}
			if blocTypePath && vs.Type == nil {
				t.Fatalf("%s : la constante %v est declaree sans type dans un bloc de voies `%s` — "+
					"le scan ne peut pas trancher si c en est une. Declarer le type explicitement",
					fset.Position(vs.Pos()), nomsDe(vs), nomDuTypeVoie)
			}
		}
	}
}

// ajouterVoies enregistre les valeurs litterales d une declaration de voies.
func ajouterVoies(t *testing.T, fset *token.FileSet, vs *ast.ValueSpec, voies map[string]string) {
	t.Helper()
	for i, nom := range vs.Names {
		if i >= len(vs.Values) {
			t.Fatalf("%s : la voie %s n a pas de valeur litterale", fset.Position(vs.Pos()), nom.Name)
		}
		lit, ok := vs.Values[i].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			t.Fatalf("%s : la voie %s n est pas une chaine litterale — le scan ne peut pas en "+
				"deriver la valeur de fil", fset.Position(vs.Pos()), nom.Name)
		}
		valeur, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("%s : valeur de %s illisible : %v", fset.Position(vs.Pos()), nom.Name, err)
		}
		voies[valeur] = nom.Name
	}
}

// nomsDe : les noms declares par un spec, pour un message d erreur lisible.
func nomsDe(vs *ast.ValueSpec) []string {
	out := make([]string, 0, len(vs.Names))
	for _, n := range vs.Names {
		out = append(out, n.Name)
	}
	return out
}

// nomDuTypeVoie : le type dont les constantes sont des voies de lecture. Le renommer sans
// toucher ici fait echouer le test (plus aucune voie trouvee), ce qui est le comportement
// voulu — c est une modification qui doit se voir.
const nomDuTypeVoie = "Path"

// cheminSourceDecodeur : le paquet du decodeur, RELATIF a ce fichier de test.
const cheminSourceDecodeur = "../../games/halo_infinite/film/killsource"

// sourceDuDecodeur : le repertoire du source de `killsource`, resolu depuis l emplacement de
// CE fichier — pas depuis le repertoire de travail, qui varie selon l invocation.
func sourceDuDecodeur(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue : impossible de localiser le source du decodeur")
	}
	dir := filepath.Join(filepath.Dir(ici), cheminSourceDecodeur)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("source du decodeur introuvable en %s (%v) — le paquet a demenage : mettre a "+
			"jour cheminSourceDecodeur, sinon ce verrou ne verrouille plus rien", dir, err)
	}
	return dir
}
