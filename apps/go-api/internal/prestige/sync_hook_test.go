package prestige

import (
	"context"
	"errors"
	"testing"
)

// Mock minimal de Service pour valider que le hook respecte le feature flag.
type mockService struct {
	called bool
}

func (m *mockService) CreateChallenge(ctx context.Context, _ CreateChallengeRequest) (Challenge, error) {
	return Challenge{}, nil
}

func (m *mockService) UpdateChallenge(ctx context.Context, _ string, _ UpdateChallengePatch) (Challenge, error) {
	return Challenge{}, nil
}

func (m *mockService) AbandonChallenge(ctx context.Context, _ string) error { return nil }

func (m *mockService) GetChallenge(ctx context.Context, _ string) (Challenge, error) {
	return Challenge{}, nil
}

func (m *mockService) ListActiveChallenges(ctx context.Context, _, _ string) ([]Challenge, error) {
	return nil, nil
}

func (m *mockService) ListChallenges(ctx context.Context, _, _ string, _ []ChallengeStatus) ([]Challenge, error) {
	return nil, nil
}

func (m *mockService) EvaluateForUser(ctx context.Context, _, _ string) ([]EvaluationOutcome, error) {
	m.called = true
	return nil, nil
}

func (m *mockService) GetUserPrestige(ctx context.Context, _, _ string) (UserPrestige, error) {
	return UserPrestige{}, nil
}

func (m *mockService) SuggestTemplates(ctx context.Context, _, _ string, _ int) ([]Template, error) {
	return nil, nil
}

func (m *mockService) SuggestNext(ctx context.Context, _ string) ([]Template, error) {
	return nil, nil
}

func (m *mockService) CreateArc(ctx context.Context, _ CreateArcRequest) (Arc, error) {
	return Arc{}, nil
}
func (m *mockService) ListArcs(ctx context.Context, _, _ string) ([]Arc, error) { return nil, nil }
func (m *mockService) GetArc(ctx context.Context, _ string) (Arc, error)        { return Arc{}, nil }
func (m *mockService) DeleteArc(ctx context.Context, _, _ string, _ DeleteArcOptions) error {
	return nil
}
func (m *mockService) ListArcPresets(ctx context.Context, _, _ string) ([]PresetArc, error) {
	return nil, nil
}
func (m *mockService) AdoptPresetArc(ctx context.Context, _, _, _ string) (Arc, error) {
	return Arc{}, nil
}
func (m *mockService) CreateSquadChallenge(ctx context.Context, _ CreateSquadChallengeRequest) (SquadChallenge, error) {
	return SquadChallenge{}, nil
}
func (m *mockService) JoinSquadChallenge(ctx context.Context, _, _ string, _ Tier, _ bool) error {
	return nil
}
func (m *mockService) GetSquadChallenge(ctx context.Context, _ string) (SquadChallenge, error) {
	return SquadChallenge{}, nil
}
func (m *mockService) ListSquadChallenges(ctx context.Context, _, _ string) ([]SquadChallengeView, error) {
	return nil, nil
}
func (m *mockService) AbandonSquadChallenge(ctx context.Context, _, _ string) error {
	return nil
}
func (m *mockService) RefreshSquadPool(ctx context.Context, _, _, _ string) ([]Template, error) {
	return nil, nil
}
func (m *mockService) CreateSquad(ctx context.Context, _ CreateSquadRequest) (Squad, error) {
	return Squad{}, nil
}
func (m *mockService) ListSquadsForUser(ctx context.Context, _ string) ([]Squad, error) {
	return nil, nil
}
func (m *mockService) GetSquad(ctx context.Context, _ string) (Squad, error) {
	return Squad{}, nil
}
func (m *mockService) ListSquadMembers(ctx context.Context, _ string) ([]SquadMember, error) {
	return nil, nil
}
func (m *mockService) AddSquadMember(ctx context.Context, _ string, _ SquadMember, _ string) error {
	return nil
}
func (m *mockService) RemoveSquadMember(ctx context.Context, _, _, _ string) error {
	return nil
}
func (m *mockService) RenameSquad(ctx context.Context, _, _, _ string) error {
	return nil
}
func (m *mockService) DeleteSquad(ctx context.Context, _, _ string) error {
	return nil
}
func (m *mockService) SquadUsualContexts(ctx context.Context, _ []string, _ string) ([]string, []string, error) {
	return nil, nil, nil
}
func (m *mockService) EvaluateSquadChallenge(ctx context.Context, _, _ string) ([]SquadParticipantProgress, error) {
	return nil, nil
}
func (m *mockService) SquadOrientation(ctx context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockService) EnablePilotMode(ctx context.Context, _, _ string) (PilotModeAttribution, error) {
	return PilotModeAttribution{}, nil
}
func (m *mockService) DisablePilotMode(ctx context.Context, _, _ string) error {
	return nil
}

func TestIsEnabled_Defaults(t *testing.T) {
	t.Setenv(FeatureFlagEnv, "")
	if !IsEnabled("") {
		t.Error("expected enabled when env var empty (default-on)")
	}
	t.Setenv(FeatureFlagEnv, "false")
	if IsEnabled("") {
		t.Error("expected disabled when explicitly false")
	}
}

func TestIsEnabled_FalsyValues(t *testing.T) {
	for _, v := range []string{"0", "false", "FALSE", "no", "off"} {
		t.Setenv(FeatureFlagEnv, v)
		if IsEnabled("") {
			t.Errorf("expected disabled for %q", v)
		}
	}
}

func TestIsEnabled_TruthyValues(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv(FeatureFlagEnv, v)
		if !IsEnabled("") {
			t.Errorf("expected enabled for %q", v)
		}
	}
}

func TestRunPostSyncHook_EnabledCallsService(t *testing.T) {
	// C7 : le hook ne re-lit plus le flag (gate unique chez l'appelant) ;
	// il appelle toujours le service non-nil.
	mock := &mockService{}
	RunPostSyncHook(context.Background(), mock, "u1", "halo_infinite")
	if !mock.called {
		t.Error("service should be called when flag is on")
	}
}

func TestRunPostSyncHook_NilServiceSurvives(t *testing.T) {
	t.Setenv(FeatureFlagEnv, "true")
	// Doit pas paniquer ni planter
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("hook panicked with nil service: %v", r)
		}
	}()
	RunPostSyncHook(context.Background(), nil, "u1", "halo_infinite")
}

// Mock qui retourne une erreur pour vérifier que le hook log mais ne propage pas.
type failingService struct {
	mockService
}

func (f *failingService) EvaluateForUser(ctx context.Context, _, _ string) ([]EvaluationOutcome, error) {
	return nil, errors.New("simulated failure")
}

func TestRunPostSyncHook_ServiceErrorIsLoggedNotPropagated(t *testing.T) {
	t.Setenv(FeatureFlagEnv, "true")
	// Pas de retour d'erreur — RunPostSyncHook est void par design (best-effort).
	RunPostSyncHook(context.Background(), &failingService{}, "u1", "halo_infinite")
}
