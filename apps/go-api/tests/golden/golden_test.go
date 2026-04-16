package golden_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// goldenDir retourne le chemin absolu du dossier golden_values.
func goldenDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Join(filepath.Dir(file), "..", "fixtures", "golden_values")
}

// loadGolden charge un fichier golden et le décode en map[string]any.
func loadGolden(t *testing.T, filename string) map[string]any {
	t.Helper()
	path := filepath.Join(goldenDir(t), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read golden file %s: %v", filename, err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("cannot parse golden file %s: %v", filename, err)
	}
	return result
}

// assertKeyExists vérifie qu'une clé existe dans la réponse.
func assertKeyExists(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Errorf("expected key %q in response", key)
	}
}

// assertStringField vérifie qu'un champ string a la valeur attendue.
func assertStringField(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("missing key %q", key)
		return
	}
	got, ok := v.(string)
	if !ok {
		t.Errorf("key %q: expected string, got %T", key, v)
		return
	}
	if got != want {
		t.Errorf("key %q: got %q, want %q", key, got, want)
	}
}

// ---------------------------------------------------------------------------
// Tests squelettes — vérifient la structure des golden values.
// Ces tests ne nécessitent PAS de serveur Go en marche : ils valident
// uniquement que les fixtures sont bien formées et conformes au schéma.
// ---------------------------------------------------------------------------

func TestGolden_HealthOK_Structure(t *testing.T) {
	m := loadGolden(t, "health_ok.json")
	assertKeyExists(t, m, "status")
	assertStringField(t, m, "status", "ok")
	assertKeyExists(t, m, "match_count")
	assertKeyExists(t, m, "player_count")
}

func TestGolden_BootstrapNoAuth_Structure(t *testing.T) {
	m := loadGolden(t, "bootstrap_no_auth.json")
	assertKeyExists(t, m, "setup_required")
	assertKeyExists(t, m, "auth_state")
	assertKeyExists(t, m, "setup_state")
	assertKeyExists(t, m, "available_players")

	players, ok := m["available_players"].([]any)
	if !ok {
		t.Fatal("available_players should be an array")
	}
	if len(players) == 0 {
		t.Skip("no players in golden fixture")
	}
	first, ok := players[0].(map[string]any)
	if !ok {
		t.Fatal("first player should be an object")
	}
	for _, key := range []string{"player_slug", "gamertag", "xuid"} {
		assertKeyExists(t, first, key)
	}
}

func TestGolden_PlayersList_Structure(t *testing.T) {
	m := loadGolden(t, "players_list.json")
	assertKeyExists(t, m, "items")
	assertKeyExists(t, m, "default_player_slug")
}

func TestGolden_FiltersResolveAll_Structure(t *testing.T) {
	m := loadGolden(t, "filters_resolve_all.json")
	assertKeyExists(t, m, "effective")
	assertKeyExists(t, m, "available_options")
	assertKeyExists(t, m, "counts")
}

func TestGolden_CareerPage_Structure(t *testing.T) {
	m := loadGolden(t, "career_page_chocoboflor.json")
	assertKeyExists(t, m, "summary")
}

func TestGolden_MatchHistoryPage1_Structure(t *testing.T) {
	m := loadGolden(t, "match_history_page1_nofilter.json")
	assertKeyExists(t, m, "summary")
	assertKeyExists(t, m, "table")
}

func TestGolden_GamertagSearchCho_Structure(t *testing.T) {
	m := loadGolden(t, "gamertag_search_cho.json")
	assertKeyExists(t, m, "query")
	assertKeyExists(t, m, "items")
}

func TestGolden_MatchViewSlayer_Structure(t *testing.T) {
	m := loadGolden(t, "match_view_slayer.json")
	assertKeyExists(t, m, "header")
	assertKeyExists(t, m, "summary_tab")
}

// ---------------------------------------------------------------------------
// Multi-titre : golden value pour Halo Infinite (titre par défaut).
// Quand un second titre sera actif, ajouter ici un test assertant
// que les golden values du second titre sont isolées.
// ---------------------------------------------------------------------------

func TestGolden_AllFixturesLoadable(t *testing.T) {
	dir := goldenDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read golden dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("cannot read %s: %v", entry.Name(), err)
			}
			var m any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Errorf("invalid JSON in %s: %v", entry.Name(), err)
			}
		})
	}
}
