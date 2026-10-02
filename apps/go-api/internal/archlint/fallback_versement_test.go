package archlint

// fallback_versement_test.go — LES REPLIS COMPTES EN DONNEES, ET LA DIRECTION (E) DU REGISTRE
// (lot J8.7 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, 2026-09-27).
//
// # CE QUE LE LOT A CHANGE AU MODELE DES DIRECTIONS
//
// Jusqu au lot J8.7, un repli branche se comptait par `Declenche(fallback.NomX)` AU SITE, et la
// direction (C) relisait ces appels. Les replis des couches qui ne peuvent pas importer le registre
// (`grammar`, `profile`) ou ne le doivent pas (`killsource`, `objectives`, DU-2 (c)) se comptent
// desormais EN DONNEES, et une seule fonction de `replay` les verse par une TABLE
// (`replay/versement_des_replis.go`) : son `DeclencheN(v.nom, …)` porte un nom VARIABLE, que (C)
// refusait. Ce fichier ferme les deux trous que ce modele ouvrirait :
//
//   - (C) DANS UNE TABLE : les noms que la table CITE sont des declenchements, relus comme tels —
//     chaque entree versee doit citer la table en site, exactement comme un site de `Declenche` ;
//   - (E) BRANCHE -> DECLENCHE : une entree `CompteurBranche: true` doit avoir sa constante, et
//     cette constante doit etre DECLENCHEE (argument de `Declenche`/`DeclencheN`) ou VERSEE (ligne
//     d une table) quelque part en production. Sans elle, remettre une entree a `true` sans la
//     cabler — ou retirer la ligne de table qui la versait — passait vert : l ancre du site de
//     decision survit au decablage.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - retirer une ligne de `versementsDesReplis` : (E) rougit (« branche, mais jamais declenche ») ;
//   - ajouter a la table une ligne d un repli dont l entree ne cite pas la table : (C) rougit ;
//   - passer `CompteurBranche: true` sur une entree sans constante : (E) rougit.
//
// Jouees et restaurees par nom le 2026-09-27 (lot J8.7).

