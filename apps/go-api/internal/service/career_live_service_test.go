// Package service — career_live_service_test.go : tests unitaires du
// CareerLiveService. Couvre :
//
//   - La matrice de fallback à 4 cas (live OK / live partiel / live KO+DB / vide)
//   - Le per-field merge avec la dernière row DB
//   - L'INSERT-if-changed
//   - Le bypass du fetcher quand les tokens sont absents
//
// Les tests d'intégration DuckDB sont dans career_live_repo_test.go
// (build tag integration).
package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// gateResolver implémente games.EndpointResolver + games.CapabilityResolver pour
// piloter le gating title-aware du live carrière dans les tests service.
type gateResolver struct {
	caps map[string]games.CapabilityMap
}

func (gateResolver) HostFor(string, games.EndpointKey) (string, bool) { return "", false }

func (g gateResolver) CapabilitiesFor(slug string) (games.CapabilityMap, bool) {
	c, ok := g.caps[slug]
	return c, ok
}

// --- mocks ---

type mockCareerFetcher struct {
	progress    *syncpkg.CareerRankData
	progressErr error
	custom      *syncpkg.SpartanCustomizationData
	customErr   error
	progCalls   int
	customCalls int
}

func (m *mockCareerFetcher) GetCareerProgress(_ context.Context, _ string) (*syncpkg.CareerRankData, error) {
	m.progCalls++
	return m.progress, m.progressErr
}
func (m *mockCareerFetcher) GetSpartanCustomization(_ context.Context, _ string) (*syncpkg.SpartanCustomizationData, error) {
	m.customCalls++
	return m.custom, m.customErr
}

type mockCareerLiveRepo struct {
	last        *duckdb.CareerRankRow
	loadErr     error
	enrichErr   error
	enrichCalls int
	inserted    []*duckdb.CareerRankRow
	insertErr   error
	insertOK    bool
	// Phase 2/3 PLAN_V2 : suivi des INSERT partials.
	insertedPartials []*duckdb.CareerProgressionPartial
	insertPartialErr error
	insertPartialOK  bool
}

func (m *mockCareerLiveRepo) LoadLastCareerRank(_ context.Context, _ string) (*duckdb.CareerRankRow, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	if m.last == nil {
		return nil, nil
	}
	// Retourner une copie pour éviter les effets de bord.
	cp := *m.last
	return &cp, nil
}

func (m *mockCareerLiveRepo) EnrichFromMetadata(_ context.Context, row *duckdb.CareerRankRow) error {
	m.enrichCalls++
	if m.enrichErr != nil {
		return m.enrichErr
	}
	// Stub : pose des valeurs prévisibles pour vérifier l'appel.
	if row != nil && row.Rank > 0 {
		if row.RankName == "" {
			row.RankName = "MockRank"
		}
		if row.XPForNextRank == 0 {
			row.XPForNextRank = 5000
		}
	}
	return nil
}

func (m *mockCareerLiveRepo) InsertCareerProgressionIfChanged(_ context.Context, _ string, row *duckdb.CareerRankRow) (bool, error) {
	if m.insertErr != nil {
		return false, m.insertErr
	}
	cp := *row
	m.inserted = append(m.inserted, &cp)
	return m.insertOK, nil
}

func (m *mockCareerLiveRepo) InsertCareerProgressionPartial(_ context.Context, _ string, p *duckdb.CareerProgressionPartial) (bool, error) {
	if m.insertPartialErr != nil {
		return false, m.insertPartialErr
	}
	if p == nil {
		return false, nil
	}
	// Copie défensive : tests vérifient ce qu'on a reçu sans risque de mutation.
	cp := *p
	m.insertedPartials = append(m.insertedPartials, &cp)
	return m.insertPartialOK, nil
}

type mockIdentityBuilder struct {
	receivedRow          *duckdb.CareerRankRow
	receivedIncludePeaks bool
	returnNil            bool
}

func (m *mockIdentityBuilder) BuildSpartanIdentityFromCareerRow(_ context.Context, row *duckdb.CareerRankRow, includePeaks bool) *domain.HomeSpartanIdentityRow {
	m.receivedRow = row
	m.receivedIncludePeaks = includePeaks
	if m.returnNil || row == nil {
		return nil
	}
	id := &domain.HomeSpartanIdentityRow{
		RankNumber: row.Rank,
		CurrentXP:  row.CurrentXP,
	}
	if row.SpartanID != "" {
		s := row.SpartanID
		id.SpartanID = &s
	}
	if row.BannerImageURL != "" {
		b := row.BannerImageURL
		id.BannerImageURL = &b
	}
	if row.EmblemImageURL != "" {
		e := row.EmblemImageURL
		id.EmblemImageURL = &e
	}
	if row.BackdropImageURL != "" {
		bd := row.BackdropImageURL
		id.BackdropImageURL = &bd
	}
	return id
}

// TestCareerLive_IncludePeaks_OnlyForOwner verrouille le volet B du fix
// explorer-target-profile-auth : les skill peaks (lus sur la player DB du
// propriétaire de la page) ne doivent être calculés QUE si le sujet de
// l'identité est ce propriétaire (xuid == HaloXUID du ctx). Pour un joueur
// tiers (cas Explorer cible), includePeaks doit être false — sinon on
// afficherait les peaks du mauvais joueur et on paierait 2 scans inutiles.
func TestCareerLive_IncludePeaks_OnlyForOwner(t *testing.T) {
	const ownerXUID = "1234567890123456" // == ctxWithTokens xuid
	const otherXUID = "9999999999999999"

	newWithRow := func() (*CareerLiveService, *mockIdentityBuilder) {
		repo := &mockCareerLiveRepo{last: &duckdb.CareerRankRow{Rank: 10}}
		builder := &mockIdentityBuilder{}
		return newService(t, nil, repo, builder), builder
	}

	t.Run("owner via GetSpartanIdentityFor → peaks inclus", func(t *testing.T) {
		svc, builder := newWithRow()
		if _, err := svc.GetSpartanIdentityFor(ctxWithTokens(t, true), ownerXUID); err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}
		if !builder.receivedIncludePeaks {
			t.Fatal("attendu includePeaks=true pour le propriétaire")
		}
	})

	t.Run("tiers via GetSpartanIdentityFor → peaks exclus", func(t *testing.T) {
		svc, builder := newWithRow()
		if _, err := svc.GetSpartanIdentityFor(ctxWithTokens(t, true), otherXUID); err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}
		if builder.receivedIncludePeaks {
			t.Fatal("attendu includePeaks=false pour un joueur tiers")
		}
	})
}

