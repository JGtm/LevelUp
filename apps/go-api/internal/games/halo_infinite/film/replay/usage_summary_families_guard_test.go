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
	"strings"
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
	// QUATRIEME LISTE ECRITE (lot 4.3) : le PERIMETRE DU BILAN d'equipement. Depuis que la
	// jointure rang -> famille lit `abilityLabels[].family` au lieu d'une racine de libelle,
	// une faute de frappe ici ne se voit plus a la lecture — la famille ne s'apparie
	// simplement a rien, et la ligne d'issue disparait en silence.
	for _, f := range equipmentOutcomeFamilies {
		out[f] = true
	}
	return out
}

func TestUsageFamiliesMatchManifestValidator(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	// internal/games/halo_infinite/film/replay -> games -> games/mappings.
	loaderPath := filepath.Join(wd, "..", "..", "..", "mappings",
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

// famillesEngendrantUnePieceDuManifeste lit le manifeste des libellés de rejeu et rend
// les familles dont AU MOINS UN objet porte `kind = "deployed"` — la nature « n'existe
// qu'une fois déployé », que le valideur n'autorise qu'avec la provenance `sofa_parent`.
//
// Lecture LIGNE À LIGNE et non par expression sur tout le fichier : l'en-tête du
// manifeste cite `[[equipment_objects]]` dans ses commentaires, et un découpage sur le
// littéral y ouvrirait un bloc fantôme.
func famillesEngendrantUnePieceDuManifeste(t *testing.T, raw []byte) map[string]bool {
	t.Helper()
	champ := regexp.MustCompile(`^\s*(family|kind)\s*=\s*"([a-z0-9_]+)"`)
	out, objets := map[string]bool{}, 0
	var famille, nature string
	ferme := func() {
		if famille == "" {
			return
		}
		objets++
		if nature == "deployed" {
			out[famille] = true
		}
		famille, nature = "", ""
	}
	for _, ligne := range strings.Split(string(raw), "\n") {
		switch {
		case strings.TrimSpace(ligne) == "[[equipment_objects]]":
			ferme()
		case strings.HasPrefix(strings.TrimSpace(ligne), "["):
			ferme() // une autre table commence : le bloc courant est clos
		default:
			if m := champ.FindStringSubmatch(ligne); m != nil {
				if m[1] == "family" {
					famille = m[2]
				} else {
					nature = m[2]
				}
			}
		}
	}
	ferme()
	if objets < 15 {
		t.Fatalf("le garde-rail n'a lu que %d objets d'équipement — extraction cassée ?", objets)
	}
	return out
}

// TestUsageFamiliesWithSpawnedPieceMatchManifest — CINQUIÈME LISTE ÉCRITE (lot 5.5) :
// les familles qui ENGENDRENT UNE PIÈCE, seules dont le canal des poses voit le
// déploiement (usageFamiliesWithSpawnedPiece). Elle décide du côté « utilisé » de tout
// le bilan d'équipement, et sa donnée source est le MANIFESTE : la famille de chaque
// objet `kind = "deployed"`.
//
// CE GARDE-RAIL LIT LE MANIFESTE, pas une seconde table figée : un panneau ajouté à une
// autre famille là-bas (un jour où un second équipement engendrerait une pièce) doit
// FAIRE ÉCHOUER ce test, pas passer inaperçu — la famille resterait alors lue sur ses
// consommations alors que ses poses la mesurent. L'inverse aussi : retirer le `kind`
// des panneaux du mur ferait basculer le mur sur `spent` sans que rien ne le dise.
func TestUsageFamiliesWithSpawnedPieceMatchManifest(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	// internal/games/halo_infinite/film/replay -> games -> internal -> go-api -> apps -> racine du dépôt.
	manifest := filepath.Join(wd, "..", "..", "..", "..", "..", "..", "..",
		"config", "titles", "halo_infinite", "mappings", "replay_labels.toml")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("lecture du manifeste des libellés de rejeu: %v", err)
	}

	duManifeste := famillesEngendrantUnePieceDuManifeste(t, raw)
	if len(duManifeste) == 0 {
		t.Fatal("aucune famille `kind = deployed` au manifeste — le garde-rail doit être " +
			"adapté, pas supprimé : la règle d'usage_summary_outcomes.go n'aurait plus de source")
	}
	for f := range duManifeste {
		if !usageFamiliesWithSpawnedPiece[f] {
			t.Errorf("la famille %q engendre une pièce au manifeste (`kind = deployed`) mais "+
				"n'est pas dans usageFamiliesWithSpawnedPiece : son « utilisé » est lu sur ses "+
				"CONSOMMATIONS alors que ses POSES le mesurent (cf. usage_summary_families.go)", f)
		}
	}
	for f := range usageFamiliesWithSpawnedPiece {
		if !duManifeste[f] {
			t.Errorf("la famille %q est déclarée engendrer une pièce, mais AUCUN objet "+
				"`kind = deployed` du manifeste ne la porte — son « utilisé » serait lu sur des "+
				"poses qui mesurent un lâcher volontaire (rapport E0, question 5)", f)
		}
		// Et elle reste une famille du bilan : une famille hors bilan n'a pas de côté
		// « utilisé » à décider.
		if !estFamilleDuBilan(f) {
			t.Errorf("la famille %q engendre une pièce mais ne porte aucune ligne d'issue", f)
		}
	}
}
