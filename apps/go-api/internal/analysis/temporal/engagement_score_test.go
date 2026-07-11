package temporal_test

import (
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// =============================================================================
// Helpers de construction d'inputs
// =============================================================================

// makeEvents construit une slice d'events avec un type donne et des TimeMS
// explicites. Tous les events portent le meme XUID (vide par defaut).
func makeEvents(eventType canonical.HighlightEventType, timesMS ...int64) []canonical.HighlightEvent {
	events := make([]canonical.HighlightEvent, len(timesMS))
	for i, t := range timesMS {
		events[i] = canonical.HighlightEvent{
			EventType: string(eventType),
			TimeMS:    t,
		}
	}
	return events
}

// concatEvents fusionne plusieurs slices d'events.
func concatEvents(slices ...[]canonical.HighlightEvent) []canonical.HighlightEvent {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	merged := make([]canonical.HighlightEvent, 0, total)
	for _, s := range slices {
		merged = append(merged, s...)
	}
	return merged
}

// makeHistory genere un historique synthetique de N matchs avec residus
// distribues lineairement entre min et max.
func makeHistory(n int, minBrut, maxBrut float64) []domain.HistoricalEngagementBrut {
	if n == 0 {
		return nil
	}
	hist := make([]domain.HistoricalEngagementBrut, n)
	if n == 1 {
		hist[0] = domain.HistoricalEngagementBrut{Brut: (minBrut + maxBrut) / 2}
		return hist
	}
	step := (maxBrut - minBrut) / float64(n-1)
	for i := 0; i < n; i++ {
		hist[i] = domain.HistoricalEngagementBrut{Brut: minBrut + float64(i)*step}
	}
	return hist
}

// =============================================================================
// Cas degeneres / boundaries
// =============================================================================

func TestComputeEngagementScore_InvalidBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		start int64
		end   int64
	}{
		{"end_before_start", 1000, 500},
		{"start_negative", -1, 1000},
		{"end_negative", 0, -1},
		{"equal_boundaries", 500, 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
				MatchStartMS: tc.start,
				MatchEndMS:   tc.end,
			})
			if !errors.Is(err, temporal.ErrInvalidBoundaries) {
				t.Errorf("expected ErrInvalidBoundaries, got %v", err)
			}
		})
	}
}

func TestComputeEngagementScore_MatchTooShort(t *testing.T) {
	// Match de 2 minutes (sous le seuil 3 min).
	_, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		MatchStartMS: 0,
		MatchEndMS:   120_000,
	})
	if !errors.Is(err, temporal.ErrMatchTooShort) {
		t.Errorf("expected ErrMatchTooShort, got %v", err)
	}
}

func TestComputeEngagementScore_NoEvents(t *testing.T) {
	// Match assez long mais aucun event (lobby vide / inactif total).
	_, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		MatchStartMS: 0,
		MatchEndMS:   720_000, // 12 min
	})
	if !errors.Is(err, temporal.ErrInsufficientData) {
		t.Errorf("expected ErrInsufficientData, got %v", err)
	}
}

// TestComputeEngagementScore_PaceTeamIncludesPlayer vérifie que la ligne
// "Équipe réelle" (pace_team) inclut le joueur cible au numérateur (cohérence
// avec NTeam qui le compte au dénominateur). Scénario : coéquipiers sans aucun
// event, joueur actif → pace_team doit être > 0 (avec l'ancienne logique
// "coéquipiers seuls" il serait 0 partout).
func TestComputeEngagementScore_PaceTeamIncludesPlayer(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000, 180_000, 240_000)
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     nil, // aucun coéquipier actif
		LobbyEvents:    playerEvents,
		NTeam:          2, // joueur + 1 coéquipier
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     360_000,
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MeanPaceTeam <= 0 {
		t.Errorf("MeanPaceTeam doit être > 0 (joueur inclus dans l'équipe), got %f", result.MeanPaceTeam)
	}
	// L'attendu (coef=1.0 × pace_team incl. joueur) doit aussi être > 0.
	var sumAttendu float64
	for _, p := range result.EngagementCurve {
		sumAttendu += p.PaceAttendu
	}
	if sumAttendu <= 0 {
		t.Errorf("PaceAttendu doit suivre pace_team (joueur inclus), somme=%f", sumAttendu)
	}
}

// =============================================================================
// Confidence vs taille de l'historique
// =============================================================================

