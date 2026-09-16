package archlint

// film_function_length_test.go — LE RATCHET DE LONGUEUR DES FONCTIONS DU DECODEUR ET DE SES
// VOISINS (item 2.10.5, decision utilisateur V16 option B, 2026-09-16).
//
// # POURQUOI CE RATCHET EXISTE : `funlen` NE VOIT RIEN
//
// CLAUDE.md regle 5 fixe le seuil a 80 lignes par fonction. Decouverte D1 (2.7p) du
// 2026-09-16 : ce seuil n est garde par AUCUN instrument. `golangci-lint` v2 met
// `funlen.ignore-comments` a `true` par defaut, et la mesure l a dit sans ambiguite —
// `golangci-lint run --enable=funlen` aux seuils du depot (80 lignes / 80 instructions) rendait
// 0 issue sur `replay/` alors que `BuildFromPositions` y faisait 474 LIGNES PHYSIQUES,
// `decimateTracks` 107 et `BuildMapObjectives` 89. Sur un code aussi commente que celui du
// decodeur, ignorer les commentaires revient a ne plus rien mesurer : les trois fonctions de
// l item 2.7.1 ont ete trouvees a la main, puis par un `awk` sur les accolades.
//
// Basculer `ignore-comments` a `false` dans `.golangci.yml` aurait rougi la baseline de lint sur
// TOUT le depot. L utilisateur a tranche (V16, option B) : un instrument dedie ici, borne aux
// paquets du chantier, avec une table datee — et `funlen` reste tel quel.
//
// # LA REGLE, EN DEUX LIGNES
//
//	fonction hors table   plafond 80 lignes PHYSIQUES
//	fonction de la table  plafond = sa longueur du jour de sa mise en table, JAMAIS accrue
//
// « Lignes physiques » = de la ligne du mot-cle `func` a la ligne de l accolade fermante, bornes
// comprises, commentaires et lignes vides interieurs compris. C est exactement ce que `funlen`
// ne compte pas, et c est ce que lit un humain qui doit tenir la fonction dans sa tete.
//
// Une fonction qui descend sous son plafond n est pas une erreur — c est le sens de la marche.
// Quand elle repasse sous 80, son entree se RETIRE de la table : la table n a plus de raison de
// la porter, et une entree perimee finit par autoriser n importe quoi
// (`TestPlafondsDeFonctionNeSontPasPerimes` le dit lui-meme).
//
// # PERIMETRE : LA PRODUCTION SEULEMENT, ET LA MESURE DIT POURQUOI
//
// Mesure du 2026-09-16 sur les quatre racines — 1 332 fichiers `.go` (410 de production),
// 10 197 fonctions et methodes (2 516 de production) :
//
//	fonctions > 80 lignes physiques   production : 8      tests : 93
//
// Les dix plus longues du jour, pour situer l echelle : 1 070, 230, 200, 197, 185, 182, 177,
// 171, 162, 162 — TOUTES des fonctions de test. La plus longue fonction de PRODUCTION des quatre
// racines fait 146 lignes.
//
// Les 8 fonctions de production sont TOUTES dans `grammar` (sept aiguillages de composants plus
// la table d invariants du profil) et tiennent dans la table ci-dessous. Les 93 fonctions de
// test, elles, ne sont pas mises sous ratchet, et ce n est pas un oubli :
//
//  1. LE VOLUME ET SA ROTATION. Une table de 93 entrees serait douze fois celle de production,
//     et 63 de ces 93 (68 %) vivent dans des `*_research_test.go` — des instruments de mesure a
//     usage unique, ecrits pour une question, dont ce chantier produit plusieurs par semaine. Un
//     ratchet dont la table bouge a chaque lot cesse d etre lu et se fait amender par reflexe :
//     c est la pourriture d allowlist contre laquelle les autres ratchets de ce paquet mettent
//     en garde.
//  2. LE RISQUE N EST PAS LE MEME DES DEUX COTES. Ce que la regle 5 protege, c est la fonction
//     ou une branche ajoutee ligne 300 interagit avec un etat construit ligne 40 — les
//     aiguillages de `grammar` sont precisement cela. Une fonction de test longue est une suite
//     d assertions independantes ou une table litterale : pas d etat accumule a se tromper, et
//     quand elle casse elle le dit elle-meme, puisqu elle EST l oracle.
//  3. LE COTE TEST EST DEJA BORNE, AU GRAIN DU FICHIER. `film_file_size_test.go` couvre les
//     tests (22 de ses entrees sont des `_test.go`), et une fonction de test ne peut pas depasser
//     500 lignes sans que son fichier les depasse aussi. Le cas le plus lourd mesure — les
//     1 070 lignes de `replay/structure_test.go:TestStructureIsOptionalInDocument` — est deja
//     gele par l entree `structure_test.go: 1151` de cet autre ratchet : il ne peut plus grossir.
//
// Si le compte des tests redevenait raisonnable (scission des balayages de recherche), une
// SECONDE table datee pourrait les prendre sans toucher a celle-ci. La porte reste ouverte ; on
// ne l ouvre pas sur 93 entrees.
//
// # LES FICHIERS `//go:build research` SONT EXCLUS
//
// Deux raisons, dans cet ordre :
//
//  1. AUCUN GATE NE LES COMPILE. Le tag `research` n est passe ni par `go test ./...`, ni par la
//     CI, ni par `make gate-push` : ces fichiers sont des instruments de mesure a usage unique,
//     gardes comme archive d une mesure, que personne ne refactorera. Les mettre sous ratchet
//     ferait sonner un garde-rail sur du code qu aucun compilateur ne voit.
//  2. MESURE, pour que l exclusion ne soit pas une amnistie deguisee : sur les 23 fichiers
//     `research` des quatre racines, ZERO fonction depasse 80 lignes (la plus longue,
//     `e191c_typeversions_research_test.go:TestE191cVersionsParTypeAlignees`, en fait 61).
//     L exclusion ne retire donc RIEN de la table aujourd hui — c est une regle sur l avenir,
//     pas un pardon sur le present.
//
// DIT HONNETEMENT : les 23 fichiers `research` d aujourd hui sont TOUS des `*_test.go`, donc le
// perimetre « production seulement » les ecarte deja. La regle est gardee a part parce que les
// deux criteres sont independants — le jour ou un instrument de recherche s ecrit hors d un
// fichier de test, `research` doit continuer de le sortir — et elle est exercee par
// `TestExclusionDuTagResearch`, qui la tient hors du musee du code mort.
//
// Les fichiers `//go:build integration`, eux, sont bien couverts : ceux-la, le depot les compile
// (`go test -tags=integration ./...`, obligatoire avant toute livraison sync/persist).
//
// # CE QUE LE RATCHET NE FAIT PAS, ET C EST VOULU
//
// Il ne demande a personne de scinder les huit fonctions de la table. Il interdit de les
// AGRANDIR. La scission se decide lot par lot, avec la preuve d equivalence qui va avec — le
// lot 2.7 l a faite pour `replay/` (474 lignes ramenees a zero fonction au-dela de 80 dans tout
// le paquet) ; les aiguillages de `grammar` attendront le lot qui les rouvrira.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// seuilLignesParFonction : le seuil du depot (CLAUDE.md regle 5).
const seuilLignesParFonction = 80

