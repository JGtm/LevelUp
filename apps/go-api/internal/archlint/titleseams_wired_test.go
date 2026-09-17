package archlint

// titleseams_wired_test.go — TOUT `package main` QUI IMPORTE LE MOTEUR DE SYNC CÂBLE LES SEAMS.
//
// POURQUOI (2026-09-16). Les seams title-owned (provider des étapes de migration, racine des
// jalons Halo 5, traductions de rangs, classifiers LUSR et famille objectif) n'étaient posés
// que par `cmd/server/main.go`. Mesure du jour sur un `levelup sync-full` : `post-sync: PANIC
// récupéré … classifier LUSR non câblé` (fail-loud MT-15,
// `internal/sync/skill/skill_chain_provider.go`) → `perf_scores=0 lusr=0 citations=0
// dominance=0` sur toute la passe. Tout binaire CLI qui touche au moteur avait ce trou ; les
// `backfill` le masquaient en recalculant après coup.
//
// LA RÈGLE. Un répertoire de `cmd/` dont un fichier NON-test importe `internal/sync` ou un de
// ses sous-paquets contient, dans un fichier non-test du MÊME répertoire, un appel
// `titleseams.RegisterAll(`. L'allowlist est VIDE et doit le rester : un binaire qui touche au
// moteur sans câbler les seams produit des scores muets, pas une erreur.
//
// Mutation qui doit le faire rougir : retirer l'appel de `cmd/levelup/main.go`.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// paquetsFailLoud : les paquets dont la présence dans les dépendances TRANSITIVES d.un binaire
// y fait entrer un fail-loud title-owned — le moteur lui-même (étapes de migration title-owned,
// post-sync) et la chaîne de classification LUSR (panic MT-15,
// `internal/sync/skill/skill_chain_provider.go`).
//
// La correspondance est EXACTE, pas par préfixe : 42 binaires dépendent de ces deux paquets,
// 84 d.un sous-paquet quelconque de `internal/sync/`. Les 42 autres ne tirent que des briques
// sans seam (`haloclient`, `matchflags`, `schemadrift`, …) : leur imposer `RegisterAll` ne
// protégerait de rien et diluerait le ratchet en bruit (mesuré le 2026-09-16).
var paquetsFailLoud = []string{
	"levelup/go-api/internal/sync",
	"levelup/go-api/internal/sync/skill",
}

// prefixeCmd : préfixe des chemins d.import des binaires du module.
const prefixeCmd = "levelup/go-api/cmd/"

// appelSeams : la forme exacte attendue dans le binaire.
const appelSeams = "titleseams.RegisterAll("

// binairesSansSeamsAutorises — ALLOWLIST VIDE (ratchet anti-résurrection). Une entrée ici
// signifie « ce binaire touche au moteur mais ne câble pas les seams » : elle doit porter une
// date et la démonstration qu'aucun chemin de post-sync ni de migration n'est atteint.
var binairesSansSeamsAutorises = map[string]bool{}

// TestBinairesSyncCablentLesSeams — LE RATCHET.
//
// CRITÈRE : les dépendances TRANSITIVES, mesurées par un seul `go list` (2,7 s mesuré le
// 2026-09-16, Windows compris). L'ancien critère ne regardait que l'import DIRECT de
// `internal/sync` dans les fichiers de `cmd/<binaire>/` : dix binaires embarquaient le moteur
// par un intermédiaire (`internal/ops`, `internal/games/halo_5/...`, `internal/api/...`) et
// passaient sous le radar — ils ont reçu `titleseams.RegisterAll("")` avec ce changement.
func TestBinairesSyncCablentLesSeams(t *testing.T) {
	racine := racineGoAPI(t)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("`go` absent du PATH : le critère transitif exige `go list` (ratchet non exécuté)")
	}

	// UNE SEULE exécution pour tout cmd/. `-e` tolère les paquets dont les contraintes de
	// build excluent tous les fichiers sur cette plateforme : ils rendent une ligne sans
	// dépendances, ce qui est le verdict correct (ils n'embarquent rien).
	cmd := exec.Command("go", "list", "-e", "-f", "{{.ImportPath}} {{.Name}} {{.Deps}}", "./cmd/...")
	cmd.Dir = racine
	sortie, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list ./cmd/... : %v", err)
	}

	importe := map[string]bool{}
	for _, ligne := range strings.Split(string(sortie), "\n") {
		champs := strings.Fields(strings.TrimSpace(ligne))
		if len(champs) < 2 || champs[1] != "main" {
			continue
		}
		dossier := strings.TrimPrefix(champs[0], prefixeCmd)
		if dossier == champs[0] {
			continue // paquet main hors de cmd/
		}
		if i := strings.Index(dossier, "/"); i >= 0 {
			dossier = dossier[:i]
		}
		for _, dep := range champs[2:] {
			dep = strings.Trim(dep, "[]")
			if slices.Contains(paquetsFailLoud, dep) {
				importe[dossier] = true
				break
			}
		}
	}

	cable := binairesQuiCablent(t, filepath.Join(racine, "cmd"))

	if len(importe) == 0 {
		t.Fatal("aucun binaire de cmd/ ne dépend du moteur de sync — le balayage s'est cassé")
	}

	var fautifs []string
	for dossier := range importe {
		if cable[dossier] || binairesSansSeamsAutorises[dossier] {
			continue
		}
		fautifs = append(fautifs, dossier)
	}
	sort.Strings(fautifs)
	for _, dossier := range fautifs {
		t.Errorf("cmd/%s dépend du moteur de sync (transitivement ou non) sans câbler les seams "+
			"title-owned — ajouter `titleseams.RegisterAll(\"\")` (ou la racine config/titles/{slug} "+
			"si le binaire seed des catalogues) au début de main(), import "+
			"\"levelup/go-api/internal/games/titleseams\" (plan 2026-09-16, étape 1 ; critère "+
			"transitif : plan robustesse, étape 5)", dossier)
	}
	t.Logf("binaires de cmd/ dépendant du moteur de sync : %d (tous câblent les seams)", len(importe))
}

