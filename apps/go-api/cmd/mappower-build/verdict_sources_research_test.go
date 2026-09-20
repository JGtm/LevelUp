//go:build research

package main

// verdict_sources_research_test.go — CE QUE LE VERDICT LIT (etape 2, 2026-09-20).
//
// Trois sources, trois fichiers, aucune base de donnees :
//
//  1. L'ORACLE — `.ai/V7.5/positions_de_force/ORACLE_PRO_2026-09-20.md`, sections 3 (table
//     finale, une ligne par position attendue) et 3.1 (contre-exemples, les lieux que les
//     guides DECONSEILLENT). Le document est la piece, ce fichier n'en est que le lecteur.
//  2. LES POSITIONS CALCULEES — `mesures_2026-09-20/positions.json`, ecrit par la passe
//     `--mesure` avec le reglage FIGE. Le type est celui du producteur (`SortiePositions`) :
//     le verdict ne redefinit pas la forme de ce qu'il juge.
//  3. LES ZONES NOMMEES — `data/titles/{slug}/reference/map_callouts.json`, par la MEME
//     cascade que le service et que les planches de controle : module installe d'abord,
//     map_id ensuite.
//
// AUCUNE LECTURE DE BASE. Le serveur de developpement peut tenir la base partagee en
// ecriture ; le verdict n'en a pas besoin, tout ce qu'il juge est deja sur disque.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// Chemins et variables d'environnement du verdict.
const (
	// verdictDataRootEnv designe la racine qui CONTIENT `data/` (le depot de travail n'en a
	// pas : les bases et les references vivent dans le depot principal). Sans elle, le test
	// SAUTE — meme regime que les autres instruments de recherche du depot.
	verdictDataRootEnv = "MAPPOWER_DATA_ROOT"
	verdictDossier     = ".ai/V7.5/positions_de_force"
	verdictMesures     = "mesures_2026-09-20"
	verdictOracleNom   = "ORACLE_PRO_2026-09-20.md"
	verdictSortieNom   = "VERDICT_ORACLE_2026-09-20.md"
	verdictTitleSlug   = "halo_infinite"
)

// Confiances de l'oracle.
const (
	confianceForte  = "forte"
	confianceFaible = "faible"
)

// zoneInconnue est la valeur de `zone_en` d'une ligne que l'oracle n'a pas su rattacher au
// vocabulaire du depot. Regle de lecture ecrite en tete de la section 3 de l'oracle : ces
// lignes ne comptent NI au rappel NI a la precision.
const zoneInconnue = "?"

// ligneOracle est une ligne des tables 3 ou 3.1.
type ligneOracle struct {
	CarteCle  string
	ZoneEN    string
	NomGuide  string
	Confiance string
	Raison    string
}

// armeSeule dit si la seule raison invoquee est la presence d'une arme de puissance.
// Reserve du pilote (section 7 de l'oracle) : ce sont des emplacements d'ARME, pas
// forcement des positions TENABLES — les manquer n'est pas la meme faute que manquer une
// hauteur, et le verdict les rapporte a part.
func (l ligneOracle) armeSeule() bool { return strings.TrimSpace(l.Raison) == "arme" }

// racineDepot rend la racine du depot de travail (celle qui porte `.ai/`).
func racineDepot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("repertoire courant illisible : %v", err)
	}
	for dir := wd; ; {
		if _, err := os.Stat(filepath.Join(dir, verdictDossier)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("dossier %s introuvable au-dessus de %s", verdictDossier, wd)
		}
		dir = parent
	}
}

// marqueursOracle delimite, dans un document d'oracle, la table des positions et celle des
// contre-exemples : le titre qui ouvre chacune, et celui qui ferme la seconde. L'oracle v1
// (sections 3 / 3.1 / 4) et l'oracle v2 (sections 6 / 7 / 8) ont la meme table a des
// numeros differents — un seul lecteur, deux jeux de marqueurs.
type marqueursOracle struct {
	DebutPositions string
	DebutPieges    string
	Fin            string
}

// marqueursOracleV1 : les sections de l'oracle v1 (`ORACLE_PRO_2026-09-20.md`).
var marqueursOracleV1 = marqueursOracle{
	DebutPositions: "## 3. TABLE FINALE", DebutPieges: "### 3.1", Fin: "## 4.",
}

// litOracle extrait les lignes des deux tables de l'oracle v1.
func litOracle(t *testing.T, chemin string) (positions, contreExemples []ligneOracle) {
	t.Helper()
	return litOracleEntre(t, chemin, marqueursOracleV1)
}

// litOracleEntre extrait les lignes des deux tables d'un oracle, entre ses marqueurs.
func litOracleEntre(t *testing.T, chemin string, m marqueursOracle) (positions, contreExemples []ligneOracle) {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("oracle illisible (%s) : %v", chemin, err)
	}
	section := ""
	for _, brute := range strings.Split(string(blob), "\n") {
		ligne := strings.TrimSpace(brute)
		switch {
		case strings.HasPrefix(ligne, m.DebutPositions):
			section = "positions"
			continue
		case strings.HasPrefix(ligne, m.DebutPieges):
			section = "pieges"
			continue
		case strings.HasPrefix(ligne, m.Fin):
			section = ""
			continue
		}
		l, ok := decoupeLigne(ligne)
		if !ok {
			continue
		}
		switch section {
		case "positions":
			positions = append(positions, l)
		case "pieges":
			contreExemples = append(contreExemples, l)
		}
	}
	if len(positions) == 0 || len(contreExemples) == 0 {
		t.Fatalf("oracle non parsable (%s) : %d positions, %d contre-exemples",
			chemin, len(positions), len(contreExemples))
	}
	return positions, contreExemples
}

