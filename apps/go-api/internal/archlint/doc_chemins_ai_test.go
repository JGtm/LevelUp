package archlint

// doc_chemins_ai_test.go — deux ratchets de DOCUMENTATION sur tout le module (J12.5 du plan
// `.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, 2026-09-30).
//
// Un commentaire qui renvoie à un document ou à une commande est un contrat : le lecteur le
// suit. L'audit du décodeur (`.ai/AUDIT_DECODEUR_FILM_2026-09-24.md`, faiblesse 11) comptait des
// chemins `.ai/` morts (les documents clos déménagent sous `.ai/V7.5/` ou `.ai/archive/`) et 170
// commandes `go test` qui visaient un paquet disparu. Rien ne les voyait : un commentaire ne
// compile pas.
//
//   - TestCheminsAiCitesDansLeCodeExistent : tout chemin `.ai/…` cité dans un COMMENTAIRE Go du
//     module existe dans le dépôt. Les chaînes de caractères ne sont pas lues (ce sont du code :
//     les changer changerait un comportement).
//   - TestAucunCommentaireNeViseLAncienPaquetFilmdec : le littéral de l'ancien chemin du
//     décodeur n'apparaît plus dans aucun fichier Go du module.

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// cheminAiMotif : un chemin `.ai/…` dans un commentaire. Il s'arrête au premier caractère qui
// ne peut pas appartenir à un nom de fichier du dépôt (blanc, ponctuation de phrase, `:` d'un
// numéro de ligne, `§`, backtick, parenthèse).
var cheminAiMotif = regexp.MustCompile("\\.ai/[A-Za-z0-9_.\\-/]*")

// racinesLocalesAi : préfixes `.ai/` IGNORÉS par git (`.gitignore`) — des dépôts locaux de
// données brutes qu'aucun clone ne porte. Un commentaire peut les nommer (il décrit où poser le
// dump), le ratchet ne peut pas vérifier leur existence en CI. Entrée datée :
//
//   - `.ai/re_dump/` (2026-09-30) : dumps `.mvar` et navmesh extraits de l'installation du jeu,
//     `.gitignore` ligne « .ai/re_dump/ ».
var racinesLocalesAi = []string{".ai/re_dump/"}

// cheminsAiMortsTolérés : couples (fichier, chemin) morts que ce lot n'a PAS le droit de
// corriger. Datés, à vider.
//
//   - 2026-09-30, J12.5 : les fichiers `*_research_test.go` et `film/research/` sont modifiés au
//     même moment par le lot J12.7 (tag `research`, autre worktree) ; leurs chemins morts sont
//     rendus au rapport du lot et se corrigent à la fusion des deux lots. Retrait : quand J12.7
//     est fusionné (critère : cette table vide, le test ci-dessous signale toute entrée qui ne
//     sert plus).
var cheminsAiMortsTolérés = map[string]bool{
	"internal/games/halo_infinite/film/internal/grammar/f0_103_contexte_research_test.go|.ai/PLAN_FINITIONS_2026-09-13.md":                true,
	"internal/games/halo_infinite/film/internal/grammar/vitesse_filtre_research_test.go|.ai/PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md": true,
	"internal/games/halo_infinite/film/replay/duels_ouverture_research_test.go|.ai/PLAN_DUELS_PORTEE_2026-09-06.md":                       true,
	"internal/games/halo_infinite/film/replay/ground_weapon_research_test.go|.ai/SUIVI_REPLAY_2D.md":                                      true,
	"internal/games/halo_infinite/film/research/grenadeids/doc.go|.ai/PLAN_DECODEUR_FILM_2026-09-13.md":                                   true,
	"internal/games/halo_infinite/film/research/grenadeids/doc.go|.ai/ADDENDUM_ETAT_DE_L_ART_2026-07-26.md":                               true,
	"internal/sync/killcollector/duels_bouclier_research_test.go|.ai/PLAN_DUELS_PORTEE_2026-09-06.md":                                     true,
}

