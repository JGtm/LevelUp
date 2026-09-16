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
// # LES TROIS DIRECTIONS, ET POURQUOI IL EN FAUT TROIS
//
//	(A) CODE -> REGISTRE. Tout identifiant Go DÉCLARÉ dans le décodeur qui se NOMME comme un
//	    repli doit être au registre — ET DANS LE FICHIER OÙ IL EST DÉCLARÉ. Sans cette
//	    direction, on écrirait un repli nommé sans jamais lui donner de critère de retrait.
//
//	(B) REGISTRE -> CODE. Toute entrée du registre doit pointer un site qui EXISTE : le fichier
//	    est là, et il porte l'ancre. Sans cette direction, le registre survivrait au code —
//	    et c'est précisément ce qui fait pourrir un inventaire. Cette direction est aussi ce
//	    qui rend D14 (d) MÉCANIQUE : quand un lot 1.9.x convertit un repli, son ancre
//	    disparaît, ce test rougit, et l'entrée DOIT sortir du registre dans le même commit.
//
//	(C) DÉCLENCHEMENT -> SITE. Tout appel `Declenche(NomX)` / `DeclencheN(NomX, …)` du dépôt
//	    vit dans un fichier que les [fallback.Site] de l'entrée X CITENT. Sans cette direction,
//	    un repli peut se déclencher depuis un endroit que le registre ne décrit pas : c'est
//	    exactement ce qui est arrivé à `repli_vie_coupee_au_trou_de_replication`, déclenché
//	    depuis `tracks_publication.go` pendant que son entrée ne citait que `lives_decoupe.go`
//	    — deux conditions d'ouverture différentes, une seule décrite (revue de jalon M1,
//	    2026-09-15). (B) ne l'attrapait pas : l'ancre qu'elle relit existait bel et bien.
//
// # LA COUVERTURE EST INDEXÉE PAR FICHIER, ET C'EST (A) QUI EN DÉPEND
//
// Constat L6-c4 de la revue de jalon M1 (2026-09-15) : la couverture était un ensemble GLOBAL
// d'identifiants, tirés de TOUTES les ancres du registre, testé sans regarder le fichier. Témoin
// joué par le relecteur — `func locateFallback() int { return 0 }` ajouté dans
// `filmdec/varwidth.go` passait VERT, parce que `locateFallback` est cité par l'ancre d'une
// entrée qui pointe `killsource/`. Un repli neuf qui reprend un nom déjà employé ailleurs entrait
// donc en production sans entrée. Un identifiant n'est désormais couvert que DANS le fichier que
// l'ancre cite.
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
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// (A) Ajouter, dans n'importe quel fichier non-test du périmètre :
//
//	func repliBidon() int { return 0 }
//
// La direction (A) rougit : « repliBidon (…) n'est déclaré nulle part au registre ». Mutation
// jouée et restaurée par nom au lot 1.9.0 (sortie consignée en §5 du plan).
//
// (A bis) `func locateFallback() int { return 0 }` dans `filmdec/varwidth.go` — le témoin du
// constat L6-c4 : VERT avant l'indexation par fichier, ROUGE depuis.
//
// (C) Déplacer un `fb.Declenche(fallback.NomXxx)` dans un fichier que les sites de l'entrée X
// ne citent pas : « déclenché depuis (…) que l'entrée ne cite pas ».
//
// (D) Laisser dans [replisDeLEcrivainDuJeu] une entrée dont l'identifiant n'est plus déclaré par
// le fichier cité : « exemption sans site ».

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
// plus `grammar` pour ses inférences qui décident en production.
//
// Chemins relatifs à `apps/go-api/`. Un répertoire neuf du décodeur s'ajoute ici : c'est le bon
// sens de la faute (oublier d'étendre le périmètre laisse passer, l'oublier au registre rougit).
var perimetreReplis = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/sync/killcollector",
	"internal/games/halo_infinite/film/facts/objectives",
}

