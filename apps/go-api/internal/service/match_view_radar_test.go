package service

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// TestObjectiveColumnValues_CoverFamilyTables : garde-rail anti-dérive entre le mapping
// ObjectiveRaw → colonnes (vue Match) et les tables de narrative — toute colonne d'action, de durée
// ou de dénominateur VIP que le noyau lit doit être produite (une colonne absente vaudrait 0 sans
// bruit), et la classification (dont le split KOTH / Bases par ticks) doit être correcte.
func TestObjectiveColumnValues_CoverFamilyTables(t *testing.T) {
	t.Parallel()
	ip := func(v int) *int { return &v }
	fp := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		obj  domain.ObjectiveRaw
		want narrative.ObjectiveFamily
	}{
		{"ctf", domain.ObjectiveRaw{
			FlagCaptures: ip(1), FlagCaptureAssists: ip(1), FlagGrabs: ip(2), FlagSecures: ip(1),
			FlagSteals: ip(1), FlagReturns: ip(2), FlagCarriersKilled: ip(1), FlagReturnersKilled: ip(1),
			KillsAsFlagCarrier: ip(3), KillsAsFlagReturner: ip(1), TimeAsFlagCarrierSeconds: fp(30),
		}, narrative.FamilyCTF},
		{"strongholds", domain.ObjectiveRaw{
			ZoneCaptures: ip(4), ZoneSecures: ip(2), ZoneOffensiveKills: ip(3), ZoneDefensiveKills: ip(2),
			ZoneScoringTicks: ip(0), TimeInZonesSeconds: fp(100),
		}, narrative.FamilyZonesStrongholds},
		{"koth", domain.ObjectiveRaw{
			ZoneCaptures: ip(4), ZoneSecures: ip(2), ZoneOffensiveKills: ip(3), ZoneDefensiveKills: ip(2),
			ZoneScoringTicks: ip(12), TimeInZonesSeconds: fp(100),
		}, narrative.FamilyZonesKOTH},
		{"oddball", domain.ObjectiveRaw{
			SkullGrabs: ip(3), SkullCarriersKilled: ip(2), SkullScoringTicks: ip(40),
			KillsAsSkullCarrier: ip(1), TimeAsSkullCarrierSeconds: fp(70),
		}, narrative.FamilyOddball},
		{"stockpile", domain.ObjectiveRaw{
			PowerSeedsDeposited: ip(4), PowerSeedsStolen: ip(2), PowerSeedCarriersKilled: ip(2),
			KillsAsPowerSeedCarrier: ip(1), TimeAsPowerSeedCarrierSeconds: fp(40), TimeAsPowerSeedDriverSeconds: fp(20),
		}, narrative.FamilyStockpile},
		{"extraction", domain.ObjectiveRaw{
			ExtractionConversionsCompleted: ip(2), ExtractionConversionsDenied: ip(1),
			ExtractionInitiationsCompleted: ip(3), ExtractionInitiationsDenied: ip(1), SuccessfulExtractions: ip(2),
		}, narrative.FamilyExtraction},
		{"vip", domain.ObjectiveRaw{
			KillsAsVip: ip(2), VipKills: ip(3), VipAssists: ip(1), TimesSelectedAsVip: ip(2),
			MaxKillingSpreeAsVip: ip(3), TimeAsVipSeconds: fp(60),
		}, narrative.FamilyVIP},
	}
	for _, tc := range cases {
		row, ok := objectiveTeamRowFromScoreboard(domain.ScoreboardRaw{XUID: "x1", Obj: tc.obj, TimePlayed: fp(600)})
		if !ok || row.Family != tc.want || row.TimePlayedSeconds != 600 {
			t.Errorf("%s: (%+v, %v), want famille %s, durée 600", tc.name, row, ok, tc.want)
			continue
		}
		for col := range narrative.ObjectiveFamilyActionWeights[tc.want] {
			if _, ok := row.Values[col]; !ok {
				t.Errorf("%s: colonne d'action %q de %s non produite", tc.name, col, tc.want)
			}
		}
		for _, col := range narrative.ObjectiveFamilyHoldColumns[tc.want] {
			if _, ok := row.Values[col]; !ok {
				t.Errorf("%s: colonne de durée %q de %s non produite", tc.name, col, tc.want)
			}
		}
		if _, ok := row.Values[narrative.ObjectiveColTimesSelectedAsVIP]; !ok {
			t.Errorf("%s: dénominateur VIP non produit", tc.name)
		}
	}
	// Slayer (aucun bloc) ou sans identité : pas de ligne.
	if _, ok := objectiveTeamRowFromScoreboard(domain.ScoreboardRaw{XUID: "x1", TimePlayed: fp(600)}); ok {
		t.Error("slayer : aucune ligne d'objectif attendue")
	}
	if _, ok := objectiveTeamRowFromScoreboard(domain.ScoreboardRaw{Obj: cases[0].obj}); ok {
		t.Error("sans xuid : aucune ligne d'objectif attendue")
	}
}

