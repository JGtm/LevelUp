package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// mockHighlightEventsRepo implémente highlightEventsLoader (même seam que
// Timeseries) pour le golden T0 : renvoie un set fixe d'events canoniques (au
// référentiel film, le service applique CorrectEvents).
type mockHighlightEventsRepo struct {
	events []canonical.HighlightEvent
}

func (m *mockHighlightEventsRepo) Load(
	_ context.Context, _ port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	return m.events, nil
}

// cadenceBucketForXUID retourne l'index de phase (bucket) où la cadence place
// les kills de xuid, ou -1 si aucun. Inspecte CombatTab.Cadence (ChartPointStacked
// par phase, Components = xuid -> kills).
func cadenceBucketForXUID(resp domain.MatchViewResponse, xuid string) int {
	if resp.CombatTab.Cadence == nil {
		return -1
	}
	for i, dp := range resp.CombatTab.Cadence.Datapoints {
		if dp.Components[xuid] > 0 {
			return i
		}
	}
	return -1
}

// TestMatchViewService_T0_EventsHonorRealMatchStart est le golden de HIGH-A :
// il verrouille que TOUTES les sources d'events de Match View sont recalées sur
// le vrai début de match (T0) en un point unique — la cadence (canonicalEvents)
// ET le kill-feed/event_time_ms (d.events, qui pilote l'axe X des charts
// KD-cumul / frag-diff / scatter tug) + les badges d'impact.
//
// Montage : T0Ms = 30 000 ms (countdown 30 s), phase de cadence = 30 s. Deux
// frags du même joueur :
//   - gameplay à 35 000 ms (film) → corrigé 5 000 ms → phase_00, event_time_ms = 5 000.
//   - countdown à 10 000 ms (film) → corrigé -20 000 ms → EXCLU (skip < 0) du
//     kill-feed et des badges (sinon le front rendrait "-1m00s").
//
// La correction de d.events est CENTRALE et INCONDITIONNELLE : elle s'applique
// que l'events repo soit câblé (chemin canonical) ou non (fallback legacy).
func TestMatchViewService_T0_EventsHonorRealMatchStart(t *testing.T) {
	const (
		matchID     = "m-t0"
		killerXUID  = "xuid-killer"
		t0Ms        = int64(30_000)
		gameplayMS  = int64(35_000) // film → corrigé 5 000 (gameplay)
		countdownMS = int64(10_000) // film → corrigé -20 000 (pré-T0, à exclure)
	)
	now := time.Now()
	playable := int64(120) // 120 s → 5 buckets de 30 s (phase_00..phase_04)
	t0, kx := t0Ms, killerXUID
	gp, cd := gameplayMS, countdownMS

	newRepo := func() *mockMatchViewRepo {
		mapN, pairN := aquariusMap, slayerMode
		return &mockMatchViewRepo{
			meta: &domain.MatchMetaRaw{
				MatchID:                 matchID,
				StartTime:               &now,
				MapName:                 &mapN,
				PairName:                &pairN,
				PlayableDurationSeconds: &playable,
				T0Ms:                    &t0,
			},
			board: []domain.ScoreboardRaw{
				{XUID: killerXUID, Gamertag: "Killer", Kills: 2},
			},
			// evtList + badges lisent d.events (Q21) : 1 frag gameplay + 1 countdown.
			events: []domain.EventRaw{
				{EventType: "kill", XUID: &kx, TimeMS: &gp},
				{EventType: "kill", XUID: &kx, TimeMS: &cd},
			},
		}
	}
	canonicalEvents := []canonical.HighlightEvent{
		{MatchID: matchID, EventType: string(canonical.EventKill), TimeMS: gameplayMS, KillerXUID: &kx},
		{MatchID: matchID, EventType: string(canonical.EventKill), TimeMS: countdownMS, KillerXUID: &kx},
	}

	// --- Câblé (config prod) : events repo wiré ---
	wiredSvc := NewMatchViewService(newRepo(), killerXUID).
		WithTitleSlug("halo_infinite").
		WithHighlightEventsRepo(&mockHighlightEventsRepo{events: canonicalEvents})
	wiredResp, err := wiredSvc.GetMatchView(context.Background(), matchID)
	if err != nil {
		t.Fatalf("wired GetMatchView: %v", err)
	}
	// (a) kill-feed : event_time_ms recalé sur le gameplay (35 000 → 5 000) et
	// le frag de countdown (corrigé -20 000) exclu.
	if n := len(wiredResp.CombatTab.HighlightEvents); n != 1 {
		t.Fatalf("HighlightEvents : attendu 1 (frag countdown exclu), obtenu %d", n)
	}
	if got := wiredResp.CombatTab.HighlightEvents[0].EventTimeMS; got == nil || *got != 5_000 {
		t.Fatalf("event_time_ms : attendu 5000 (recalé T0), obtenu %v", got)
	}
	// (b) cadence : le frag gameplay tombe en phase_00 (vrai début de match).
	if b := cadenceBucketForXUID(wiredResp, killerXUID); b != 0 {
		t.Fatalf("cadence : frag gameplay attendu en phase_00, obtenu bucket %d", b)
	}

	// --- Legacy (events repo NON câblé) : la correction T0 de d.events est
	// centrale/inconditionnelle → le kill-feed reste recalé même sans canonicalEvents. ---
	legacyResp, err := NewMatchViewService(newRepo(), killerXUID).GetMatchView(context.Background(), matchID)
	if err != nil {
		t.Fatalf("legacy GetMatchView: %v", err)
	}
	if n := len(legacyResp.CombatTab.HighlightEvents); n != 1 {
		t.Fatalf("legacy HighlightEvents : attendu 1 (frag countdown exclu), obtenu %d", n)
	}
	if got := legacyResp.CombatTab.HighlightEvents[0].EventTimeMS; got == nil || *got != 5_000 {
		t.Fatalf("legacy event_time_ms : attendu 5000 (correction T0 centrale inconditionnelle), obtenu %v", got)
	}
}
