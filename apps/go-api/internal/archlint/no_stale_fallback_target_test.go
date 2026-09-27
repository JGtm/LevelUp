package archlint

// no_stale_fallback_target_test.go — UNE CIBLE DE RETRAIT QUI NOMME UN LOT DEJA CLOS EST ROUGE
// (D14 d du plan `.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md`, ADR 0034 ; pose le 2026-09-16 par la
// revue de jalon M1, lentille L3).
//
// # LE DEFAUT QUE CE GARDE-RAIL FERME
//
// Une entree du registre des replis porte une CIBLE DE RETRAIT — le lot du plan qui doit faire
// disparaitre ce repli (regle 11 du depot : pas de repli sans date cible). Quand ce lot est
// fusionne SANS avoir retire le repli, la cible devient un mensonge : elle designe un travail
// deja fait, donc elle ne designe plus rien, et le repli survit sans echeance. Le registre
// continue pourtant de se lire comme si tout etait sous controle.
//
// CONSTAT QUI L'A FAIT NAITRE (revue de jalon M1) : huit entrees nommaient le lot 1.9.13 en
// cible — de retrait ou de comptage — alors qu'il etait fusionne depuis le 2026-09-15. Trois
// d'entre elles decidaient un fait publie en publiant 0 dans `coverage.fallbacks`, parce que
// leur compteur devait etre cable « au lot 1.9.13 ». Deux autres entrees nommaient de la meme
// facon les lots 1.9.3 et 1.6.3.
//
// # LA REGLE, ET POURQUOI ELLE EST FORMULEE AINSI
//
// Pour `CibleRetrait` et `CibleComptage` : on releve toutes les REFERENCES DE LOT du texte
// (les jetons de la forme `1.9.13`, `0.A.2`, `3.x` — un chiffre, un point, au moins un
// segment). Le champ est ROUGE quand il en porte au moins une ET que TOUTES sont closes.
//
//	- « au moins une » : une cible qui ne nomme aucun lot (« retrait sec des que le compte est
//	  nul ») est la forme que D14 autorise explicitement — une CONDITION plutot qu'un lot ;
//	- « toutes » : un texte qui cite un lot clos EN PASSANT (« la moitie est faite au lot
//	  1.9.4, restent les deux appelants ») garde une cible vivante (`lot 3.x`) et n'est pas une
//	  cible perimee. Exiger qu'aucun lot clos ne soit cite interdirait d'ecrire cette histoire,
//	  qui est precisement ce qui rend une cible comprehensible.
//
// # CE QU'IL NE TIENT PAS, ET C'EST ECRIT
//
// Une FAMILLE de lots (`lot 1.6`, `lot 0.E`) n'est pas un item du plan : le plan ne coche que
// des items (`- [x] 1.6.3`). Une cible qui nomme une famille entierement close passe donc ce
// garde-rail. C'est une limite assumee — mesurer la cloture d'une famille demanderait de
// deduire sa composition du plan, ce qui inventerait une hierarchie que le plan n'ecrit pas.
// Consigne en §4 du plan (revue M1 — L3).
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Remettre `CibleRetrait: "lot 1.9.13"` sur n'importe quelle entree du registre : le test
// rougit (« cible perimee »). Jouee et restauree par NOM le 2026-09-16.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// cheminDuPlanDecodeur : le plan est la SOURCE des cases. Chemin depuis `apps/go-api` ; le plan
// clos vit sous `.ai/V7.5/` depuis l archivage du 2026-09-23 (fe2106f4b).
const cheminDuPlanDecodeur = "../../.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md"

// planchersLotsDuPlan : le nombre minimal d'items que le balayage du plan doit voir. Mesure du
// 2026-09-16 : 96 items coches. Plancher a 60 — un parcours casse (plan renomme, format des
// cases change) rendrait 0 et le test passerait en silence, ce qui est le pire des cas pour un
// ratchet.
const planchersLotsDuPlan = 60

// caseDItemDuPlan : `- [x] 1.9.13 **Une vie...` / `- [ ] 2.1.1 ...`. Le statut et l'identifiant.
var caseDItemDuPlan = regexp.MustCompile(`^\s*-\s\[([x~!\s])\]\s+([0-9][0-9A-Za-z.]*[0-9A-Za-z])\s`)

// referenceDeLot : un identifiant de lot cite dans un texte — au moins un point, pour ne pas
// confondre avec un compte (`0 piste`, `8 builds`).
var referenceDeLot = regexp.MustCompile(`\b[0-9]+\.[0-9A-Za-z]+(?:\.[0-9A-Za-z]+)*\b`)

