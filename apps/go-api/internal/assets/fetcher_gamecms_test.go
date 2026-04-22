package assets

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"levelup/go-api/internal/domain"
)

// stubTokenProvider retourne des tokens de test.
func stubTokens(_ context.Context) (*domain.HaloTokens, error) {
	return &domain.HaloTokens{SpartanToken: "spartan-test", ClearanceToken: "clearance-test"}, nil
}

func failingTokens(_ context.Context) (*domain.HaloTokens, error) {
	return nil, errors.New("auth failed")
}

func TestGameCMSFetcher_Supports(t *testing.T) {
	f := NewGameCMSFetcher(nil, nil, "")
	supported := []Kind{
		KindMedalImage, KindChallengeBadge, KindBPTrackImage, KindBPBackground,
		KindSpartanEmblem, KindSpartanBanner, KindSpartanBackdrop, KindCareerRankImage,
		KindMedalMetadata, KindChallengeDefinition, KindRewardTrackDefinition,
	}
	for _, k := range supported {
		if !f.Supports(k) {
			t.Errorf("Supports(%q) devrait être true", k)
		}
	}
	if f.Supports(KindMapImage) {
		t.Error("Supports(KindMapImage) devrait être false (non géré par GameCMS)")
	}
	if f.Supports(KindAssetTranslation) {
		t.Error("Supports(KindAssetTranslation) devrait être false")
	}
}

func TestGameCMSFetcher_Fetch_UnsupportedKind(t *testing.T) {
	f := NewGameCMSFetcher(nil, nil, "")
	ref := Ref{Kind: KindMapImage, TitleID: "hi", ID: "LiveFire"}
	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Errorf("attendu ErrUnsupportedKind, got %v", err)
	}
}

func TestGameCMSFetcher_MedalImage_OK(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), nil, srv.URL)
	ref := Ref{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "42"}

	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	bin, ok := res.(BinaryPayload)
	if !ok {
		t.Fatal("attendu BinaryPayload")
	}
	if bin.ContentType != "image/png" {
		t.Errorf("ContentType: got %q, want image/png", bin.ContentType)
	}
	if len(bin.Bytes) != len(pngBytes) {
		t.Errorf("bytes len: got %d, want %d", len(bin.Bytes), len(pngBytes))
	}
	if bin.ETag == "" {
		t.Error("ETag ne devrait pas être vide")
	}
}

func TestGameCMSFetcher_MedalImage_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), nil, srv.URL)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "missing"}

	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, got %v", err)
	}
}

func TestGameCMSFetcher_MedalImage_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), nil, srv.URL)
	ref := Ref{Kind: KindMedalImage, TitleID: "hi", ID: "err"}

	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("attendu ErrUpstreamUnavailable, got %v", err)
	}
}

func TestGameCMSFetcher_ChallengeBadge_OK(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vérifier que les headers spartan sont envoyés.
		if r.Header.Get("x-343-authorization-spartan") != "spartan-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindChallengeBadge, TitleID: "halo_infinite", ID: "combat/EnemiesKilledMelee"}

	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if _, ok := res.(BinaryPayload); !ok {
		t.Error("attendu BinaryPayload")
	}
}

func TestGameCMSFetcher_ChallengeBadge_TokenError(t *testing.T) {
	f := NewGameCMSFetcher(nil, failingTokens, "")
	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "badge"}

	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Errorf("attendu ErrUpstreamUnavailable, got %v", err)
	}
}

func TestGameCMSFetcher_MedalMetadata_OK(t *testing.T) {
	jsonData := []byte(`{"Medals":[]}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonData)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), nil, srv.URL)
	ref := Ref{Kind: KindMedalMetadata, TitleID: "hi", ID: "metadata"}

	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	jp, ok := res.(JSONPayload)
	if !ok {
		t.Fatal("attendu JSONPayload")
	}
	if string(jp.RawJSON) != string(jsonData) {
		t.Errorf("RawJSON inattendu: %q", jp.RawJSON)
	}
}

func TestGameCMSFetcher_BPImage_OK(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindBPTrackImage, TitleID: "hi", ID: "Progression/Seasons/S1/cover.png"}

	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if _, ok := res.(BinaryPayload); !ok {
		t.Error("attendu BinaryPayload")
	}
}

func TestGameCMSFetcher_BPImage_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindBPBackground, TitleID: "hi", ID: "missing/bg.png"}

	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, got %v", err)
	}
}

func TestGameCMSFetcher_SpartanIdentityImages_OK(t *testing.T) {
	cases := []struct {
		name     string
		ref      Ref
		wantPath string
	}{
		{
			name:     "emblem waypoint path preserved",
			ref:      Ref{Kind: KindSpartanEmblem, TitleID: "halo_infinite", ID: "hi/Waypoint/file/images/emblems/test_123.png"},
			wantPath: "/hi/Waypoint/file/images/emblems/test_123.png",
		},
		{
			name:     "career rank defaults to images file",
			ref:      Ref{Kind: KindCareerRankImage, TitleID: "halo_infinite", ID: "Progression/RewardTracks/CareerRanks/platinum1-large.png"},
			wantPath: "/hi/images/file/Progression/RewardTracks/CareerRanks/platinum1-large.png",
		},
		{
			name:     "banner path preserved",
			ref:      Ref{Kind: KindSpartanBanner, TitleID: "halo_infinite", ID: "hi/images/file/progression/Nameplates/test-banner.png"},
			wantPath: "/hi/images/file/progression/Nameplates/test-banner.png",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d})
			}))
			defer srv.Close()

			f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
			res, err := f.Fetch(context.Background(), tc.ref)
			if err != nil {
				t.Fatalf("erreur inattendue: %v", err)
			}
			if _, ok := res.(BinaryPayload); !ok {
				t.Fatal("attendu BinaryPayload")
			}
			if gotPath != tc.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

func TestGameCMSFetcher_ChallengeDefinition_OK(t *testing.T) {
	jsonData := []byte(`{"Path":"Weekly/Challenges/week1.json"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(jsonData)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindChallengeDefinition, TitleID: "hi", ID: "Weekly/Challenges/week1.json"}

	res, err := f.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("erreur inattendue: %v", err)
	}
	if _, ok := res.(JSONPayload); !ok {
		t.Error("attendu JSONPayload")
	}
}

func TestGameCMSFetcher_RewardTrackDefinition_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindRewardTrackDefinition, TitleID: "hi", ID: "RewardTracks/Operations/missing.json"}

	_, err := f.Fetch(context.Background(), ref)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, got %v", err)
	}
}

func TestGameCMSFetcher_DoGet_SetsHeaders(t *testing.T) {
	var gotSpartan, gotClearance string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSpartan = r.Header.Get("x-343-authorization-spartan")
		gotClearance = r.Header.Get("343-clearance")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	}))
	defer srv.Close()

	f := NewGameCMSFetcher(srv.Client(), stubTokens, srv.URL)
	ref := Ref{Kind: KindChallengeBadge, TitleID: "hi", ID: "combat/test"}
	_, _ = f.Fetch(context.Background(), ref)

	if gotSpartan != "spartan-test" {
		t.Errorf("x-343-authorization-spartan: got %q, want %q", gotSpartan, "spartan-test")
	}
	if gotClearance != "clearance-test" {
		t.Errorf("343-clearance: got %q, want %q", gotClearance, "clearance-test")
	}
}
