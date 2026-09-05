package replay

// usage_summary_families_guard_test.go — GARDE-RAIL DE COHÉRENCE : les trois listes
// écrites de usage_summary_families.go (grenades, capacités portées, bonus) ne
// doivent porter QUE des familles que le manifeste ADMET (liste fermée
// `equipmentFamilies` du valideur games/mappings/loader_replay_labels_equipment.go)
// — une faute de frappe rendrait une entrée morte en silence. Ce garde ne vérifie
// PAS l'exhaustivité (elle est structurelle : usageFamilyIsDeployable est la
// négation des trois listes, toute famille nouvelle tombe « déployable » par
// défaut) — cf. l'en-tête de usage_summary_families.go, revue adversariale 2026-09-04.
//
// LE TEST LIT LE FICHIER GO DU VALIDEUR, comme le fait déjà le garde-rail web
// (placementFamily.guard.test.ts) : la liste fermée y est LA source, et un
// troisième exemplaire figé ici re-divergerait.

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// familleConnueDuResume dit si la famille est couverte par UNE décision écrite du
// résumé : grenade, capacité portée, bonus — ou déployable par usageFamilyIsDeployable
// (qui est la négation des trois listes : toute famille tombe donc quelque part).
// Ce que ce garde vérifie n'est PAS l'exhaustivité (elle est structurelle) mais que
// les TROIS listes écrites ne portent QUE des familles que le manifeste admet — une
// faute de frappe dans une liste la rendrait morte en silence.
func famillesEcritesDuResume() map[string]bool {
	out := map[string]bool{}
	for f := range usageGrenadeFamilies {
		out[f] = true
	}
	for f := range usageCarriedCapacityFamilies {
		out[f] = true
	}
	for f := range usagePowerupFamilies {
		out[f] = true
	}
	return out
}

func TestUsageFamiliesMatchManifestValidator(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	// internal/analysis/replay -> internal -> games/mappings.
	loaderPath := filepath.Join(wd, "..", "..", "games", "mappings",
		"loader_replay_labels_equipment.go")
	raw, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("lecture du valideur de familles: %v", err)
	}

	// Le bloc `var equipmentFamilies = map[string]bool{ ... }` : chaque clé citée.
	bloc := regexp.MustCompile(`(?s)var equipmentFamilies = map\[string\]bool\{(.*?)\n\}`).
		FindSubmatch(raw)
	if bloc == nil {
		t.Fatal("bloc equipmentFamilies introuvable dans le valideur — le garde-rail doit être adapté, pas supprimé")
	}
	admises := map[string]bool{}
	for _, m := range regexp.MustCompile(`"([a-z0-9_]+)"\s*:\s*true`).FindAllSubmatch(bloc[1], -1) {
		admises[string(m[1])] = true
	}
	// equipFamilyOther est une constante ("other"), pas un littéral du bloc.
	admises["other"] = true
	if len(admises) < 10 {
		t.Fatalf("le garde-rail n'a lu que %d familles admises — extraction cassée ?", len(admises))
	}

	for f := range famillesEcritesDuResume() {
		if !admises[f] {
			t.Errorf("la famille %q des tables du résumé n'est pas admise par le manifeste "+
				"(faute de frappe ? famille retirée ?)", f)
		}
	}

	// Sens inverse : toute famille admise doit tomber dans exactement un seau —
	// structurellement vrai (déployable = négation des trois listes), mais on
	// vérifie qu'aucune famille n'est dans DEUX listes à la fois.
	for f := range admises {
		n := 0
		if usageGrenadeFamilies[f] {
			n++
		}
		if usageCarriedCapacityFamilies[f] {
			n++
		}
		if usagePowerupFamilies[f] {
			n++
		}
		if n > 1 {
			t.Errorf("la famille %q est classée dans %d listes du résumé — une seule est permise", f, n)
		}
	}
}