// decoupeLigne rend la ligne de table si c'en est une (ni en-tete ni filet). Six colonnes
// dans l'oracle v1 ; sept dans l'oracle v2, qui ajoute `v1v2` et `sources` — deux colonnes
// de tracabilite que le verdict ne lit pas.
func decoupeLigne(ligne string) (ligneOracle, bool) {
	if !strings.HasPrefix(ligne, "|") || strings.Contains(ligne, "---") {
		return ligneOracle{}, false
	}
	champs := strings.Split(strings.Trim(ligne, "|"), "|")
	if len(champs) != 6 && len(champs) != 7 {
		return ligneOracle{}, false
	}
	for i := range champs {
		champs[i] = strings.TrimSpace(champs[i])
	}
	if champs[0] == "carte_cle" {
		return ligneOracle{}, false
	}
	return ligneOracle{CarteCle: champs[0], ZoneEN: champs[1], NomGuide: champs[2],
		Confiance: champs[3], Raison: champs[4]}, true
}

// litPositions lit la sortie de la passe de mesure et refuse une autre version de forme.
func litPositions(t *testing.T, chemin string) SortiePositions {
	t.Helper()
	doc, err := LitPositionsJSON(chemin)
	if err != nil {
		t.Fatalf("%v\nproduire d'abord : go run ./cmd/mappower-build --mesure ... (ou --fusion, ou mapgeo-build)", err)
	}
	return doc
}

// zonesDeCarte resout les zones nommees d'une carte mesuree par la cascade du service :
// module installe d'abord, puis chaque map_id du corpus (le dominant en premier).
//
// LE MODULE D'UNE CARTE FORGE EST CELUI DE SON CANEVAS (`fo11_blank` porte Solitude ET
// Empyrean) : il n'identifie pas la carte, et le catalogue ne lui attribue aucune zone. La
// cascade s'en sort d'elle-meme — le lookup par module echoue et le map_id tranche — mais il
// fallait l'ecrire, sinon la prochaine lecture croira le module fiable.
func zonesDeCarte(cat *replay.MapCalloutsCatalog, c SortieCartePos) []replay.CalloutZone {
	if e, err := cat.Lookup(c.Module); err == nil && len(e.Zones) > 0 {
		return e.Zones
	}
	for _, id := range append([]string{c.MapIDDominant}, c.MapIDs...) {
		if e, err := cat.LookupByID(id); err == nil && len(e.Zones) > 0 {
			return e.Zones
		}
	}
	return nil
}

// clesDe rend les identifiants sous lesquels l'oracle peut designer une carte mesuree.
func clesDe(c SortieCartePos) []string {
	cles := append([]string{}, c.MapIDs...)
	if c.Module != "" {
		cles = append(cles, c.Module)
	}
	return cles
}

// surfaceDeZone assemble la surface d'une zone nommee (contour, morceaux, trous).
func surfaceDeZone(z replay.CalloutZone) surface {
	anneaux := [][][2]float64{}
	if len(z.Polygon) >= 3 {
		anneaux = append(anneaux, z.Polygon)
	}
	anneaux = append(anneaux, z.Parts...)
	anneaux = append(anneaux, z.Holes...)
	return nouvelleSurface(anneaux...)
}

// nomsTries rend les libelles EN distincts et non vides des zones d'une carte, tries.
// C'est la LISTE DE REFERENCE du temoin negatif (permutation circulaire) : elle doit etre
// deterministe, sinon le temoin n'est pas rejouable.
func nomsTries(zones []replay.CalloutZone) []string {
	vus := map[string]bool{}
	var out []string
	for _, z := range zones {
		nom := strings.TrimSpace(z.EN)
		if nom == "" || vus[strings.ToLower(nom)] {
			continue
		}
		vus[strings.ToLower(nom)] = true
		out = append(out, nom)
	}
	sort.Strings(out)
	return out
}

// suivante rend le nom qui suit `nom` dans la liste, en circulaire. C'est la PERMUTATION du
// temoin negatif : la position attendue en Z_i est reputee en Z_i+1. Rend "" (et donc une
// ligne non exploitable) quand le nom n'est pas dans la liste.
func suivante(noms []string, nom string) string {
	for i, n := range noms {
		if strings.EqualFold(n, nom) {
			return noms[(i+1)%len(noms)]
		}
	}
	return ""
}

// cheminDonnees rend la racine des donnees, ou "" si l'environnement ne la donne pas.
func cheminDonnees() string { return strings.TrimSpace(os.Getenv(verdictDataRootEnv)) }

// pourcent formate une proportion pour les tables du verdict.
func pourcent(v float64) string { return fmt.Sprintf("%.2f", v) }
