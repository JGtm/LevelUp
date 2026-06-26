// Package service — tests pour les helpers extraits de buildTeamTabFull et
// buildCombatTabFull (audit #2 funlen extraction).
package service

import (
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// ---------- indexBulkMedalsByXUID ----------

func TestIndexBulkMedalsByXUID_GroupsByXUID(t *testing.T) {
	bulk := []domain.BulkMedalRaw{
		{XUID: "A", MedalID: 1, Count: 3, Label: "Killing Spree", Difficulty: "normal"},
		{XUID: "A", MedalID: 2, Count: 1, Label: "Double Kill", Difficulty: "common"},
		{XUID: "B", MedalID: 1, Count: 2, Label: "Killing Spree", Difficulty: "normal"},
	}
	got := indexBulkMedalsByXUID(bulk, nil, len(bulk))
	if len(got["A"]) != 2 {
		t.Errorf("XUID=A doit avoir 2 entrees, obtenu %d", len(got["A"]))
	}
	if len(got["B"]) != 1 {
		t.Errorf("XUID=B doit avoir 1 entree, obtenu %d", len(got["B"]))
	}
	if got["A"][0].MedalID != 1 || got["A"][0].Count != 3 {
		t.Errorf("medaille A[0] doit etre {MedalID=1, Count=3}, obtenu %+v", got["A"][0])
	}
}

func TestIndexBulkMedalsByXUID_EmptyInput(t *testing.T) {
	got := indexBulkMedalsByXUID(nil, nil, 0)
	if len(got) != 0 {
		t.Errorf("attendu map vide, obtenu %d entrees", len(got))
	}
}

// ---------- indexBulkWeaponsByXUID ----------

func TestIndexBulkWeaponsByXUID_GroupsByXUID(t *testing.T) {
	bulk := []domain.BulkWeaponKillRaw{
		{XUID: "A", WeaponID: 101, NameEN: "BR75 Battle Rifle", WeaponLabel: "BR75", Kills: 5},
		{XUID: "A", WeaponID: 102, NameEN: "AK-47", WeaponLabel: "AK-47", Kills: 2},
		{XUID: "B", WeaponID: 101, NameEN: "BR75 Battle Rifle", WeaponLabel: "BR75", Kills: 3},
	}
	got := indexBulkWeaponsByXUID(bulk, nil, len(bulk))
	if len(got["A"]) != 2 {
		t.Errorf("XUID=A doit avoir 2 armes, obtenu %d", len(got["A"]))
	}
	if got["A"][0].WeaponID != 101 || got["A"][0].Kills != 5 {
		t.Errorf("arme A[0] doit etre {101, 5 kills}, obtenu %+v", got["A"][0])
	}
}

// ---------- computeScoreboardRowCombatYield ----------

func TestComputeScoreboardRowCombatYield_NoDamageData_ReturnsNilPointers(t *testing.T) {
	s := domain.ScoreboardRaw{Kills: 5, Deaths: 3, Assists: 2}
	oc, dr, dpk, dpd := computeScoreboardRowCombatYield(s, 225)
	if oc != nil || dr != nil || dpk != nil || dpd != nil {
		t.Errorf("Tous les pointeurs doivent etre nil sans damage data : oc=%v dr=%v dpk=%v dpd=%v", oc, dr, dpk, dpd)
	}
}

func TestComputeScoreboardRowCombatYield_FullData_AllPointersSet(t *testing.T) {
	dd := 2000.0
	dt := 1500.0
	s := domain.ScoreboardRaw{
		Kills: 10, Deaths: 5, Assists: 3,
		DamageDealt: &dd,
		DamageTaken: &dt,
	}
	oc, dr, dpk, dpd := computeScoreboardRowCombatYield(s, 225)
	if oc == nil || dr == nil || dpk == nil || dpd == nil {
		t.Fatalf("Tous les pointeurs doivent etre non-nil : oc=%v dr=%v dpk=%v dpd=%v", oc, dr, dpk, dpd)
	}
	if *dpk != 200.0 { // 2000 / 10
		t.Errorf("DamagePerKill doit etre 200, obtenu %f", *dpk)
	}
	if *dpd != 300.0 { // 1500 / 5
		t.Errorf("DamagePerDeath doit etre 300, obtenu %f", *dpd)
	}
}

func TestComputeScoreboardRowCombatYield_ZeroDeaths_NoDPD(t *testing.T) {
	dd := 1000.0
	dt := 0.0
	s := domain.ScoreboardRaw{
		Kills: 4, Deaths: 0,
		DamageDealt: &dd,
		DamageTaken: &dt,
	}
	_, _, dpk, dpd := computeScoreboardRowCombatYield(s, 225)
	if dpk == nil {
		t.Error("DamagePerKill doit etre defini (kills=4 > 0)")
	}
	if dpd != nil {
		t.Errorf("DamagePerDeath doit etre nil (deaths=0), obtenu %v", *dpd)
	}
}

// TestComputeScoreboardRowCombatYield_DamageTakenNil_OCSetDRNil couvre le cas
// title-agnostic Halo 5 : damage_dealt présent mais damage_taken absent. L'OC
// (offensive_conversion) ne dépend que de damage_dealt → doit être calculé ; la
// DR (defensive_resistance) exige damage_taken → reste N/A (nil), légitime.
func TestComputeScoreboardRowCombatYield_DamageTakenNil_OCSetDRNil(t *testing.T) {
	dd := 2000.0
	s := domain.ScoreboardRaw{
		Kills: 10, Deaths: 5, Assists: 3,
		DamageDealt: &dd,
		DamageTaken: nil, // Halo 5 : pas de damage_taken
	}
	oc, dr, dpk, dpd := computeScoreboardRowCombatYield(s, 225)
	if oc == nil {
		t.Fatal("OC doit etre non-nil quand damage_dealt est present (independant de damage_taken)")
	}
	// OC = 225 * (10 + 3/3) / 2000 = 225*11/2000 = 1.2375
	if *oc < 1.2374 || *oc > 1.2376 {
		t.Errorf("OC attendu ~1.2375, obtenu %f", *oc)
	}
	if dr != nil {
		t.Errorf("DR doit etre nil sans damage_taken, obtenu %v", *dr)
	}
	if dpk == nil || *dpk != 200.0 {
		t.Errorf("DamagePerKill attendu 200 (damage_dealt present), obtenu %v", dpk)
	}
	if dpd != nil {
		t.Errorf("DamagePerDeath doit etre nil sans damage_taken, obtenu %v", *dpd)
	}
}

// ---------- buildCombatTabFull : fallback synthétique kvPairs (#8/#3) ----------

// TestBuildCombatTabFull_MedalsOnlyEvents_UsesKVPairsSynthetic couvre le cas H5 :
// highlight_events ne porte que des médailles, mais killer_victim_pairs est
// peuplé. Le combat tab doit alors synthétiser les events kill/death depuis les
// paires → KD timeline, kill-feed, cadence et killer/victim non vides.
func TestBuildCombatTabFull_MedalsOnlyEvents_UsesKVPairsSynthetic(t *testing.T) {
	me := "ME"
	enemy := "EN"
	medalT := int64(2000)
	medalX := me
	events := []domain.EventRaw{
		{EventType: "medal", TimeMS: &medalT, XUID: &medalX},
	}
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: me, VictimXUID: enemy, TimeMS: 1000, KillCount: 1},
		{KillerXUID: me, VictimXUID: enemy, TimeMS: 3000, KillCount: 1},
		{KillerXUID: enemy, VictimXUID: me, TimeMS: 5000, KillCount: 1},
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: me, Kills: 2, Deaths: 1, OutcomeCode: 2, TeamID: intPtr(0)},
		{XUID: enemy, Kills: 1, Deaths: 2, OutcomeCode: 3, TeamID: intPtr(1)},
	}
	tab := buildCombatTabFull("m1", nil, events, nil, kvPairs, scoreboard, me, 60000)

	// KD timeline du joueur : 2 kills + 1 death = 3 points.
	if len(tab.KDTimeline) == 0 {
		t.Fatal("KDTimeline vide alors que kvPairs porte les kills (fallback synthétique attendu)")
	}
	last := tab.KDTimeline[len(tab.KDTimeline)-1]
	if last.Kills != 2 || last.Deaths != 1 {
		t.Errorf("KD final attendu 2K/1D, obtenu %dK/%dD", last.Kills, last.Deaths)
	}
	// Kill-feed (HighlightEvents) : 1 médaille + 6 events synthétiques (3 paires
	// × kill+death).
	if len(tab.HighlightEvents) != 7 {
		t.Errorf("HighlightEvents attendu 7 (1 médaille + 6 synthétiques), obtenu %d", len(tab.HighlightEvents))
	}
	if tab.Cadence == nil {
		t.Error("Cadence nil alors que des kills synthétiques existent")
	}
	if len(tab.KillerVictim) == 0 {
		t.Error("KillerVictim vide alors que kvPairs est peuplé")
	}
}