func TestComputeEngagementScore_InsufficientHistory(t *testing.T) {
	// Match valide mais < HistoryMinPartial (10) matchs en historique.
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000, 180_000)
	teamEvents := makeEvents(canonical.EventKill, 30_000, 60_000, 90_000, 150_000)

	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     teamEvents,
		LobbyEvents:    concatEvents(playerEvents, teamEvents),
		NTeam:          4,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(5, -1.0, 1.0), // < 10 matchs
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EngagementScore != nil {
		t.Errorf("expected nil score on insufficient history, got %v", *result.EngagementScore)
	}
	if result.Confidence != "insufficient_history" {
		t.Errorf("expected confidence=insufficient_history, got %s", result.Confidence)
	}
	if result.NHistoryMatches != 5 {
		t.Errorf("expected NHistoryMatches=5, got %d", result.NHistoryMatches)
	}
	// La courbe doit etre presente meme sans score (utile pour Match View).
	if len(result.EngagementCurve) == 0 {
		t.Error("curve should be present even without score")
	}
}

func TestComputeEngagementScore_PartialHistory(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000, 180_000)
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     makeEvents(canonical.EventKill, 30_000, 90_000),
		LobbyEvents:    playerEvents,
		NTeam:          4,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(15, -1.0, 1.0), // entre 10 et 30
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EngagementScore == nil {
		t.Fatal("expected non-nil score with partial history")
	}
	if result.Confidence != "partial" {
		t.Errorf("expected confidence=partial, got %s", result.Confidence)
	}
}

func TestComputeEngagementScore_FullHistory(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000)
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     makeEvents(canonical.EventKill, 90_000),
		LobbyEvents:    playerEvents,
		NTeam:          4,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(50, -2.0, 2.0), // >= 30
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EngagementScore == nil {
		t.Fatal("expected non-nil score with full history")
	}
	if result.Confidence != "full" {
		t.Errorf("expected confidence=full, got %s", result.Confidence)
	}
}

// =============================================================================
// Comportement attendu sur scenarios narratifs
// =============================================================================

// TestComputeEngagementScore_PlayerAboveExpectedScoresHigh : si le joueur
// produit constamment plus que son attendu, son residual brut est positif et
// son percentile vs un historique centre sur 0 est > 50.
func TestComputeEngagementScore_PlayerAboveExpectedScoresHigh(t *testing.T) {
	// 12 events joueur uniformement repartis (1 toutes les minutes).
	playerEvents := []canonical.HighlightEvent{}
	for i := 1; i <= 12; i++ {
		playerEvents = append(playerEvents, canonical.HighlightEvent{
			EventType: string(canonical.EventKill),
			TimeMS:    int64(i) * 60_000,
		})
	}
	// Equipe (3 autres alliés) genere globalement 3 events sur 12 min : pace
	// team_per_player tres bas. Avec coef=1.0, attendu = team -> joueur tres
	// au-dessus de l'attendu.
	teamEvents := makeEvents(canonical.EventKill, 200_000, 400_000, 600_000)

	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     teamEvents,
		LobbyEvents:    concatEvents(playerEvents, teamEvents),
		NTeam:          4,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(50, -2.0, 2.0),
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResidualBrut <= 0 {
		t.Errorf("expected positive residual when player exceeds expected, got %v", result.ResidualBrut)
	}
	if result.EngagementScore == nil || *result.EngagementScore <= 50 {
		t.Errorf("expected score > 50 when player exceeds historical median, got %v", result.EngagementScore)
	}
}

// TestComputeEngagementScore_PlayerBelowExpectedScoresLow : symetrique du
// precedent. Joueur sous-engage -> percentile < 50.
func TestComputeEngagementScore_PlayerBelowExpectedScoresLow(t *testing.T) {
	// Joueur passif : seulement 2 events sur 12 min.
	playerEvents := makeEvents(canonical.EventKill, 200_000, 600_000)
	// Equipe tres active : 24 events distribués -> pace_team eleve.
	teamEvents := []canonical.HighlightEvent{}
	for i := 1; i <= 24; i++ {
		teamEvents = append(teamEvents, canonical.HighlightEvent{
			EventType: string(canonical.EventKill),
			TimeMS:    int64(i) * 30_000, // 1 toutes les 30s
		})
	}

	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     teamEvents,
		LobbyEvents:    concatEvents(playerEvents, teamEvents),
		NTeam:          4,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(50, -2.0, 2.0), // mediane = 0
		CoefTeamShare:  1.0,
		CoefLobbyShare: 1.0,
		IsTeamMode:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResidualBrut >= 0 {
		t.Errorf("expected negative residual when player below expected, got %v", result.ResidualBrut)
	}
	if result.EngagementScore == nil || *result.EngagementScore >= 50 {
		t.Errorf("expected score < 50 when player below historical median, got %v", result.EngagementScore)
	}
}

