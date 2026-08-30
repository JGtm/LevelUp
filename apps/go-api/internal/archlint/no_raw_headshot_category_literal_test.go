// Package archlint — no_raw_headshot_category_literal_test.go : ratchet G.1.
//
// Interdit tout littéral brut `"HeadshotMultiplier"` (valeur DANGEREUSE de
// `match_kill_events.source_category`) hors de ses deux propriétaires : `internal/domain/killscope`
// (le comparateur `IsHeadshotCategory`, qui l'exclut explicitement) et
// `internal/games/halo_infinite/film/killsource` (l'énumération `Category` qui la produit).
//
// POURQUOI `HeadshotMultiplier` SEUL, ET PAS AUSSI `"Headshot"` — CHOIX DÉLIBÉRÉ, PAS UN OUBLI.
// `"Headshot"` est AUSSI le nom d'une médaille du jeu (catalogue commendations/citations,
// `internal/ops/seed_citation_data.go`, `internal/ops/seed_demo_synthetic_meta.go`, fixtures de
// service) : un ratchet qui bannirait le mot bannirait pour toujours des données légitimes SANS
// AUCUN rapport avec `source_category`. `"HeadshotMultiplier"` n'a, lui, qu'UN sens dans tout le
// dépôt (constaté par grep avant d'écrire ce test) : le mesurer une seconde fois ailleurs ne
// peut être qu'une réécriture du filtre de lecture.
//
// POURQUOI CE TEST EXISTE — le rapport G.0 (2026-08-29) a mesuré l'oracle contre
// `match_participants.headshot_kills` (API officielle) : le filtre STRICT `= 'Headshot'` seul
// donne 99,3 % d'accord ; y ajouter `HeadshotMultiplier` (même famille de nom, PAS le même sens
// produit) fait chuter l'accord à 84,4 %. C'est la mesure qui coûte 15 points d'exactitude si un
// second endroit du dépôt écrit sa propre comparaison en l'incluant — silencieusement, sans
// erreur ni compteur : un kill mal classé ne fait planter rien, il ment juste.
//
// MODÈLE : internal/archlint/no_raw_kill_scope_literal_test.go (J4R-3), même mécanique (scan
// texte, propriétaires exemptés, allowlist datée et justifiée).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// headshotCategoryAllowlist : VIDE par construction (ratchet neuf, G.1 2026-08-30) — aucune
// copie ne préexistait avant ce lot. Une entrée ajoutée ici serait une seconde source de vérité
// pour une valeur qui décide de 15 points d'exactitude oracle : la justifier par écrit ET par
// une date, ou ne pas l'ajouter.
var headshotCategoryAllowlist = map[string]bool{}

// headshotCategoryRE matche la valeur en littéral Go OU SQL (simples ou doubles quotes),
// littéral ENTIER entre guillemets (donc "HeadshotMultiplier2" ou une phrase FR ne matchent pas
// — même garde que killScopeRE). PAS "Headshot" seul : cf. en-tête du fichier.
var headshotCategoryRE = regexp.MustCompile(`["'](HeadshotMultiplier)["']`)

// headshotCategoryOwners : les deux paquets qui ont le droit d'écrire cette valeur.
var headshotCategoryOwners = []string{
	"internal/domain/killscope/",
	"internal/games/halo_infinite/film/killsource/",
}

func TestNoRawHeadshotCategoryLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal
	apiRoot := filepath.Dir(internalRoot)                // .../apps/go-api

	var violations []string
	for _, racine := range []string{internalRoot, filepath.Join(apiRoot, "cmd")} {
		err := filepath.WalkDir(racine, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(apiRoot, path)
			rel = filepath.ToSlash(rel)
			for _, owner := range headshotCategoryOwners {
				if strings.HasPrefix(rel, owner) {
					return nil
				}
			}
			if headshotCategoryAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") ||
					strings.HasPrefix(trimmed, "--") {
					continue
				}
				if headshotCategoryRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", racine, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral brut de catégorie `source_category` interdit (G.1) — utiliser "+
			"`killscope.IsHeadshotCategory` / `killscope.CategoryHeadshot` (paquet "+
			"`internal/domain/killscope`, feuille sans import). Une comparaison écrite à la main "+
			"ailleurs (notamment en incluant `HeadshotMultiplier`) coûte 15 points d'exactitude "+
			"oracle (rapport G.0 : 99,3 %% avec le filtre strict, 84,4 %% avec) — SANS erreur ni "+
			"compteur :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestHeadshotCategoryAllowlistEntriesStayJustified (self-check, même leçon que
// TestKillScopeAllowlistEntriesStayJustified) — chaque clé de headshotCategoryAllowlist doit
// désigner un fichier EXISTANT qui matche RÉELLEMENT le motif. Vide aujourd'hui : ce test ne
// boucle sur rien, et c'est le comportement attendu tant que l'allowlist reste vide.
func TestHeadshotCategoryAllowlistEntriesStayJustified(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	for rel := range headshotCategoryAllowlist {
		data, err := os.ReadFile(filepath.Join(apiRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("headshotCategoryAllowlist : entrée %q pointe un fichier inexistant (%v) — sa "+
				"cause a disparu, retirer l'entrée.", rel, err)
			continue
		}
		if !headshotCategoryRE.Match(data) {
			t.Errorf("headshotCategoryAllowlist : entrée %q ne matche plus le motif — le littéral a "+
				"été retiré ou renommé, retirer l'entrée (allowlist décroissante, cible : VIDE).", rel)
		}
	}
}