// TestBuildCombatTabFull_RealKillEvents_NoSynthetic vérifie qu'avec des events
// kill/death déjà présents (cas Infinite), le fallback n'est PAS déclenché : le
// kill-feed ne contient QUE les events d'origine (aucun doublon synthétique).
func TestBuildCombatTabFull_RealKillEvents_NoSynthetic(t *testing.T) {
	me := "ME"
	enemy := "EN"
	kT := int64(1000)
	events := []domain.EventRaw{
		{EventType: "kill", TimeMS: &kT, XUID: &me},
		{EventType: "death", TimeMS: &kT, XUID: &enemy},
	}
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: me, VictimXUID: enemy, TimeMS: 1000, KillCount: 1},
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: me, Kills: 1, Deaths: 0, OutcomeCode: 2, TeamID: intPtr(0)},
		{XUID: enemy, Kills: 0, Deaths: 1, OutcomeCode: 3, TeamID: intPtr(1)},
	}
	tab := buildCombatTabFull("m1", nil, events, nil, kvPairs, scoreboard, me, 60000)
	// Pas de synthèse : exactement les 2 events d'origine dans le kill-feed.
	if len(tab.HighlightEvents) != 2 {
		t.Errorf("HighlightEvents attendu 2 (events réels, pas de synthèse), obtenu %d", len(tab.HighlightEvents))
	}
}

