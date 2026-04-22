// Package halo — battlepass_details_test.go : tests unitaires pour fetchRewardTrackDefinition.
//
// Seul le chemin P4/P5 (via assets.Resolver) est testé ici.
// Les tests DuckDB legacy ont été supprimés lors du nettoyage P6.
package halo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
)

// ---------------------------------------------------------------------------
// Mock assets.Resolver
// ---------------------------------------------------------------------------

type mockResolver struct {
	getFunc func(ctx context.Context, ref assets.Ref) (assets.Resolved, error)
}

func (m *mockResolver) Get(ctx context.Context, ref assets.Ref) (assets.Resolved, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, ref)
	}
	return assets.Resolved{}, assets.ErrNotFound
}

func (m *mockResolver) Refresh(ctx context.Context, ref assets.Ref) (assets.Resolved, error) {
	return m.Get(ctx, ref)
}

func (m *mockResolver) Warm(_ context.Context, _ ...assets.Ref) {}

func (m *mockResolver) RegisterLocalFile(_ context.Context, _ assets.Ref, _ string) error {
	return nil
}

func (m *mockResolver) Close(_ context.Context) error { return nil }

// ---------------------------------------------------------------------------
// Tests fetchRewardTrackDefinition (P4/P5 only)
// ---------------------------------------------------------------------------

func TestFetchRewardTrackDefinition_EmptyPath_ReturnsNil(t *testing.T) {
	p := DefaultHaloProvider.WithAssetResolver(&mockResolver{})
	for _, path := range []string{"", "   "} {
		result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, path)
		if result != nil {
			t.Fatalf("attendu nil pour chemin %q, got %+v", path, result)
		}
	}
}

func TestFetchRewardTrackDefinition_NilResolver_ReturnsNil(t *testing.T) {
	p := DefaultHaloProvider
	if p.assetResolver != nil {
		t.Fatal("DefaultHaloProvider devrait avoir assetResolver nil")
	}
	result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "RewardTracks/Operations/S6.json")
	if result != nil {
		t.Fatalf("attendu nil avec resolver nil, got %+v", result)
	}
}

func TestFetchRewardTrackDefinition_ResolverError_ReturnsNil(t *testing.T) {
	sentinel := errors.New("upstream unavailable")
	p := DefaultHaloProvider.WithAssetResolver(&mockResolver{
		getFunc: func(_ context.Context, _ assets.Ref) (assets.Resolved, error) {
			return assets.Resolved{}, sentinel
		},
	})
	result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "RewardTracks/Operations/S6.json")
	if result != nil {
		t.Fatalf("attendu nil sur erreur resolver, got %+v", result)
	}
}

func TestFetchRewardTrackDefinition_ValidJSONPayload_ReturnsParsed(t *testing.T) {
	raw := `{"XpPerRank":1000,"Ranks":[]}`
	p := DefaultHaloProvider.WithAssetResolver(&mockResolver{
		getFunc: func(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
			if ref.Kind != assets.KindRewardTrackDefinition {
				return assets.Resolved{}, assets.ErrNotFound
			}
			return assets.Resolved{
				Payload: assets.JSONPayload{RawJSON: json.RawMessage(raw)},
			}, nil
		},
	})
	result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "RewardTracks/Operations/S6.json")
	if result == nil {
		t.Fatal("attendu résultat non-nil pour payload JSON valide")
	}
	if result.XpPerRank != 1000 {
		t.Fatalf("attendu XpPerRank=1000, got %d", result.XpPerRank)
	}
}

func TestFetchRewardTrackDefinition_WrongPayloadType_ReturnsNil(t *testing.T) {
	p := DefaultHaloProvider.WithAssetResolver(&mockResolver{
		getFunc: func(_ context.Context, _ assets.Ref) (assets.Resolved, error) {
			return assets.Resolved{
				Payload: assets.BinaryPayload{Bytes: []byte("not-json")},
				// BinaryPayload is not a JSONPayload — wrong type.
			}, nil
		},
	})
	result := p.fetchRewardTrackDefinition(context.Background(), &domain.HaloTokens{}, "RewardTracks/Operations/S6.json")
	if result != nil {
		t.Fatalf("attendu nil pour payload non-JSON, got %+v", result)
	}
}
