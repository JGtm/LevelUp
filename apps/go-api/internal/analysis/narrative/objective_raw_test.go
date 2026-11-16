package narrative

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// objetPeuple : un bloc objectif dont chaque champ porte une valeur distincte (entiers 1..n, durées n,5).
func objetPeuple() domain.ObjectiveRaw {
	n := 0
	ip := func() *int { n++; v := n; return &v }
	fp := func() *float64 { n++; v := float64(n) + 0.5; return &v }
	return domain.ObjectiveRaw{
		FlagCaptures: ip(), FlagCaptureAssists: ip(), FlagGrabs: ip(), FlagSecures: ip(), FlagSteals: ip(),
		FlagReturns: ip(), FlagCarriersKilled: ip(), FlagReturnersKilled: ip(), KillsAsFlagCarrier: ip(),
		KillsAsFlagReturner: ip(), TimeAsFlagCarrierSeconds: fp(),
		ZoneCaptures: ip(), ZoneSecures: ip(), ZoneOffensiveKills: ip(), ZoneDefensiveKills: ip(),
		ZoneScoringTicks: ip(), TimeInZonesSeconds: fp(),
		KillsAsSkullCarrier: ip(), SkullCarriersKilled: ip(), SkullGrabs: ip(), SkullScoringTicks: ip(),
		TimeAsSkullCarrierSeconds: fp(),
		KillsAsPowerSeedCarrier:   ip(), PowerSeedCarriersKilled: ip(), PowerSeedsDeposited: ip(), PowerSeedsStolen: ip(),
		TimeAsPowerSeedCarrierSeconds: fp(), TimeAsPowerSeedDriverSeconds: fp(),
		ExtractionConversionsCompleted: ip(), ExtractionConversionsDenied: ip(), ExtractionInitiationsCompleted: ip(),
		ExtractionInitiationsDenied: ip(), SuccessfulExtractions: ip(),
		KillsAsVip: ip(), VipKills: ip(), VipAssists: ip(), TimesSelectedAsVip: ip(), MaxKillingSpreeAsVip: ip(),
		TimeAsVipSeconds: fp(),
	}
}

// Témoin de caractérisation : chaque colonne reçoit la valeur de SON champ. Valeurs relevées sur la
// fonction du paquet service avant son déplacement ici.
func TestObjectiveColumnValuesOf_ValeursParColonne(t *testing.T) {
	want := map[string]float64{
		"extraction_conversions_completed": 29, "extraction_conversions_denied": 30,
		"extraction_initiations_completed": 31, "flag_capture_assists": 2, "flag_captures": 1,
		"flag_carriers_killed": 7, "flag_returners_killed": 8, "flag_returns": 6, "flag_secures": 4,
		"flag_steals": 5, "kills_as_vip": 34, "power_seed_carriers_killed": 24, "power_seeds_deposited": 25,
		"power_seeds_stolen": 26, "skull_carriers_killed": 19, "skull_grabs": 20, "skull_scoring_ticks": 21,
		"successful_extractions": 33, "time_as_flag_carrier_seconds": 11.5,
		"time_as_power_seed_carrier_seconds": 27.5, "time_as_power_seed_driver_seconds": 28.5,
		"time_as_skull_carrier_seconds": 22.5, "time_as_vip_seconds": 39.5, "time_in_zones_seconds": 17.5,
		"times_selected_as_vip": 37, "vip_assists": 36, "vip_kills": 35, "zone_captures": 12,
		"zone_defensive_kills": 15, "zone_offensive_kills": 14, "zone_scoring_ticks": 16, "zone_secures": 13,
	}
	if got := ObjectiveColumnValuesOf(objetPeuple()); !reflect.DeepEqual(got, want) {
		t.Errorf("colonnes = %v, attendu %v", got, want)
	}
	for col, v := range ObjectiveColumnValuesOf(domain.ObjectiveRaw{}) {
		if v != 0 {
			t.Errorf("%s = %v sur un bloc vide, attendu 0", col, v)
		}
	}
}

// Précédence des blocs et split KOTH / Strongholds : zones avant CTF, ticks > 0 = KOTH.
func TestObjectiveFamilyOfRaw_PrecedenceEtSplit(t *testing.T) {
	ip := func(v int) *int { return &v }
	cas := []struct {
		nom  string
		obj  domain.ObjectiveRaw
		want ObjectiveFamily
		ok   bool
	}{
		{"zones avec ticks : KOTH", domain.ObjectiveRaw{ZoneCaptures: ip(1), ZoneScoringTicks: ip(3)}, FamilyZonesKOTH, true},
		{"zones sans tick : Strongholds", domain.ObjectiveRaw{ZoneCaptures: ip(1), ZoneScoringTicks: ip(0)}, FamilyZonesStrongholds, true},
		{"zones avant CTF", domain.ObjectiveRaw{ZoneCaptures: ip(1), FlagCaptures: ip(2)}, FamilyZonesStrongholds, true},
		{"aucun bloc", domain.ObjectiveRaw{}, "", false},
	}
	for _, c := range cas {
		if got, ok := ObjectiveFamilyOfRaw(c.obj); got != c.want || ok != c.ok {
			t.Errorf("%s : (%q, %v), attendu (%q, %v)", c.nom, got, ok, c.want, c.ok)
		}
	}
}
