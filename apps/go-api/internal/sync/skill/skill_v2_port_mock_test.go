package skill

// skill_v2_port_mock_test.go — démontre le gain du découplage (Axe 2) : la
// logique LUSR v2 dépend désormais de port.SkillV2Repository, donc testable
// avec un mock en mémoire, SANS DuckDB ni *sql.DB.

import (
	"context"
	"testing"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
)

// fakeSkillV2Repo implémente port.SkillV2Repository en mémoire (pas de DB).
type fakeSkillV2Repo struct {
	states map[string]*domain.SkillV2State // clé = xuid
}

func (f *fakeSkillV2Repo) LoadState(_ context.Context, xuid, _ string) (*domain.SkillV2State, error) {
	return f.states[xuid], nil
}
func (f *fakeSkillV2Repo) LoadAllStates(_ context.Context, _ string) ([]domain.SkillV2State, error) {
	return nil, nil
}
func (f *fakeSkillV2Repo) UpsertState(_ context.Context, _ domain.SkillV2State) error { return nil }
func (f *fakeSkillV2Repo) LoadHyperparams(_ context.Context, _ string) (map[string]float64, error) {
	return nil, nil
}

// TestLoadStatesOrSeed_MockRepo : xuid connu → état renvoyé verbatim ; xuid
// inconnu → seed depuis les priors. Tourne sans DuckDB grâce à l'interface.
func TestLoadStatesOrSeed_MockRepo(t *testing.T) {
	priors := skillv2.DefaultPriors()
	known := &domain.SkillV2State{XUID: "known", PlaylistGroup: "g", Mu: 30, Sigma: 5}
	repo := &fakeSkillV2Repo{states: map[string]*domain.SkillV2State{"known": known}}

	out, err := loadStatesOrSeed(context.Background(), repo, []string{"known", "unknown"}, "g", priors)
	if err != nil {
		t.Fatalf("loadStatesOrSeed: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	// xuid connu : renvoyé tel quel.
	if out[0].XUID != "known" || out[0].Mu != 30 || out[0].Sigma != 5 {
		t.Errorf("known state non renvoyé verbatim: %+v", out[0])
	}
	// xuid inconnu : seedé depuis les priors.
	seed := priors.NewPlayerState()
	if out[1].XUID != "unknown" || out[1].PlaylistGroup != "g" ||
		out[1].Mu != seed.Mu || out[1].Sigma != seed.Sigma {
		t.Errorf("unknown devrait être seedé depuis priors (Mu=%v Sigma=%v): %+v",
			seed.Mu, seed.Sigma, out[1])
	}
}

// TestComputeTeamSquadOffsets_NilRepo : sans repo (flag squad off), aucun
// offset — la fonction est appelable avec une interface nil sans paniquer.
func TestComputeTeamSquadOffsets_NilRepo(t *testing.T) {
	states := []domain.SkillV2State{
		{XUID: "a"}, {XUID: "b"},
	}
	offsets := computeTeamSquadOffsets(context.Background(), nil, states, "g")
	if len(offsets) != 2 || offsets[0] != 0 || offsets[1] != 0 {
		t.Errorf("repo nil devrait donner des offsets nuls, got %v", offsets)
	}
}
