package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/progression/coach_advisor"
)

// fakeAdvisorService implémente coach_advisor.Service avec un store en mémoire.
type fakeAdvisorService struct {
	proposals map[string]coach_advisor.Proposal
	// Comportements injectables
	acceptErr  error
	dismissErr error
	listErr    error
}

func newFakeAdvisor() *fakeAdvisorService {
	return &fakeAdvisorService{proposals: map[string]coach_advisor.Proposal{}}
}

func (f *fakeAdvisorService) GenerateProposals(_ context.Context, _ coach_advisor.GenerateInput) ([]coach_advisor.Proposal, error) {
	return nil, nil
}

func (f *fakeAdvisorService) ListProposals(_ context.Context, _ string, _ string, status coach_advisor.ProposalStatus) ([]coach_advisor.Proposal, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []coach_advisor.Proposal
	for _, p := range f.proposals {
		if status != "" && p.Status != status {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeAdvisorService) AcceptProposal(_ context.Context, id string) (coach_advisor.AcceptResult, error) {
	if f.acceptErr != nil {
		return coach_advisor.AcceptResult{}, f.acceptErr
	}
	p, ok := f.proposals[id]
	if !ok {
		return coach_advisor.AcceptResult{}, coach_advisor.ErrProposalNotFound
	}
	if p.Status != coach_advisor.ProposalPending {
		return coach_advisor.AcceptResult{}, coach_advisor.ErrProposalNotAcceptable
	}
	p.Status = coach_advisor.ProposalAccepted
	p.ResolvedRef = "ch_fake_" + id
	f.proposals[id] = p
	return coach_advisor.AcceptResult{ChallengeID: p.ResolvedRef}, nil
}

func (f *fakeAdvisorService) DismissProposal(_ context.Context, id string) error {
	if f.dismissErr != nil {
		return f.dismissErr
	}
	p, ok := f.proposals[id]
	if !ok {
		return coach_advisor.ErrProposalNotFound
	}
	p.Status = coach_advisor.ProposalDismissed
	f.proposals[id] = p
	return nil
}

func (f *fakeAdvisorService) ObsoletePendingForAxis(_ context.Context, _, _, _ string) (int, error) {
	return 0, nil
}

// helper : monte le handler sur un sous-routeur chi avec /players/{player_slug} préfixé.
func mountHandler(svc coach_advisor.Service, userID string, resolveErr error) http.Handler {
	resolver := func(_ context.Context, _ string) (coach_advisor.Service, string, error) {
		return svc, userID, resolveErr
	}
	h := handlers.NewCoachProposalsHandler(resolver, "halo_infinite")
	r := chi.NewRouter()
	r.Route("/api/v1/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

func mkPendingProp(id string) coach_advisor.Proposal {
	return coach_advisor.Proposal{
		ID:           id,
		UserID:       "u1",
		TitleSlug:    "halo_infinite",
		Kind:         coach_advisor.ProposalKindChallenge,
		TemplateID:   "tpl_" + id,
		SourceSignal: coach_advisor.SignalLOWESSPositive,
		SourceMetric: "accuracy",
		RadarAxis:    "combat",
		Strength:     0.7,
		Origin:       coach_advisor.OriginCatalog,
		Status:       coach_advisor.ProposalPending,
		CreatedAt:    time.Now().UTC(),
	}
}

func TestListProposals_HappyPath(t *testing.T) {
	svc := newFakeAdvisor()
	svc.proposals["p1"] = mkPendingProp("p1")
	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/players/madina97294/coach/proposals")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}

	var body struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Kind   string `json:"kind"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "p1" {
		t.Errorf("expected 1 item p1, got %+v", body.Items)
	}
}

func TestListProposals_FiltersByStatus(t *testing.T) {
	svc := newFakeAdvisor()
	p1 := mkPendingProp("p1")
	p2 := mkPendingProp("p2")
	p2.Status = coach_advisor.ProposalAccepted
	svc.proposals["p1"] = p1
	svc.proposals["p2"] = p2

	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/players/madina97294/coach/proposals?status=pending")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var body struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Items) != 1 || body.Items[0].ID != "p1" {
		t.Errorf("expected only p1 (pending), got %+v", body.Items)
	}
}

func TestAcceptProposal_HappyPath(t *testing.T) {
	svc := newFakeAdvisor()
	svc.proposals["p1"] = mkPendingProp("p1")
	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/players/madina97294/coach/proposals/p1/accept", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	var body struct {
		Status      string `json:"status"`
		ChallengeID string `json:"challenge_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Status != "accepted" {
		t.Errorf("status field: got %q, want accepted", body.Status)
	}
	if body.ChallengeID == "" {
		t.Error("expected ChallengeID in response")
	}
}

func TestAcceptProposal_NotFound_Returns404(t *testing.T) {
	svc := newFakeAdvisor()
	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/players/madina97294/coach/proposals/missing/accept", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestAcceptProposal_AlreadyAccepted_Returns409(t *testing.T) {
	svc := newFakeAdvisor()
	p := mkPendingProp("p1")
	p.Status = coach_advisor.ProposalAccepted
	svc.proposals["p1"] = p

	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/players/madina97294/coach/proposals/p1/accept", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status: got %d, want 409", resp.StatusCode)
	}
}

func TestDismissProposal_HappyPath(t *testing.T) {
	svc := newFakeAdvisor()
	svc.proposals["p1"] = mkPendingProp("p1")
	srv := httptest.NewServer(mountHandler(svc, "u1", nil))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/players/madina97294/coach/proposals/p1/dismiss", "", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	if svc.proposals["p1"].Status != coach_advisor.ProposalDismissed {
		t.Errorf("proposal should be dismissed, status=%q", svc.proposals["p1"].Status)
	}
}

func TestHandler_NilService_Returns503(t *testing.T) {
	srv := httptest.NewServer(mountHandler(nil, "", nil))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/players/madina97294/coach/proposals")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", resp.StatusCode)
	}
}

func TestHandler_PlayerResolveError_Returns404(t *testing.T) {
	srv := httptest.NewServer(mountHandler(newFakeAdvisor(), "", errors.New("not found")))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/players/missing/coach/proposals")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}
