package archlint

// no_unregistered_fallback_test.go — UN REPLI HORS REGISTRE EST ROUGE (D14 a, lot 1.9.0 du plan
// `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, ADR 0034).
//
// # LE DÉFAUT QUE CE GARDE-RAIL FERME
//
// Le REPLI ANONYME : un `if` de secours au milieu d'une fonction, qui décide un fait à la place
// de la lecture du film, sans nom, sans compteur, sans date et sans critère de retrait. L'audit
// du lot 0.E en a recensé 62 dans le décodeur, dont neuf portaient DÉJÀ un défaut mesuré. Rien,
// dans le dépôt, n'empêchait le soixante-troisième.
//
// # LES DEUX DIRECTIONS, ET POURQUOI IL EN FAUT DEUX
//
//	(A) CODE -> REGISTRE. Tout identifiant Go DÉCLARÉ dans le décodeur qui se NOMME comme un
//	    repli doit être au registre. Sans cette direction, on écrirait un repli nommé sans
//	    jamais lui donner de critère de retrait.
//
//	(B) REGISTRE -> CODE. Toute entrée du registre doit pointer un site qui EXISTE : le fichier
//	    est là, et il porte l'ancre. Sans cette direction, le registre survivrait au code —
//	    et c'est précisément ce qui fait pourrir un inventaire. Cette direction est aussi ce
//	    qui rend D14 (d) MÉCANIQUE : quand un lot 1.9.x convertit un repli, son ancre
//	    disparaît, ce test rougit, et l'entrée DOIT sortir du registre dans le même commit.
//
// # LA CONVENTION DE NOMMAGE, ET POURQUOI ELLE EST ÉTROITE
//
// Un repli se nomme avec le segment `repli` / `Repli` (français, le mot du dépôt) ou
// `fallback` / `Fallback`, à une frontière de mot camelCase : `repliDernierOccupant`,
// `equipeParRepli`, `locateFallback`. La frontière EST la convention : sans elle, le scan
// attraperait `replication`, `replique`, `replier`, `repliement` — quatre mots français
// courants du vocabulaire du décodeur, qui ne désignent aucun repli. Un identifiant qui
// contient `repli` suivi d'une minuscule est donc IGNORÉ, et c'est voulu.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter, dans n'importe quel fichier non-test du périmètre :
//
//	func repliBidon() int { return 0 }
//
// La direction (A) rougit : « repliBidon (…) n'est déclaré nulle part au registre ». Mutation
// jouée et restaurée par nom au lot 1.9.0 (sortie consignée en §5 du plan).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode"

	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// perimetreReplis : les répertoires où un repli de production peut vivre — ceux de l'audit 0.E,
// plus `filmdec` pour ses inférences qui décident en production.
//
// Chemins relatifs à `apps/go-api/`. Un répertoire neuf du décodeur s'ajoute ici : c'est le bon
// sens de la faute (oublier d'étendre le périmètre laisse passer, l'oublier au registre rougit).
var perimetreReplis = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/sync/killcollector",
	"internal/analysis/objectiveevents",
}

// replisDeLEcrivainDuJeu : les identifiants qui portent `fallback` SANS être un repli de
// LevelUp. Ils nomment le chemin de repli DE L'ÉCRIVAIN DU JEU — une branche de la grammaire
// lue dans l'exécutable (le delta prédit est absent, le lecteur prend la voie absolue). Les
// nommer autrement mentirait sur ce que le film écrit.
//
// C'EST UN RATCHET, PAS UNE PORTE : chaque ligne porte sa date et sa raison, et la liste ne
// grandit que pour un identifiant qui désigne, lui aussi, une branche de la grammaire du jeu.
// Un repli de LevelUp n'y entre JAMAIS — il entre au registre.
var replisDeLEcrivainDuJeu = map[string]string{
	"attachFallbackCoverage": "2026-09-14 — replay/fallbacks_publication.go : la pose du compte des replis sur le document, et sa journalisation. Il ne DECIDE rien.",
	"FallbackHit":            "2026-09-14 — replay/coverage.go : le type PUBLIE d'un repli et son compte (`coverage.fallbacks[]`). Il ne DECIDE rien : il transporte le compte jusqu'au document.",
	"fallbackHitsOf":         "2026-09-14 — replay/coverage.go : la projection du rapport du compteur vers ce type publie. Meme raison.",
	"logFallbacks":           "2026-09-14 — replay/build.go : la journalisation des replis declenches, une ligne par repli. Meme raison.",
	"toFallbackHits":         "2026-09-14 — service/replayview/convert_coverage.go : la projection vers le document SERVI. Meme raison.",
	"PosKindAbsFallback":     "2026-09-14 — filmdec/position_capture.go : la NATURE d'une capture de position telle que l'écrivain du jeu la produit (absolu atteint par l'absence du delta prédit). Grammaire, pas décision.",
	"absViaFallback":         "2026-09-14 — filmdec/position_capture.go : le drapeau qui marque cette même branche de l'écrivain pendant la traversée.",
}