// Les racines surveillees sont CELLES DU RATCHET DE TAILLE DE FICHIER, par reference et non par
// copie (`racinesSurveilleesTaille`, `film_file_size_test.go`) : deux listes recopiees divergent,
// et le perimetre des deux ratchets est le meme — le chantier du decodeur de film. Ajouter une
// racine la-bas la met sous les deux regles ici.

// plancherFonctionsBalayees : LE PLANCHER CONTRE UN BALAYAGE MUET. 2 516 fonctions et methodes de
// production mesurees le 2026-09-16 dans les quatre racines (410 fichiers `.go` hors tests) ; un
// balayage qui en rend nettement moins n a pas trouve l arborescence (racine renommee, chemin
// relatif casse) et ne garde plus rien. Il doit ECHOUER bruyamment, pas rendre vert sur du vide.
const plancherFonctionsBalayees = 2300

// plafondsParFonction — LA TABLE DATEE DES FONCTIONS DE PRODUCTION DEJA AU-DELA DU SEUIL, avec
// leur longueur physique du 2026-09-16 (item 2.10.5). Cle : chemin relatif a `apps/go-api` en
// slash, puis `:`, puis le nom rendu par `nomDeFonction` (le namer DEJA canonique de ce paquet,
// `no_recomputed_film_context_test.go`) — soit `consumeByName` pour une fonction, soit
// `(*FilmContext).BipedSlots` pour une methode, recepteur compris.
//
// CHAQUE VALEUR NE PEUT QUE DESCENDRE. La reponse a « mon lot ajoute dix lignes ici » est d en
// sortir dix ailleurs dans la fonction, ou de l extraire. Mettre une fonction NEUVE dans cette
// table n est pas une reponse : elle est datee et fermee au 2026-09-16, elle recense la dette
// constatee ce jour-la, pas la dette a venir.
//
// Les huit entrees sont dans `grammar` et se lisent en deux familles :
//   - les sept aiguillages de composants et la boucle d inference — de longues suites de `case`
//     sur la grammaire du film, ou chaque branche consomme des bits dans un ordre impose par le
//     format. Elles se scinderont par famille de composants, dans un lot qui portera sa preuve
//     d equivalence (zero difference au corpus gate), jamais au fil de l eau ;
//   - `tableProfilInvariants`, qui est une table litterale d invariants par build.
var plafondsParFonction = map[string]int{
	"internal/games/halo_infinite/film/internal/grammar/dispatch_player.go:consumeCrewFlockAndMusicComponent":       146,
	"internal/games/halo_infinite/film/internal/grammar/dispatch_biped.go:consumeManagedAndObjectiveComponent":      143,
	"internal/games/halo_infinite/film/internal/grammar/dispatch_item.go:consumeItemAndTacmapComponent":             139,
	"internal/games/halo_infinite/film/internal/grammar/dispatch_player.go:consumePlayerTailAndGameEngineComponent": 122,
	"internal/games/halo_infinite/film/internal/grammar/dispatch_object.go:consumeByName":                           118,
	"internal/games/halo_infinite/film/internal/grammar/frame_infer.go:decodeInferLoop":                             114,
	"internal/games/halo_infinite/film/internal/grammar/dispatch_biped.go:consumeCaptureAndBipedComponent":          112,
	"internal/games/halo_infinite/film/internal/profile/profile_table.go:tableProfilInvariants":                     89,
}

