// no_identity_bridge_outside_registry_test.go — AUCUN CALQUE NE RECONSTRUIT SON PROPRE PONT.
//
// # CE QUE CE GARDE-RAIL EMPÊCHE DE RÉÉCRIRE
//
// Avant le lot P2 (2026-09-08), quinze calques du rejeu et deux lecteurs hors rejeu lisaient
// `OwnerReport.SlotXUID` / `.Owner` ou appelaient `buildOwners` / `ResolveSlotXUID` directement.
// Chacun avait sa propre garde contre les slots ambigus — ou pas : `xuidOfPublishedTrack` servait
// le PREMIER occupant d'un slot partagé (constat C2 de la revue VIES-R1), `EntreeContexteMorts`
// avait dû corriger le même défaut de son côté (P0-2), et le collecteur de sync avait sa propre
// entrée exportée. Le même fait — « qui occupe ce slot » — se répondait donc de trois façons.
//
// LA RÈGLE POSÉE : `internal/analysis/replay/identity_registry.go` est le SEUL fichier autorisé à
// toucher les tables brutes du pont. Tout le reste passe par les accesseurs du registre, qui
// portent déjà leurs gardes (`PontEpure` retire les slots ambigus, `XUIDAt` préfère la vie qui
// couvre l'instant). Un lecteur ne peut plus oublier une garde : il n'a plus de quoi l'enfreindre.
//
// # POURQUOI CES MOTIFS-LÀ, ET PAS `.Owner`
//
// `.Owner` est un nom de champ TROP COMMUN dans ce paquet : `EquipmentPlacement.Owner` (le poseur
// d'un équipement) et `ZoneSpan.Owner` (le camp qui tient une zone) n'ont aucun rapport avec le
// pont. Le motif retenu est donc `.own.` — le nom du champ PRIVÉ du registre, qui ne désigne que
// lui — plus le TYPE `OwnerReport` en signature de fonction, plus les trois portes historiques.
// Mesuré au HEAD du lot : `.own.` n'apparaît nulle part ailleurs, et les trois portes ont disparu
// de tous les appelants.
//
// # ALLOWLIST : UNE SEULE ENTRÉE, DATÉE
//
// `identity_registry.go`, le registre lui-même. Toute autre entrée exige une justification écrite
// et datée ici — c'est ce qui a manqué au prédicat bot, passé de 8 à 36 copies après une
// centralisation sans garde-rail (règle 6 du CLAUDE.md).
//
// LES `_test.go` NE SONT PAS BALAYÉS, et c'est délibéré : les tests de calque montent des ponts
// synthétiques (`regDe(OwnerReport{...})`) pour exercer un cas que le film ne produit pas à la
// demande. Ils ne sont pas du code de production, et les forcer à passer par un film complet
// remplacerait un test précis par un test lent et vague.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// identityBridgeScope : les paquets où le pont d'identité circule — la cuisson du rejeu et le
// collecteur de sync, les deux producteurs que la décision D11 autorise.
var identityBridgeScope = []string{
	filepath.Join("analysis", "replay"),
	filepath.Join("sync", "killcollector"),
}

// identityBridgeAllowlist : les fichiers autorisés à toucher les tables brutes du pont.
//
//	identity_registry.go             2026-09-08, lot P2 — LE REGISTRE. Type, construction,
//	                                 accesseurs. Seul producteur.
//	identity_registry_mutations.go   2026-09-08, lot P2-bis — LES POSEURS, extraits du fichier
//	                                 ci-dessus qui atteignait 471 L pour un seuil de 500
//	                                 (découverte 3 du lot P2) au moment d'ajouter une seconde
//	                                 voie de déduction. Ce sont les deux MOITIÉS du même
//	                                 producteur, pas une seconde porte : aucune RÈGLE de nommage
//	                                 n'y vit — les décideurs (`_elimination.go`, `_exclusion.go`)
//	                                 restent hors allowlist et passent par ces poseurs.
var identityBridgeAllowlist = map[string]string{
	"identity_registry.go":           "2026-09-08, lot P2 — le registre, seul producteur du pont",
	"identity_registry_mutations.go": "2026-09-08, lot P2-bis — les poseurs du registre",
}

// identityBridgePatterns : les motifs interdits hors de l'allowlist, avec ce qu'ils protègent.
var identityBridgePatterns = []struct {
	re  *regexp.Regexp
	dit string
}{
	{regexp.MustCompile(`\.own\.`),
		"lecture directe des tables brutes du pont (`IdentityRegistry.own`)"},
	{regexp.MustCompile(`\bfunc\b[^\n]*\bOwnerReport\b`),
		"`OwnerReport` en signature de fonction — les calques prennent `IdentityRegistry`"},
	{regexp.MustCompile(`\bbuildOwners\s*\(`),
		"appel à `buildOwners` — seul le registre construit le pont"},
	{regexp.MustCompile(`\bResolveSlotXUID\s*\(`),
		"appel à `ResolveSlotXUID` — porte supprimée au lot P2 (`BuildIdentityRegistry`)"},
	{regexp.MustCompile(`\.NamingBridge\s*\(`),
		"appel à `NamingBridge` — l'accesseur du registre est `PontEpure`"},
	{regexp.MustCompile(`\.SlotXUID\b`),
		"lecture de `SlotXUID` — l'accesseur du registre est `PontParSlot`"},
}

func TestNoIdentityBridgeOutsideRegistry(t *testing.T) {
	_, thisFile, ok := callerFile()
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal

	var violations []string
	for _, scope := range identityBridgeScope {
		root := filepath.Join(internalRoot, scope)
		if _, err := os.Stat(root); err != nil {
			// Le paquet peut avoir été déplacé (le décodeur doit migrer sous
			// `games/halo_infinite/film/` après v7.5.0) : ce n'est pas l'objet de ce test.
			continue
		}
		if err := filepath.Walk(root, visiteurDuPont(&violations)); err != nil {
			t.Fatalf("balayage de %s : %v", scope, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("le pont d'identité est reconstruit hors du registre :\n  %s\n\n"+
			"Le registre (`internal/analysis/replay/identity_registry.go`) est le SEUL producteur : "+
			"passez par ses accesseurs (`PontEpure`, `PontParSlot`, `XUIDAt`, `IndexParSlot`, "+
			"`Vies`, `SanteDuPont`). Ajouter une entrée à l'allowlist exige une justification "+
			"écrite et datée dans ce fichier.", strings.Join(violations, "\n  "))
	}
}

// visiteurDuPont rend la fonction de parcours : elle lit chaque `.go` de production et relève
// les motifs interdits, commentaires exclus.
func visiteurDuPont(violations *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") {
			return nil
		}
		if _, autorise := identityBridgeAllowlist[base]; autorise {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(src), "\n") {
			// LES COMMENTAIRES ONT LE DROIT DE NOMMER LE PONT : c'est ainsi qu'ils expliquent
			// pourquoi il a changé de porte. Seul le code est interdit.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, p := range identityBridgePatterns {
				if p.re.MatchString(line) {
					*violations = append(*violations,
						filepath.Base(filepath.Dir(path))+"/"+base+":"+itoa(i+1)+" — "+p.dit)
				}
			}
		}
		return nil
	}
}

// callerFile isole `runtime.Caller` pour que le test lise comme sa règle.
func callerFile() (int, string, bool) {
	_, file, line, ok := runtime.Caller(1)
	return line, file, ok
}