// ---------- convertTugBinsToDomain ----------

func TestConvertTugBinsToDomain_SplitsAllyEnemy(t *testing.T) {
	bins := []analysis.TugOfWarBin{
		{BinStartMS: 0, BinEndMS: 30000, Delta: 3, CumDelta: 3},      // ally +3
		{BinStartMS: 30000, BinEndMS: 60000, Delta: -2, CumDelta: 1}, // enemy +2
	}
	got := convertTugBinsToDomain(bins)
	if len(got) != 2 {
		t.Fatalf("attendu 2 bins, obtenu %d", len(got))
	}
	if got[0].TeamKills != 3 || got[0].EnemyKills != 0 {
		t.Errorf("Bin 0 : Team=3 Enemy=0 attendu, obtenu Team=%d Enemy=%d", got[0].TeamKills, got[0].EnemyKills)
	}
	if got[1].TeamKills != 0 || got[1].EnemyKills != 2 {
		t.Errorf("Bin 1 : Team=0 Enemy=2 attendu, obtenu Team=%d Enemy=%d", got[1].TeamKills, got[1].EnemyKills)
	}
	if got[0].BinStart != 0 || got[0].BinEnd != 30 {
		t.Errorf("Bin 0 : conversion ms -> seconds doit donner BinStart=0 BinEnd=30, obtenu %d / %d", got[0].BinStart, got[0].BinEnd)
	}
}

// ---------- convertImpactBadgesToDomain ----------

func TestConvertImpactBadgesToDomain_TimeMSPointerOptional(t *testing.T) {
	badges := []analysis.ImpactBadge{
		{BadgeKey: "first_blood", BadgeFR: "Premier sang", PlayerXUID: "A", TimeMS: 5000},
		{BadgeKey: "top_killer", BadgeFR: "Meilleur tueur", PlayerXUID: "B", TimeMS: 0},
	}
	got := convertImpactBadgesToDomain(badges)
	if len(got) != 2 {
		t.Fatalf("attendu 2 badges, obtenu %d", len(got))
	}
	if got[0].TimeMS == nil || *got[0].TimeMS != 5000 {
		t.Errorf("Badge 0 TimeMS doit etre *5000, obtenu %v", got[0].TimeMS)
	}
	if got[1].TimeMS != nil {
		t.Errorf("Badge 1 TimeMS doit etre nil (TimeMS=0), obtenu %v", *got[1].TimeMS)
	}
}

// ---------- convertKDPointsToDomain ----------

func TestConvertKDPointsToDomain_MStoSeconds(t *testing.T) {
	points := []analysis.KDTimelinePoint{
		{TimeMS: 0, CumKills: 0, CumDeaths: 0},
		{TimeMS: 15500, CumKills: 2, CumDeaths: 1},
		{TimeMS: 60000, CumKills: 5, CumDeaths: 3},
	}
	got := convertKDPointsToDomain(points)
	if len(got) != 3 {
		t.Fatalf("attendu 3 points, obtenu %d", len(got))
	}
	if got[1].TimeSeconds != 15 { // 15500 / 1000 = 15 (int truncation)
		t.Errorf("Point 1 TimeSeconds=15 attendu, obtenu %d", got[1].TimeSeconds)
	}
	if got[1].Kills != 2 || got[1].Deaths != 1 {
		t.Errorf("Point 1 : Kills=2 Deaths=1 attendu, obtenu Kills=%d Deaths=%d", got[1].Kills, got[1].Deaths)
	}
}
