package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockCitationsRepo struct {
	totals      []domain.CitationTotalRow
	totalsErr   error
	mappings    []domain.CitationMappingRow
	mappingsErr error
	medals      []domain.MedalEarnedRow
	medalsErr   error
	medalMaps   []domain.MedalCitationRow
	medalMapErr error
}

func (m *mockCitationsRepo) LoadCitationTotals(_ context.Context) ([]domain.CitationTotalRow, error) {
	return m.totals, m.totalsErr
}
func (m *mockCitationsRepo) LoadCitationMappings(_ context.Context) ([]domain.CitationMappingRow, error) {
	return m.mappings, m.mappingsErr
}
func (m *mockCitationsRepo) LoadMedalTotals(_ context.Context, _ string) ([]domain.MedalEarnedRow, error) {
	return m.medals, m.medalsErr
}
func (m *mockCitationsRepo) LoadMedalCitationMappings(_ context.Context) ([]domain.MedalCitationRow, error) {
	return m.medalMaps, m.medalMapErr
}
func (m *mockCitationsRepo) LoadCitationMedalMappings(_ context.Context) ([]domain.CitationMedalMapping, error) {
	return nil, nil
}
func (m *mockCitationsRepo) LoadMatchCitationsForView(_ context.Context, _ string) ([]domain.CitationMatchViewRow, error) {
	return nil, nil
}
func (m *mockCitationsRepo) LoadMatchCitationsRich(_ context.Context, _ string) ([]domain.HomeMatchCitationRaw, error) {
	return nil, nil
}
func (m *mockCitationsRepo) LoadMatchCommendationsRich(_ context.Context, _, _ string) ([]domain.HomeMatchCommendationRaw, error) {
	return nil, nil
}

// --- tests ---

func TestCitationsService_GetCitationsPage_OK(t *testing.T) {
	repo := &mockCitationsRepo{
		totals: []domain.CitationTotalRow{
			{NameNorm: "kills", Total: 42},
		},
		mappings: []domain.CitationMappingRow{
			{NameNorm: "kills", NameDisplay: "Kills", Category: "Combat"},
		},
	}
	svc := NewCitationsService(repo)

	resp, err := svc.GetCitationsPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCitationsService_GetCitationsPage_Empty(t *testing.T) {
	repo := &mockCitationsRepo{
		totals:   []domain.CitationTotalRow{},
		mappings: []domain.CitationMappingRow{},
	}
	svc := NewCitationsService(repo)

	resp, err := svc.GetCitationsPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCitationsService_GetCitationsPage_TotalsError(t *testing.T) {
	repo := &mockCitationsRepo{totalsErr: errors.New("db fail")}
	svc := NewCitationsService(repo)

	_, err := svc.GetCitationsPage(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

func TestCitationsService_GetCitationsPage_MappingsError_Graceful(t *testing.T) {
	repo := &mockCitationsRepo{
		totals:      []domain.CitationTotalRow{{NameNorm: "kills", Total: 1}},
		mappingsErr: errors.New("mappings fail"),
	}
	svc := NewCitationsService(repo)

	resp, err := svc.GetCitationsPage(context.Background())
	if err != nil {
		t.Fatalf("expected graceful handling, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response despite mappings error")
	}
}

func TestCitationsService_GetCommendationsPage_OK(t *testing.T) {
	repo := &mockCitationsRepo{
		medals:    []domain.MedalEarnedRow{{MedalID: 1, TotalCount: 5}},
		medalMaps: []domain.MedalCitationRow{{MedalID: 1, NameDisplay: "Double Kill", Category: "Multi"}},
	}
	svc := NewCitationsService(repo)

	resp, err := svc.GetCommendationsPage(context.Background(), "xuid123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCitationsService_GetCommendationsPage_MedalsError(t *testing.T) {
	repo := &mockCitationsRepo{medalsErr: errors.New("fail")}
	svc := NewCitationsService(repo)

	_, err := svc.GetCommendationsPage(context.Background(), "xuid123")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCitationsService_GetCommendationsPage_MappingsError_Graceful(t *testing.T) {
	repo := &mockCitationsRepo{
		medals:      []domain.MedalEarnedRow{{MedalID: 1, TotalCount: 3}},
		medalMapErr: errors.New("fail"),
	}
	svc := NewCitationsService(repo)

	resp, err := svc.GetCommendationsPage(context.Background(), "xuid123")
	if err != nil {
		t.Fatalf("expected graceful, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