// --- helpers ---

func ctxWithTokens(t *testing.T, hasTokens bool) context.Context {
	t.Helper()
	ctx := context.Background()
	if hasTokens {
		ctx = ctxkeys.WithHaloAuth(ctx, &domain.HaloTokens{SpartanToken: "spartan-token-xxx"}, "1234567890123456")
	} else {
		ctx = ctxkeys.WithHaloAuth(ctx, nil, "1234567890123456")
	}
	return ctx
}

func newService(t *testing.T, fetcher *mockCareerFetcher, repo *mockCareerLiveRepo, builder CareerIdentityBuilder) *CareerLiveService {
	t.Helper()
	factory := func(_ context.Context) CareerFetcher {
		if fetcher == nil {
			return nil
		}
		return fetcher
	}
	return NewCareerLiveService(repo, builder, factory, nil) // pas de cache pour les tests
}

// --- TestCareerLive_MergeCareerRow_Matrix ---
//
// Matrice du merge per-field : c'est le contrat utilisateur final
// "le bloc Spartan ne doit jamais être vide si la DB porte une row".
// Teste mergeCareerRow directement (fonction pure, déterministe) sur les
// combinaisons live × DB qui correspondent aux états possibles du cache
// après refresh background.

func TestCareerLive_MergeCareerRow_Matrix(t *testing.T) {
	dbRowFull := &duckdb.CareerRankRow{
		Rank: 30, CurrentXP: 500, SpartanID: "SR-DB-001",
		BannerImageURL: "https://db/banner.png", EmblemImageURL: "https://db/emblem.png",
		BackdropImageURL: "https://db/backdrop.png",
	}

	cases := []struct {
		name          string
		progress      *syncpkg.CareerRankData
		custom        *syncpkg.SpartanCustomizationData
		dbRow         *duckdb.CareerRankRow
		wantMerged    bool
		wantSpartanID string
		wantEmblemURL string
		wantRank      int
		wantCurrentXP int
	}{
		{
			name:          "cache hit live OK full",
			progress:      &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			custom:        &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99", EmblemImageURL: "https://live/emblem.png"},
			dbRow:         dbRowFull,
			wantMerged:    true,
			wantSpartanID: "SR-LIVE-99",
			wantEmblemURL: "https://live/emblem.png",
			wantRank:      42,
			wantCurrentXP: 1500,
		},
		{
			name:          "cache hit custom partielle → per-field carry-forward DB",
			progress:      &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			custom:        &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99"}, // emblem vide
			dbRow:         dbRowFull,
			wantMerged:    true,
			wantSpartanID: "SR-LIVE-99",
			wantEmblemURL: "https://db/emblem.png", // carry-forward
			wantRank:      42,
			wantCurrentXP: 1500,
		},
		{
			name:          "cache hit progress only → custom carry-forward DB",
			progress:      &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			custom:        nil,
			dbRow:         dbRowFull,
			wantMerged:    true,
			wantSpartanID: "SR-DB-001",
			wantEmblemURL: "https://db/emblem.png",
			wantRank:      42,
			wantCurrentXP: 1500,
		},
		{
			name:          "cache miss + DB full → DB only (SwR fallback)",
			progress:      nil,
			custom:        nil,
			dbRow:         dbRowFull,
			wantMerged:    true,
			wantSpartanID: "SR-DB-001",
			wantEmblemURL: "https://db/emblem.png",
			wantRank:      30,
			wantCurrentXP: 500,
		},
		{
			name:       "cache miss + DB vide → nil (joueur jamais sync'd)",
			progress:   nil,
			custom:     nil,
			dbRow:      nil,
			wantMerged: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged := mergeCareerRow(tc.progress, tc.custom, tc.dbRow)
			if !tc.wantMerged {
				if merged != nil {
					t.Errorf("merged attendu nil, obtenu %+v", merged)
				}
				return
			}
			if merged == nil {
				t.Fatal("merged attendu non-nil")
			}
			if merged.Rank != tc.wantRank {
				t.Errorf("rank = %d, want %d", merged.Rank, tc.wantRank)
			}
			if merged.CurrentXP != tc.wantCurrentXP {
				t.Errorf("current_xp = %d, want %d", merged.CurrentXP, tc.wantCurrentXP)
			}
			if tc.wantSpartanID != "" && merged.SpartanID != tc.wantSpartanID {
				t.Errorf("spartan_id = %q, want %q", merged.SpartanID, tc.wantSpartanID)
			}
			if tc.wantEmblemURL != "" && merged.EmblemImageURL != tc.wantEmblemURL {
				t.Errorf("emblem_image_url = %q, want %q", merged.EmblemImageURL, tc.wantEmblemURL)
			}
		})
	}
}