import (
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// tablesDeVersement : LES FICHIERS DONT LE DECLENCHEMENT PORTE UN NOM VARIABLE, lu dans une table
// du meme fichier. C EST UNE ALLOWLIST FERMEE ET DATEE : une table de plus est une porte de plus, et
// elle ne s ouvre que pour un versement de comptes rendus EN DONNEES par une couche qui ne peut pas
// declencher elle-meme. Relue par [TestChaqueTableDeVersementATouJoursSonSite].
var tablesDeVersement = map[string]exemptionEcrivain{
	"internal/games/halo_infinite/film/replay/versement_des_replis.go": {
		Fichier: "internal/games/halo_infinite/film/replay/versement_des_replis.go",
		Date:    "2026-09-27",
		Raison: "la seule porte entre les comptes de replis que grammar, profile, killsource et objectives rendent EN DONNEES et le " +
			"compteur de la cuisson (decision 1 du superviseur, lot J8.7) : une ligne par entree, un champ source par ligne.",
	},
}

// replisComptesAuDocumentServi : les entrees BRANCHEES dont le compte n est PAS un declenchement du
// compteur de cuisson mais un champ du document SERVI — une regle du service, jouee a la lecture de
// l artefact, APRES la cuisson : son compte ne peut pas entrer dans `coverage.fallbacks`. `Fichier`
// porte le site du compte et `Ancre` le champ compte ; [TestChaqueRepliBrancheEstDeclencheOuVerse] les
// relit. Decouverts par la direction (E) a sa pose (2026-09-27) : ils etaient branches sans constante.
var replisComptesAuDocumentServi = map[decfilm.Nom]struct{ Fichier, Ancre, Date, Raison string }{
	"repli_decor_carte_sans_zone_affiche": {"internal/service/replay_vehicle_scenery_rule.go", "out.ZoneUnknown++", "2026-09-27",
		"compte dans `vehicleScenery.zoneUnknown` du document servi (lot M7 des retours rejeu) : regle du service, pas de la cuisson"},
	"repli_decor_sous_le_sol_foule_du_match": {"internal/service/replay_vehicle_scenery_rule.go", "return sceneryReasonBelowPlayedFloor", "2026-09-27",
		"compte dans `vehicleScenery.hidden` sous la raison `below_played_floor` du document servi : regle du service, pas de la cuisson"},
	"repli_decor_tenu_en_l_air_a_vide": {"internal/service/replay_vehicle_scenery_rule.go", "Reason: sceneryReasonAloftUnoccupied", "2026-10-02",
		"compte dans `vehicleScenery.hidden` sous la raison `aloft_unoccupied` du document servi (chantier Falcon de Behemoth) : regle du service, pas de la cuisson"},
}

// perimetreDeclenchementsHorsDecodeur : les repertoires HORS du perimetre du decodeur ou un repli du
// registre se declenche quand meme — une passe de `sync` qui juge un fait du rejeu. Ils ne sont
// balayes QUE pour la direction (E) : les directions (A) a (C) restent sur `perimetreReplis`.
var perimetreDeclenchementsHorsDecodeur = []string{
	"internal/sync/replayartifacts",
}

// declenchementsDuFichierOuDeSaTable : les declenchements d un fichier pour la direction (C). Pour
// une TABLE DE VERSEMENT, l appel a nom variable est ecarte et les noms que la table CITE le
// remplacent — ce sont eux que (C) confronte aux sites.
func declenchementsDuFichierOuDeSaTable(rel string, f *ast.File) []string {
	appels := declenchementsDuFichier(f)
	if _, table := tablesDeVersement[rel]; !table {
		return appels
	}
	out := make([]string, 0, len(appels))
	for _, a := range appels {
		if strings.HasPrefix(a, "Nom") {
			out = append(out, a)
		}
	}
	return append(out, nomsCitesParUneTable(f)...)
}

// nomsCitesParUneTable rend les constantes de nom (`fallback.NomX`) que les LITTERAUX COMPOSITES du
// fichier citent — les lignes d une table de versement.
func nomsCitesParUneTable(f *ast.File) []string {
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, e := range lit.Elts {
			sel, ok := e.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(sel.Sel.Name, "Nom") {
				continue
			}
			if x, ok := sel.X.(*ast.Ident); ok && (x.Name == "fallback" || x.Name == "decfilm") {
				out = append(out, sel.Sel.Name)
			}
		}
		return true
	})
	return out
}

// TestChaqueRepliBrancheEstDeclencheOuVerse — DIRECTION (E) : branche -> declenche.
func TestChaqueRepliBrancheEstDeclencheOuVerse(t *testing.T) {
	racine := racineGoAPI(t)
	valeurDeLaConstante := nomsDuPaquetFallback(t, racine)
	constanteDuNom := map[decfilm.Nom]string{}
	for id, nom := range valeurDeLaConstante {
		constanteDuNom[nom] = id
	}
	vus := map[string]bool{}
	for _, sous := range append(append([]string{}, perimetreReplis...), perimetreDeclenchementsHorsDecodeur...) {
		parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
			for _, a := range declenchementsDuFichierOuDeSaTable(rel, f) {
				vus[a] = true
			}
		}, racine)
	}
	var fautes []string
	branches := 0
	for _, r := range decfilm.Table() {
		if !r.CompteurBranche {
			continue
		}
		branches++
		if ex, servi := replisComptesAuDocumentServi[r.Nom]; servi {
			if pb := siteDuCompteServi(racine, ex.Fichier, ex.Ancre, ex.Date, ex.Raison); pb != "" {
				fautes = append(fautes, string(r.Nom)+" : "+pb)
			}
			continue
		}
		id, ok := constanteDuNom[r.Nom]
		switch {
		case !ok:
			fautes = append(fautes, string(r.Nom)+" : branche, mais aucune constante dans `fallback/noms.go`")
		case !vus[id]:
			fautes = append(fautes, string(r.Nom)+" : branche, mais "+id+" n est ni declenche ni verse en production")
		}
	}
	if branches < plancherReplisBranches {
		t.Fatalf("seulement %d entrees branchees (plancher %d) : le registre a-t-il ete vide ?", branches, plancherReplisBranches)
	}
	sort.Strings(fautes)
	for _, f := range fautes {
		t.Errorf("DIRECTION (E) — %s.\n  Un compteur « branche » que rien ne declenche publie un zero qui n est pas une mesure :\n"+
			"  D-10 (regle 4) supprimerait un repli actif. Cabler le site, ou remettre `CompteurBranche: false`.", f)
	}
}

