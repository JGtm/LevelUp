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
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// importsMoteurSync : les chemins d'import qui font entrer le moteur de sync (et donc les seams
// title-owned) dans un binaire. Le second motif couvre les sous-paquets (`internal/sync/skill`,
// `internal/sync/v2`, …) dont les fail-loud sont les mêmes.
var importsMoteurSync = []string{
	`"levelup/go-api/internal/sync"`,
	`"levelup/go-api/internal/sync/`,
}

// appelSeams : la forme exacte attendue dans le binaire.
const appelSeams = "titleseams.RegisterAll("

// binairesSansSeamsAutorises — ALLOWLIST VIDE (ratchet anti-résurrection). Une entrée ici
// signifie « ce binaire touche au moteur mais ne câble pas les seams » : elle doit porter une
// date et la démonstration qu'aucun chemin de post-sync ni de migration n'est atteint.
var binairesSansSeamsAutorises = map[string]bool{}

// TestBinairesSyncCablentLesSeams — LE RATCHET.
func TestBinairesSyncCablentLesSeams(t *testing.T) {
	racine := racineGoAPI(t)
	cmdRacine := filepath.Join(racine, "cmd")

	// dossier -> importe le moteur ; dossier -> câble les seams.
	importe := map[string]bool{}
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
			texte := string(brut)
			for _, imp := range importsMoteurSync {
				if strings.Contains(texte, imp) {
					importe[dossier] = true
				}
			}
			if strings.Contains(texte, appelSeams) {
				cable[dossier] = true
			}
		}
	}

	if len(importe) == 0 {
		t.Fatal("aucun binaire de cmd/ n'importe le moteur de sync — le balayage s'est cassé")
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
		t.Errorf("cmd/%s importe le moteur de sync sans câbler les seams title-owned — "+
			"ajouter `titleseams.RegisterAll(\"\")` (ou la racine config/titles/{slug} si le "+
			"binaire seed des catalogues) au début de main(), import "+
			"\"levelup/go-api/internal/games/titleseams\" (plan 2026-09-16, étape 1)", dossier)
	}
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
