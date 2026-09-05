package sessionusage

// usage_test.go — le contrat de calcul de ComputeUsage : couverture partielle,
// attribution par camp, parités sur effectifs inégaux, cadences sur la durée
// MESURÉE seule, étendue + matchs au-dessus de la parité DU match, lignes
// d'escouade, FFA (parts d'équipe nil, jamais 0), ventilations socles/bonus.

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

const eps = 1e-9

func closeTo(got *float64, want float64) bool {
	return got != nil && math.Abs(*got-want) < 1e-6
}

func intp(v int) *int { return &v }

// sessionDeTest — 3 matchs, m3 NON mesuré (et porteur de valeurs énormes qui ne
// doivent JAMAIS compter). Camps : joueur P + allié A (+ B au m2) contre E1/E2.
func sessionDeTest() Input {
	return Input{
		PlayerXUID: "P",
		SquadXUIDs: []string{"A"},
		Matches: []MatchInput{
			{
				MatchID: "m1", Measured: true, DurationSeconds: 600,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "A": 0, "E1": 1, "E2": 1},
				TeamSize:   2, LobbySize: 4,
				PadUnnamed:     3,
				PowerupPickups: map[string]int{"powerup_camo": 2},
				Players: []PlayerRow{
					{MatchID: "m1", XUID: "P", PadPickups: 1},
					{MatchID: "m1", XUID: "A", PadPickups: 2, DeployedByFamily: map[string]int{"wall": 1}},
					{MatchID: "m1", XUID: "E1", PadPickups: 3},
				},
			},
			{
				MatchID: "m2", Measured: true, DurationSeconds: 300,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "A": 0, "B": 0, "E1": 1, "E2": 1},
				TeamSize:   3, LobbySize: 5,
				PadUnnamed:     1,
				PowerupPickups: map[string]int{"powerup_camo": 1, "powerup_overshield": 1},
				Players: []PlayerRow{
					{MatchID: "m2", XUID: "P", PadPickups: 4},
					{MatchID: "m2", XUID: "B", PadPickups: 1},
					{MatchID: "m2", XUID: "E1", PadPickups: 1},
				},
			},
			{
				MatchID: "m3", Measured: false,
				Players: []PlayerRow{{MatchID: "m3", XUID: "P", PadPickups: 999}},
			},
		},
	}
}

// findMetric retourne la métrique de clé donnée, ou échoue.
func findMetric(t *testing.T, metrics []domain.SessionUsageMetric, key string) domain.SessionUsageMetric {
	t.Helper()
	for i := range metrics {
		if metrics[i].Key == key {
			return metrics[i]
		}
	}
	t.Fatalf("métrique %q absente : %+v", key, metrics)
	return domain.SessionUsageMetric{}
}

func TestComputeUsage_CouvertureEtSommesSurMesuresSeuls(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	if !out.Available {
		t.Fatal("Available = false, attendu true")
	}
	if out.MatchesMeasured != 2 || out.MatchesTotal != 3 {
		t.Errorf("couverture = %d/%d, attendu 2/3", out.MatchesMeasured, out.MatchesTotal)
	}
	if out.MeasuredDurationSeconds != 900 {
		t.Errorf("durée mesurée = %v, attendu 900 (m3 non mesuré exclu)", out.MeasuredDurationSeconds)
	}
	if out.PadUnnamedTotal != 4 {
		t.Errorf("pad_unnamed_total = %d, attendu 4", out.PadUnnamedTotal)
	}
	m := findMetric(t, out.Metrics, MetricPadPickups)
	if m.PlayerTotal != 5 || !closeTo(m.TeamTotal, 8) || m.LobbyTotal != 12 {
		t.Errorf("(joueur, camp, lobby) = (%v, %v, %v), attendu (5, 8, 12) — m3 (999) exclu",
			m.PlayerTotal, m.TeamTotal, m.LobbyTotal)
	}
	if !closeTo(m.TeamShareOfLobbyPct, 100*8.0/12) {
		t.Errorf("camp/lobby = %v, attendu 66.67", m.TeamShareOfLobbyPct)
	}
	if !closeTo(m.PlayerShareOfTeamPct, 62.5) || !closeTo(m.PlayerShareOfLobbyPct, 100*5.0/12) {
		t.Errorf("parts joueur = (%v, %v), attendu (62.5, 41.67)", m.PlayerShareOfTeamPct, m.PlayerShareOfLobbyPct)
	}
}

