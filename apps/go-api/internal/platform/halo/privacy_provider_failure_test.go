package halo

// privacy_provider_failure_test.go — mémoire de l'échec de l'appel privacy (plan perf
// 2026-09-23, lot L5b, D5b.7) : échec Waypoint mémorisé 5 min puis réessayé, fin de
// contexte jamais mémorisée. Transport HTTP factice, horloge du cache pilotée.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/observability/timing"
)

// stubPrivacyTransport compte ses appels et répond `status` (0 = attendre la fin du
// contexte de la requête, comme un Waypoint qui ne répond pas).
type stubPrivacyTransport struct {
	calls  atomic.Int32
	status atomic.Int32
	body   string
}

func (s *stubPrivacyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls.Add(1)
	code := int(s.status.Load())
	if code == 0 {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newPrivacyTestProvider(tr http.RoundTripper, now *time.Time) *HaloProvider {
	p := NewHaloProvider()
	p.client = &http.Client{Transport: tr}
	p.maxRetries = 1
	p.privacyCache.now = func() time.Time { return *now }
	return p
}

// TestGetMatchPrivacy_FailureMemorizedThenRetried : un échec Waypoint (403) est resservi
// sans appel réseau pendant PrivacyFailureTTL, puis réessayé ; la réussite suivante
// est gardée PrivacyCacheTTL.
func TestGetMatchPrivacy_FailureMemorizedThenRetried(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	tr := &stubPrivacyTransport{body: `{"AllMatchesPrivacy":"Private"}`}
	tr.status.Store(http.StatusForbidden)
	p := newPrivacyTestProvider(tr, &now)
	ctx := privacyCtx("xuid-1")

	info, _ := p.GetMatchPrivacy(ctx, "xuid-1")
	if info.Hint != errHintFetchError || !info.IsPartial {
		t.Fatalf("échec : %+v, want repli fetch_error", info)
	}
	tr.status.Store(http.StatusOK) // Waypoint rétabli : la mémoire de l'échec doit primer
	now = now.Add(PrivacyFailureTTL - time.Minute)
	if info, _ = p.GetMatchPrivacy(ctx, "xuid-1"); info.Hint != errHintFetchError {
		t.Fatalf("pendant la mémoire de l'échec : %+v, want le repli mémorisé", info)
	}
	if got := tr.calls.Load(); got != 1 {
		t.Fatalf("appels Waypoint pendant la mémoire de l'échec = %d, want 1", got)
	}

	now = now.Add(2 * time.Minute) // PrivacyFailureTTL écoulé
	if info, _ = p.GetMatchPrivacy(ctx, "xuid-1"); !info.IsPrivate {
		t.Fatalf("après l'échéance : %+v, want la réponse Waypoint (privé)", info)
	}
	now = now.Add(10 * time.Minute)
	_, _ = p.GetMatchPrivacy(ctx, "xuid-1")
	if got := tr.calls.Load(); got != 2 {
		t.Errorf("appels Waypoint = %d, want 2 (réussite gardée %v)", got, PrivacyCacheTTL)
	}
}

// TestGetMatchPrivacy_ContextEndNotMemorized : un appel interrompu par l'échéance ou
// l'annulation de l'appelant n'est pas un verdict de Waypoint — l'appel suivant réessaie.
func TestGetMatchPrivacy_ContextEndNotMemorized(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	tr := &stubPrivacyTransport{body: `{"AllMatchesPrivacy":"Open"}`}
	p := newPrivacyTestProvider(tr, &now) // status 0 : Waypoint ne répond pas

	deadlineCtx, cancel := context.WithTimeout(privacyCtx("xuid-2"), 30*time.Millisecond)
	defer cancel()
	if info, _ := p.GetMatchPrivacy(deadlineCtx, "xuid-2"); info.Hint != errHintFetchError {
		t.Fatalf("échéance : %+v, want repli fetch_error", info)
	}
	canceledCtx, cancelNow := context.WithCancel(privacyCtx("xuid-2"))
	cancelNow()
	_, _ = p.GetMatchPrivacy(canceledCtx, "xuid-2")

	tr.status.Store(http.StatusOK)
	info, _ := p.GetMatchPrivacy(privacyCtx("xuid-2"), "xuid-2")
	if info.Hint != "" || info.IsPartial {
		t.Errorf("après des fins de contexte : %+v, want la réponse Waypoint (non mémorisées)", info)
	}
}

func TestGetMatchPrivacy_TimingMarkers(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	tr := &stubPrivacyTransport{body: `{"AllMatchesPrivacy":"Open"}`}
	tr.status.Store(http.StatusOK)
	p := newPrivacyTestProvider(tr, &now)
	ctx, tm := timing.WithTimings(privacyCtx("xuid-3"))
	_, _ = p.GetMatchPrivacy(ctx, "xuid-3")
	_, _ = p.GetMatchPrivacy(ctx, "xuid-3")
	calls := map[string]int{}
	for _, s := range tm.Snapshot() {
		calls[s.Name] = s.Calls
	}
	if calls["privacy_cache_miss"] != 1 || calls["privacy_cache_hit"] != 1 {
		t.Errorf("marqueurs = %v, want miss=1 hit=1", calls)
	}
}