// radarCTFRow : un participant d'un match Drapeau (camp `team`), avec ses actions d'objectif.
func radarCTFRow(xuid string, team int, obj domain.ObjectiveRaw) domain.ScoreboardRaw {
	tp, dd, dt, ps := 600.0, 2500.0, 2000.0, 2000.0
	return domain.ScoreboardRaw{
		XUID: xuid, Gamertag: "GT_" + xuid, TeamID: &team, Kills: 10, Deaths: 5, Assists: 2,
		DamageDealt: &dd, DamageTaken: &dt, PersonalScore: &ps, TimePlayed: &tp, Obj: obj,
	}
}

// ctfActions : un bloc Drapeau avec les actions données (les autres colonnes à 0, mais présentes).
func ctfActions(captures, steals, returns, secures int) domain.ObjectiveRaw {
	z := 0
	return domain.ObjectiveRaw{
		FlagCaptures: &captures, FlagSteals: &steals, FlagReturns: &returns, FlagSecures: &secures,
		FlagCaptureAssists: &z, FlagCarriersKilled: &z, FlagReturnersKilled: &z,
	}
}

func radarAxisOf(series []MatchViewRadarSeries, axis narrative.ParticipationAxis) *narrative.ParticipationScore {
	for i := range series[0].Axes {
		if series[0].Axes[i].Axis == axis {
			return &series[0].Axes[i]
		}
	}
	return nil
}

// TestBuildMatchRadarFromScoreboard_ObjectifDansCeMatch : l'axe Objectif est la note d'objectif DU
// MATCH SEUL, la part du joueur dans l'objectif de SON équipe (les deux camps de la feuille), prise et
// soutien compris. Il ne dépend d'aucun historique (la fonction n'en reçoit aucun) ni de l'adversaire.
func TestBuildMatchRadarFromScoreboard_ObjectifDansCeMatch(t *testing.T) {
	t.Parallel()
	// Équipe 0 : « me » et trois coéquipiers à 2 captures et 2 vols chacun ; équipe 1 : un adversaire
	// très actif, qui ne doit pas entrer dans la part.
	board := []domain.ScoreboardRaw{
		radarCTFRow("me", 0, ctfActions(2, 2, 0, 0)),
		radarCTFRow("b", 0, ctfActions(2, 2, 0, 0)),
		radarCTFRow("c", 0, ctfActions(2, 2, 0, 0)),
		radarCTFRow("d", 0, ctfActions(2, 2, 0, 0)),
		radarCTFRow("foe", 1, ctfActions(9, 9, 9, 9)),
	}
	series := BuildMatchRadarFromScoreboard(board, "me", 500, "ctf", games.DefaultEffectiveHpToKill, 0.90)
	obj := radarAxisOf(series, narrative.AxisObjective)
	if obj == nil {
		t.Fatal("axe objective absent sur un match Drapeau")
	}
	// Part équitable (1/4 de l'équipe) = 50, quel que soit l'adversaire.
	if math.Abs(obj.Value-50) > 1e-9 || math.Abs(obj.Raw-50) > 1e-9 {
		t.Errorf("objective: want 50 (part équitable), got raw %v value %v", obj.Raw, obj.Value)
	}
	// Résiduel Score : PSA soustrait (décision 1) — 2000 − 10×100 − 2×50 − 500 = 400.
	if sc := radarAxisOf(series, narrative.AxisScore); sc == nil || sc.Raw != 400 {
		t.Errorf("score residuel: want 400, got %+v", sc)
	}
}

