package teammates

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// from_squad_service_test.go — tests TeammatesService + helpers purs (safeDiv)
// extraits de service/squad_service_test.go lors de l'extraction du sous-package
// teammates (K3b).

func TestTeammatesService_GetPage_OK(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "x1", Gamertag: "Ally1", GamesTogether: 30, WinsTogether: 20, WinRate: 0.67, AvgKDA: 1.5},
			{XUID: "x2", Gamertag: "Ally2", GamesTogether: 10, WinsTogether: 4, WinRate: 0.4, AvgKDA: 0.8},
		},
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp
}

func TestTeammatesService_GetPage_Error(t *testing.T) {
	repo := &mockSquadRepo{topErr: errors.New("fail")}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test")

	_, err := svc.GetPage(context.Background(), "xuid", domain.TeammatesQueryRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestSafeDiv_ZeroDivisor(t *testing.T) {
	if safeDiv(10, 0) != 10 {
		t.Error("expected a when b=0")
	}
}

func TestSafeDiv_Normal(t *testing.T) {
	got := safeDiv(10, 3)
	if got < 3.33 || got > 3.34 {
		t.Errorf("safeDiv(10,3) = %f, want ~3.33", got)
	}
}