func TestComputeUsage_PariteSurEffectifsInegaux(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	if math.Abs(out.TeamSizeAvg-2.5) > eps || math.Abs(out.LobbySizeAvg-4.5) > eps {
		t.Errorf("effectifs moyens = (%v, %v), attendu (2.5, 4.5)", out.TeamSizeAvg, out.LobbySizeAvg)
	}
	if !closeTo(out.TeamParityPct, 40) {
		t.Errorf("parité équipe = %v, attendu 40 (100/2.5)", out.TeamParityPct)
	}
	if !closeTo(out.LobbyParityPct, 100/4.5) {
		t.Errorf("parité lobby = %v, attendu 22.22 (100/4.5)", out.LobbyParityPct)
	}
}

func TestComputeUsage_CadenceSurDureeMesureeSeule(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	m := findMetric(t, out.Metrics, MetricPadPickups)
	// 5 prises sur 900 s mesurées = 3.333/10 min. Si m3 comptait dans la durée
	// (ou dans les prises), la cadence serait fausse.
	if !closeTo(m.PlayerPer10Min, 5*600.0/900) {
		t.Errorf("cadence joueur = %v, attendu 3.333", m.PlayerPer10Min)
	}
	if !closeTo(m.TeamPer10Min, 8*600.0/900) || !closeTo(m.LobbyPer10Min, 12*600.0/900) {
		t.Errorf("cadences camp/lobby = (%v, %v), attendu (5.333, 8)", m.TeamPer10Min, m.LobbyPer10Min)
	}
}

func TestComputeUsage_EtendueEtMatchsAuDessusDeLaParite(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	m := findMetric(t, out.Metrics, MetricPadPickups)
	// m1 : joueur/équipe = 1/3 = 33.33 (parité du match 50 → pas au-dessus) ;
	// m2 : 4/5 = 80 (parité 33.33 → au-dessus).
	if !closeTo(m.PlayerShareOfTeamMinPct, 100.0/3) || !closeTo(m.PlayerShareOfTeamMaxPct, 80) {
		t.Errorf("étendue = (%v, %v), attendu (33.33, 80)", m.PlayerShareOfTeamMinPct, m.PlayerShareOfTeamMaxPct)
	}
	if m.MatchesAboveTeamParity == nil || *m.MatchesAboveTeamParity != 1 {
		t.Errorf("au-dessus parité équipe = %v, attendu 1 (parité DU match, pas de la session)", m.MatchesAboveTeamParity)
	}
	// m1 : joueur/lobby = 1/6 = 16.67 (parité 25 → non) ; m2 : 4/6 = 66.67 (parité 20 → oui).
	if m.MatchesAboveLobbyParity != 1 {
		t.Errorf("au-dessus parité lobby = %d, attendu 1", m.MatchesAboveLobbyParity)
	}
	if len(m.PerMatch) != 2 || m.PerMatch[0].MatchID != "m1" || m.PerMatch[1].MatchID != "m2" {
		t.Fatalf("per_match = %+v, attendu [m1, m2] dans l'ordre de la session", m.PerMatch)
	}
}

func TestComputeUsage_LignesEscouade(t *testing.T) {
	out := ComputeUsage(sessionDeTest())
	m := findMetric(t, out.Metrics, MetricPadPickups)
	if len(m.Squad) != 1 || m.Squad[0].XUID != "A" {
		t.Fatalf("lignes escouade = %+v, attendu une ligne pour A", m.Squad)
	}
	l := m.Squad[0]
	if l.Total != 2 {
		t.Errorf("total A = %v, attendu 2 (m1 seulement, m3 exclu)", l.Total)
	}
	if !closeTo(l.ShareOfTeamPct, 25) || !closeTo(l.ShareOfLobbyPct, 100*2.0/12) {
		t.Errorf("parts A = (%v, %v), attendu (25, 16.67)", l.ShareOfTeamPct, l.ShareOfLobbyPct)
	}
	if !closeTo(l.Per10Min, 2*600.0/900) {
		t.Errorf("cadence A = %v, attendu 1.333", l.Per10Min)
	}
	// Métrique dynamique (deployed_wall observée sur A) : ligne escouade aussi.
	w := findMetric(t, out.Metrics, MetricDeployedPrefix+"wall")
	if len(w.Squad) != 1 || w.Squad[0].Total != 1 {
		t.Errorf("deployed_wall escouade = %+v, attendu total 1 pour A", w.Squad)
	}
}