// replisDeLEcrivainDuJeu : les identifiants qui portent `fallback` SANS être un repli de
// LevelUp. Ils nomment le chemin de repli DE L'ÉCRIVAIN DU JEU — une branche de la grammaire
// lue dans l'exécutable (le delta prédit est absent, le lecteur prend la voie absolue). Les
// nommer autrement mentirait sur ce que le film écrit.
//
// C'EST UN RATCHET, PAS UNE PORTE : chaque ligne porte sa date et sa raison, et la liste ne
// grandit que pour un identifiant qui désigne, lui aussi, une branche de la grammaire du jeu.
// Un repli de LevelUp n'y entre JAMAIS — il entre au registre.
// L'EXEMPTION PORTE SON FICHIER DANS UN CHAMP, PAS DANS SA PHRASE (revue de jalon M1,
// 2026-09-15, constat arch C2). La map n'était lue qu'en consultation : une exemption dont le
// site avait disparu ne rougissait pas, et le fichier cité vivait dans une chaîne libre que rien
// ne relisait. Trois entrées sur sept mentaient — `logFallbacks` n'existait nulle part (le
// journaliseur réel est `attachFallbackCoverage`), `FallbackHit` et `fallbackHitsOf` étaient
// annotés `replay/coverage.go` alors qu'ils vivent dans `replay/fallbacks_publication.go`.
// Le ratchet frère `no_title_package_in_analysis_test.go` le dit depuis longtemps : « une
// exemption qui survit à son site finit par en couvrir un autre ».
type exemptionEcrivain struct {
	// Fichier : le fichier qui DÉCLARE cet identifiant, relatif à `apps/go-api/`. Relu par
	// [TestChaqueExemptionDeLEcrivainATouJoursSonSite].
	Fichier string
	// Date : le jour où l'exemption a été posée, `AAAA-MM-JJ`.
	Date string
	// Raison : pourquoi cet identifiant nomme une branche de la grammaire DU JEU, et non une
	// décision de LevelUp.
	Raison string
}

var replisDeLEcrivainDuJeu = map[string]exemptionEcrivain{
	"attachFallbackCoverage": {"internal/games/halo_infinite/film/replay/fallbacks_publication.go", "2026-09-14",
		"la pose du compte des replis sur le document, et sa journalisation. Il ne DECIDE rien."},
	"FallbackHit": {"internal/games/halo_infinite/film/replay/fallbacks_publication.go", "2026-09-14",
		"le type PUBLIE d'un repli et son compte (`coverage.fallbacks[]`). Il ne DECIDE rien : il transporte le compte jusqu'au document."},
	"fallbackHitsOf": {"internal/games/halo_infinite/film/replay/fallbacks_publication.go", "2026-09-14",
		"la projection du rapport du compteur vers ce type publie. Meme raison."},
	"toFallbackHits": {"internal/service/replayview/convert_coverage.go", "2026-09-14",
		"la projection vers le document SERVI. Meme raison."},
	"PosKindAbsFallback": {"internal/games/halo_infinite/film/grammar/position_capture.go", "2026-09-14",
		"la NATURE d'une capture de position telle que l'ecrivain du jeu la produit (absolu atteint par l'absence du delta predit). Grammaire, pas decision."},
	"viaRepli": {"internal/games/halo_infinite/film/grammar/position_capture.go", "2026-09-14",
		"le drapeau qui marque cette meme branche de l'ecrivain pendant la traversee. " +
			"S'appelait `absViaFallback` jusqu'au lot 2.3 (2026-09-17), qui en a fait un CHAMP de " +
			"`captureDePosition` au lieu d'une variable de paquet — meme branche, meme raison."},
}

// TestToutReplinNommeEstAuRegistre — DIRECTION (A) : code -> registre, FICHIER PAR FICHIER.
func TestToutReplinNommeEstAuRegistre(t *testing.T) {
	racine := racineGoAPI(t)
	connus := couvertureParFichier()
	var orphelins []string
	fichiersVus := 0
	for _, sous := range perimetreReplis {
		parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
			fichiersVus++
			for _, id := range identifiantsDeclares(f) {
				if !ressembleAUnRepli(id) || connus[rel][id] {
					continue
				}
				if ex, toleré := replisDeLEcrivainDuJeu[id]; toleré && ex.Fichier == rel {
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

// couvertureParFichier : les identifiants que le registre NOMME, INDEXÉS PAR LE FICHIER DU SITE
// qui les cite — parce qu'ils apparaissent dans son ancre, ou parce qu'ils sont le nom de
// l'entrée elle-même.
//
// L'ANCRE EST LE LIEN, et c'est ce qui rend les directions cohérentes : un repli nommé dans le
// code entre au registre en citant son identifiant dans l'ancre de son site, ce qui satisfait
// (A) et (B) d'un seul geste.
//
// L'INDEX PAR FICHIER EST CE QUI REND (A) EXACTE (constat L6-c4, 2026-09-15). Un ensemble global
// laissait passer tout identifiant qu'une ancre citait QUELQUE PART : `locateFallback` déclaré
// dans `filmdec/varwidth.go` passait pour couvert par une entrée qui pointe `killsource/`. Le
// registre ne dit pas « ce nom existe », il dit « ce nom décide ICI » ; la couverture le dit donc
// aussi.
func couvertureParFichier() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, r := range fallback.Table() {
		for _, s := range r.Sites {
			if out[s.Fichier] == nil {
				out[s.Fichier] = map[string]bool{}
			}
			out[s.Fichier][string(r.Nom)] = true
			for _, mot := range decouperEnIdentifiants(s.Ancre) {
				out[s.Fichier][mot] = true
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