// binairesQuiCablent rend l'ensemble des dossiers de cmd/ dont un fichier non-test appelle
// titleseams.RegisterAll(.
func binairesQuiCablent(t *testing.T, cmdRacine string) map[string]bool {
	t.Helper()
	cable := map[string]bool{}
	entrees, err := os.ReadDir(cmdRacine)
	if err != nil {
		t.Fatalf("lecture de cmd/ : %v", err)
	}
	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		dossier := e.Name()
		fichiers, err := os.ReadDir(filepath.Join(cmdRacine, dossier))
		if err != nil {
			t.Fatalf("lecture de cmd/%s : %v", dossier, err)
		}
		for _, f := range fichiers {
			nom := f.Name()
			if f.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
				continue
			}
			brut, err := os.ReadFile(filepath.Join(cmdRacine, dossier, nom))
			if err != nil {
				t.Fatalf("lecture de cmd/%s/%s : %v", dossier, nom, err)
			}
			if strings.Contains(string(brut), appelSeams) {
				cable[dossier] = true
			}
		}
	}
	return cable
}

// TestSeamsPosesUniquementParTitleseams — aucun binaire ne repose les seams à la main.
//
// Le corollaire du ratchet précédent : si un `cmd/` réécrit `SetLUSRChainClassifier(` lui-même,
// la liste des huit seams se remet à diverger entre binaires (c'est exactement l'état d'avant le
// 2026-09-16, où seul le serveur en tenait une copie complète).
func TestSeamsPosesUniquementParTitleseams(t *testing.T) {
	racine := racineGoAPI(t)
	// Les quatre seams de CLASSIFICATION : ceux dont l'absence produit le panic MT-15 et
	// dont la liste divergeait entre binaires (certains posaient le défaut Infinite sans la
	// variante h5 — tous les modes h5 collapsés dans arena_slayer, sans un mot). Les seams
	// SetTitleStepsProvider / SetCareerRankTranslationsProvider ne sont PAS ici : des
	// binaires qui n'embarquent pas le moteur de sync (h5-metadata-fetch, h5-read-smoke,
	// seed-rank-translations, snapshot-world-leaderboard) les posent légitimement seuls,
	// et les leur imposer via RegisterAll y ferait entrer internal/sync pour rien.
	motifs := []string{
		"SetLUSRChainClassifier(",
		"SetLUSRChainClassifierForTitle(",
		"SetObjectiveFamilyClassifier(",
		"SetObjectiveFamilyClassifierForTitle(",
	}
	err := filepath.Walk(filepath.Join(racine, "cmd"), func(chemin string, info os.FileInfo, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if info.IsDir() || !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return nil
		}
		brut, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(racine, chemin)
		rel = filepath.ToSlash(rel)
		for _, motif := range motifs {
			if strings.Contains(string(brut), motif) {
				t.Errorf("%s pose le seam %q à la main — passer par titleseams.RegisterAll "+
					"(source unique, plan 2026-09-16, étape 1)", rel, motif)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de cmd/ : %v", err)
	}
}
