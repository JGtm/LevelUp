package main

// deux_passes_test.go — LES GARDES DU MODE S8, SANS UN OCTET DE FILM.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/replaybuild"
)

// TestVerifierLaBrancheServie : LA GARDE ANTI-EQUIVALENCE-VACUANTE.
//
// Les deux sens comptent. Une passe `faits` qui aurait redecode en silence ferait comparer deux
// decodages — identiques par construction — et rendrait un vert qui ne prouve rien ; une passe
// `film` qui aurait relu les faits ne mesurerait pas le decodage.
func TestVerifierLaBrancheServie(t *testing.T) {
	etape := replaybuild.EtapeRejeuDepuisLesFaits
	cas := []struct {
		nom      string
		passe    string
		booleens map[string]bool
		refuse   bool
	}{
		{"hors mode S8, rien n est exige", "", nil, false},
		{"passe film, decodage servi", passeFilm, map[string]bool{etape: false}, false},
		{"passe faits, rejeu servi", passeFaits, map[string]bool{etape: true}, false},
		{"passe faits, REDECODAGE silencieux", passeFaits, map[string]bool{etape: false}, true},
		{"passe film, rejeu servi malgre le forcage", passeFilm, map[string]bool{etape: true}, true},
		{"etape de branche JAMAIS observee", passeFaits, map[string]bool{}, true},
	}
	for _, c := range cas {
		err := verifierLaBrancheServie(c.passe, c.booleens)
		if (err != nil) != c.refuse {
			t.Errorf("%s : erreur = %v, refus attendu = %v", c.nom, err, c.refuse)
		}
	}
}

// TestEtapeDeBrancheVientDeLaProduction : le nom n est pas recopie.
//
// Une copie du nom d etape ici se serait desynchronisee au premier renommage, et le harnais aurait
// alors compte la branche comme une divergence sur les vingt films — un faux positif sur tout le
// corpus, a chaque passe.
func TestEtapeDeBrancheVientDeLaProduction(t *testing.T) {
	if etapeDeBranche() != replaybuild.EtapeRejeuDepuisLesFaits {
		t.Fatalf("etapeDeBranche = %q, production %q", etapeDeBranche(),
			replaybuild.EtapeRejeuDepuisLesFaits)
	}
	if !contient(etapesAttendues(), etapeDeBranche()) {
		t.Errorf("l etape de branche %q n est pas dans les etapes attendues : le harnais ne la "+
			"verrait pas passer", etapeDeBranche())
	}
}

// TestClasserLesEcartsSepareLaBrancheDuContenu : LES DEUX CLASSES, ET LA BRANCHE HORS DES DEUX.
//
// Le mode de panne que ce test ferme est un FAUX ROUGE de masse : la passe-faits n emet aucune des
// quarante etapes du balayage (elle ne le rejoue pas — c est l objet du lot), et les compter comme
// divergences faisait dire au bilan « 0 identique a l octet » alors que les dix films rendaient le
// MEME artefact. Le mode de panne symetrique — avaler un ecart reel avec les absences — est ferme
// par les deux derniers cas.
func TestClasserLesEcartsSepareLaBrancheDuContenu(t *testing.T) {
	memeChose := func(v string) map[string]string {
		out := map[string]string{}
		for _, e := range etapesAttendues() {
			out[e] = v
		}
		return out
	}
	a, b := memeChose("1 aaa"), memeChose("1 aaa")
	// La branche differe TOUJOURS entre les deux passes : c est son travail.
	b[etapeDeBranche()] = "1 bbb"
	if absences, got := classerLesEcarts(a, b); len(got) != 0 || absences != 0 {
		t.Errorf("l etape de branche est comptee : absences=%d ecarts=%v", absences, got)
	}
	// LA PASSE-FAITS REELLE : aucune etape du balayage. Rien a classer, tout a compter.
	faits := memeChose("1 aaa")
	for _, e := range replay.BuildFromFilmSteps {
		delete(faits, e)
	}
	absences, got := classerLesEcarts(a, faits)
	if len(got) != 0 {
		t.Errorf("les etapes du balayage sont classees comme ecarts : %v", got)
	}
	if absences != len(replay.BuildFromFilmSteps) {
		t.Errorf("absences de branche = %d, attendu %d", absences, len(replay.BuildFromFilmSteps))
	}
	// UN ECART DE CONTENU ne se dilue pas dans les absences.
	faits["killsource"] = "1 ccc"
	if _, got := classerLesEcarts(a, faits); len(got) != 1 || got[0] != "killsource" {
		t.Errorf("ecarts = %v, attendu [killsource]", got)
	}
	// UNE ABSENCE HORS BALAYAGE n est pas expliquee par la branche : elle se classe.
	horsBalayage := memeChose("1 aaa")
	delete(horsBalayage, "zones")
	if absences, got := classerLesEcarts(a, horsBalayage); absences != 0 ||
		!contient(got, "zones(absente d une passe)") {
		t.Errorf("absence hors balayage : absences=%d ecarts=%v", absences, got)
	}
	// UNE ABSENCE DU COTE FILM non plus, meme sur une etape du balayage : le decodage DOIT les
	// emettre, et son silence est une anomalie, pas une definition.
	if absences, got := classerLesEcarts(faits, a); absences != 0 || len(got) == 0 {
		t.Errorf("absence du cote FILM : absences=%d ecarts=%v", absences, got)
	}
}

