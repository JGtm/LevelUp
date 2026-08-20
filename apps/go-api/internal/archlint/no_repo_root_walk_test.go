// no_repo_root_walk_test.go : interdit de RÉ-IMPLÉMENTER la remontée « depuis le cwd
// jusqu'au marqueur db_profiles.json ». Le helper canonique est title.FindRepoRoot()
// (internal/domain/title/repo_root.go) ; il existait déjà en deux copies identiques
// (cmd/mapquant-build, cmd/replay-build) au moment où une troisième allait naître —
// règle CLAUDE.md n°6 : à la 3e copie, on centralise ET on pose le garde-rail.
//
// Le fichier porte TROIS gardes, une par facon de se tromper de racine :
//   - TestNoDuplicateRepoRootWalk        : code de PRODUCTION — pas de remontee maison.
//   - TestNoAdHocRepoRootLadderInTests   : TESTS — pas d'echelle « ../../../.. » en dur.
//   - TestNoProdRepoRootHelperInTests    : TESTS — ni title.FindRepoRoot() ni la variable
//     d'environnement, qui font skipper en silence sur un checkout CI (ajoutee le
//     2026-08-19 : les deux premieres n'attrapaient pas le motif qui a cause R1-1).
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// repoRootWalkAllowlist : fichiers (chemin relatif depuis apps/go-api/) autorisés à porter
// une remontée. Le helper canonique, plus deux variantes ANTÉRIEURES (2026-07-26) qui ne
// sont pas des copies du même code : elles partent de l'exécutable, bornent la remontée et
// cherchent un autre marqueur. Les migrer exigerait de changer leur comportement au
// démarrage du serveur — hors périmètre du chantier rejeu ; à traiter avec le prochain
// passage sur la config.
var repoRootWalkAllowlist = map[string]bool{
	"internal/domain/title/repo_root.go":   true, // helper canonique
	"internal/config/config.go":            true, // autoDetectRepoRoot serveur (exe + cwd, marqueur .example)
	"cmd/migrate-to-shared-social/main.go": true, // remontée bornée depuis l'exe
}

// repoRootWalkMarkers : la remontée se reconnaît à la conjonction du marqueur de racine et
// de la boucle vers le parent. Les fichiers qui lisent seulement LEVELUP_REPO_ROOT ne sont
// pas visés (ils ne dupliquent rien).
var repoRootWalkMarkers = []string{"db_profiles.json", "filepath.Dir(dir)"}

func TestNoDuplicateRepoRootWalk(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	var violations []string
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if repoRootWalkAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			body := stripComments(string(data))
			for _, m := range repoRootWalkMarkers {
				if !strings.Contains(body, m) {
					return nil
				}
			}
			violations = append(violations, rel)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("remontée maison vers la racine du dépôt interdite — appeler "+
			"title.FindRepoRoot() :\n  %s", strings.Join(violations, "\n  "))
	}
}

// stripComments retire les lignes de commentaire : une explication citant db_profiles.json
// n'est pas une réimplémentation.
func stripComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// repoRootLadderAllowlist : fichiers de TEST (chemin relatif depuis apps/go-api/) encore
// autorises a ecrire l'echelle de remontee a la main. Les deux entrees sont ANTERIEURES au
// garde-rail (relevees le 2026-08-19) et hors du perimetre de gate du lot C-ter volet 2 :
// les migrer sans rejouer leur paquet serait un fix aveugle. Condition de reprise : au
// prochain passage sur ces paquets, remplacer l'echelle par testutil.RepoRoot() et retirer
// l'entree ici.
var repoRootLadderAllowlist = map[string]bool{
	"internal/analysis/filmdec/map_bounds_test.go": true, // echelle + Skip sur map_quant_bounds.json (versionne)
	"internal/ops/seed_citation_assets_test.go":    true, // const citationRepoRoot, racine des image_path seedes
}

