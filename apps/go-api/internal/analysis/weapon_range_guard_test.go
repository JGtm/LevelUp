package analysis

// weapon_range_guard_test.go — LES DEUX SEUILS DE LA PORTÉE NE S'ÉCRIVENT QU'UNE FOIS.
//
// POURQUOI CE GARDE-RAIL. `WeaponRangeMinMeasured` (8 mesures) et `WeaponRangeLevelBandM`
// (1,0 m) sont des RÈGLES PRODUIT, pas des détails de calcul : le premier décide ce que la
// section publie, le second décide ce que « d'en haut » veut dire. Ils traversent quatre
// couches (analysis, repo DuckDB, service, web). La règle n°6 du dépôt le dit sans
// ambiguïté : à la troisième copie on centralise ET on pose un garde-rail — sans quoi la
// dette re-croît (leçon chiffrée : prédicat bot passé de 8 à 36 copies APRÈS
// centralisation). Ici on pose le garde-rail AVANT la deuxième copie.
//
// CE QU'IL COUVRE : tout le Go sous `internal/`, pas seulement ce paquet — la copie qui
// divergerait viendrait du repo DuckDB (un `WHERE kp.killer_z - kp.victim_z > 1.0` écrit en
// SQL) ou du service, jamais d'ici.
//
// IL PORTE SON PROPRE CONTRÔLE POSITIF (`TestSeuilsPorteeDetecteUneCopie`). Un garde-rail qui
// n'affirme que « zéro fautif » est vrai aussi d'un détecteur mort : la mutation
// `if false && fautifPortee(...)` le laissait vert (revue du 2026-09-06). Le détecteur prend
// donc sa RACINE en paramètre, et le contrôle positif lui plante de vraies copies dans un
// répertoire temporaire.
//
// CE QU'IL NE CAPTE PAS, ET C'EST CONSIGNÉ PLUTÔT QUE CORRIGÉ — les motifs sont des regex de
// LIGNE, elles ne comprennent pas le SQL :
//
//   - une requête MULTILIGNE qui coupe entre la colonne et sa comparaison
//     (`... kp.killer_z - kp.victim_z\n    > 1.0 ...`) : les 80 caractères de contexte ne
//     franchissent pas le retour à la ligne ;
//   - un ALIAS qui masque la colonne (`SELECT killer_z - victim_z AS dz ... WHERE dz > 1.0`) —
//     le motif `delta_?z` ne connaît pas `dz`, et l'élargir à deux lettres ferait tomber la
//     moitié du dépôt ;
//   - le seuil de publication écrit sans son nom (`HAVING count(*) >= 8`).
//
// Ces trois trous sont ACCEPTÉS parce que le lot 3 a pour consigne de n'écrire AUCUN seuil ni
// comparaison de dénivelé en SQL : le repo rend les frags MESURÉS, l'agrégat et ses seuils
// restent ici. Le jour où cette consigne bougerait, ce commentaire dit ce qu'il faudrait
// renforcer — et un garde-rail de nommage (`kill_measured.go`, lot 3.2) le couvrira mieux
// qu'une regex de littéral.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// weaponRangeOwner : le fichier qui a le droit d'écrire les deux seuils, vu depuis la
// racine de la marche (`..` = `internal/`).
const weaponRangeOwner = "analysis/weapon_range.go"

// racineInterne : la racine marchée par le garde-rail réel.
const racineInterne = ".."