// =============================================================================
// MatchIntensity
// =============================================================================

func TestComputeEngagementScore_MatchIntensityComputed(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000)
	lobbyEvents := []canonical.HighlightEvent{}
	for i := 1; i <= 80; i++ {
		lobbyEvents = append(lobbyEvents, canonical.HighlightEvent{
			EventType: string(canonical.EventKill),
			TimeMS:    int64(i) * 9_000,
		})
	}
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:  playerEvents,
		TeamEvents:    makeEvents(canonical.EventKill, 30_000),
		LobbyEvents:   lobbyEvents,
		NTeam:         4,
		NHumansLobby:  8,
		MatchStartMS:  0,
		MatchEndMS:    720_000,
		History:       makeHistory(50, -1.0, 1.0),
		CoefTeamShare: 1.0,
		IsTeamMode:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 80 events / 8 humains / 12 min = ~0.833 events/min/joueur.
	expected := 80.0 / 8.0 / 12.0
	if math.Abs(result.MatchIntensity-expected) > 0.01 {
		t.Errorf("expected MatchIntensity ~%v, got %v", expected, result.MatchIntensity)
	}
}

// =============================================================================
// Mode FFA (fallback lobby)
// =============================================================================

func TestComputeEngagementScore_FFAFallbackToLobby(t *testing.T) {
	// FFA : NTeam=1 et IsTeamMode=false. L'attendu doit etre calcule via
	// lobby_share au lieu de team_share. On verifie que le calcul ne panique
	// pas et produit une courbe coherente.
	playerEvents := makeEvents(canonical.EventKill, 60_000, 120_000, 180_000)
	lobbyEvents := []canonical.HighlightEvent{}
	for i := 1; i <= 30; i++ {
		lobbyEvents = append(lobbyEvents, canonical.HighlightEvent{
			EventType: string(canonical.EventKill),
			TimeMS:    int64(i) * 20_000,
		})
	}

	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:   playerEvents,
		TeamEvents:     nil,
		LobbyEvents:    lobbyEvents,
		NTeam:          1,
		NHumansLobby:   8,
		MatchStartMS:   0,
		MatchEndMS:     720_000,
		History:        makeHistory(50, -1.0, 1.0),
		CoefTeamShare:  0, // ignore en FFA
		CoefLobbyShare: 1.05,
		IsTeamMode:     false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.EngagementCurve) == 0 {
		t.Error("expected non-empty curve in FFA mode")
	}
}

// =============================================================================
// Attendu ancre lobby : bins d'intensite / fallback global / cold-start
// =============================================================================

// meanAttendu calcule la moyenne de PaceAttendu sur la courbe.
func meanAttendu(curve []domain.EngagementPoint) float64 {
	if len(curve) == 0 {
		return 0
	}
	var s float64
	for _, p := range curve {
		s += p.PaceAttendu
	}
	return s / float64(len(curve))
}

// activeLobby genere un lobby actif (60 events sur 12 min) — meanLobby largement
// au-dessus de 0.01, pour tomber dans le bin chaotique des tests ci-dessous.
func activeLobby() []canonical.HighlightEvent {
	ev := make([]canonical.HighlightEvent, 0, 60)
	for i := 1; i <= 60; i++ {
		ev = append(ev, canonical.HighlightEvent{EventType: string(canonical.EventKill), TimeMS: int64(i) * 12_000})
	}
	return ev
}