// TestCareerLive_MergeCustom_BannerNeverEmpty cadenasse la directive produit
// apparence côté merge (2026-07-08, cas JGtm emblème 3806589 sans nameplate
// upstream) : les champs sont indépendants — la bannière dbLast est
// carry-forwardée même si l'emblème live a changé ; on affiche toujours une
// bannière (l'actuelle sinon la dernière connue), jamais un bloc vide.
func TestCareerLive_MergeCustom_BannerNeverEmpty(t *testing.T) {
	dbLast := &duckdb.CareerRankRow{
		Rank: 30, CurrentXP: 500, SpartanID: "SR-DB",
		BannerImageURL: "https://db/banner-old.png", EmblemImageURL: "https://db/emblem-old.png",
		BackdropImageURL: "https://db/backdrop.png",
	}

	t.Run("emblème changé + banner irrésoluble → carry de la dernière bannière connue", func(t *testing.T) {
		custom := &syncpkg.SpartanCustomizationData{
			SpartanID:      "SR-LIVE",
			EmblemImageURL: "https://live/emblem-new.png",
			// BannerImageURL vide : nameplate absente upstream.
		}
		merged := mergeCareerRow(nil, custom, dbLast)
		if merged == nil {
			t.Fatal("merged attendu non-nil")
		}
		if merged.EmblemImageURL != "https://live/emblem-new.png" {
			t.Errorf("emblem = %q, want live emblem-new", merged.EmblemImageURL)
		}
		if merged.BannerImageURL != "https://db/banner-old.png" {
			t.Errorf("banner = %q, want dernière bannière connue (jamais vide)", merged.BannerImageURL)
		}
		if merged.BackdropImageURL != "https://db/backdrop.png" {
			t.Errorf("backdrop = %q, want carry-forward DB", merged.BackdropImageURL)
		}
	})

	t.Run("même emblème + banner vide transitoire → carry bannière (anti-flicker)", func(t *testing.T) {
		custom := &syncpkg.SpartanCustomizationData{
			SpartanID:      "SR-LIVE",
			EmblemImageURL: "https://db/emblem-old.png", // inchangé
		}
		merged := mergeCareerRow(nil, custom, dbLast)
		if merged == nil {
			t.Fatal("merged attendu non-nil")
		}
		if merged.BannerImageURL != "https://db/banner-old.png" {
			t.Errorf("banner = %q, want carry-forward DB (même emblème)", merged.BannerImageURL)
		}
	})

	t.Run("custom nil → paire dbLast entière carry-forwardée", func(t *testing.T) {
		merged := mergeCareerRow(nil, nil, dbLast)
		if merged == nil {
			t.Fatal("merged attendu non-nil")
		}
		if merged.BannerImageURL != "https://db/banner-old.png" || merged.EmblemImageURL != "https://db/emblem-old.png" {
			t.Errorf("paire = (%q, %q), want paire dbLast complète", merged.BannerImageURL, merged.EmblemImageURL)
		}
	})
}