// bandeDeniveleInline matche une COMPARAISON entre un dénivelé et un littéral d'un mètre,
// dans les deux sens d'écriture. Le motif vise la comparaison et non la mention : une DDL
// qui déclare `killer_z DOUBLE` ou un commentaire qui parle du dénivelé ne sont pas des
// copies du seuil.
var bandeDeniveleInline = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(killer_z|victim_z|delta_?z)[^\n]{0,80}[<>]=?\s*-?\s*1(\.0+)?\b`),
	regexp.MustCompile(`(?i)-?\s*1(\.0+)?\s*[<>]=?[^\n]{0,80}(killer_z|victim_z|delta_?z)`),
}

// seuilPublicationInline matche un seuil de publication de 8 écrit à côté d'un
// « minMeasured » / « min_measured » ailleurs que chez le propriétaire.
var seuilPublicationInline = regexp.MustCompile(`(?i)min_?measured[^\n]{0,60}\b8\b`)

// TestSeuilsPorteeDefinisUneSeuleFois : les deux constantes vivent dans weapon_range.go, et
// aucun autre fichier Go de `internal/` ne réécrit leur littéral.
func TestSeuilsPorteeDefinisUneSeuleFois(t *testing.T) {
	verifierProprietairePortee(t)
	offenders := scanSeuilsPortee(t, racineInterne)
	if len(offenders) > 0 {
		t.Fatalf("seuil de portée RÉÉCRIT hors de %s : %v.\n"+
			"Utiliser analysis.WeaponRangeMinMeasured (D9) et analysis.WeaponRangeLevelBandM (D4) —"+
			" un seuil produit recopié diverge (règle n°6 du dépôt)", weaponRangeOwner, offenders)
	}
}

// TestSeuilsPorteeDetecteUneCopie est le CONTRÔLE POSITIF : sans lui, le test ci-dessus reste
// vert même si le détecteur ne détecte plus rien. Chacun des deux littéraux interdits est
// planté dans un fichier `.go` non-test d'un répertoire temporaire, qui doit sortir fautif ;
// un troisième fichier, licite, ne doit PAS sortir — sinon le garde-rail crierait sur des
// DDL et des commentaires, et finirait désactivé.
//
// Le répertoire est celui de `t.TempDir()`, supprimé par le framework à la fin du test : rien
// n'est écrit dans l'arbre du dépôt, donc rien ne peut y rester si le test échoue en cours de
// route.
func TestSeuilsPorteeDetecteUneCopie(t *testing.T) {
	racine := t.TempDir()
	fautifs := map[string]string{
		"copie_denivele.go": "package faux\n\nfunc f(row struct{ deltaZ float64 }) bool {\n" +
			"\treturn row.deltaZ > 1.0\n}\n",
		"copie_seuil.go": "package faux\n\nconst minMeasured = 8\n",
	}
	licite := "ddl_innocente.go"
	contenus := map[string]string{
		licite: "package faux\n\n// Le dénivelé se lit chez analysis, pas ici.\n" +
			"const ddl = `CREATE TABLE kill_positions (killer_z DOUBLE, victim_z DOUBLE)`\n",
	}
	for nom, src := range fautifs {
		contenus[nom] = src
	}
	for nom, src := range contenus {
		if err := os.WriteFile(filepath.Join(racine, nom), []byte(src), 0o600); err != nil {
			t.Fatalf("écriture de la copie %s : %v", nom, err)
		}
	}

	vus := map[string]bool{}
	for _, f := range scanSeuilsPortee(t, racine) {
		vus[f] = true
	}
	for nom := range fautifs {
		if !vus[nom] {
			t.Errorf("copie NON DÉTECTÉE : %s — le détecteur ne détecte plus rien", nom)
		}
	}
	if vus[licite] {
		t.Errorf("%s est licite (DDL + commentaire) et ne doit pas être signalé", licite)
	}
}

// verifierProprietairePortee échoue tout de suite si le propriétaire ne porte plus les
// constantes : un garde-rail qui ne garde plus rien est pire qu'aucun garde-rail.
func verifierProprietairePortee(t *testing.T) {
	t.Helper()
	owner, err := os.ReadFile(filepath.Clean(filepath.Join(racineInterne, weaponRangeOwner)))
	if err != nil {
		t.Fatalf("lecture du propriétaire %s : %v", weaponRangeOwner, err)
	}
	for _, decl := range []string{"WeaponRangeMinMeasured = 8", "WeaponRangeLevelBandM = 1.0"} {
		if !strings.Contains(string(owner), decl) {
			t.Fatalf("« %s » a disparu de %s : le garde-rail ne vérifie plus rien",
				decl, weaponRangeOwner)
		}
	}
}

// scanSeuilsPortee marche la racine donnée et rend les fichiers fautifs, en chemin RELATIF à
// cette racine. La racine est un paramètre pour que le contrôle positif puisse lui soumettre
// de vraies copies sans les écrire dans le dépôt.
func scanSeuilsPortee(t *testing.T, racine string) []string {
	t.Helper()
	var offenders []string
	err := filepath.WalkDir(racine, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDirPorteee(d.Name())
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, racine+string(filepath.Separator)))
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") ||
			rel == weaponRangeOwner {
			return nil
		}
		if fautifPortee(t, path) {
			offenders = append(offenders, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("marche de %s : %v", racine, err)
	}
	return offenders
}

// skipDirPorteee écarte ce qui n'est pas du code du dépôt.
func skipDirPorteee(name string) error {
	if name == "vendor" || name == "testdata" || name == "node_modules" {
		return fs.SkipDir
	}
	return nil
}

// fautifPortee dit si le fichier réécrit l'un des deux seuils.
func fautifPortee(t *testing.T, path string) bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("lecture %s : %v", path, err)
	}
	if seuilPublicationInline.Match(raw) {
		return true
	}
	for _, re := range bandeDeniveleInline {
		if re.Match(raw) {
			return true
		}
	}
	return false
}