// TestToutReplinNommeEstAuRegistre — DIRECTION (A) : code -> registre.
func TestToutReplinNommeEstAuRegistre(t *testing.T) {
	racine := racineGoAPI(t)
	connus := identifiantsCouvertsParLeRegistre()
	var orphelins []string
	fichiersVus := 0
	for _, sous := range perimetreReplis {
		parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
			fichiersVus++
			for _, id := range identifiantsDeclares(f) {
				if !ressembleAUnRepli(id) || connus[id] {
					continue
				}
				if _, toleré := replisDeLEcrivainDuJeu[id]; toleré {
					continue
				}
				orphelins = append(orphelins, id+" ("+rel+")")
			}
		}, racine)
	}
	if fichiersVus < plancherFichiersPerimetreReplis {
		t.Fatalf("le parcours n'a vu que %d fichiers de production (plancher %d) : un ratchet qui "+
			"ne scanne rien passe en silence", fichiersVus, plancherFichiersPerimetreReplis)
	}
	if len(orphelins) == 0 {
		return
	}
	sort.Strings(orphelins)
	t.Errorf("REPLI HORS REGISTRE (%d) :\n  %s\n\n"+
		"Un repli est NOMMÉ, ordonné, compté, daté, et il porte son critère de retrait (D14).\n"+
		"Ajouter son entrée dans `internal/games/halo_infinite/film/replay/fallback/registre_*.go`,\n"+
		"avec une ancre qui cite ce site. Si l'identifiant nomme une branche de la grammaire DU JEU\n"+
		"et non une décision de LevelUp, l'inscrire dans `replisDeLEcrivainDuJeu` avec sa date.",
		len(orphelins), strings.Join(orphelins, "\n  "))
}

// plancherFichiersPerimetreReplis : le nombre minimal de fichiers de production que le parcours
// doit voir. Mesuré le 2026-09-14 : 351. Plancher à 250 — assez serré pour qu'un parcours cassé
// échoue, assez lâche pour ne pas devenir un compteur à maintenir.
const plancherFichiersPerimetreReplis = 250

// TestToutSiteDuRegistreExiste — DIRECTION (B) : registre -> code.
func TestToutSiteDuRegistreExiste(t *testing.T) {
	racine := racineGoAPI(t)
	cache := map[string]string{}
	for _, r := range fallback.Table() {
		for _, s := range r.Sites {
			contenu, ok := cache[s.Fichier]
			if !ok {
				b, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(s.Fichier))) //nolint:gosec // chemin issu du registre versionné
				if err != nil {
					t.Errorf("%s : le fichier %s n'existe plus (%v) — le repli a-t-il été converti ? "+
						"Dans ce cas, RETIRER l'entrée du registre dans le même commit (D14 d)",
						r.Nom, s.Fichier, err)
					continue
				}
				contenu = string(b)
				cache[s.Fichier] = contenu
			}
			if !strings.Contains(contenu, s.Ancre) {
				t.Errorf("%s : l'ancre est absente de %s.\n  ancre attendue : %q\n"+
					"  Si le repli a été CONVERTI, retirer son entrée du registre (D14 d).\n"+
					"  Si le code a seulement bougé, mettre l'ancre à jour dans le même commit.",
					r.Nom, s.Fichier, s.Ancre)
			}
		}
	}
}