func TestComputeUsage_FFAPartsEquipeNilJamaisZero(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{{
			MatchID: "ffa", Measured: true, DurationSeconds: 300,
			PlayerTeam: nil, LobbySize: 3,
			Players: []PlayerRow{
				{MatchID: "ffa", XUID: "P", GrapplePulls: 2},
				{MatchID: "ffa", XUID: "E1", GrapplePulls: 1},
			},
		}},
	}
	out := ComputeUsage(in)
	m := findMetric(t, out.Metrics, MetricGrapplePulls)
	if m.TeamShareOfLobbyPct != nil || m.PlayerShareOfTeamPct != nil {
		t.Errorf("parts d'équipe FFA = (%v, %v), attendu nil (jamais 0)", m.TeamShareOfLobbyPct, m.PlayerShareOfTeamPct)
	}
	if m.TeamTotal != nil {
		t.Errorf("camp FFA = %v, attendu nil (aucun camp connu — jamais un 0 inventé)", *m.TeamTotal)
	}
	// C2 : une cadence d'équipe sur une session sans camp connu serait inventée.
	if m.TeamPer10Min != nil {
		t.Errorf("cadence d'équipe FFA = %v, attendu nil (jamais &0)", *m.TeamPer10Min)
	}
	if m.MatchesAboveTeamParity != nil {
		t.Errorf("au-dessus parité équipe FFA = %v, attendu nil", *m.MatchesAboveTeamParity)
	}
	if !closeTo(m.PlayerPer10Min, 2*600.0/300) {
		t.Errorf("cadence joueur FFA = %v, attendu 4 (le joueur, lui, mesure)", m.PlayerPer10Min)
	}
	if !closeTo(m.PlayerShareOfLobbyPct, 100*2.0/3) {
		t.Errorf("part lobby = %v, attendu 66.67 (le lobby, lui, mesure)", m.PlayerShareOfLobbyPct)
	}
	if out.TeamParityPct != nil {
		t.Errorf("parité équipe FFA = %v, attendu nil", out.TeamParityPct)
	}
	if !closeTo(out.LobbyParityPct, 100.0/3) {
		t.Errorf("parité lobby = %v, attendu 33.33", out.LobbyParityPct)
	}
}