// siteDuCompteServi rend le diagnostic d une exemption de [replisComptesAuDocumentServi], ou "".
func siteDuCompteServi(racine, fichier, ancre, date, raison string) string {
	if !dateExemptionConforme(date) || strings.TrimSpace(raison) == "" {
		return "exemption sans date ou sans raison"
	}
	b, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(fichier))) //nolint:gosec // chemin fige dans le code
	if err != nil || !strings.Contains(string(b), ancre) {
		return "compte au document servi introuvable (" + fichier + " : " + ancre + ") — l exemption survit a son site, la RETIRER"
	}
	return ""
}

// plancherReplisBranches : mesure du 2026-09-27 (sous-lot grammar du lot J8.7) — 70 entrees
// branchees. Plancher a 40 : un parcours casse rendrait zero et passerait en silence.
const plancherReplisBranches = 40

// TestChaqueTableDeVersementATouJoursSonSite : une table de versement retiree laisse sinon une porte
// ouverte a un `DeclencheN` a nom variable dans un fichier du meme nom.
func TestChaqueTableDeVersementATouJoursSonSite(t *testing.T) {
	racine := racineGoAPI(t)
	for rel, ex := range tablesDeVersement {
		if ex.Fichier != rel || !dateExemptionConforme(ex.Date) || strings.TrimSpace(ex.Raison) == "" {
			t.Errorf("table de versement %s : fichier, date ou raison incoherents", rel)
		}
		trouve := false
		parcourirGoProduction(t, filepath.Dir(filepath.Join(racine, filepath.FromSlash(rel))), func(r string, f *ast.File) {
			if r == rel && len(nomsCitesParUneTable(f)) > 0 {
				trouve = true
			}
		}, racine)
		if !trouve {
			t.Errorf("table de versement %s : le fichier n existe plus ou ne cite plus aucun nom — RETIRER la ligne", rel)
		}
	}
}

// TestLesReplisHorsProductionNOntQueLeursOutilsPourAppelants — LA PREUVE DE LA CATEGORIE « OUTIL HORS
// PRODUCTION » (lot J8.7, decision 5 du superviseur, 2026-09-27).
//
// Une entree sort du ratchet des compteurs parce que son code ne tourne QUE dans un outil. Cette
// affirmation se VERIFIE : aucun fichier non-test du module n appelle `decfilm.<Symbole>` ni
// `objectives.<Symbole>` hors des repertoires qu elle nomme (la facade qui le renvoie et le paquet qui
// le definit exceptes). Le jour ou une cuisson ou une passe l appelle, l entree doit etre CABLEE.
//
// MUTATION JOUEE (2026-09-27) : un appel `decfilm.Extract(...)` ajoute dans `internal/replaybuild`
// fait rougir ce test.
func TestLesReplisHorsProductionNOntQueLeursOutilsPourAppelants(t *testing.T) {
	racine := racineGoAPI(t)
	for _, r := range decfilm.Table() {
		h := r.HorsProduction
		if h == nil {
			continue
		}
		for _, sous := range []string{"internal", "cmd"} {
			parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
				if appelantAdmis(rel, h.Appelants) {
					return
				}
				ast.Inspect(f, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != h.Symbole {
						return true
					}
					if x, ok := sel.X.(*ast.Ident); ok && (x.Name == "decfilm" || x.Name == "objectives") {
						t.Errorf("%s : %s appelle %s.%s — le repli %s n est plus d un outil hors production : le CABLER",
							r.Nom, rel, x.Name, h.Symbole, r.Nom)
					}
					return true
				})
			}, racine)
		}
	}
}

// appelantAdmis : le fichier vit sous un appelant nomme, ou dans la facade qui renvoie le symbole.
func appelantAdmis(rel string, appelants []string) bool {
	if strings.HasPrefix(rel, "internal/games/halo_infinite/film/decfilm/") {
		return true
	}
	for _, a := range appelants {
		if strings.HasPrefix(rel, strings.TrimSuffix(a, "/")+"/") {
			return true
		}
	}
	return false
}