// TestDeuxPassesEtUpdateSontExclusifs : le mode S8 ne re-fige rien, et ne peut pas etre confondu.
//
// C est le point du mode : le regime ordinaire compare a une reference FIGEE, et un fige se
// regenere par accident (« -update pour voir »). Ici il n y a rien a regenerer, et la ligne de
// commande le refuse explicitement plutot que d ignorer le drapeau.
func TestDeuxPassesEtUpdateSontExclusifs(t *testing.T) {
	o := options{repoRoot: "C:/depot", deuxPasses: true, update: true}
	if err := valider(o); err == nil {
		t.Fatal("-deux-passes -update accepte : le mode S8 ne lit et n ecrit AUCUNE reference")
	} else if !strings.Contains(err.Error(), "exclusifs") {
		t.Errorf("le refus doit dire POURQUOI ; obtenu : %v", err)
	}
	if err := valider(options{repoRoot: "C:/depot", passe: "autre"}); err == nil {
		t.Error("une passe inconnue est acceptee : le harnais jouerait une branche au hasard")
	}
	for _, p := range []string{"", passeFilm, passeFaits} {
		if err := valider(options{repoRoot: "C:/depot", passe: p}); err != nil {
			t.Errorf("passe %q refusee : %v", p, err)
		}
	}
}

func contient(liste []string, v string) bool {
	for _, s := range liste {
		if s == v {
			return true
		}
	}
	return false
}

// TestModeS8NePeutPasEcrireUneReference : LE MODE S8 NE TOUCHE JAMAIS LES 20 TSV DE REFERENCE.
//
// # CE QU IL GARDE, ET POURQUOI PAR LA SOURCE
//
// Les references du regime ordinaire (`film/replay/testdata/equivalence/<short8>.tsv`) sont
// l ORACLE de tout le chantier : les re-figer par accident est le mode de panne le plus couteux du
// harnais, parce qu il ne se voit pas — un oracle re-fige compare pour toujours du faux a du faux.
// Le mode S8 n a AUCUNE raison d y ecrire (il compare deux passes du meme commit), et ce test
// l etablit de deux facons independantes :
//
//  1. LE CHEMIN. Le seul `os.WriteFile` vers le dossier des references vit dans `traiterFilm`
//     (`parent.go`), sous `if p.update`. Ni `parentDeuxPasses` ni ses fonctions n appellent
//     `traiterFilm`, ne construisent une `passe` et ne citent `dossierEquivalence`.
//  2. LA LIGNE DE COMMANDE. `-deux-passes -update` est refuse (cf.
//     `TestDeuxPassesEtUpdateSontExclusifs`), donc meme la branche gardee est hors d atteinte.
//
// Les enfants, eux, ecrivent leur TSV dans le dossier que le PARENT leur designe (`-out`) : un
// dossier temporaire, ou celui de `-out-dir`. Jamais celui des references.
func TestModeS8NePeutPasEcrireUneReference(t *testing.T) {
	src, err := os.ReadFile("deux_passes.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "deux_passes.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	// LES APPELS, PAR AST ET NON PAR GREP : ce fichier CITE `traiterFilm` et `dossierEquivalence`
	// dans son en-tete — c est ainsi qu il explique la regle — et un grep rougirait sur sa propre
	// documentation.
	interdits := map[string]string{
		"traiterFilm":        "la seule fonction qui ECRIT une reference (sous `if p.update`)",
		"dossierEquivalence": "le dossier des references ET des faits figes",
		"WriteFile":          "toute ecriture de fichier depuis le parent S8",
	}
	ast.Inspect(f, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var nom string
		switch fn := appel.Fun.(type) {
		case *ast.Ident:
			nom = fn.Name
		case *ast.SelectorExpr:
			nom = fn.Sel.Name
		}
		if raison, interdit := interdits[nom]; interdit {
			t.Errorf("deux_passes.go:%d appelle %q (%s) : le mode S8 ne doit RIEN ecrire dans "+
				"`film/replay/testdata/equivalence/` — ses references sont l oracle du chantier, "+
				"et un oracle re-fige par accident compare du faux a du faux pour toujours.",
				fset.Position(appel.Pos()).Line, nom, raison)
		}
		return true
	})
	// La `passe` (qui PORTE le dossier des references et le drapeau `update`) ne doit pas non plus
	// se construire ici.
	if bytes.Contains(src, []byte("passe{")) {
		t.Error("deux_passes.go construit une `passe` : c est elle qui porte `dir` (le dossier " +
			"des references) et `update`. Le mode S8 n en a pas besoin.")
	}
}
