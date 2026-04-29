// Package sync — engagement_regression_test.go : tests régression B1-B4
// (revue 2026-04-29 P3.2).
//
// 4 bugs critiques corrigés le 2026-04-29 (cf. thought_log entry "engagement:
// backfill all-players + 4 bugs critiques") qui n'avaient pas de test de
// non-régression. Ce fichier comble le gap.
//
// Stratégie : tests source-grep + JSON roundtrip — pas de DuckDB requis.
// Catch toute régression qui réintroduirait les patterns cassés.
//
// Bugs couverts :
//   - B1 : match_registry.is_pve n'existe pas (utiliser is_firefight)
//   - B2 : match_participants.is_bot n'existe pas (utiliser xuid LIKE 'bid(%')
//   - B3 : highlight_events.time_ms est relatif au début du match (0 à
//     duration_ms), pas un epoch UTC. MatchStartMS=0 et MatchEndMS=durationMS
//     doivent être passés à ComputeEngagementScore.
//   - B4 : EngagementScoreResult + EngagementPoint exposent des tags JSON
//     snake_case (frontend attend snake_case).
package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// ─── B1 : pas de référence à match_registry.is_pve ──────────────────────

func TestRegressionB1_NoIsPveColumn(t *testing.T) {
	source := readSourceFile(t, "engagement.go")
	repoSource := readRepoSource(t)

	// Patterns SQL spécifiques (le champ Go IsPvE est OK — c'est juste un
	// nom de champ, pas un nom de colonne SQL).
	badSQLPatterns := []string{
		"mr.is_pve",
		"match_registry.is_pve",
		"is_pve,",
		"is_pve)",
		" is_pve ", // dans une SELECT/WHERE
	}
	for _, pattern := range badSQLPatterns {
		if strings.Contains(source, pattern) {
			t.Errorf("régression B1 : engagement.go contient SQL `%s` — utiliser `is_firefight`", pattern)
		}
		if strings.Contains(repoSource, pattern) {
			t.Errorf("régression B1 : engagement_score_repo_queries.go contient SQL `%s`", pattern)
		}
	}
	if !strings.Contains(source, "is_firefight") {
		t.Errorf("régression B1 : engagement.go ne référence pas `is_firefight` — schema canonique perdu")
	}
}

// ─── B2 : pas de référence à match_participants.is_bot ──────────────────

func TestRegressionB2_NoIsBotColumn(t *testing.T) {
	source := readSourceFile(t, "engagement.go")
	repoSource := readRepoSource(t)

	for _, name := range []string{"engagement.go", "engagement_score_repo_queries.go"} {
		s := source
		if name == "engagement_score_repo_queries.go" {
			s = repoSource
		}
		if strings.Contains(s, "is_bot") {
			t.Errorf("régression B2 : %s contient `is_bot` — utiliser `xuid LIKE 'bid(%%'`", name)
		}
		if !strings.Contains(s, "bid(") {
			t.Errorf("régression B2 : %s ne référence pas `bid(` (filtre bots manquant)", name)
		}
	}
}

// ─── B3 : MatchStartMS=0 et MatchEndMS=durationMS (temps relatif) ──────

func TestRegressionB3_RelativeTimeWindow(t *testing.T) {
	source := readSourceFile(t, "engagement.go")

	// Le code DOIT passer MatchStartMS:0 (pas un epoch UTC) parce que
	// highlight_events.time_ms est relatif au début du match.
	if !strings.Contains(source, "MatchStartMS:   0,") &&
		!strings.Contains(source, "MatchStartMS: 0,") &&
		!strings.Contains(source, "MatchStartMS:0,") {
		t.Errorf("régression B3 : engagement.go ne passe plus MatchStartMS:0 — risque de fenêtre vide")
	}

	// MatchEndMS doit être durationMS (relatif), pas m.EndTimeMS (absolu).
	if strings.Contains(source, "MatchEndMS:     m.EndTimeMS,") ||
		strings.Contains(source, "MatchEndMS: m.EndTimeMS,") {
		t.Errorf("régression B3 : engagement.go passe `MatchEndMS: m.EndTimeMS` (epoch UTC) — utiliser `durationMS`")
	}
	if !strings.Contains(source, "durationMS") {
		t.Errorf("régression B3 : engagement.go ne calcule plus `durationMS` (m.EndTimeMS - m.StartTimeMS)")
	}
}

// ─── B4 : tags JSON snake_case sur EngagementScoreResult + Point ───────

func TestRegressionB4_JSONTagsSnakeCase(t *testing.T) {
	// Construit une instance représentative et vérifie le snake_case en
	// sérialisation. Couvre les 2 types touchés par B4.
	score := 75.5
	result := domain.EngagementScoreResult{
		EngagementScore: &score,
		ResidualBrut:    1.23,
		EngagementCurve: []domain.EngagementPoint{
			{
				TimeMS:         15000,
				PaceJoueur:     2.5,
				PaceTeam:       2.0,
				PaceAttendu:    2.1,
				PaceLobby:      1.8,
				PostDeathFlag:  false,
				IsPassiveDeath: false,
			},
		},
		MatchIntensity:  3.4,
		Confidence:      "full",
		NHistoryMatches: 47,
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal EngagementScoreResult: %v", err)
	}
	encoded := string(raw)

	expectedSnakeKeys := []string{
		`"engagement_score"`,
		`"residual_brut"`,
		`"engagement_curve"`,
		`"match_intensity"`,
		`"confidence"`,
		`"n_history_matches"`,
		// EngagementPoint
		`"time_ms"`,
		`"pace_joueur"`,
		`"pace_team"`,
		`"pace_attendu"`,
		`"pace_lobby"`,
		`"post_death_flag"`,
		`"is_passive_death"`,
	}
	for _, key := range expectedSnakeKeys {
		if !strings.Contains(encoded, key) {
			t.Errorf("régression B4 : clé snake_case %s absente du JSON sérialisé", key)
		}
	}

	// Vérifier qu'aucune clé PascalCase n'est sérialisée (avant fix B4 :
	// EngagementScore, ResidualBrut, etc.).
	pascalLeaks := []string{
		`"EngagementScore"`,
		`"ResidualBrut"`,
		`"EngagementCurve"`,
		`"MatchIntensity"`,
		`"NHistoryMatches"`,
		`"TimeMS"`,
		`"PaceJoueur"`,
		`"PostDeathFlag"`,
		`"IsPassiveDeath"`,
	}
	for _, key := range pascalLeaks {
		if strings.Contains(encoded, key) {
			t.Errorf("régression B4 : clé PascalCase %s sérialisée — frontend attend snake_case", key)
		}
	}
}

// ─── Helpers source-grep ──────────────────────────────────────────────

// readSourceFile lit un fichier source du package sync. Retourne string.
func readSourceFile(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	syncDir := filepath.Dir(thisFile)
	path := filepath.Join(syncDir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lire %s : %v", name, err)
	}
	return string(raw)
}

// readRepoSource lit le repo queries (un cran plus haut dans l'arbo).
func readRepoSource(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	syncDir := filepath.Dir(thisFile)
	path := filepath.Join(syncDir, "..", "platform", "duckdb", "engagement_score_repo_queries.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		// Le fichier peut avoir été déplacé — log mais ne crash pas le test.
		t.Logf("repo queries introuvable %s : %v", path, err)
		return ""
	}
	return string(raw)
}