// TestCareerLive_GetIdentity_SwRBehavior valide le contrat stale-while-
// revalidate côté service :
//   - cache vide → DB servie immédiatement (SANS attendre live)
//   - cache plein → cache + DB mergées immédiatement
//   - cache miss + DB vide → nil
//   - pas de tokens → DB only, jamais d'appel HTTP
func TestCareerLive_GetIdentity_SwRBehavior(t *testing.T) {
	dbRow := &duckdb.CareerRankRow{Rank: 30, CurrentXP: 500, SpartanID: "SR-DB"}

	t.Run("cache miss → DB served + bg refresh kicked", func(t *testing.T) {
		fetcher := &mockCareerFetcher{
			progress: &syncpkg.CareerRankData{CurrentRank: 99, CurrentXP: 9999},
		}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
		if err != nil {
			t.Fatalf("GetSpartanIdentity: %v", err)
		}
		if got == nil || got.RankNumber != 30 {
			t.Errorf("cache miss attendu DB rank=30, obtenu %+v", got)
		}
	})

	t.Run("cache hit → cache mergé avec DB", func(t *testing.T) {
		fetcher := &mockCareerFetcher{}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		// Pré-populate cache pour simuler un précédent refresh background
		cache.PutProgress("1234567890123456", &syncpkg.CareerRankData{CurrentRank: 99, CurrentXP: 9999})
		cache.PutCustomization("1234567890123456", &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE"})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
		if err != nil {
			t.Fatalf("GetSpartanIdentity: %v", err)
		}
		if got == nil || got.RankNumber != 99 || got.SpartanID == nil || *got.SpartanID != "SR-LIVE" {
			t.Errorf("cache hit attendu rank=99 spartan=SR-LIVE, obtenu %+v", got)
		}
	})

	t.Run("pas de tokens → DB only, fetcher jamais appelé", func(t *testing.T) {
		fetcher := &mockCareerFetcher{}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		svc := newService(t, fetcher, repo, builder)

		got, err := svc.GetSpartanIdentity(ctxWithTokens(t, false))
		if err != nil {
			t.Fatalf("GetSpartanIdentity: %v", err)
		}
		if got == nil || got.RankNumber != 30 {
			t.Errorf("no-tokens attendu DB rank=30, obtenu %+v", got)
		}
		if fetcher.progCalls != 0 || fetcher.customCalls != 0 {
			t.Errorf("fetcher invoqué sans tokens : progress=%d custom=%d",
				fetcher.progCalls, fetcher.customCalls)
		}
	})

	t.Run("cache miss + DB empty → nil", func(t *testing.T) {
		fetcher := &mockCareerFetcher{}
		repo := &mockCareerLiveRepo{last: nil}
		builder := &mockIdentityBuilder{}
		svc := newService(t, fetcher, repo, builder)

		got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
		if err != nil {
			t.Fatalf("GetSpartanIdentity: %v", err)
		}
		if got != nil {
			t.Errorf("attendu nil (joueur jamais sync'd), obtenu %+v", got)
		}
	})
}

// TestCareerLive_MergePreservesLiveXPEvenIfZero vérifie qu'on n'écrase
// JAMAIS un current_xp=0 live (palier juste franchi) avec le DB carry-
// forward — c'est un cas piège qui ferait régresser l'affichage XP.
func TestCareerLive_MergePreservesLiveXPEvenIfZero(t *testing.T) {
	live := &syncpkg.CareerRankData{CurrentRank: 50, CurrentXP: 0, IsMaxRank: false}
	db := &duckdb.CareerRankRow{Rank: 49, CurrentXP: 4900, SpartanID: "SR-001"}

	merged := mergeCareerRow(live, nil, db)
	if merged == nil {
		t.Fatal("merge attendu non-nil")
	}
	if merged.Rank != 50 {
		t.Errorf("rank = %d, want 50 (live)", merged.Rank)
	}
	if merged.CurrentXP != 0 {
		t.Errorf("current_xp = %d, want 0 (live ne doit pas être écrasé par DB carry)", merged.CurrentXP)
	}
	if merged.SpartanID != "SR-001" {
		t.Errorf("spartan_id = %q, want SR-001 (DB carry-forward)", merged.SpartanID)
	}
}

// TestCareerLive_NoTokens_NoFetcherCall vérifie que sans tokens dans le
// contexte, on ne tente jamais d'appel HTTP (préserve rate limit + évite
// les 401 inutiles).
func TestCareerLive_NoTokens_NoFetcherCall(t *testing.T) {
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{CurrentRank: 42},
	}
	repo := &mockCareerLiveRepo{last: &duckdb.CareerRankRow{Rank: 30, CurrentXP: 100}}
	builder := &mockIdentityBuilder{}
	svc := newService(t, fetcher, repo, builder)

	// hasTokens=false : ctxkeys.HaloTokens(ctx) retournera nil
	got, err := svc.GetSpartanIdentity(ctxWithTokens(t, false))
	if err != nil {
		t.Fatalf("GetSpartanIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("identity attendue non-nil (fallback DB)")
	}
	if got.RankNumber != 30 {
		t.Errorf("rank = %d, want 30 (DB)", got.RankNumber)
	}
	if fetcher.progCalls != 0 || fetcher.customCalls != 0 {
		t.Errorf("fetcher appelé sans tokens : progress=%d custom=%d",
			fetcher.progCalls, fetcher.customCalls)
	}
}

// TestCareerLive_NoXUID_TriggersFallbackPath vérifie qu'un contexte sans
// xuid (cas extrême : middleware mal configuré) bascule sur le fallback DB
// nominal sans crasher. Le mock repo retournera nil parce que xuid="" ne
// peut pas vraiment matcher une row, mais le service ne doit pas exploser.
func TestCareerLive_NoXUID_TriggersFallbackPath(t *testing.T) {
	fetcher := &mockCareerFetcher{}
	repo := &mockCareerLiveRepo{last: nil}
	builder := &mockIdentityBuilder{}
	svc := newService(t, fetcher, repo, builder)

	// Contexte sans xuid : pas de fetcher invoqué, fallback DB direct.
	got, err := svc.GetSpartanIdentity(context.Background())
	if err != nil {
		t.Fatalf("GetSpartanIdentity sans xuid: %v", err)
	}
	if got != nil {
		t.Errorf("identity attendue nil sans xuid + DB vide, obtenu %+v", got)
	}
}

// TestCareerLive_TitleSansCatalogue_SkipLiveFetch verrouille le fix S1 :
// pour un titre qui n'expose PAS le catalogue de rangs de carrière
// (career.rank_catalog absent, ex. Halo 5 dont le SR vient de la carnage),
// le live careerranks (endpoint economy Halo Infinite) NE doit JAMAIS être
// appelé — même avec des tokens valides et un cache pré-rempli — sinon on
// écrirait la carrière INFINITE dans la player DB du titre (contamination
// cross-titre). La row DB du titre reste servie telle quelle.
//
// Comparaison de contrôle : sans gating (slug par défaut halo_infinite, où
// career.rank_catalog est supported), le chemin live normal s'exécute.
func TestCareerLive_TitleSansCatalogue_SkipLiveFetch(t *testing.T) {
	// Resolver partagé : halo_infinite a le catalogue, halo_5 ne l'a pas.
	prev := games.DefaultEndpointResolver()
	games.SetDefaultEndpointResolver(gateResolver{caps: map[string]games.CapabilityMap{
		"halo_infinite": {games.CapCareerRankCatalog: games.CapSupported},
		"halo_5":        {games.CapCareerProgression: games.CapSupported}, // pas de rank_catalog
	}})
	t.Cleanup(func() { games.SetDefaultEndpointResolver(prev) })

	const xuid = "1234567890123456" // == ctxWithTokens xuid (owner)
	dbRow := &duckdb.CareerRankRow{Rank: 152, CurrentXP: 4200, SpartanID: "SR-H5"}

	t.Run("halo_5 → live careerranks court-circuité, DB servie", func(t *testing.T) {
		fetcher := &mockCareerFetcher{
			progress: &syncpkg.CareerRankData{CurrentRank: 272, CurrentXP: 99999}, // carrière HINF polluante
		}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		// Cache pré-rempli avec une progression HINF : le gating doit l'ignorer.
		cache.PutProgress(xuid, &syncpkg.CareerRankData{CurrentRank: 272, CurrentXP: 99999})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		ctx := ctxkeys.WithTitleSlug(ctxWithTokens(t, true), "halo_5")
		got, err := svc.GetSpartanIdentityFor(ctx, xuid)
		if err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}
		if got == nil || got.RankNumber != 152 {
			t.Fatalf("attendu DB rank=152 (SR H5), obtenu %+v", got)
		}
		if fetcher.progCalls != 0 || fetcher.customCalls != 0 {
			t.Errorf("live careerranks NE doit PAS être appelé pour halo_5 : progress=%d custom=%d",
				fetcher.progCalls, fetcher.customCalls)
		}
	})

	t.Run("halo_infinite → chemin live normal (contrôle)", func(t *testing.T) {
		fetcher := &mockCareerFetcher{}
		repo := &mockCareerLiveRepo{last: &duckdb.CareerRankRow{Rank: 30, CurrentXP: 500}}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		cache.PutProgress(xuid, &syncpkg.CareerRankData{CurrentRank: 99, CurrentXP: 9999})
		cache.PutCustomization(xuid, &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE"})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		ctx := ctxkeys.WithTitleSlug(ctxWithTokens(t, true), "halo_infinite")
		got, err := svc.GetSpartanIdentityFor(ctx, xuid)
		if err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}
		// Cache hit live → rank 99 servi (chemin live normal préservé).
		if got == nil || got.RankNumber != 99 {
			t.Errorf("halo_infinite attendu chemin live (rank=99), obtenu %+v", got)
		}
	})
}