// TestLongueurDesFonctionsDuFilmNeCroitPas : aucune fonction des racines surveillees ne depasse
// son plafond — 80 lignes physiques par defaut, sa longueur figee si elle est dans la table.
func TestLongueurDesFonctionsDuFilmNeCroitPas(t *testing.T) {
	longueurs := balayerLongueursFonctionsFilm(t)
	if len(longueurs) < plancherFonctionsBalayees {
		t.Fatalf("balayage muet : %d fonctions vues dans %v, plancher %d. "+
			"L arborescence a bouge ou le chemin relatif est casse — ce ratchet ne garde "+
			"plus rien et doit echouer bruyamment.",
			len(longueurs), racinesSurveilleesTaille, plancherFonctionsBalayees)
	}
	cles := make([]string, 0, len(longueurs))
	for cle := range longueurs {
		cles = append(cles, cle)
	}
	sort.Strings(cles)
	for _, cle := range cles {
		n := longueurs[cle]
		plafond, fige := plafondsParFonction[cle]
		if !fige {
			plafond = seuilLignesParFonction
		}
		if n <= plafond {
			continue
		}
		if fige {
			t.Errorf("%s : %d lignes physiques, plafond fige a %d (2026-09-16, item 2.10.5).\n"+
				"UN PLAFOND NE MONTE PAS. Sortir autant de lignes ailleurs dans la fonction, "+
				"ou l extraire — la scission d un aiguillage se fait dans un lot qui porte sa "+
				"preuve d equivalence, pas au fil de l eau.", cle, n, plafond)
			continue
		}
		t.Errorf("%s : %d lignes physiques, seuil %d (CLAUDE.md regle 5).\n"+
			"Extraire une sous-fonction. Mettre l entree dans plafondsParFonction N EST PAS "+
			"une reponse : cette table est datee et fermee au 2026-09-16, elle recense la "+
			"dette constatee ce jour-la, pas la dette a venir.\n"+
			"Rappel : `funlen` ne le verra jamais (ignore-comments vaut true en v2, D1 (2.7p)) "+
			"— ce ratchet est le seul instrument du depot sur ce seuil.",
			cle, n, seuilLignesParFonction)
	}
}

// TestPlafondsDeFonctionNeSontPasPerimes : une entree de la table qui designe une fonction
// disparue (renommee, deplacee, supprimee), ou qui est repassee sous le seuil, se RETIRE. Une
// allowlist perimee finit par autoriser n importe quoi (meme regle que les autres ratchets de ce
// paquet).
func TestPlafondsDeFonctionNeSontPasPerimes(t *testing.T) {
	longueurs := balayerLongueursFonctionsFilm(t)
	entrees := make([]string, 0, len(plafondsParFonction))
	for cle := range plafondsParFonction {
		entrees = append(entrees, cle)
	}
	sort.Strings(entrees)
	for _, cle := range entrees {
		n, vue := longueurs[cle]
		if !vue {
			t.Errorf("%s est au plafond mais n existe plus (renommee, deplacee ou supprimee) : "+
				"retirer l entree, ou la reecrire au nouveau chemin / nouveau nom.", cle)
			continue
		}
		if n <= seuilLignesParFonction {
			t.Errorf("%s : %d lignes physiques, soit sous le seuil de %d — retirer son entree "+
				"de plafondsParFonction. La fonction est rentree dans la regle, la table n a "+
				"plus de raison de la porter.", cle, n, seuilLignesParFonction)
		}
	}
}