// TestComputeEngagementScore_BinChaotiqueLowExpected : un joueur dont le bin
// chaotique a un coef bas (repond mal aux matchs intenses) doit avoir, sur un
// match intense, un attendu BAS (coef du bin chaotique), et ExpectedBasis "bin".
func TestComputeEngagementScore_BinChaotiqueLowExpected(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 180_000, 300_000, 480_000)
	lobby := activeLobby()
	// Bornes basses pour que meanLobby (match actif) tombe dans le bin chaotique.
	bins := &domain.EngagementResponseBins{
		Bins: []domain.EngagementIntensityBin{
			{Bin: temporal.IntensityBinCalme, LowerBound: 0, UpperBound: 0.001, CoefLobby: 1.5, NMatches: 20},
			{Bin: temporal.IntensityBinStandard, LowerBound: 0.001, UpperBound: 0.002, CoefLobby: 1.0, NMatches: 20},
			{Bin: temporal.IntensityBinChaotique, LowerBound: 0.002, UpperBound: 1000, CoefLobby: 0.3, NMatches: 20},
		},
	}
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:       playerEvents,
		LobbyEvents:        lobby,
		NTeam:              4,
		NHumansLobby:       8,
		MatchStartMS:       0,
		MatchEndMS:         720_000,
		History:            makeHistory(50, -1.0, 1.0),
		CoefLobbyShare:     1.2,
		HasGlobalLobbyCoef: true, // present mais le bin doit primer
		ResponseBins:       bins,
		IsTeamMode:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExpectedBasis != domain.ExpectedBasisBin {
		t.Errorf("ExpectedBasis want bin, got %q", result.ExpectedBasis)
	}
	if result.IntensityBin != temporal.IntensityBinChaotique {
		t.Errorf("IntensityBin want chaotique, got %q", result.IntensityBin)
	}
	// Attendu = 0.3 x pace_lobby ; comparer a ce que donnerait le coef calme (1.5).
	got := meanAttendu(result.EngagementCurve)
	wantChaotique := 0.3 * result.MeanPaceLobby
	if math.Abs(got-wantChaotique) > 1e-9 {
		t.Errorf("mean attendu want %v (0.3 x lobby), got %v", wantChaotique, got)
	}
	if got >= 1.5*result.MeanPaceLobby {
		t.Errorf("attendu chaotique doit etre bas (< coef calme x lobby)")
	}
}

// TestComputeEngagementScore_FallbackGlobal : sans bins exploitables mais avec
// un coef lobby global, l'attendu utilise le coef global (ExpectedBasis "global").
func TestComputeEngagementScore_FallbackGlobal(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 180_000, 300_000)
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:       playerEvents,
		LobbyEvents:        activeLobby(),
		NTeam:              4,
		NHumansLobby:       8,
		MatchStartMS:       0,
		MatchEndMS:         720_000,
		History:            makeHistory(50, -1.0, 1.0),
		CoefLobbyShare:     0.8,
		HasGlobalLobbyCoef: true,
		ResponseBins:       nil, // aucun bin persiste
		IsTeamMode:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExpectedBasis != domain.ExpectedBasisGlobal {
		t.Errorf("ExpectedBasis want global, got %q", result.ExpectedBasis)
	}
	if result.IntensityBin != "" {
		t.Errorf("IntensityBin want empty hors bin, got %q", result.IntensityBin)
	}
	if got, want := meanAttendu(result.EngagementCurve), 0.8*result.MeanPaceLobby; math.Abs(got-want) > 1e-9 {
		t.Errorf("mean attendu want %v (0.8 x lobby), got %v", want, got)
	}
}

// TestComputeEngagementScore_ColdStartBasis : sans bins ni coef global,
// ExpectedBasis "cold_start" et attendu = 1.0 x pace_lobby.
func TestComputeEngagementScore_ColdStartBasis(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 180_000, 300_000)
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:       playerEvents,
		LobbyEvents:        activeLobby(),
		NTeam:              4,
		NHumansLobby:       8,
		MatchStartMS:       0,
		MatchEndMS:         720_000,
		History:            makeHistory(50, -1.0, 1.0),
		HasGlobalLobbyCoef: false,
		ResponseBins:       nil,
		IsTeamMode:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExpectedBasis != domain.ExpectedBasisColdStart {
		t.Errorf("ExpectedBasis want cold_start, got %q", result.ExpectedBasis)
	}
	if got, want := meanAttendu(result.EngagementCurve), 1.0*result.MeanPaceLobby; math.Abs(got-want) > 1e-9 {
		t.Errorf("mean attendu want %v (1.0 x lobby), got %v", want, got)
	}
}

