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
	"errors"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

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

type mockIdentityBuilder struct {
	receivedRow *duckdb.CareerRankRow
	returnNil   bool
}

func (m *mockIdentityBuilder) BuildSpartanIdentityFromCareerRow(_ context.Context, row *duckdb.CareerRankRow) *domain.HomeSpartanIdentityRow {
	m.receivedRow = row
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
	if row.EmblemImageURL != "" {
		e := row.EmblemImageURL
		id.EmblemImageURL = &e
	}
	return id
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

func newService(t *testing.T, fetcher *mockCareerFetcher, repo *mockCareerLiveRepo, builder *mockIdentityBuilder) *CareerLiveService {
	t.Helper()
	factory := func(_ context.Context) CareerFetcher {
		if fetcher == nil {
			return nil
		}
		return fetcher
	}
	return NewCareerLiveService(repo, builder, factory, nil) // pas de cache pour les tests
}

// --- TestCareerLive_GetIdentity_FallbackMatrix ---
//
// Matrice de fallback : c'est le test qui acte le contrat utilisateur final
// "le bloc Spartan ne doit jamais être vide si la DB porte une row".

func TestCareerLive_GetIdentity_FallbackMatrix(t *testing.T) {
	dbRowFull := &duckdb.CareerRankRow{
		Rank: 30, CurrentXP: 500, SpartanID: "SR-DB-001",
		BannerImageURL: "https://db/banner.png", EmblemImageURL: "https://db/emblem.png",
		BackdropImageURL: "https://db/backdrop.png",
	}

	cases := []struct {
		name              string
		progress          *syncpkg.CareerRankData
		progressErr       error
		custom            *syncpkg.SpartanCustomizationData
		customErr         error
		dbRow             *duckdb.CareerRankRow
		hasTokens         bool
		wantIdentity      bool
		wantSpartanID     string
		wantEmblemURL     string
		wantRank          int
		wantInsertCalled  bool
	}{
		{
			name:             "live OK full",
			progress:         &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			custom:           &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99", EmblemImageURL: "https://live/emblem.png"},
			dbRow:            dbRowFull,
			hasTokens:        true,
			wantIdentity:     true,
			wantSpartanID:    "SR-LIVE-99",
			wantEmblemURL:    "https://live/emblem.png",
			wantRank:         42,
			wantInsertCalled: true,
		},
		{
			name:             "live customization partielle → per-field merge",
			progress:         &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			custom:           &syncpkg.SpartanCustomizationData{SpartanID: "SR-LIVE-99"}, // emblem vide
			dbRow:            dbRowFull,
			hasTokens:        true,
			wantIdentity:     true,
			wantSpartanID:    "SR-LIVE-99",
			wantEmblemURL:    "https://db/emblem.png", // carry-forward depuis DB
			wantRank:         42,
			wantInsertCalled: true,
		},
		{
			name:             "live customization KO → carry-forward complet",
			progress:         &syncpkg.CareerRankData{CurrentRank: 42, CurrentXP: 1500},
			customErr:        errors.New("customization API 500"),
			dbRow:            dbRowFull,
			hasTokens:        true,
			wantIdentity:     true,
			wantSpartanID:    "SR-DB-001",
			wantEmblemURL:    "https://db/emblem.png",
			wantRank:         42,
			wantInsertCalled: true,
		},
		{
			name:             "live progress KO + custom KO → fallback DB row",
			progressErr:      errors.New("progress API 500"),
			customErr:        errors.New("custom API 500"),
			dbRow:            dbRowFull,
			hasTokens:        true,
			wantIdentity:     true,
			wantSpartanID:    "SR-DB-001",
			wantEmblemURL:    "https://db/emblem.png",
			wantRank:         30,
			wantInsertCalled: true, // INSERT-if-changed sera appelé mais skipera (mock ne le sait pas)
		},
		{
			name:             "pas de tokens → fallback DB direct",
			dbRow:            dbRowFull,
			hasTokens:        false,
			wantIdentity:     true,
			wantSpartanID:    "SR-DB-001",
			wantEmblemURL:    "https://db/emblem.png",
			wantRank:         30,
			wantInsertCalled: true, // path tokens=present mais factory→nil fetcher: même chemin merge
		},
		{
			name:             "live KO + DB vide → nil (joueur jamais sync'd)",
			progressErr:      errors.New("progress API 500"),
			customErr:        errors.New("custom API 500"),
			dbRow:            nil,
			hasTokens:        true,
			wantIdentity:     false,
			wantInsertCalled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := &mockCareerFetcher{
				progress:    tc.progress,
				progressErr: tc.progressErr,
				custom:      tc.custom,
				customErr:   tc.customErr,
			}
			repo := &mockCareerLiveRepo{last: tc.dbRow}
			builder := &mockIdentityBuilder{}
			svc := newService(t, fetcher, repo, builder)

			got, err := svc.GetSpartanIdentity(ctxWithTokens(t, tc.hasTokens))
			if err != nil {
				t.Fatalf("GetSpartanIdentity: erreur inattendue %v", err)
			}

			if !tc.wantIdentity {
				if got != nil {
					t.Errorf("identity attendue nil, obtenue %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("identity attendue non-nil, obtenue nil")
			}
			if got.RankNumber != tc.wantRank {
				t.Errorf("rank = %d, want %d", got.RankNumber, tc.wantRank)
			}
			if tc.wantSpartanID != "" && (got.SpartanID == nil || *got.SpartanID != tc.wantSpartanID) {
				t.Errorf("spartan_id = %v, want %q", got.SpartanID, tc.wantSpartanID)
			}
			if tc.wantEmblemURL != "" && (got.EmblemImageURL == nil || *got.EmblemImageURL != tc.wantEmblemURL) {
				t.Errorf("emblem_image_url = %v, want %q", got.EmblemImageURL, tc.wantEmblemURL)
			}
			if tc.wantInsertCalled && len(repo.inserted) == 0 {
				t.Errorf("InsertCareerProgressionIfChanged attendu appelé, %d insertions", len(repo.inserted))
			}
		})
	}
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