// TestExclusionDuTagResearch exerce la garde du tag : elle n est pas une intention ecrite dans
// un commentaire, elle se verifie. Les trois cas qui comptent sont le prologue reconnu, la
// contrainte composee (`research && …`), et le PIEGE : un commentaire de CORPS qui cite le tag
// ne doit pas sortir le fichier du ratchet — c est pour cela que la lecture s arrete a `package`.
func TestExclusionDuTagResearch(t *testing.T) {
	cas := []struct {
		nom    string
		source string
		veut   bool
	}{
		{"prologue simple", "//go:build research\n\npackage x\n", true},
		{"contrainte composee", "//go:build research && !windows\n\npackage x\n", true},
		{"autre tag", "//go:build integration\n\npackage x\n", false},
		{"aucun tag", "package x\n", false},
		{"citation apres la clause package", "package x\n\n// cf. //go:build research\nfunc f() {}\n", false},
		{"prefixe trompeur", "//go:build researchy\n\npackage x\n", false},
	}
	for _, c := range cas {
		if got := estSousTagResearch([]byte(c.source)); got != c.veut {
			t.Errorf("%s : estSousTagResearch = %v, attendu %v", c.nom, got, c.veut)
		}
	}
}

// balayerLongueursFonctionsFilm rend, par cle `chemin/relatif.go:Nom` (le nom vient du namer
// canonique du paquet, recepteur compris pour une methode), la longueur PHYSIQUE de chaque
// fonction et methode de PRODUCTION des racines surveillees : ligne de l accolade fermante moins
// ligne du mot-cle `func`, plus un. Le commentaire de documentation qui precede la fonction n en
// fait PAS partie (il n est pas dans le corps) ; les commentaires et lignes vides INTERIEURS, si.
//
// Trois exclusions, toutes justifiees dans l en-tete :
//   - les `*_test.go` (le perimetre est la production — mesure des 93 contre 8) ;
//   - les fichiers sous `//go:build research` (instruments jetables qu aucun gate ne compile) ;
//   - les declarations sans corps (`func` externes, assembleur) : rien a mesurer.
func balayerLongueursFonctionsFilm(t *testing.T) map[string]int {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	out := map[string]int{}
	for _, racine := range racinesSurveilleesTaille {
		base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				nom := d.Name()
				if chemin != base && (strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			return releverFonctionsDuFichier(goAPIRoot, chemin, out)
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", base, err)
		}
	}
	return out
}

// releverFonctionsDuFichier ajoute a `out` les fonctions d un fichier. Le fichier est lu une
// seule fois : ses octets servent d abord au tag de build, puis au parseur (mode 0, sans les
// commentaires — `fd.Pos()` reste le mot-cle `func`, ce qui est bien la borne voulue).
func releverFonctionsDuFichier(goAPIRoot, chemin string, out map[string]int) error {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du paquet
	if err != nil {
		return err
	}
	if estSousTagResearch(blob) {
		return nil
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, blob, 0)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(goAPIRoot, chemin)
	if err != nil {
		return err
	}
	prefixe := filepath.ToSlash(rel) + ":"
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		n := fset.Position(fd.End()).Line - fset.Position(fd.Pos()).Line + 1
		cle := prefixe + nomDeFonction(fd)
		// Une cle peut se repeter dans un seul cas legal : plusieurs `func init()` dans le meme
		// fichier. On garde la plus longue — c est celle que le plafond doit borner.
		if precedent, deja := out[cle]; !deja || n > precedent {
			out[cle] = n
		}
	}
	return nil
}

// estSousTagResearch dit si le prologue du fichier porte `//go:build research`. La ligne de
// contrainte precede obligatoirement la clause `package`, donc la lecture s arrete la : chercher
// plus loin ferait mordre un commentaire de corps qui cite le tag.
func estSousTagResearch(blob []byte) bool {
	for _, ligne := range bytes.Split(blob, []byte("\n")) {
		ligne = bytes.TrimSpace(ligne)
		if bytes.HasPrefix(ligne, []byte("package ")) {
			return false
		}
		if !bytes.HasPrefix(ligne, []byte("//go:build")) {
			continue
		}
		for _, mot := range strings.Fields(string(ligne[len("//go:build"):])) {
			if mot == "research" {
				return true
			}
		}
	}
	return false
}
