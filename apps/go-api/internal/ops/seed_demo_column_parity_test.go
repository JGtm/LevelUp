package ops

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/persist"
)

// insertColumnsUnion extrait l'UNION des colonnes de tous les `INSERT INTO <table> (
// ... )` d'un blob source (multi-ligne). Une table écrite par plusieurs INSERT
// (ex. match_skill_rank : chemins CSR classé + LUSR) voit ses colonnes fusionnées.
func insertColumnsUnion(src, table string) map[string]bool {
	out := map[string]bool{}
	re := regexp.MustCompile(`INSERT\s+(?:OR\s+IGNORE\s+)?INTO\s+` + regexp.QuoteMeta(table) + `\s*\(`)
	for _, loc := range re.FindAllStringIndex(src, -1) {
		rest := src[loc[1]:]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			continue
		}
		for _, col := range strings.Split(rest[:end], ",") {
			if c := strings.TrimSpace(col); c != "" {
				out[c] = true
			}
		}
	}
	return out
}

func diff(a map[string]bool, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func setOf(cols []string) map[string]bool {
	m := map[string]bool{}
	for _, c := range cols {
		m[c] = true
	}
	return m
}

// TestSeedDemoColumnParity (F7, revue 2026-07-17) : garde-rail anti-rot du corpus
// démo. Pour chaque table critique, l'ensemble des colonnes écrites par le SEEDER
// synthétique doit correspondre au contrat persist (persist.Match*Columns), aux
// divergences INTENTIONNELLES documentées près (allowlists ci-dessous). Une colonne
// AJOUTÉE côté persist (recette ADR 0026) et absente du seeder, hors allowlist, casse
// ce test → force la mise à jour du seeder (ou l'ajout justifié à l'allowlist).
func TestSeedDemoColumnParity(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	dir := filepath.Dir(thisFile)
	var srcBuilder strings.Builder
	for _, f := range []string{"seed_demo_synthetic_shared.go", "seed_demo_synthetic_player.go"} {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("lecture %s: %v", f, err)
		}
		srcBuilder.Write(data)
		srcBuilder.WriteByte('\n')
	}
	seederSrc := srcBuilder.String()

	cases := []struct {
		table   string
		persist map[string]bool
		// seederExtras : colonnes que le seeder écrit EN PLUS de persist (déterminisme
		// démo : written_at/start_time ancrés ; libellés FR non peuplés par persist).
		seederExtras map[string]bool
		// persistOnly : colonnes persist que le seeder OMET volontairement (valeurs par
		// défaut suffisantes pour la démo : version_id, stats MMR/expected, timestamps).
		persistOnly map[string]bool
	}{
		{
			table:        "match_registry",
			persist:      setOf(persist.MatchRegistryColumns),
			seederExtras: setOf([]string{"playlist_name_fr", "map_name_fr", "pair_name_fr"}),
			persistOnly: setOf([]string{
				"playlist_version_id", "map_version_id", "pair_version_id",
				"game_variant_id", "game_variant_version_id", "is_firefight", "season_id",
				"playable_duration_seconds", "real_start_time", "team_0_ps_score", "team_1_ps_score",
				// Manches : le corpus démo ne fabrique aucun mode à manches (pas de variante
				// Oddball dans son catalogue), donc rien à seeder — NULL y est la mesure juste
				// (« on ne sait pas »), et l'affichage retombe sur les points comme en prod.
				"team_0_rounds_won", "team_1_rounds_won", "rounds_total",
				"match_intensity", "backfill_completed", "first_sync_by", "first_sync_at",
				"last_updated_at", "created_at", "updated_at",
			}),
		},
		{
			table:        "match_participants",
			persist:      setOf(persist.MatchParticipantsColumns),
			seederExtras: map[string]bool{},
			persistOnly: setOf([]string{
				"avg_life_seconds", "kills_expected", "deaths_expected", "kills_stddev", "deaths_stddev",
				"grenade_kills", "assassination_kills", "ground_pound_kills", "shoulder_bash_kills",
				"joined_in_progress", "left_in_progress", "first_joined_time", "last_leave_time",
				"backfill_bits", "created_at",
			}),
		},
		{
			table:        "match_skill_rank",
			persist:      setOf(persist.MatchSkillRankColumns),
			seederExtras: setOf([]string{"start_time", "written_at"}),
			persistOnly:  setOf([]string{"rating_deviation"}),
		},
		{
			table:        "match_csrs",
			persist:      setOf(persist.MatchCSRColumns),
			seederExtras: setOf([]string{"written_at"}),
			persistOnly:  map[string]bool{},
		},
	}

	for _, tc := range cases {
		seeder := insertColumnsUnion(seederSrc, tc.table)
		if len(seeder) == 0 {
			t.Errorf("%s : aucun INSERT trouvé dans le seeder (extracteur ou renommage ?)", tc.table)
			continue
		}
		// Colonnes du seeder inconnues de persist et non déclarées comme extras.
		for _, c := range diff(seeder, tc.persist) {
			if !tc.seederExtras[c] {
				t.Errorf("%s : le seeder écrit %q absent du contrat persist et de seederExtras", tc.table, c)
			}
		}
		// Colonnes persist absentes du seeder et non déclarées comme omissions.
		for _, c := range diff(tc.persist, seeder) {
			if !tc.persistOnly[c] {
				t.Errorf("%s : colonne persist %q non écrite par le seeder (rot ADR 0026 ?) — "+
					"mettre à jour le seeder ou l'allowlist persistOnly", tc.table, c)
			}
		}
	}
}
