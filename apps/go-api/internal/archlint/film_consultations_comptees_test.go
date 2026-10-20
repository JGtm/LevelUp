package archlint

// film_consultations_comptees_test.go — AUCUNE LECTURE DE PRODUCTION NE CONSULTE SANS ENREGISTREUR
// (lot J8.7-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, 2026-09-28).
//
// Le repli qui se declenche a la LECTURE des series nommees (`repli_emission_hors_domaine_jetee`) se
// note dans l enregistreur du document (`objectives.ReplisALaConsultation`), passe en DERNIER argument des
// entrees publiques ci-dessous. Nil ne note rien : c est la valeur des outils hors production. Dans la
// cuisson (`film/replay`, `replaybuild`), un nil litteral serait une lecture qui decide des faits SANS
// les compter — ce que la regle utilisateur interdit (un repli n est jamais silencieux).
//
// MUTATION JOUEE (2026-09-28) : remplacer `opt.consultations()` par `nil` dans l appel de
// `NamedEventsFrom` de `replay/vip_crown.go` fait rougir ce test, qui nomme le fichier et l appel.

import (
	"go/ast"
	"path/filepath"
	"testing"
)

// lecturesComptees : les entrees publiques qui consultent les series nommees ou le resolveur, et dont
// le DERNIER argument est l enregistreur du document.
var lecturesComptees = map[string]bool{
	"SeriesByRound": true, "SeriesTotal": true, "NamedEventsFrom": true,
	"ResolveRoundIdentity": true, "SlotIdentityFrom": true,
}

// perimetreDesConsultationsComptees : la cuisson — les deux paquets qui assemblent un document.
var perimetreDesConsultationsComptees = []string{
	"internal/games/halo_infinite/film/replay",
	"internal/replaybuild",
}

func TestAucuneLectureDeLaCuissonNeConsulteSansEnregistreur(t *testing.T) {
	racine := racineGoAPI(t)
	vus := 0
	for _, sous := range perimetreDesConsultationsComptees {
		parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
			ast.Inspect(f, func(n ast.Node) bool {
				appel, ok := n.(*ast.CallExpr)
				if !ok || len(appel.Args) == 0 {
					return true
				}
				sel, ok := appel.Fun.(*ast.SelectorExpr)
				if !ok || !lecturesComptees[sel.Sel.Name] {
					return true
				}
				vus++
				if id, nul := appel.Args[len(appel.Args)-1].(*ast.Ident); nul && id.Name == "nil" {
					t.Errorf("%s : `%s(..., nil)` — une lecture de la cuisson consulte sans l enregistreur du "+
						"document : ses replis a la consultation ne seraient pas comptes. Passer "+
						"`opt.consultations()` (replay) ou `pont.cons` (replaybuild).", rel, sel.Sel.Name)
				}
				return true
			})
		}, racine)
	}
	t.Logf("%d lectures de la cuisson vues", vus)
	// PLANCHER ANTI-MUET : mesure du 2026-09-28, 18 lectures dans le perimetre.
	if vus < 10 {
		t.Fatalf("le parcours n a vu que %d lectures (plancher 10) : un garde-rail qui ne voit plus rien passe en silence", vus)
	}
}