// TestCheminsAiCitesDansLeCodeExistent — tout chemin `.ai/…` cité dans un commentaire Go du
// module existe.
//
// Un chemin qui finit par `_` ou `-`, ou suivi de `*`, `<` ou `{`, est un GABARIT
// (`NOTE_3_6_*`, `ENGAGEMENT_CALIBRATION_<titre>_<date>.md`) : il vaut s'il désigne au moins un
// fichier existant. Un nom de dossier qui contient une espace (`.ai/archive/V7/LUSR v2/`) est
// recollé à son suffixe avant d'être déclaré mort.
//
// Mutation qui doit le faire rougir : ajouter à `internal/sync/scope.go` un commentaire qui cite
// un plan `.ai/` absent (vérifié le 2026-09-30).
func TestCheminsAiCitesDansLeCodeExistent(t *testing.T) {
	goAPI, depot := racinesDuModule(t)
	morts := map[string]bool{}
	vus := 0
	balayerGo(t, goAPI, func(rel, chemin string) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, chemin, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Errorf("%s : analyse impossible : %v", rel, err)
			return
		}
		for _, groupe := range f.Comments {
			for _, c := range groupe.List {
				for _, cite := range cheminsAiDuTexte(c.Text) {
					vus++
					if cheminAiVivant(depot, cite) {
						continue
					}
					cle := rel + "|" + cite.chemin
					if cheminsAiMortsTolérés[cle] {
						morts[cle] = true
						continue
					}
					t.Errorf("%s:%d cite %s, qui n'existe pas dans le dépôt — pointer vers "+
						"l'emplacement actuel du document (souvent sous .ai/V7.5/ ou .ai/archive/)",
						rel, fset.Position(c.Pos()).Line, cite.chemin)
				}
			}
		}
	})
	if vus < 300 {
		t.Errorf("%d chemin(s) .ai/ lus dans les commentaires, plancher 300 (mesure du "+
			"2026-09-30 : plus de 700) — le balayage ne lit plus le module", vus)
	}
	var perimees []string
	for cle := range cheminsAiMortsTolérés {
		if !morts[cle] {
			perimees = append(perimees, cle)
		}
	}
	sort.Strings(perimees)
	for _, cle := range perimees {
		t.Errorf("entrée tolérée qui ne sert plus (chemin corrigé ou fichier déplacé) : %s — "+
			"la retirer de cheminsAiMortsTolérés", cle)
	}
}

// ancienPaquetFilmdec : l'ancien chemin du décodeur, découpé pour que ce fichier ne le porte
// pas lui-même. Le paquet a éclaté en `film/internal/{source,profile,grammar,facts}`,
// `film/replay` et `film/decfilm` (ADR 0034, M2).
var ancienPaquetFilmdec = "film/" + "filmdec/"

// TestAucunCommentaireNeViseLAncienPaquetFilmdec — aucun fichier Go du module ne cite
// l'ancien chemin du décodeur. Ces citations sont des commandes `go test` d'en-tête : copiées
// telles quelles, elles échouaient (« directory not found »).
//
// Mutation qui doit le faire rougir : réécrire une commande d'en-tête de
// `film/internal/grammar/fuzz_records_test.go` vers l'ancien chemin (vérifié le 2026-09-30).
func TestAucunCommentaireNeViseLAncienPaquetFilmdec(t *testing.T) {
	goAPI, _ := racinesDuModule(t)
	tolere := 0
	balayerGo(t, goAPI, func(rel, chemin string) {
		buf, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
		if err != nil {
			t.Errorf("%s : %v", rel, err)
			return
		}
		n := strings.Count(string(buf), ancienPaquetFilmdec)
		if n == 0 {
			return
		}
		if fichierDuLotResearch(rel) {
			tolere += n
			return
		}
		t.Errorf("%s cite %d fois %q, qui n'existe plus — viser le paquet actuel "+
			"(film/internal/grammar/, film/replay/, …)", rel, n, ancienPaquetFilmdec)
	})
	// Plafond daté (2026-09-30, J12.5) : les 62 citations restantes vivent dans des fichiers de
	// recherche modifiés au même moment par le lot J12.7 ; elles se corrigent à la fusion des
	// deux lots, puis ce plafond passe à 0 et `fichierDuLotResearch` disparaît.
	if tolere > 62 {
		t.Errorf("%d citation(s) de %q dans les fichiers de recherche, plafond 62 (2026-09-30) "+
			"— n'en ajouter aucune", tolere, ancienPaquetFilmdec)
	}
}

