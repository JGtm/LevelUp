package grammar

// instruments_partages_inventaire_test.go — le corpus et les variables d'environnement de
// la mesure des trous d'inventaire (`invTrous*`), dont se sert la garde
// `inventory_position_i22_test.go`.
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	invTrousFilmsEnv  = "INV_FILMS"
	invTrousCacheEnv  = "INV_CACHE"
	invTrousSampleEnv = "INV_SAMPLE"
	invTrousOutEnv    = "INV_OUT"
	invTrousDigEnv    = "INV_DIG"
	invTrousI48Env    = "INV_I48"
)

// invTrousCorpus resout la liste des dossiers de film a mesurer.
func invTrousCorpus(t *testing.T) []string {
	t.Helper()
	if raw := strings.TrimSpace(os.Getenv(invTrousFilmsEnv)); raw != "" {
		var out []string
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	root := strings.TrimSpace(os.Getenv(invTrousCacheEnv))
	if root == "" {
		return nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("racine de cache illisible %s : %v", root, err)
	}
	var shorts []string
	for _, e := range ents {
		if e.IsDir() {
			shorts = append(shorts, e.Name())
		}
	}
	sort.Strings(shorts)
	n := invTrousEnvInt(invTrousSampleEnv, 20)
	if n <= 0 || n >= len(shorts) {
		n = len(shorts)
	}
	// ECHANTILLON REPARTI : un pas constant sur les prefixes tries, pas les n premiers — les
	// prefixes sont des hachages, mais prendre une tranche contigue reste un choix arbitraire
	// que rien ne justifie.
	step := len(shorts) / n
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, filepath.Join(root, shorts[i*step]))
	}
	return out
}

func invTrousEnvInt(key string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return def
}