// TestNoAdHocRepoRootLadderInTests : un test qui lit un fichier VERSIONNE localise la
// racine par testutil.RepoRoot(), pas par une echelle de « .. » ecrite a la main.
//
// Pourquoi c'est une regle et pas un gout : l'echelle en dur se trompe en SILENCE (elle
// tombe a cote, le test skippe, la garde ne tourne plus) — c'est le defaut R1-1 de la revue
// ronde 1 du lot C-ter volet 2, ou trois gardes centrales n'ont jamais tourne en CI. La
// remontee jusqu'a un marqueur gitignore (title.FindRepoRoot) a le meme defaut cote CI ;
// elle reste le helper des OUTILS, pas des tests de fichiers versionnes.
func TestNoAdHocRepoRootLadderInTests(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	// Motif CONSTRUIT (et non ecrit en clair) pour que ce fichier de garde ne se signale
	// pas lui-meme : quatre « .. » = la racine du depot depuis un paquet sous internal/.
	echelle := strings.Repeat("../", 3) + ".."

	var violations []string
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if repoRootLadderAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(stripComments(string(data)), echelle) {
				violations = append(violations, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("échelle de remontée vers la racine écrite à la main dans un test — "+
			"appeler testutil.RepoRoot() :\n  %s", strings.Join(violations, "\n  "))
	}
}

// repoRootInTestsAllowlist : fichiers de TEST (chemin relatif depuis apps/go-api/) encore
// autorises a situer la racine du depot autrement que par testutil.RepoRoot(). Releve DATE
// du 2026-08-19 (revue adversariale ronde 2, R2-2) ; chaque entree porte sa raison ET sa
// condition de reprise — une allowlist sans date se perime en silence.
var repoRootInTestsAllowlist = map[string]bool{
	// Instrument de RECHERCHE sous garde d'environnement (CTF_RESEARCH_FILMS) : il ne
	// tourne jamais en CI, donc le helper de production y est sans consequence. Reprise :
	// si la garde d'environnement tombe, migrer vers testutil.RepoRoot().
	"internal/analysis/replay/ctf_research_test.go": true,
	// Lit data/cache/replays/halo_infinite/000d5950.json — un CACHE, NON versionne
	// (git ls-files vide au 2026-08-19). Ici le skip est le comportement JUSTE : le fichier
	// est vraiment absent d'un checkout. Reprise : s'il devient versionne, migrer + Fatal.
	"internal/himap/carte_oracle_gamefiles_test.go": true,
	// (2026-08-20, lot hygiene H4) internal/mapdecoupe/oracle_corpus_test.go et
	// cmd/mapcallouts-build/classify_test.go migres vers testutil.RepoRoot() + t.Fatalf —
	// les deux entrees qui etaient ici ont disparu avec la cause (cf. historique git).
}

// repoRootInTestsMotifs : les deux voies par lesquelles un test retombe sur une racine
// INTROUVABLE en integration. title.FindRepoRoot() cherche db_profiles.json (gitignore,
// absent d'un checkout neuf) ou LEVELUP_REPO_ROOT (jamais pose en CI) ; lire la variable a
// la main a exactement le meme defaut. Les deux menent au meme accident : t.Skip.
//
// C'est l'APPEL qui est vise, pas l'import de `title` — ce paquet sert aussi DefaultSlug et
// PathResolver, qui n'ont rien a voir ; interdire l'import ferait tomber des dizaines de
// tests legitimes. Un test ne peut pas appeler le helper sans ecrire cet appel.
var repoRootInTestsMotifs = []string{
	"title.FindRepoRoot",
	`os.Getenv("LEVELUP_REPO_ROOT")`,
}

// TestNoProdRepoRootHelperInTests : un test ne situe PAS la racine du depot avec le helper
// de production ni avec la variable d'environnement.
//
// Pourquoi cette garde en plus de TestNoAdHocRepoRootLadderInTests : celle-ci ne cherchait
// que l'echelle « ../../../.. » ecrite a la main. Elle n'a donc PAS attrape le motif qui a
// cause R1-1 — title.FindRepoRoot() suivi d'un t.Skip — et cinq gardes centrales du lot
// C-ter volet 2 (catalogues map_callouts.json / map_quant_bounds.json, fonds de carte) sont
// restees muettes en CI jusqu'au 2026-08-19, certaines sous un commentaire affirmant
// l'inverse. Une garde qui n'attrape pas le defaut qu'elle documente ne garde rien.
func TestNoProdRepoRootHelperInTests(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	// CE fichier CITE les motifs interdits (messages d'erreur, table des motifs) sans en
	// appeler aucun : il s'exclut par son chemin — meme intention que le motif CONSTRUIT de
	// TestNoAdHocRepoRootLadderInTests, exprimee une seule fois.
	moi, _ := filepath.Rel(goAPIRoot, thisFile)
	moi = filepath.ToSlash(moi)

	violations := map[string][]string{}
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if rel == moi || repoRootInTestsAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			body := stripComments(string(data))
			for _, m := range repoRootInTestsMotifs {
				if strings.Contains(body, m) {
					violations[rel] = append(violations[rel], m)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) == 0 {
		return
	}
	fichiers := make([]string, 0, len(violations))
	for rel := range violations {
		fichiers = append(fichiers, rel+" ("+strings.Join(violations[rel], ", ")+")")
	}
	sort.Strings(fichiers)
	t.Errorf("racine du dépôt situee par le helper de PRODUCTION dans un test — le test se "+
		"skippe alors en silence en CI ; appeler testutil.RepoRoot() et échouer par "+
		"t.Fatalf :\n  %s", strings.Join(fichiers, "\n  "))
}
