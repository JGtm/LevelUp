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
// CE QU'IL NE COUVRE PAS, ET C'EST ASSUMÉ : le TypeScript de `apps/web` (hors module Go) et
// les `_test.go`. Les fixtures d'un test portent légitimement +1,0 et -1,0 EXACTEMENT —
// c'est ainsi que les bornes de la classe « à niveau » se testent (weapon_range_test.go).

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
	offenders := scanSeuilsPortee(t)
	if len(offenders) > 0 {
		t.Fatalf("seuil de portée RÉÉCRIT hors de %s : %v.\n"+
			"Utiliser analysis.WeaponRangeMinMeasured (D9) et analysis.WeaponRangeLevelBandM (D4) —"+
			" un seuil produit recopié diverge (règle n°6 du dépôt)", weaponRangeOwner, offenders)
	}
}

// scanSeuilsPortee marche `internal/` et rend les fichiers fautifs. Il échoue tout de suite
// si le propriétaire ne porte plus les constantes : un garde-rail qui ne garde plus rien est
// pire qu'aucun garde-rail.
func scanSeuilsPortee(t *testing.T) []string {
	t.Helper()
	owner, err := os.ReadFile(filepath.Clean(filepath.Join("..", weaponRangeOwner)))
	if err != nil {
		t.Fatalf("lecture du propriétaire %s : %v", weaponRangeOwner, err)
	}
	for _, decl := range []string{"WeaponRangeMinMeasured = 8", "WeaponRangeLevelBandM = 1.0"} {
		if !strings.Contains(string(owner), decl) {
			t.Fatalf("« %s » a disparu de %s : le garde-rail ne vérifie plus rien",
				decl, weaponRangeOwner)
		}
	}
	var offenders []string
	err = filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDirPorteee(d.Name())
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, ".."+string(filepath.Separator)))
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
		t.Fatalf("marche de internal/ : %v", err)
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