// C1 : une session qui MÉLANGE match en équipe et FFA ne doit jamais croiser
// les scopes. Avant correctif : joueur sommé sur les deux matchs (6) contre une
// équipe sommée sur le seul match à camp connu (1) → part joueur/équipe 600 %,
// et camp/lobby dilué par le lobby FFA (1/12 = 8,33 %). Après : les grandeurs
// d'équipe se calculent sur le seul m1 (numérateurs ET dénominateurs).
func TestComputeUsage_SessionMixteEquipeEtFFAResteDansLeScope(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{
			{
				MatchID: "m1", Measured: true, DurationSeconds: 600,
				PlayerTeam: intp(0),
				TeamOf:     map[string]int{"P": 0, "E1": 1},
				TeamSize:   1, LobbySize: 2,
				Players: []PlayerRow{
					{MatchID: "m1", XUID: "P", GrapplePulls: 1, PadPickupsByFamily: map[string]int{"aabbccdd": 1}},
					{MatchID: "m1", XUID: "E1", GrapplePulls: 1},
				},
			},
			{
				MatchID: "m2", Measured: true, DurationSeconds: 300,
				PlayerTeam: nil, LobbySize: 5,
				Players: []PlayerRow{
					{MatchID: "m2", XUID: "P", GrapplePulls: 5, PadPickupsByFamily: map[string]int{"aabbccdd": 5}},
					{MatchID: "m2", XUID: "X", GrapplePulls: 5},
				},
			},
		},
	}
	out := ComputeUsage(in)
	m := findMetric(t, out.Metrics, MetricGrapplePulls)
	// Scope complet pour le joueur et le lobby.
	if m.PlayerTotal != 6 || m.LobbyTotal != 12 || !closeTo(m.PlayerShareOfLobbyPct, 50) {
		t.Errorf("joueur/lobby = (%v, %v, %v), attendu (6, 12, 50 %%)",
			m.PlayerTotal, m.LobbyTotal, m.PlayerShareOfLobbyPct)
	}
	// Scope camp connu (m1 seul) pour tout ce qui est relatif à l'équipe.
	if !closeTo(m.TeamTotal, 1) {
		t.Errorf("camp = %v, attendu 1 (m1 seul)", m.TeamTotal)
	}
	if !closeTo(m.PlayerShareOfTeamPct, 100) {
		t.Errorf("joueur/équipe = %v, attendu 100 %% — un croisement de scopes donnerait 600 %%", m.PlayerShareOfTeamPct)
	}
	if !closeTo(m.TeamShareOfLobbyPct, 50) {
		t.Errorf("camp/lobby = %v, attendu 50 %% (1/2 sur m1) — un croisement donnerait 8.33 %%", m.TeamShareOfLobbyPct)
	}
	// Cadence d'équipe sur la durée des matchs à camp connu (600 s), pas 900.
	if !closeTo(m.TeamPer10Min, 1) {
		t.Errorf("cadence d'équipe = %v, attendu 1 (1 sur 600 s) — sur 900 s elle serait 0.667", m.TeamPer10Min)
	}
	if m.MatchesAboveTeamParity == nil || *m.MatchesAboveTeamParity != 0 {
		t.Errorf("au-dessus parité équipe = %v, attendu 0 (connu, pas nil : m1 a un camp)", m.MatchesAboveTeamParity)
	}
	// Cadences joueur/lobby : scope complet (900 s).
	if !closeTo(m.PlayerPer10Min, 4) || !closeTo(m.LobbyPer10Min, 8) {
		t.Errorf("cadences joueur/lobby = (%v, %v), attendu (4, 8)", m.PlayerPer10Min, m.LobbyPer10Min)
	}
	// Même règle sur la ventilation par famille (computePadFamilies).
	if len(out.PadFamilies) != 1 {
		t.Fatalf("pad_families = %+v, attendu la seule famille aabbccdd", out.PadFamilies)
	}
	fam := out.PadFamilies[0]
	if fam.PlayerTotal != 6 || !closeTo(fam.TeamTotal, 1) || !closeTo(fam.PlayerShareOfTeamPct, 100) {
		t.Errorf("famille = (joueur %v, camp %v, joueur/équipe %v), attendu (6, 1, 100 %%)",
			fam.PlayerTotal, fam.TeamTotal, fam.PlayerShareOfTeamPct)
	}
}