// TestAucuneCibleDeRepliNeNommeUnLotClos — LE RATCHET.
func TestAucuneCibleDeRepliNeNommeUnLotClos(t *testing.T) {
	clos, ouverts := lotsDuPlan(t)
	if len(clos)+len(ouverts) < planchersLotsDuPlan {
		t.Fatalf("le balayage du plan n'a vu que %d item(s) (plancher %d) : le plan a-t-il "+
			"change de nom ou de format de cases ? Chemin lu : %s",
			len(clos)+len(ouverts), planchersLotsDuPlan, cheminDuPlanDecodeur)
	}
	var perimees []string
	for _, r := range decfilm.Table() {
		for _, champ := range []struct{ nom, valeur string }{
			{"CibleRetrait", r.CibleRetrait},
			{"CibleComptage", r.CibleComptage},
		} {
			if pb := ciblePerimee(champ.valeur, clos, ouverts); pb != "" {
				perimees = append(perimees, string(r.Nom)+" ."+champ.nom+" : "+pb)
			}
		}
	}
	if len(perimees) == 0 {
		return
	}
	sort.Strings(perimees)
	t.Errorf("CIBLE PERIMEE (%d) — le registre des replis nomme un lot que le plan a coche :\n  %s\n\n"+
		"Un lot fusionne qui n'a pas retire son repli laisse l'entree SANS echeance (D14 d,\n"+
		"regle 11 du depot). Reecrire la cible : un lot ENCORE OUVERT, ou une condition mesurable\n"+
		"sans lot (« retrait sec des que le compte est nul »). Si le lot a fait son travail sans\n"+
		"retirer le repli, le dire dans un commentaire au-dessus de l'entree et donner la cible\n"+
		"suivante, datee.", len(perimees), strings.Join(perimees, "\n  "))
}

// ciblePerimee rend le diagnostic d'un champ, ou "" quand il tient.
//
// # LE PLAN DU DECODEUR EST CLOS, ET LES CIBLES SYMBOLIQUES AVEC LUI (lot J8.7, 2026-09-27)
//
// Jusqu au lot J8.7, un item `[ ]` ou `[!]` du plan se lisait « ouvert », une FAMILLE (`lot 3.4`) ou
// une famille GENERIQUE (`1.9.x`, `3.x`) se lisait vivante, et un JALON (`M2`, « pas 2 de M2 ») n etait
// meme pas releve. Or le PLAN_DECODEUR_FILM est CLOS — cloture de M4 et du chantier le 2026-09-18
// (ADR 0034), plan archive sous `.ai/V7.5/` — et la campagne « retours rejeu », qui numerotait ses
// lots `M1` a `M8`, l est aussi (handoff du 2026-09-25) : aucun lot de ces deux plans ne fera plus
// rien. Une cible qui en nomme un ne designe PLUS RIEN, et c est exactement la cible sans echeance
// que ce garde-rail existe pour refuser. D ou trois familles de references perimees :
//
//	numerique  un item du plan (quel que soit son statut) ou une famille d items (`3.4`, `1.8`)
//	symbolique un jalon (`M2`, `M4b`) ou une famille generique (`1.9.x`, `3.x`)
//
// La cible vivante nomme un lot du PLAN VIVANT (`J11`, `J8.7`), ou une condition mesurable sans lot
// — la forme de la regle 4 de D-10 : « retrait au jalon suivant si compte nul au corpus gate de J11 ».
func ciblePerimee(valeur string, clos, ouverts map[string]bool) string {
	var vus []string
	for _, ref := range referenceDeLot.FindAllString(valeur, -1) {
		if clos[ref] || ouverts[ref] || estFamilleDuPlan(ref, clos, ouverts) {
			vus = append(vus, ref)
		}
	}
	vus = append(vus, referenceSymbolique.FindAllString(valeur, -1)...)
	if len(vus) == 0 {
		return "" // aucun lot du plan clos : la cible est une condition, ou un lot du plan vivant
	}
	return "lot(s) ou jalon(s) d un plan CLOS : " + strings.Join(vus, ", ") + " — dans " + tronquer(valeur)
}

// referenceSymbolique : un JALON (`M2`, `M4b`) ou une FAMILLE GENERIQUE (`1.9.x`, `3.x`) — les deux
// formes que la regle numerique ne voyait pas (cf. [ciblePerimee]).
var referenceSymbolique = regexp.MustCompile(`\bM[0-9]+[a-z]?\b|\b[0-9]+(?:\.[0-9]+)*\.x\b`)

// estFamilleDuPlan : `ref` prefixe au moins un item du plan (`3.4` pour `3.4.1`).
func estFamilleDuPlan(ref string, clos, ouverts map[string]bool) bool {
	for _, ens := range []map[string]bool{clos, ouverts} {
		for id := range ens {
			if strings.HasPrefix(id, ref+".") {
				return true
			}
		}
	}
	return false
}

// tronquer borne la citation du champ dans le message d'erreur.
func tronquer(s string) string {
	if len(s) <= 120 {
		return "«" + s + "»"
	}
	return "«" + s[:120] + "…»"
}

// lotsDuPlan relit les cases du plan et rend les identifiants CLOS et OUVERTS.
//
// UN IDENTIFIANT VU DEUX FOIS AVEC DEUX STATUTS EST OUVERT. Le plan en porte (`0.D.1` apparait
// coche puis `[!]`) : le lire comme clos ferait rougir une cible qui designe le travail restant.
func lotsDuPlan(t *testing.T) (clos, ouverts map[string]bool) {
	t.Helper()
	racine := racineGoAPI(t)
	brut, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(cheminDuPlanDecodeur))) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("plan du decodeur illisible (%s) : %v", cheminDuPlanDecodeur, err)
	}
	clos, ouverts = map[string]bool{}, map[string]bool{}
	for _, ligne := range strings.Split(string(brut), "\n") {
		m := caseDItemDuPlan.FindStringSubmatch(ligne)
		if m == nil {
			continue
		}
		id := m[2]
		if m[1] == "x" || m[1] == "~" {
			clos[id] = true
			continue
		}
		ouverts[id] = true
	}
	for id := range ouverts {
		delete(clos, id)
	}
	return clos, ouverts
}
