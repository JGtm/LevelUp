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

// ---------------------------------------------------------------------------
// Tests collectTrackItemPaths
// ---------------------------------------------------------------------------

func TestCollectTrackItemPaths_DeduplicatesAcrossBuckets(t *testing.T) {
	p := DefaultHaloProvider
	def := &battlepassTrackDefinitionRaw{
		Ranks: []battlepassRankDefRaw{
			{
				Rank: 1,
				FreeRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/A.json"},
					{InventoryItemPath: "Inventory/B.json"},
				}},
				PaidRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/A.json"}, // doublon
				}},
			},
			{
				Rank: 2,
				FreeRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "  "}, // chemin vide → ignoré
				}},
				PaidRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/C.json"},
				}},
			},
		},
	}

	paths := p.collectTrackItemPaths(def)

	got := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		got[path] = struct{}{}
	}
	for _, want := range []string{"Inventory/A.json", "Inventory/B.json", "Inventory/C.json"} {
		if _, ok := got[want]; !ok {
			t.Errorf("chemin attendu manquant : %q", want)
		}
	}
	if len(paths) != 3 {
		t.Errorf("len(paths) = %d, want 3 (pas de doublons)", len(paths))
	}
}

func TestCollectTrackItemPaths_EmptyRanks_ReturnsEmpty(t *testing.T) {
	p := DefaultHaloProvider
	def := &battlepassTrackDefinitionRaw{}
	if got := p.collectTrackItemPaths(def); len(got) != 0 {
		t.Errorf("attendu slice vide, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Tests resolveAndPersistItem
// ---------------------------------------------------------------------------

// mockItemPersister capture les appels à UpsertItemDefinition.
type mockItemPersister struct {
	calls []struct {
		path string
		raw  []byte
	}
}

func (m *mockItemPersister) UpsertItemDefinition(_ context.Context, path string, raw []byte) error {
	m.calls = append(m.calls, struct {
		path string
		raw  []byte
	}{path, raw})
	return nil
}

func TestResolveAndPersistItem_CallsPersisterWithJSON(t *testing.T) {
	itemJSON := `{"CommonData":{"Quality":"Legendary"}}`
	persister := &mockItemPersister{}
	p := DefaultHaloProvider.
		WithAssetResolver(&mockResolver{
			getFunc: func(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
				if ref.Kind != assets.KindBPItemDefinition {
					return assets.Resolved{}, assets.ErrNotFound
				}
				return assets.Resolved{
					Payload: assets.JSONPayload{RawJSON: json.RawMessage(itemJSON)},
				}, nil
			},
		}).
		WithItemDefPersister(persister)

	p.resolveAndPersistItem(context.Background(), "Inventory/Coat-01.json")

	if len(persister.calls) != 1 {
		t.Fatalf("attendu 1 appel UpsertItemDefinition, got %d", len(persister.calls))
	}
	if persister.calls[0].path != "Inventory/Coat-01.json" {
		t.Errorf("path = %q, want %q", persister.calls[0].path, "Inventory/Coat-01.json")
	}
}

func TestResolveAndPersistItem_ResolverMiss_NoPersist(t *testing.T) {
	persister := &mockItemPersister{}
	p := DefaultHaloProvider.
		WithAssetResolver(&mockResolver{
			getFunc: func(_ context.Context, _ assets.Ref) (assets.Resolved, error) {
				return assets.Resolved{}, assets.ErrNotFound
			},
		}).
		WithItemDefPersister(persister)

	p.resolveAndPersistItem(context.Background(), "Inventory/Missing.json")

	if len(persister.calls) != 0 {
		t.Errorf("attendu 0 appels UpsertItemDefinition sur miss resolver, got %d", len(persister.calls))
	}
}

func TestResolveAndPersistItem_NilPersister_NoError(t *testing.T) {
	p := DefaultHaloProvider.
		WithAssetResolver(&mockResolver{
			getFunc: func(_ context.Context, _ assets.Ref) (assets.Resolved, error) {
				return assets.Resolved{
					Payload: assets.JSONPayload{RawJSON: json.RawMessage(`{}`)},
				}, nil
			},
		})
	// itemDefPersister est nil — ne doit pas paniquer
	p.resolveAndPersistItem(context.Background(), "Inventory/Coat-01.json")
}

func TestResolveAndPersistItem_BinaryPayload_NoPersist(t *testing.T) {
	persister := &mockItemPersister{}
	p := DefaultHaloProvider.
		WithAssetResolver(&mockResolver{
			getFunc: func(_ context.Context, _ assets.Ref) (assets.Resolved, error) {
				return assets.Resolved{
					Payload: assets.BinaryPayload{Bytes: []byte("not-json")},
				}, nil
			},
		}).
		WithItemDefPersister(persister)

	p.resolveAndPersistItem(context.Background(), "Inventory/Coat-01.json")

	if len(persister.calls) != 0 {
		t.Errorf("attendu 0 appels sur payload binaire, got %d", len(persister.calls))
	}
}

// ---------------------------------------------------------------------------
// Tests warmBPTrackAssets
// ---------------------------------------------------------------------------

func TestWarmBPTrackAssets_NilDef_NoOp(t *testing.T) {
	p := DefaultHaloProvider.WithAssetResolver(&mockResolver{})
	p.warmBPTrackAssets(context.Background(), nil) // ne doit pas paniquer
}

func TestWarmBPTrackAssets_NilResolver_NoOp(t *testing.T) {
	p := DefaultHaloProvider // assetResolver nil
	def := &battlepassTrackDefinitionRaw{BattlePassImage: "progression/track.png"}
	p.warmBPTrackAssets(context.Background(), def) // ne doit pas paniquer
}

func TestWarmBPTrackAssets_WarmsImagesAndPersistsItems(t *testing.T) {
	var warmedRefs []assets.Ref
	var resolvedItems []string
	persister := &mockItemPersister{}

	itemJSON := `{"CommonData":{"Quality":"Rare","Type":"ArmorCoating"}}`

	resolver := &mockResolver{
		getFunc: func(_ context.Context, ref assets.Ref) (assets.Resolved, error) {
			if ref.Kind == assets.KindBPItemDefinition {
				resolvedItems = append(resolvedItems, ref.ID)
				return assets.Resolved{
					Payload: assets.JSONPayload{RawJSON: json.RawMessage(itemJSON)},
				}, nil
			}
			return assets.Resolved{}, assets.ErrNotFound
		},
	}
	// Override Warm pour capturer les refs
	warmingResolver := &capturingResolver{mockResolver: resolver, warmed: &warmedRefs}

	p := DefaultHaloProvider.
		WithAssetResolver(warmingResolver).
		WithItemDefPersister(persister)

	def := &battlepassTrackDefinitionRaw{
		BattlePassImage:     "progression/track.png",
		BackgroundImagePath: "progression/bg.png",
		Ranks: []battlepassRankDefRaw{
			{
				Rank: 1,
				FreeRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/Item-A.json"},
				}},
				PaidRewards: battlepassRewardBucketRaw{InventoryRewards: []battlepassInventoryRewardRaw{
					{InventoryItemPath: "Inventory/Item-B.json"},
				}},
			},
		},
	}

	p.warmBPTrackAssets(context.Background(), def)

	// Les images de track doivent avoir été Warm()ed
	wantImages := map[string]bool{
		"progression/track.png": false,
		"progression/bg.png":    false,
	}
	for _, ref := range warmedRefs {
		wantImages[ref.ID] = true
	}
	for id, found := range wantImages {
		if !found {
			t.Errorf("image attendue dans Warm() : %q", id)
		}
	}

	// Les items doivent avoir été résolus et persistés
	if len(persister.calls) != 2 {
		t.Errorf("attendu 2 appels UpsertItemDefinition, got %d", len(persister.calls))
	}
}

// capturingResolver enregistre les appels à Warm tout en déléguant à mockResolver.
type capturingResolver struct {
	*mockResolver
	warmed *[]assets.Ref
}

func (c *capturingResolver) Warm(_ context.Context, refs ...assets.Ref) {
	*c.warmed = append(*c.warmed, refs...)
}