// TestRegistreDesReplisEstValide : les invariants structurels du registre, revérifiés depuis
// `archlint`. Le paquet `fallback` les tient déjà ; les rejouer ici garantit qu'un contournement
// ne passe pas par la suppression du test voisin.
func TestRegistreDesReplisEstValide(t *testing.T) {
	for _, pb := range fallback.VerifierRegistre() {
		t.Errorf("registre des replis : %s", pb)
	}
}

// identifiantsCouvertsParLeRegistre : les identifiants que le registre NOMME — parce qu'ils
// apparaissent dans une ancre, ou parce qu'ils sont le nom de l'entrée elle-même.
//
// L'ANCRE EST LE LIEN, et c'est ce qui rend les deux directions cohérentes : un repli nommé dans
// le code entre au registre en citant son identifiant dans l'ancre de son site, ce qui satisfait
// (A) et (B) d'un seul geste.
func identifiantsCouvertsParLeRegistre() map[string]bool {
	out := map[string]bool{}
	for _, r := range fallback.Table() {
		out[string(r.Nom)] = true
		for _, s := range r.Sites {
			for _, mot := range decouperEnIdentifiants(s.Ancre) {
				out[mot] = true
			}
		}
	}
	return out
}

// decouperEnIdentifiants extrait d'une ancre les suites de caractères d'identifiant Go.
func decouperEnIdentifiants(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
}

// ressembleAUnRepli applique la convention de nommage : `repli` ou `fallback`, à une frontière
// de mot camelCase, suivi d'une majuscule, d'un chiffre, d'un `_` ou de la fin de l'identifiant.
func ressembleAUnRepli(id string) bool {
	for _, racine := range []string{"repli", "Repli", "fallback", "Fallback"} {
		for i := 0; i+len(racine) <= len(id); i++ {
			if id[i:i+len(racine)] != racine {
				continue
			}
			if i > 0 && estMinusculeOuChiffre(rune(id[i-1])) && racine[0] >= 'a' && racine[0] <= 'z' {
				continue // `xrepli...` : pas une frontière de mot
			}
			if j := i + len(racine); j == len(id) || !estMinusculeOuChiffre(rune(id[j])) {
				return true
			}
		}
	}
	return false
}

func estMinusculeOuChiffre(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// identifiantsDeclares rend les noms DÉCLARÉS par un fichier : fonctions et méthodes, types,
// constantes, variables de paquet, et champs de struct.
//
// LES CHAMPS DE STRUCT EN FONT PARTIE, et c'est le point : `Repli bool` dans une structure de
// couverture est exactement le repli qu'on veut voir déclaré.
func identifiantsDeclares(f *ast.File) []string {
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.FuncDecl:
			out = append(out, d.Name.Name)
		case *ast.TypeSpec:
			out = append(out, d.Name.Name)
		case *ast.ValueSpec:
			for _, nom := range d.Names {
				if nom.Name != "_" {
					out = append(out, nom.Name)
				}
			}
		case *ast.Field:
			for _, nom := range d.Names {
				out = append(out, nom.Name)
			}
		}
		return true
	})
	return out
}

// parcourirGoProduction applique `visite` à chaque fichier Go NON-test d'une arborescence,
// parsé en AST. `testdata` est écarté : ce ne sont pas des sources du décodeur.
func parcourirGoProduction(t *testing.T, dir string, visite func(rel string, f *ast.File), racine string) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(chemin string, d os.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if d.IsDir() {
			// `fallback` EST le registre : ses propres types (`Repli`, `Ordre`...) nomment les
			// replis sans en être. Le scanner reviendrait à demander au registre de se déclarer
			// lui-même.
			if d.Name() == "testdata" || d.Name() == "vendor" || d.Name() == "fallback" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, chemin, nil, 0)
		if err != nil {
			t.Errorf("parse de %s : %v", chemin, err)
			return nil
		}
		rel, errRel := filepath.Rel(racine, chemin)
		if errRel != nil {
			rel = chemin
		}
		visite(filepath.ToSlash(rel), f)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", dir, err)
	}
}
