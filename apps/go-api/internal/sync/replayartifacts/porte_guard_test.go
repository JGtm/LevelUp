package replayartifacts

// porte_guard_test.go — LE RATCHET DE LA PORTE DE CAPABILITY.
//
// Le motif « lire les capabilities du titre, WARN si le TOML est illisible, DEBUG si la clé
// manque » a vécu en QUATRE exemplaires dans ce paquet avant d'être centralisé le 2026-09-13
// (revue adversariale, constat C4). Une factorisation sans garde-rail re-diverge : la
// cinquième famille recopierait la troisième, et la nuance « incident ≠ configuration » se
// perdrait dans l'une des copies sans que rien ne rougisse.
//
// CE TEST INTERDIT DONC LE LITTÉRAL `LoadCapabilityMap` hors de `porte.go`, dans les fichiers
// de PRODUCTION de ce paquet. Les tests, eux, restent libres : un test qui construit sa
// fixture de capabilities n'ouvre aucune porte.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// porteSourceUnique : le SEUL fichier de production autorisé à appeler LoadCapabilityMap.
const porteSourceUnique = "porte.go"

func TestPorteCapability_UneSeuleLecture(t *testing.T) {
	t.Parallel()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet: %v", err)
	}
	var fautifs []string
	vuDansLeHelper := false
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(filepath.Clean(nom))
		if rerr != nil {
			t.Fatalf("lecture de %s: %v", nom, rerr)
		}
		if !strings.Contains(string(src), "LoadCapabilityMap") {
			continue
		}
		if nom == porteSourceUnique {
			vuDansLeHelper = true
			continue
		}
		fautifs = append(fautifs, nom)
	}
	if len(fautifs) > 0 {
		t.Errorf("LoadCapabilityMap appelé hors de %s : %v\n"+
			"La porte de capability vit en UN seul exemplaire (porteCapability) : une copie "+
			"reperd la distinction « TOML illisible = incident » / « clé absente = "+
			"configuration », et c'est elle qui décide si le lot compte en échecs.",
			porteSourceUnique, fautifs)
	}
	if !vuDansLeHelper {
		t.Errorf("%s n'appelle plus LoadCapabilityMap — le garde-rail ne garde plus rien "+
			"(helper renommé ou déplacé sans mettre ce test à jour)", porteSourceUnique)
	}
}
