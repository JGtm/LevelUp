package timeline

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ev(matchID string, timeMS int64) canonical.HighlightEvent {
	return canonical.HighlightEvent{MatchID: matchID, TimeMS: timeMS, EventType: "kill"}
}

func TestCorrectEvents_NilInput(t *testing.T) {
	if got := CorrectEvents(nil, nil); got != nil {
		t.Errorf("nil events should return nil, got %v", got)
	}
}

func TestCorrectEvents_EmptyTimelinesIsIdentity(t *testing.T) {
	in := []canonical.HighlightEvent{ev("m1", 1000), ev("m2", 5000)}
	out := CorrectEvents(in, nil)
	if len(out) != 2 || out[0].TimeMS != 1000 || out[1].TimeMS != 5000 {
		t.Errorf("nil timelines should be identity, got %+v", out)
	}
}

func TestCorrectEvents_MissingMatchIsIdentity(t *testing.T) {
	in := []canonical.HighlightEvent{ev("unknown", 1234)}
	timelines := map[string]domain.MatchTimeline{
		"other": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].TimeMS != 1234 {
		t.Errorf("match absent from timelines must stay unchanged, got %d", out[0].TimeMS)
	}
}

func TestCorrectEvents_AppliesT0PerMatch(t *testing.T) {
	in := []canonical.HighlightEvent{
		ev("fortress", 30000), // T0=28000 → 2000
		ev("btb", 50000),      // T0=49000 → 1000
		ev("notrack", 7777),   // absent → unchanged
	}
	timelines := map[string]domain.MatchTimeline{
		"fortress": domain.NewMatchTimeline(447000, 28000),
		"btb":      domain.NewMatchTimeline(576000, 49000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].TimeMS != 2000 {
		t.Errorf("fortress: got %d, want 2000", out[0].TimeMS)
	}
	if out[1].TimeMS != 1000 {
		t.Errorf("btb: got %d, want 1000", out[1].TimeMS)
	}
	if out[2].TimeMS != 7777 {
		t.Errorf("notrack: got %d, want 7777 (unchanged)", out[2].TimeMS)
	}
}

func TestCorrectEvents_DoesNotMutateInput(t *testing.T) {
	in := []canonical.HighlightEvent{ev("m1", 30000)}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	_ = CorrectEvents(in, timelines)
	if in[0].TimeMS != 30000 {
		t.Errorf("input must not be mutated, got %d", in[0].TimeMS)
	}
}

func TestCorrectEvents_PreservesOtherFields(t *testing.T) {
	killer := "killer-xuid"
	in := []canonical.HighlightEvent{{
		MatchID:    "m1",
		TimeMS:     30000,
		EventType:  "kill",
		KillerXUID: &killer,
	}}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	if out[0].EventType != "kill" || out[0].KillerXUID == nil || *out[0].KillerXUID != "killer-xuid" {
		t.Errorf("non-time fields must be preserved, got %+v", out[0])
	}
	if out[0].TimeMS != 2000 {
		t.Errorf("TimeMS: got %d, want 2000", out[0].TimeMS)
	}
}

// Property: badge winner invariance — correcting by a constant per match
// preserves the relative order of events, so MIN/MAX-based selections are stable.
func TestCorrectImpactEvents_AppliesT0AndPreservesFields(t *testing.T) {
	in := []domain.ImpactEventRow{
		{MatchID: "fortress", XUID: "x1", Gamertag: "GT1", EventType: "kill", TimeMS: 30000}, // T0=28000 → 2000
		{MatchID: "btb", XUID: "x2", EventType: "death", TimeMS: 50000},                      // T0=49000 → 1000
		{MatchID: "notrack", XUID: "x3", EventType: "kill", TimeMS: 7777},                    // absent → inchangé
	}
	timelines := map[string]domain.MatchTimeline{
		"fortress": domain.NewMatchTimeline(447000, 28000),
		"btb":      domain.NewMatchTimeline(576000, 49000),
	}
	out := CorrectImpactEvents(in, timelines)
	if out[0].TimeMS != 2000 {
		t.Errorf("fortress: got %d, want 2000", out[0].TimeMS)
	}
	if out[1].TimeMS != 1000 {
		t.Errorf("btb: got %d, want 1000", out[1].TimeMS)
	}
	if out[2].TimeMS != 7777 {
		t.Errorf("notrack (absent): got %d, want 7777 (inchangé)", out[2].TimeMS)
	}
	if out[0].XUID != "x1" || out[0].Gamertag != "GT1" || out[0].EventType != "kill" {
		t.Errorf("champs non-temporels doivent être préservés, got %+v", out[0])
	}
	if in[0].TimeMS != 30000 {
		t.Errorf("l'input ne doit pas être muté, got %d", in[0].TimeMS)
	}
}

func TestCorrectImpactEvents_NilAndPreGameplayNegative(t *testing.T) {
	if got := CorrectImpactEvents(nil, nil); got != nil {
		t.Errorf("nil events should return nil, got %v", got)
	}
	// Event pendant le countdown (TimeMS < T0) → corrigé négatif (au caller de filtrer).
	in := []domain.ImpactEventRow{{MatchID: "m1", EventType: "kill", TimeMS: 10000}}
	timelines := map[string]domain.MatchTimeline{"m1": domain.NewMatchTimeline(447000, 28000)}
	out := CorrectImpactEvents(in, timelines)
	if out[0].TimeMS != -18000 {
		t.Errorf("pre-T0 event: got %d, want -18000", out[0].TimeMS)
	}
}

// ptrI64 alloue un *int64 (helper de test pour EventRaw.TimeMS).
func ptrI64(v int64) *int64 { return &v }

func TestCorrectEventRaws_NilInput(t *testing.T) {
	if got := CorrectEventRaws(nil, domain.NewMatchTimeline(0, 0)); got != nil {
		t.Errorf("nil events should return nil, got %v", got)
	}
}

func TestCorrectEventRaws_AppliesT0_NilPreserved(t *testing.T) {
	in := []domain.EventRaw{
		{EventType: "kill", TimeMS: ptrI64(35000)}, // T0=30000 → 5000
		{EventType: "medal", TimeMS: nil},          // nil → préservé
	}
	tl := domain.NewMatchTimeline(600000, 30000)
	out := CorrectEventRaws(in, tl)
	if out[0].TimeMS == nil || *out[0].TimeMS != 5000 {
		t.Errorf("event 0: got %v, want 5000", out[0].TimeMS)
	}
	if out[1].TimeMS != nil {
		t.Errorf("event 1 nil TimeMS doit rester nil, got %v", out[1].TimeMS)
	}
}

// Piège load-bearing : EventRaw.TimeMS est *int64 ; copy() partage le pointeur.
// CorrectEventRaws DOIT réallouer pour ne pas corrompre l'input (sinon un 2e
// appel re-soustrairait T0).
func TestCorrectEventRaws_DoesNotMutateInput(t *testing.T) {
	in := []domain.EventRaw{{EventType: "kill", TimeMS: ptrI64(35000)}}
	tl := domain.NewMatchTimeline(600000, 30000)
	_ = CorrectEventRaws(in, tl)
	if in[0].TimeMS == nil || *in[0].TimeMS != 35000 {
		t.Fatalf("input muté : TimeMS source = %v, want 35000 (pointeur partagé non réalloué)", in[0].TimeMS)
	}
}

func TestCorrectEventRaws_ZeroTimelineIsIdentity(t *testing.T) {
	in := []domain.EventRaw{{EventType: "kill", TimeMS: ptrI64(12345)}}
	out := CorrectEventRaws(in, domain.MatchTimeline{}) // T0=0 → identité
	if out[0].TimeMS == nil || *out[0].TimeMS != 12345 {
		t.Errorf("tl zéro-value doit être identité, got %v", out[0].TimeMS)
	}
}

func TestCorrectEventRaws_PreGameplayNegative(t *testing.T) {
	// Frag de countdown (TimeMS < T0) → corrigé négatif (au caller de filtrer).
	in := []domain.EventRaw{{EventType: "kill", TimeMS: ptrI64(10000)}}
	tl := domain.NewMatchTimeline(600000, 30000)
	out := CorrectEventRaws(in, tl)
	if out[0].TimeMS == nil || *out[0].TimeMS != -20000 {
		t.Errorf("pre-T0 event: got %v, want -20000", out[0].TimeMS)
	}
}

// TestCorrectKillSourceRaws_AppliqueT0SansMuterLInput : les sources Q21b (arme du
// kill feed) doivent vivre dans le même référentiel que les events corrigés —
// c'est la clé d'appariement (xuid, time_ms) de decorateKillFeed. Valeurs du bug
// réel (000d5950 : T0 = 18465 ms, premier kill film 35306 → gameplay 16841).
func TestCorrectKillSourceRaws_AppliqueT0SansMuterLInput(t *testing.T) {
	if got := CorrectKillSourceRaws(nil, domain.NewMatchTimeline(0, 0)); got != nil {
		t.Errorf("nil rows doit rendre nil, got %v", got)
	}
	in := []domain.KillSourceRaw{{XUID: "A", TimeMS: 35306, SourceTag: 0x11}}
	tl := domain.NewMatchTimeline(600000, 18465)
	out := CorrectKillSourceRaws(in, tl)
	if out[0].TimeMS != 16841 {
		t.Errorf("TimeMS corrigé = %d, attendu 16841", out[0].TimeMS)
	}
	if out[0].XUID != "A" || out[0].SourceTag != 0x11 {
		t.Errorf("champs non-temporels doivent être préservés, got %+v", out[0])
	}
	if in[0].TimeMS != 35306 {
		t.Errorf("l'input ne doit pas être muté, got %d", in[0].TimeMS)
	}
	// tl zéro-value → identité (match sans T0 connu).
	id := CorrectKillSourceRaws(in, domain.MatchTimeline{})
	if id[0].TimeMS != 35306 {
		t.Errorf("tl zéro-value doit être identité, got %d", id[0].TimeMS)
	}
}

// TestCorrectKillAssistRaws_AppliqueT0SansMuterLInput : même contrat pour Q21c
// (assistant + parts de dégâts) — les champs pointeurs sont partagés, jamais
// modifiés.
func TestCorrectKillAssistRaws_AppliqueT0SansMuterLInput(t *testing.T) {
	if got := CorrectKillAssistRaws(nil, domain.NewMatchTimeline(0, 0)); got != nil {
		t.Errorf("nil rows doit rendre nil, got %v", got)
	}
	gt := "Bob"
	pct := 63
	in := []domain.KillAssistRaw{{XUID: "A", TimeMS: 35306, AssistGamertag: &gt, KillerDamagePct: &pct}}
	tl := domain.NewMatchTimeline(600000, 18465)
	out := CorrectKillAssistRaws(in, tl)
	if out[0].TimeMS != 16841 {
		t.Errorf("TimeMS corrigé = %d, attendu 16841", out[0].TimeMS)
	}
	if out[0].AssistGamertag == nil || *out[0].AssistGamertag != "Bob" ||
		out[0].KillerDamagePct == nil || *out[0].KillerDamagePct != 63 {
		t.Errorf("champs d'assistance doivent être préservés, got %+v", out[0])
	}
	if in[0].TimeMS != 35306 {
		t.Errorf("l'input ne doit pas être muté, got %d", in[0].TimeMS)
	}
}

func TestCorrectEvents_PreservesRelativeOrder(t *testing.T) {
	in := []canonical.HighlightEvent{
		ev("m1", 50000),
		ev("m1", 10000),
		ev("m1", 30000),
	}
	timelines := map[string]domain.MatchTimeline{
		"m1": domain.NewMatchTimeline(447000, 28000),
	}
	out := CorrectEvents(in, timelines)
	// Order of slice is preserved; values shifted by -28000.
	if out[0].TimeMS != 22000 || out[1].TimeMS != -18000 || out[2].TimeMS != 2000 {
		t.Errorf("unexpected shifted values: %d %d %d", out[0].TimeMS, out[1].TimeMS, out[2].TimeMS)
	}
	// MIN before correction is m1@10000; after, it is still the same event (-18000).
	minIdx := 0
	for i := range out {
		if out[i].TimeMS < out[minIdx].TimeMS {
			minIdx = i
		}
	}
	if minIdx != 1 {
		t.Errorf("MIN event index should remain 1, got %d", minIdx)
	}
}