// Une escorte (retours et sécurisations, aucun temps de port, aucune capture) est mesurée sur sa part
// du SOUTIEN de l'équipe : l'ancien indice par famille, qui dépendait du temps de port et d'un repère
// de population, la notait ~31 ; la note de ce match vaut 100 (tout le soutien de son équipe).
func TestBuildMatchRadarFromScoreboard_EscorteMesureeSurLeSoutien(t *testing.T) {
	t.Parallel()
	board := []domain.ScoreboardRaw{
		radarCTFRow("escorte", 0, ctfActions(0, 0, 3, 2)), // objectif : 6,5 points, tous de soutien
		radarCTFRow("porteur", 0, ctfActions(2, 2, 0, 0)), // objectif : 14 points, aucun soutien
		radarCTFRow("c", 0, ctfActions(0, 0, 0, 0)),
		radarCTFRow("d", 0, ctfActions(0, 0, 0, 0)),
		radarCTFRow("foe", 1, ctfActions(4, 4, 4, 4)),
	}
	series := BuildMatchRadarFromScoreboard(board, "escorte", 0, "ctf", games.DefaultEffectiveHpToKill, 0.90)
	obj := radarAxisOf(series, narrative.AxisObjective)
	if obj == nil || math.Abs(obj.Value-100) > 1e-9 {
		t.Fatalf("escorte : %+v, want 100 (soutien : part de 100 %% de son équipe de 4)", obj)
	}
	// Le porteur n'a aucun soutien : sa note suit sa part de l'objectif (14 / 20,5 x 4 / 2 > 1).
	series = BuildMatchRadarFromScoreboard(board, "porteur", 0, "ctf", games.DefaultEffectiveHpToKill, 0.90)
	if p := radarAxisOf(series, narrative.AxisObjective); p == nil || p.Value != 100 {
		t.Errorf("porteur : %+v, want 100", p)
	}
	// Un joueur de l'équipe qui n'a rien fait de l'objectif a 0 : mesuré, pas absent.
	series = BuildMatchRadarFromScoreboard(board, "c", 0, "ctf", games.DefaultEffectiveHpToKill, 0.90)
	if c := radarAxisOf(series, narrative.AxisObjective); c == nil || c.Value != 0 {
		t.Errorf("joueur sans action : %+v, want 0 mesuré", c)
	}
}

// Slayer, Halo 5 (aucun bloc d'objectif) ou joueur sans camp : l'axe est RETIRÉ, jamais 0.
func TestBuildMatchRadarFromScoreboard_ObjectifAbsentRetire(t *testing.T) {
	t.Parallel()
	slayer := []domain.ScoreboardRaw{radarCTFRow("x1", 0, domain.ObjectiveRaw{}), radarCTFRow("x2", 1, domain.ObjectiveRaw{})}
	series := BuildMatchRadarFromScoreboard(slayer, "x1", 0, "slayer", games.DefaultEffectiveHpToKill, 0.90)
	if len(series) != 1 {
		t.Fatalf("want 1 série, got %d", len(series))
	}
	if obj := radarAxisOf(series, narrative.AxisObjective); obj != nil {
		t.Errorf("axe objective sur un match Slayer: want retiré, got %+v", obj)
	}
	// Un bloc d'objectif mais aucun camp (donnée incomplète) : retiré aussi.
	noTeam := radarCTFRow("x1", 0, ctfActions(1, 1, 1, 1))
	noTeam.TeamID = nil
	series = BuildMatchRadarFromScoreboard([]domain.ScoreboardRaw{noTeam}, "x1", 0, "ctf", games.DefaultEffectiveHpToKill, 0.90)
	if obj := radarAxisOf(series, narrative.AxisObjective); obj != nil {
		t.Errorf("joueur sans camp: want axe retiré, got %+v", obj)
	}
}

func TestMatchModeFamilyFromMeta(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pairName string
		want     string
	}{
		{"Team Slayer", "slayer"},
		{"CTF Big Team", "ctf"},
		{"Strongholds Ranked", "strongholds"},
		{"Oddball Pro", "oddball"},
		{"Neutral Bomb", "oddball"},
		{"Custom Game", ""},
	}
	for _, tc := range cases {
		name := tc.pairName
		meta := &domain.MatchMetaRaw{PairName: &name}
		got := matchModeFamilyFromMeta(meta)
		if got != tc.want {
			t.Errorf("matchModeFamilyFromMeta(%q) want %q, got %q", tc.pairName, tc.want, got)
		}
	}
}

func TestMatchModeFamilyFromMeta_NilHandling(t *testing.T) {
	t.Parallel()
	if got := matchModeFamilyFromMeta(nil); got != "" {
		t.Errorf("nil meta: want empty, got %q", got)
	}
	if got := matchModeFamilyFromMeta(&domain.MatchMetaRaw{}); got != "" {
		t.Errorf("nil pair: want empty, got %q", got)
	}
}