// C6 : un match mesuré SANS échelle de temps (durée 0, aucun repli) reste dans
// les totaux et les parts mais sort des cadences — numérateur ET dénominateur.
// Avant correctif, ses prises entraient au numérateur sans durée au
// dénominateur : cadence joueur 5*600/600 = 5 au lieu de 1.
func TestComputeUsage_MatchSansDureeExcluDesCadences(t *testing.T) {
	in := Input{
		PlayerXUID: "P",
		Matches: []MatchInput{
			{
				MatchID: "m1", Measured: true, DurationSeconds: 600,
				PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0, "E1": 1},
				TeamSize: 1, LobbySize: 2,
				Players: []PlayerRow{
					{MatchID: "m1", XUID: "P", PadPickups: 1},
					{MatchID: "m1", XUID: "E1", PadPickups: 1},
				},
			},
			{
				MatchID: "m2", Measured: true, DurationSeconds: 0, // artefact sans échelle de temps
				PlayerTeam: intp(0), TeamOf: map[string]int{"P": 0, "E1": 1},
				TeamSize: 1, LobbySize: 2,
				Players: []PlayerRow{
					{MatchID: "m2", XUID: "P", PadPickups: 4},
					{MatchID: "m2", XUID: "E1", PadPickups: 2},
				},
			},
		},
	}
	out := ComputeUsage(in)
	// La durée mesurée ne compte que les matchs à durée connue.
	if out.MeasuredDurationSeconds != 600 {
		t.Errorf("durée mesurée = %v, attendu 600 (m2 sans échelle de temps)", out.MeasuredDurationSeconds)
	}
	m := findMetric(t, out.Metrics, MetricPadPickups)
	// Totaux et parts : m2 reste compté.
	if m.PlayerTotal != 5 || m.LobbyTotal != 8 || !closeTo(m.TeamTotal, 5) {
		t.Errorf("totaux = (joueur %v, lobby %v, camp %v), attendu (5, 8, 5)",
			m.PlayerTotal, m.LobbyTotal, m.TeamTotal)
	}
	if len(m.PerMatch) != 2 {
		t.Errorf("per_match = %+v, attendu m1 ET m2 (les parts d'un match sans durée restent valides)", m.PerMatch)
	}
	// Cadences : m2 exclu du numérateur ET du dénominateur.
	if !closeTo(m.PlayerPer10Min, 1) {
		t.Errorf("cadence joueur = %v, attendu 1 (1 sur 600 s) — 5 = numérateur gonflé par m2", m.PlayerPer10Min)
	}
	if !closeTo(m.TeamPer10Min, 1) || !closeTo(m.LobbyPer10Min, 2) {
		t.Errorf("cadences camp/lobby = (%v, %v), attendu (1, 2)", m.TeamPer10Min, m.LobbyPer10Min)
	}
}

func TestComputeUsage_SessionSansMatchMesure(t *testing.T) {
	out := ComputeUsage(Input{PlayerXUID: "P", Matches: []MatchInput{{MatchID: "m1"}}})
	if !out.Available || out.MatchesMeasured != 0 || out.MatchesTotal != 1 {
		t.Errorf("bloc = %+v, attendu Available 0/1 (jamais nil : l'écran dit 0 mesuré)", out)
	}
	if len(out.Metrics) != 0 {
		t.Errorf("métriques = %+v, attendu aucune", out.Metrics)
	}
}

func TestComputeUsage_VentilationsSoclesEtBonus(t *testing.T) {
	in := sessionDeTest()
	in.Matches[0].Players[0].PadPickupsByFamily = map[string]int{"aabbccdd": 1}
	in.Matches[0].Players[2].PadPickupsByFamily = map[string]int{"aabbccdd": 2, "eeff0011": 1}
	out := ComputeUsage(in)
	if len(out.PadFamilies) != 2 || out.PadFamilies[0].FamilyKey != "aabbccdd" {
		t.Fatalf("pad_families = %+v, attendu aabbccdd (volume 3) en premier", out.PadFamilies)
	}
	f := out.PadFamilies[0]
	if f.PlayerTotal != 1 || !closeTo(f.TeamTotal, 1) || f.LobbyTotal != 3 {
		t.Errorf("famille aabbccdd = (%v, %v, %v), attendu (1, 1, 3)", f.PlayerTotal, f.TeamTotal, f.LobbyTotal)
	}
	// Bonus : anonymes par famille, total + cadence, JAMAIS de part.
	if len(out.PowerupPickups) != 2 ||
		out.PowerupPickups[0].FamilyKey != "powerup_camo" || out.PowerupPickups[0].Occupations != 3 ||
		out.PowerupPickups[1].FamilyKey != "powerup_overshield" || out.PowerupPickups[1].Occupations != 1 {
		t.Fatalf("powerup_pickups = %+v, attendu camo=3 puis overshield=1", out.PowerupPickups)
	}
	if !closeTo(out.PowerupPickups[0].Per10Min, 3*600.0/900) {
		t.Errorf("cadence camo = %v, attendu 2", out.PowerupPickups[0].Per10Min)
	}
}
