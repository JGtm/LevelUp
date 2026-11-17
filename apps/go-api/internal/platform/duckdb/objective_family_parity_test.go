//go:build cgo

package duckdb

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// discriminantCols : les colonnes de match_objective_stats que lit `objectiveFamilyCaseSQL`, posées depuis
// les champs non nil du bloc (un champ nil = colonne NULL, donc omise de l'INSERT).
func discriminantCols(o domain.ObjectiveRaw) map[string]any {
	cols := map[string]any{}
	set := func(name string, p *int) {
		if p != nil {
			cols[name] = *p
		}
	}
	set("zone_scoring_ticks", o.ZoneScoringTicks)
	set("zone_captures", o.ZoneCaptures)
	set("flag_grabs", o.FlagGrabs)
	set("flag_captures", o.FlagCaptures)
	set("skull_grabs", o.SkullGrabs)
	set("skull_scoring_ticks", o.SkullScoringTicks)
	set("power_seeds_deposited", o.PowerSeedsDeposited)
	set("power_seed_carriers_killed", o.PowerSeedCarriersKilled)
	set("successful_extractions", o.SuccessfulExtractions)
	set("extraction_initiations_completed", o.ExtractionInitiationsCompleted)
	set("times_selected_as_vip", o.TimesSelectedAsVip)
	set("kills_as_vip", o.KillsAsVip)
	return cols
}

// TestObjectiveFamilyCaseSQL_ParityWithGoFunction : l'expression SQL `objectiveFamilyCaseSQL` et la fonction
// Go `narrative.ObjectiveFamilyOfRaw` sont deux écritures de la MÊME règle ; elles rendent la même famille
// sur chaque ligne d'une table de cas (schéma réel de match_objective_stats, DuckDB mémoire) : un cas par
// famille, chaque colonne discriminante prise seule, la précédence des blocs, le split KOTH / Strongholds
// et la ligne sans objectif.
func TestObjectiveFamilyCaseSQL_ParityWithGoFunction(t *testing.T) {
	_, db := newObjectiveStatsRepoOnMem(t, true)
	ip := func(v int) *int { return &v }

	cas := []struct {
		nom string
		obj domain.ObjectiveRaw
	}{
		{"koth : ticks de zone", domain.ObjectiveRaw{ZoneCaptures: ip(2), ZoneScoringTicks: ip(12)}},
		{"strongholds : zero tick", domain.ObjectiveRaw{ZoneCaptures: ip(2), ZoneScoringTicks: ip(0)}},
		{"strongholds : captures seules", domain.ObjectiveRaw{ZoneCaptures: ip(2)}},
		{"zones : ticks seuls non nuls", domain.ObjectiveRaw{ZoneScoringTicks: ip(4)}},
		{"zones : ticks seuls a zero", domain.ObjectiveRaw{ZoneScoringTicks: ip(0)}},
		{"ctf : prises", domain.ObjectiveRaw{FlagGrabs: ip(3)}},
		{"ctf : captures", domain.ObjectiveRaw{FlagCaptures: ip(1)}},
		{"oddball : prises", domain.ObjectiveRaw{SkullGrabs: ip(3)}},
		{"oddball : ticks", domain.ObjectiveRaw{SkullScoringTicks: ip(40)}},
		{"stockpile : depots", domain.ObjectiveRaw{PowerSeedsDeposited: ip(4)}},
		{"stockpile : porteurs tues", domain.ObjectiveRaw{PowerSeedCarriersKilled: ip(2)}},
		{"extraction : reussies", domain.ObjectiveRaw{SuccessfulExtractions: ip(2)}},
		{"extraction : initiations", domain.ObjectiveRaw{ExtractionInitiationsCompleted: ip(3)}},
		{"vip : selections", domain.ObjectiveRaw{TimesSelectedAsVip: ip(2)}},
		{"vip : frags", domain.ObjectiveRaw{KillsAsVip: ip(2)}},
		{"precedence : zones avant CTF", domain.ObjectiveRaw{ZoneCaptures: ip(1), FlagCaptures: ip(2)}},
		{"precedence : KOTH avant le reste", domain.ObjectiveRaw{ZoneScoringTicks: ip(5), SkullGrabs: ip(1), KillsAsVip: ip(1)}},
		{"precedence : CTF avant oddball", domain.ObjectiveRaw{FlagGrabs: ip(1), SkullGrabs: ip(1)}},
		{"precedence : oddball avant stockpile", domain.ObjectiveRaw{SkullGrabs: ip(1), PowerSeedsDeposited: ip(1)}},
		{"precedence : stockpile avant extraction", domain.ObjectiveRaw{PowerSeedsDeposited: ip(1), SuccessfulExtractions: ip(1)}},
		{"precedence : extraction avant VIP", domain.ObjectiveRaw{SuccessfulExtractions: ip(1), KillsAsVip: ip(1)}},
		{"sans objectif", domain.ObjectiveRaw{}},
	}
	familles := map[narrative.ObjectiveFamily]bool{}
	for i, c := range cas {
		matchID := "m" + string(rune('a'+i))
		insertObjectiveRow(t, db, matchID, "x1", discriminantCols(c.obj))

		var got sql.NullString
		q := "SELECT " + objectiveFamilyCaseSQL() + " FROM match_objective_stats o WHERE o.match_id = ?"
		if err := db.QueryRow(q, matchID).Scan(&got); err != nil {
			t.Fatalf("%s : lecture de la famille SQL : %v", c.nom, err)
		}
		want, ok := narrative.ObjectiveFamilyOfRaw(c.obj)
		if ok {
			familles[want] = true
		}
		if got.Valid != ok || (ok && got.String != string(want)) {
			t.Errorf("%s : SQL = (%q, valide=%v), Go = (%q, %v)", c.nom, got.String, got.Valid, want, ok)
		}
	}
	for _, fam := range narrative.AllObjectiveFamilies() {
		if !familles[fam] {
			t.Errorf("la table de cas ne couvre pas la famille %s : le test de parité ne la garderait pas", fam)
		}
	}
}