// fichierDuLotResearch : les fichiers que le lot J12.7 modifie en parallèle (2026-09-30).
func fichierDuLotResearch(rel string) bool {
	return strings.HasSuffix(rel, "_research_test.go") ||
		strings.Contains(rel, "halo_infinite/film/research/")
}

type citationAi struct {
	chemin  string
	gabarit bool
	suite   string // texte qui suit le chemin, pour recoller un nom de dossier à espace
}

// cheminsAiDuTexte extrait les chemins `.ai/…` d'un commentaire.
func cheminsAiDuTexte(texte string) []citationAi {
	var out []citationAi
	for _, loc := range cheminAiMotif.FindAllStringIndex(texte, -1) {
		if loc[0] > 0 && estLettreOuChiffre(texte[loc[0]-1]) {
			continue // `claude.ai/…`, `openai/…` : un domaine, pas le dossier du dépôt
		}
		chemin := strings.TrimRight(texte[loc[0]:loc[1]], ".")
		suite := texte[loc[0]+len(chemin):]
		suivant := ""
		if loc[1] < len(texte) {
			suivant = texte[loc[1] : loc[1]+1]
		}
		gabarit := strings.HasSuffix(chemin, "_") || strings.HasSuffix(chemin, "-") ||
			suivant == "*" || suivant == "<" || suivant == "{"
		chemin = strings.TrimRight(chemin, "/")
		if chemin == ".ai" || chemin == ".ai/" {
			continue
		}
		out = append(out, citationAi{chemin: chemin, gabarit: gabarit, suite: suite})
	}
	return out
}

func estLettreOuChiffre(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// cheminAiVivant dit si la citation désigne un fichier ou un dossier du dépôt.
func cheminAiVivant(depot string, c citationAi) bool {
	for _, r := range racinesLocalesAi {
		if strings.HasPrefix(c.chemin+"/", r) {
			return true
		}
	}
	abs := filepath.Join(depot, filepath.FromSlash(c.chemin))
	if _, err := os.Stat(abs); err == nil {
		return true
	}
	if c.gabarit {
		correspondances, _ := filepath.Glob(abs + "*")
		return len(correspondances) > 0
	}
	// Dossier à espace : « …/V7/LUSR v2/X.md » est lu « …/V7/LUSR » suivi de « v2/X.md ».
	if strings.HasPrefix(c.suite, " ") {
		reste := cheminAiMotif.FindString(".ai/" + strings.TrimPrefix(c.suite, " "))
		if reste != "" {
			recolle := c.chemin + " " + strings.TrimRight(strings.TrimPrefix(reste, ".ai/"), ".")
			if _, err := os.Stat(filepath.Join(depot, filepath.FromSlash(recolle))); err == nil {
				return true
			}
		}
	}
	return false
}

// racinesDuModule rend `apps/go-api` et la racine du dépôt.
func racinesDuModule(t *testing.T) (goAPI, depot string) {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPI = filepath.Dir(filepath.Dir(filepath.Dir(ici)))
	return goAPI, filepath.Dir(filepath.Dir(goAPI))
}

// balayerGo appelle `visiter` pour chaque fichier `.go` du module (tests compris), avec son
// chemin relatif à `apps/go-api` en slash. Mêmes répertoires sautés que `balayerTests`.
func balayerGo(t *testing.T, goAPI string, visiter func(rel, chemin string)) {
	t.Helper()
	err := filepath.WalkDir(goAPI, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != goAPI && (dossiersInvisiblesAuGo[nom] ||
				strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(goAPI, chemin)
		visiter(filepath.ToSlash(rel), chemin)
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module (%s) : %v", goAPI, err)
	}
}