// TestComputeEngagementScore_FFASameExpectedAsTeam : l'attendu ne depend plus du
// mode (ancre lobby unifiee D1). A lobby identique, FFA (NTeam=1) et mode equipe
// produisent le meme attendu.
func TestComputeEngagementScore_FFASameExpectedAsTeam(t *testing.T) {
	playerEvents := makeEvents(canonical.EventKill, 60_000, 180_000, 300_000)
	lobby := activeLobby()
	base := temporal.EngagementScoreInput{
		PlayerEvents:       playerEvents,
		LobbyEvents:        lobby,
		NHumansLobby:       8,
		MatchStartMS:       0,
		MatchEndMS:         720_000,
		History:            makeHistory(50, -1.0, 1.0),
		CoefLobbyShare:     1.1,
		HasGlobalLobbyCoef: true,
	}
	teamInput := base
	teamInput.NTeam = 4
	teamInput.IsTeamMode = true
	ffaInput := base
	ffaInput.NTeam = 1
	ffaInput.IsTeamMode = false

	teamRes, err := temporal.ComputeEngagementScore(teamInput)
	if err != nil {
		t.Fatalf("team: %v", err)
	}
	ffaRes, err := temporal.ComputeEngagementScore(ffaInput)
	if err != nil {
		t.Fatalf("ffa: %v", err)
	}
	if math.Abs(meanAttendu(teamRes.EngagementCurve)-meanAttendu(ffaRes.EngagementCurve)) > 1e-9 {
		t.Errorf("attendu team vs FFA doivent etre identiques (ancre lobby) : %v vs %v",
			meanAttendu(teamRes.EngagementCurve), meanAttendu(ffaRes.EngagementCurve))
	}
}

// =============================================================================
// EventsObjectifEstimes (helper modes asymetriques)
// =============================================================================

func TestEventsObjectifEstimes(t *testing.T) {
	cases := []struct {
		name          string
		personalScore int
		kills         int
		assists       int
		want          float64
	}{
		{"slayer_pur_no_objectif", 1500, 15, 0, 0},
		{"slayer_avec_assists_no_objectif", 1100, 10, 2, 0}, // 1100 - 1000 - 100 = 0
		{"oddball_objectif_dominant", 2000, 5, 0, 60.0},     // (2000-500)/25 = 60
		{"ctf_quelques_captures", 1300, 10, 2, 8.0},         // (1300-1000-100)/25 = 8
		{"score_below_kills_clamped_zero", 800, 10, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := temporal.EventsObjectifEstimes(tc.personalScore, tc.kills, tc.assists)
			if math.Abs(got-tc.want) > 0.001 {
				t.Errorf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

// =============================================================================
// Annotations deaths (passive vs active)
// =============================================================================

func TestComputeEngagementScore_PassiveDeathDetected(t *testing.T) {
	// Joueur fait un kill a 30s, puis silence 60s, puis death a 90s.
	// La mort est PASSIVE (creux > 30s avant la mort).
	playerEvents := []canonical.HighlightEvent{
		{EventType: string(canonical.EventKill), TimeMS: 30_000},
		{EventType: string(canonical.EventDeath), TimeMS: 90_000},
	}
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:  playerEvents,
		TeamEvents:    makeEvents(canonical.EventKill, 60_000),
		LobbyEvents:   playerEvents,
		NTeam:         4,
		NHumansLobby:  8,
		MatchStartMS:  0,
		MatchEndMS:    720_000,
		History:       makeHistory(50, -1.0, 1.0),
		CoefTeamShare: 1.0,
		IsTeamMode:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Trouver le point qui correspond a la mort (90s = TimeMS 90_000).
	foundPassive := false
	for _, p := range result.EngagementCurve {
		if p.IsPassiveDeath {
			foundPassive = true
			break
		}
	}
	if !foundPassive {
		t.Error("expected at least one IsPassiveDeath flag set on the death point")
	}
}

func TestComputeEngagementScore_ActiveDeathNotMarkedPassive(t *testing.T) {
	// Joueur fait un kill a 90s, puis death a 100s (10s seulement apres).
	// Mort ACTIVE (creux < 30s).
	playerEvents := []canonical.HighlightEvent{
		{EventType: string(canonical.EventKill), TimeMS: 90_000},
		{EventType: string(canonical.EventDeath), TimeMS: 100_000},
	}
	result, err := temporal.ComputeEngagementScore(temporal.EngagementScoreInput{
		PlayerEvents:  playerEvents,
		TeamEvents:    makeEvents(canonical.EventKill, 60_000),
		LobbyEvents:   playerEvents,
		NTeam:         4,
		NHumansLobby:  8,
		MatchStartMS:  0,
		MatchEndMS:    720_000,
		History:       makeHistory(50, -1.0, 1.0),
		CoefTeamShare: 1.0,
		IsTeamMode:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range result.EngagementCurve {
		if p.IsPassiveDeath {
			t.Error("expected no IsPassiveDeath flag (active death scenario)")
			break
		}
	}
}