// slowFetcher simule un fetcher qui bloque plus longtemps que le budget,
// jusqu'à la cancellation du ctx. Utilisé pour valider le découplage
// home / live (budget timeout → fallback DB sans bloquer la home).
type slowFetcher struct {
	progressDelay time.Duration
	customDelay   time.Duration
}

func (s *slowFetcher) GetCareerProgress(ctx context.Context, _ string) (*syncpkg.CareerRankData, error) {
	select {
	case <-time.After(s.progressDelay):
		return &syncpkg.CareerRankData{CurrentRank: 99}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *slowFetcher) GetSpartanCustomization(ctx context.Context, _ string) (*syncpkg.SpartanCustomizationData, error) {
	select {
	case <-time.After(s.customDelay):
		return &syncpkg.SpartanCustomizationData{SpartanID: "SR-SLOW"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TestCareerLive_SlowFetcher_DoesNotBlockHome est LE test qui acte la
// garantie "la home ne hang jamais sur Halo" via le pattern stale-while-
// revalidate : peu importe la lenteur du fetcher live, GetSpartanIdentity
// doit retourner instantanément avec la dernière row DB connue. Le
// refresh live happen en background goroutine, sans pénaliser le caller.
//
// C'est le test de régression directe pour le bug "home charge pas en 60s"
// reporté après le câblage initial. Avant la refacto SwR, ce test prenait
// 2.5s (budget). Après, < 100ms.
func TestCareerLive_SlowFetcher_DoesNotBlockHome(t *testing.T) {
	slow := &slowFetcher{
		progressDelay: 30 * time.Second, // bien au-delà de tout budget raisonnable
		customDelay:   30 * time.Second,
	}
	dbRow := &duckdb.CareerRankRow{
		Rank: 30, CurrentXP: 500, SpartanID: "SR-DB-001",
		EmblemImageURL: "https://db/emblem.png",
	}
	repo := &mockCareerLiveRepo{last: dbRow}
	builder := &mockIdentityBuilder{}

	factory := func(_ context.Context) CareerFetcher { return slow }
	cache := NewCareerLiveCache(CareerLiveCacheConfig{}) // cache nécessaire pour qu'on déclenche bg refresh
	svc := NewCareerLiveService(repo, builder, factory, cache)

	start := time.Now()
	got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetSpartanIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("identity attendue non-nil (fallback DB)")
	}
	if got.RankNumber != 30 || got.SpartanID == nil || *got.SpartanID != "SR-DB-001" {
		t.Errorf("identity = %+v, want rank=30 spartan=SR-DB-001 (DB fallback)", got)
	}
	// SwR : la home doit retourner quasi-instantanément, même si le fetcher
	// est lent. 500 ms est large pour absorber l'overhead test + le spawn de
	// la goroutine background.
	if elapsed > 500*time.Millisecond {
		t.Errorf("GetSpartanIdentity prit %v, attendu < 500ms (SwR ne doit pas bloquer sur live)", elapsed)
	}
}

// --- TestOverlayIdentityFromFallback ---
//
// Anti-régression « les bannières vont et viennent » (revue 2026-05-20).
// Contrat UI-first : un fetch live qui rend BannerImageURL=nil ne doit JAMAIS
// écraser la valeur DB historique. La fonction overlayIdentityFromFallback
// est le filet final qui garantit ça.

func sPtr(s string) *string { return &s }

func TestOverlayIdentityFromFallback(t *testing.T) {
	cases := []struct {
		name     string
		identity *domain.HomeSpartanIdentityRow
		fallback *domain.HomeSpartanIdentityRow
		wantNil  bool
		wantBann *string
		wantEmbl *string
		wantBack *string
		wantSp   *string
	}{
		{
			name:     "identity nil + fallback non-nil → fallback",
			identity: nil,
			fallback: &domain.HomeSpartanIdentityRow{
				BannerImageURL:   sPtr("/db/banner.png"),
				EmblemImageURL:   sPtr("/db/emblem.png"),
				BackdropImageURL: sPtr("/db/backdrop.png"),
				SpartanID:        sPtr("OKLM"),
			},
			wantBann: sPtr("/db/banner.png"),
			wantEmbl: sPtr("/db/emblem.png"),
			wantBack: sPtr("/db/backdrop.png"),
			wantSp:   sPtr("OKLM"),
		},
		{
			name:     "identity + fallback tous deux nil → nil",
			identity: nil,
			fallback: nil,
			wantNil:  true,
		},
		{
			name: "identity sans banner + fallback avec banner → patche depuis fallback",
			identity: &domain.HomeSpartanIdentityRow{
				SpartanID: sPtr("OKLM-live"),
				// pas de BannerImageURL (live a rendu nil)
			},
			fallback: &domain.HomeSpartanIdentityRow{
				BannerImageURL: sPtr("/db/banner.png"),
				SpartanID:      sPtr("OKLM-db-old"), // pas écrasé car live l'a
			},
			wantBann: sPtr("/db/banner.png"),
			wantSp:   sPtr("OKLM-live"),
		},
		{
			name: "identity avec banner live + fallback différent → garde live",
			identity: &domain.HomeSpartanIdentityRow{
				BannerImageURL: sPtr("/live/banner-new.png"),
			},
			fallback: &domain.HomeSpartanIdentityRow{
				BannerImageURL: sPtr("/db/banner-old.png"),
			},
			wantBann: sPtr("/live/banner-new.png"),
		},
		{
			// Directive « jamais vide » + champs indépendants (2026-07-08, cas
			// JGtm emblème 3806589) : l'emblème live a changé et la nameplate est
			// irrésoluble upstream → la dernière bannière connue (fallback) doit
			// quand même être patchée.
			name: "emblème changé sans banner → patch de la dernière bannière connue",
			identity: &domain.HomeSpartanIdentityRow{
				EmblemImageURL: sPtr("/live/emblem-new.png"),
				// pas de BannerImageURL (nameplate absente upstream)
			},
			fallback: &domain.HomeSpartanIdentityRow{
				EmblemImageURL: sPtr("/db/emblem-old.png"),
				BannerImageURL: sPtr("/db/banner-old.png"),
			},
			wantBann: sPtr("/db/banner-old.png"),
			wantEmbl: sPtr("/live/emblem-new.png"),
		},
		{
			name: "identity complète + fallback nil → garde identity",
			identity: &domain.HomeSpartanIdentityRow{
				BannerImageURL: sPtr("/live/b.png"),
			},
			fallback: nil,
			wantBann: sPtr("/live/b.png"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := overlayIdentityFromFallback(tc.identity, tc.fallback)
			if tc.wantNil {
				if got != nil {
					t.Errorf("got %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil, want non-nil")
			}
			if !strPtrEq(got.BannerImageURL, tc.wantBann) {
				t.Errorf("BannerImageURL = %v, want %v", strPtrDeref(got.BannerImageURL), strPtrDeref(tc.wantBann))
			}
			if tc.wantEmbl != nil && !strPtrEq(got.EmblemImageURL, tc.wantEmbl) {
				t.Errorf("EmblemImageURL = %v, want %v", strPtrDeref(got.EmblemImageURL), strPtrDeref(tc.wantEmbl))
			}
			if tc.wantBack != nil && !strPtrEq(got.BackdropImageURL, tc.wantBack) {
				t.Errorf("BackdropImageURL = %v, want %v", strPtrDeref(got.BackdropImageURL), strPtrDeref(tc.wantBack))
			}
			if tc.wantSp != nil && !strPtrEq(got.SpartanID, tc.wantSp) {
				t.Errorf("SpartanID = %v, want %v", strPtrDeref(got.SpartanID), strPtrDeref(tc.wantSp))
			}
		})
	}
}

func strPtrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func strPtrDeref(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

// TestCareerLive_GetIdentity_LiveBannerNil_DBHasOne cadenasse le contrat
// end-to-end : quand la cache live retourne `cachedCustom != nil` mais
// `BannerImageURL=""` (mapping resolver hits rate-limit / résolution échoue),
// et la DB a une bannière historique, la home doit servir la bannière DB.
//
// Reproduit le scénario de prod : background refresh re-fetch la
// customization, écrit `cachedCustom.BannerImageURL = ""` dans le cache,
// la requête suivante voit cachedCustom non-nil → mergeCareerRow ne fallback
// PAS sur dbLast pour la banner (puisque custom est officiellement non-nil
// avec une valeur "rien") — bug latent corrigé par overlayIdentityFromFallback.
func TestCareerLive_GetIdentity_LiveBannerNil_DBHasOne(t *testing.T) {
	dbRow := &duckdb.CareerRankRow{
		Rank:             42,
		CurrentXP:        1500,
		SpartanID:        "SR-DB-001",
		BannerImageURL:   "https://db/banner-known-good.png",
		EmblemImageURL:   "https://db/emblem.png",
		BackdropImageURL: "https://db/backdrop.png",
	}
	fetcher := &mockCareerFetcher{
		progress: &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
		// Custom non-nil mais BannerImageURL="" : simulé le cas "mapping.json
		// resolver retourne '' parce qu'upstream HTTP failed / rate-limit".
		custom: &syncpkg.SpartanCustomizationData{
			SpartanID: "SR-LIVE-99",
			// pas de BannerImageURL → ""
		},
	}
	repo := &mockCareerLiveRepo{last: dbRow}
	builder := &realBuilderForOverlay{} // un builder qui copie tous les champs
	svc := newService(t, fetcher, repo, builder)

	got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
	if err != nil {
		t.Fatalf("GetSpartanIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("identity attendue non-nil")
	}
	if got.BannerImageURL == nil || *got.BannerImageURL != "https://db/banner-known-good.png" {
		t.Errorf("BannerImageURL = %v, want %q (DB last-known-good)", strPtrDeref(got.BannerImageURL), "https://db/banner-known-good.png")
	}
}

// realBuilderForOverlay copie tous les champs de CareerRankRow vers
// HomeSpartanIdentityRow — comportement réaliste pour tester l'overlay.
type realBuilderForOverlay struct{}

func (b *realBuilderForOverlay) BuildSpartanIdentityFromCareerRow(_ context.Context, row *duckdb.CareerRankRow, _ bool) *domain.HomeSpartanIdentityRow {
	if row == nil {
		return nil
	}
	out := &domain.HomeSpartanIdentityRow{
		RankNumber: row.Rank,
		CurrentXP:  row.CurrentXP,
	}
	if row.SpartanID != "" {
		s := row.SpartanID
		out.SpartanID = &s
	}
	if row.BannerImageURL != "" {
		s := row.BannerImageURL
		out.BannerImageURL = &s
	}
	if row.EmblemImageURL != "" {
		s := row.EmblemImageURL
		out.EmblemImageURL = &s
	}
	if row.BackdropImageURL != "" {
		s := row.BackdropImageURL
		out.BackdropImageURL = &s
	}
	return out
}

// TestCareerLive_NilAPIResponse_NotCached vérifie que si GetCareerProgress retourne
// (nil, nil) — réponse silencieuse de l'API (401/403 ou payload non parseable) —
// on ne met PAS nil en cache. Sans ce garde-fou, le cache stocke une entrée nil
// qui retourne hit=true sur la requête suivante et bloque définitivement le
// background refresh pour les 5 prochaines minutes.
func TestCareerLive_NilAPIResponse_NotCached(t *testing.T) {
	done := make(chan struct{})
	fetcher := &nilSignalingFetcher{done: done}
	dbRow := &duckdb.CareerRankRow{Rank: 50, CurrentXP: 1000, SpartanID: "SR-DB-NIL"}
	repo := &mockCareerLiveRepo{last: dbRow}
	builder := &mockIdentityBuilder{}

	factory := func(_ context.Context) CareerFetcher { return fetcher }
	cache := NewCareerLiveCache(CareerLiveCacheConfig{})
	svc := NewCareerLiveService(repo, builder, factory, cache)

	// Premier appel : cache miss → DB servie + background refresh déclenché.
	got, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
	if err != nil {
		t.Fatalf("GetSpartanIdentity: %v", err)
	}
	if got == nil || got.RankNumber != 50 {
		t.Errorf("attendu DB rank=50, obtenu %+v", got)
	}

	// Attendre que la goroutine background ait terminé (fetcher signale via done).
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh n'a pas terminé dans les temps")
	}

	// Le cache ne doit PAS contenir une entrée nil pour progress.
	// Si nil était caché, GetProgress retournerait (nil, true) et needRefresh
	// resterait false → le refresh ne se redéclencherait jamais.
	if p, hit := cache.GetProgress("1234567890123456"); hit {
		t.Errorf("nil progress ne devrait pas être caché (got hit=true, data=%v)", p)
	}

	// Deuxième appel : doit rester cache miss → background refresh se redéclenche.
	// On vérifie indirectement en vérifiant que le rank DB est toujours servi
	// (et pas un rank=0 issu d'un merge avec nil caché).
	got2, err := svc.GetSpartanIdentity(ctxWithTokens(t, true))
	if err != nil {
		t.Fatalf("GetSpartanIdentity 2e appel: %v", err)
	}
	if got2 == nil || got2.RankNumber != 50 {
		t.Errorf("2e appel : attendu DB rank=50, obtenu %+v", got2)
	}
}

// TestCareerLive_GetSpartanIdentityFor_ThirdPartyXUID couvre le contrat du
// path Explorer : un xuid tiers (différent du user connecté) doit :
//   - être servi avec les données live + DB (lecture OK)
//   - NE PAS déclencher kickoffBackgroundRefresh (pas de persistance dans
//     la career_progression de la player DB du user connecté)
//
// Le mock fetcher porte un canal `done` pour détecter si le path background
// a été déclenché : un read dans le canal après un court délai signifie que
// la goroutine background a tourné — comportement à proscrire pour xuid tiers.
func TestCareerLive_GetSpartanIdentityFor_ThirdPartyXUID(t *testing.T) {
	const (
		userXUID   = "1234567890123456"
		targetXUID = "9999888877776666"
	)
	dbRow := &duckdb.CareerRankRow{Rank: 30, CurrentXP: 500, SpartanID: "SR-DB"}

	t.Run("xuid tiers : kickoff background non déclenché malgré cache miss", func(t *testing.T) {
		bgDone := make(chan struct{}, 4)
		fetcher := &bgSignalingFetcher{
			progress: &syncpkg.CareerRankData{CurrentRank: 99, CurrentXP: 9999},
			custom:   &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99"},
			bgDone:   bgDone,
		}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		ctx := ctxkeys.WithHaloAuth(context.Background(),
			&domain.HaloTokens{SpartanToken: "spartan-tok"}, userXUID)
		got, err := svc.GetSpartanIdentityFor(ctx, targetXUID)
		if err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}
		if got == nil {
			t.Fatal("identity attendue non-nil (DB fallback dispo)")
		}

		// Tolérance : on laisse un court délai pour qu'une éventuelle goroutine
		// background ait le temps de tourner. Si elle tourne, le test échoue.
		select {
		case <-bgDone:
			t.Errorf("kickoffBackgroundRefresh déclenché pour xuid tiers — fuite de persistance")
		case <-time.After(150 * time.Millisecond):
			// OK : pas de background refresh, comme attendu.
		}

		// Vérification additionnelle : pas d'INSERT dans career_progression.
		if len(repo.insertedPartials) > 0 {
			t.Errorf("InsertCareerProgressionPartial appelé %d fois — devait rester à 0 pour xuid tiers",
				len(repo.insertedPartials))
		}
	})

	t.Run("xuid user connecté : kickoff background normalement déclenché", func(t *testing.T) {
		bgDone := make(chan struct{}, 4)
		fetcher := &bgSignalingFetcher{
			progress: &syncpkg.CareerRankData{CurrentRank: 99, CurrentXP: 9999},
			custom:   &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99"},
			bgDone:   bgDone,
		}
		repo := &mockCareerLiveRepo{last: dbRow}
		builder := &mockIdentityBuilder{}
		factory := func(_ context.Context) CareerFetcher { return fetcher }
		cache := NewCareerLiveCache(CareerLiveCacheConfig{})
		svc := NewCareerLiveService(repo, builder, factory, cache)

		ctx := ctxkeys.WithHaloAuth(context.Background(),
			&domain.HaloTokens{SpartanToken: "spartan-tok"}, userXUID)
		// Appel avec userXUID == ctxkeys.HaloXUID(ctx) → allowPersist=true
		if _, err := svc.GetSpartanIdentityFor(ctx, userXUID); err != nil {
			t.Fatalf("GetSpartanIdentityFor: %v", err)
		}

		// Le background refresh doit tourner pour le user lui-même.
		select {
		case <-bgDone:
			// OK : kickoff déclenché.
		case <-time.After(500 * time.Millisecond):
			t.Errorf("kickoffBackgroundRefresh non déclenché pour le user connecté — régression")
		}
	})
}

// TestCareerLive_FetchLiveIdentity_ThirdPartyAppearance vérifie le chemin Explorer :
// pour un xuid TIERS sans ligne DB locale, FetchLiveIdentity fait un fetch live
// SYNCHRONE et construit l'identité depuis la customisation seule (career rank
// player-gated → progress nil ; appearance servie via la vue publique). Aucune
// persistance ne doit être déclenchée.
func TestCareerLive_FetchLiveIdentity_ThirdPartyAppearance(t *testing.T) {
	const targetXUID = "2535427927026623"

	fetcher := &mockCareerFetcher{
		progress: nil, // adversaire : /careerranks player-gated
		custom: &syncpkg.SpartanCustomizationData{
			SpartanID:        "MELG",
			EmblemImageURL:   "Inventory/Spartan/Emblems/104-001.json",
			BackdropImageURL: "Inventory/Spartan/BackdropImages/103-000.json",
		},
	}
	repo := &mockCareerLiveRepo{last: nil} // cible non suivie localement
	builder := &realBuilderForOverlay{}
	factory := func(_ context.Context) CareerFetcher { return fetcher }
	cache := NewCareerLiveCache(CareerLiveCacheConfig{})
	svc := NewCareerLiveService(repo, builder, factory, cache)

	ctx := ctxkeys.WithHaloAuth(context.Background(),
		&domain.HaloTokens{SpartanToken: "spartan-tok"}, "1111222233334444")

	got, err := svc.FetchLiveIdentity(ctx, targetXUID)
	if err != nil {
		t.Fatalf("FetchLiveIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("identité attendue non-nil (customisation live disponible)")
	}
	if got.SpartanID == nil || *got.SpartanID != "MELG" {
		t.Errorf("SpartanID = %v, attendu MELG", got.SpartanID)
	}
	if got.EmblemImageURL == nil {
		t.Error("EmblemImageURL attendu non-nil")
	}
	if got.BackdropImageURL == nil {
		t.Error("BackdropImageURL attendu non-nil")
	}
	if got.RankNumber != 0 {
		t.Errorf("RankNumber = %d, attendu 0 (career rank gated pour un adversaire)", got.RankNumber)
	}
	if fetcher.customCalls == 0 {
		t.Error("GetSpartanCustomization jamais appelé — fetch synchrone attendu")
	}
	// Garde-fou : aucune écriture player DB pour un xuid tiers.
	if len(repo.insertedPartials) > 0 || len(repo.inserted) > 0 {
		t.Errorf("persistance déclenchée (%d partials, %d full) — interdit pour xuid tiers",
			len(repo.insertedPartials), len(repo.inserted))
	}
}

// TestCareerLive_FetchLiveIdentity_NoAuth vérifie qu'en l'absence de tokens, on
// ne tente aucun fetch live et on sert l'identité DB locale si elle existe.
func TestCareerLive_FetchLiveIdentity_NoAuth(t *testing.T) {
	const targetXUID = "9999888877776666"
	dbRow := &duckdb.CareerRankRow{Rank: 25, CurrentXP: 300, SpartanID: "SR-DB"}

	fetcher := &mockCareerFetcher{custom: &syncpkg.SpartanCustomizationData{SpartanID: "SHOULD-NOT-FETCH"}}
	repo := &mockCareerLiveRepo{last: dbRow}
	builder := &realBuilderForOverlay{}
	factory := func(_ context.Context) CareerFetcher { return fetcher }
	cache := NewCareerLiveCache(CareerLiveCacheConfig{})
	svc := NewCareerLiveService(repo, builder, factory, cache)

	got, err := svc.FetchLiveIdentity(context.Background(), targetXUID) // pas de ctx auth
	if err != nil {
		t.Fatalf("FetchLiveIdentity: %v", err)
	}
	if got == nil || got.SpartanID == nil || *got.SpartanID != "SR-DB" {
		t.Fatalf("identité DB attendue (SR-DB), got=%v", got)
	}
	if fetcher.customCalls != 0 || fetcher.progCalls != 0 {
		t.Errorf("fetch live déclenché sans auth (custom=%d prog=%d) — interdit",
			fetcher.customCalls, fetcher.progCalls)
	}
}

// bgSignalingFetcher étend mockCareerFetcher avec un canal qui signale chaque
// appel — utilisé pour détecter si kickoffBackgroundRefresh a tourné.
type bgSignalingFetcher struct {
	progress *syncpkg.CareerRankData
	custom   *syncpkg.SpartanCustomizationData
	bgDone   chan struct{}
}

func (f *bgSignalingFetcher) GetCareerProgress(_ context.Context, _ string) (*syncpkg.CareerRankData, error) {
	select {
	case f.bgDone <- struct{}{}:
	default:
	}
	return f.progress, nil
}

func (f *bgSignalingFetcher) GetSpartanCustomization(_ context.Context, _ string) (*syncpkg.SpartanCustomizationData, error) {
	select {
	case f.bgDone <- struct{}{}:
	default:
	}
	return f.custom, nil
}

// nilSignalingFetcher retourne nil pour progress (simule un skip API silencieux)
// et signale via done quand il a été appelé.
type nilSignalingFetcher struct {
	done chan struct{}
}

func (f *nilSignalingFetcher) GetCareerProgress(_ context.Context, _ string) (*syncpkg.CareerRankData, error) {
	defer func() {
		select {
		case f.done <- struct{}{}:
		default:
		}
	}()
	return nil, nil // API silent skip
}

func (f *nilSignalingFetcher) GetSpartanCustomization(_ context.Context, _ string) (*syncpkg.SpartanCustomizationData, error) {
	return nil, nil
}
