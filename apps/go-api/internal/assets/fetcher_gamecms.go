package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"levelup/go-api/internal/domain"
)

const defaultGameCMSBase = "https://gamecms-hacs.svc.halowaypoint.com"

// GameCMSFetcher implémente Fetcher pour tous les assets GameCMS :
// images de médailles, badges de défis, images BP, métadonnées médailles,
// définitions de défis et de tracks Battle Pass.
type GameCMSFetcher struct {
	httpClient *http.Client
	tokens     TokenProvider
	baseURL    string // défaut : defaultGameCMSBase
}

// NewGameCMSFetcher crée un GameCMSFetcher.
// tokens peut être nil pour les assets publics (médailles, maps).
func NewGameCMSFetcher(client *http.Client, tokens TokenProvider, baseURL string) *GameCMSFetcher {
	if baseURL == "" {
		baseURL = defaultGameCMSBase
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &GameCMSFetcher{httpClient: client, tokens: tokens, baseURL: strings.TrimRight(baseURL, "/")}
}

// Supports retourne true pour les kinds servis par GameCMS.
func (f *GameCMSFetcher) Supports(k Kind) bool {
	switch k {
	case KindMedalImage, KindChallengeBadge, KindBPTrackImage, KindBPBackground,
		KindMedalMetadata, KindChallengeDefinition, KindRewardTrackDefinition:
		return true
	}
	return false
}

// Fetch récupère l'asset depuis GameCMS.
func (f *GameCMSFetcher) Fetch(ctx context.Context, ref Ref) (Payload, error) {
	switch ref.Kind {
	case KindMedalImage:
		return f.fetchMedalImage(ctx, ref)
	case KindChallengeBadge:
		return f.fetchChallengeBadge(ctx, ref)
	case KindBPTrackImage, KindBPBackground:
		return f.fetchBPImage(ctx, ref)
	case KindMedalMetadata:
		return f.fetchMedalMetadata(ctx, ref)
	case KindChallengeDefinition:
		return f.fetchChallengeDefinition(ctx, ref)
	case KindRewardTrackDefinition:
		return f.fetchRewardTrackDefinition(ctx, ref)
	}
	return nil, ErrUnsupportedKind
}

// fetchMedalImage récupère l'image d'une médaille.
// Retourne URLPayload (redirection) car les images sont publiques sur CDN.
// Le fallback spritesheet est géré par ChainFetcher.
func (f *GameCMSFetcher) fetchMedalImage(ctx context.Context, ref Ref) (Payload, error) {
	url := fmt.Sprintf("%s/hi/Progression/file/medals/%s/%s.png", f.baseURL, ref.TitleID, ref.ID)
	resp, err := f.doGet(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: medal image GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: medal image status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return BinaryPayload{
		ContentType: "image/png",
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

// fetchChallengeBadge récupère l'image d'un badge de défi.
// ref.ID contient le stem (ex: "combat/EnemiesKilledMelee").
func (f *GameCMSFetcher) fetchChallengeBadge(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	url := fmt.Sprintf("%s/hi/waypoint/file/images/%s.png", f.baseURL, ref.ID)
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: challenge badge GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: challenge badge status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return BinaryPayload{
		ContentType: "image/png",
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

// fetchBPImage récupère une image Battle Pass (track image ou background).
// ref.ID contient le chemin GameCMS complet (ex: "Progression/Seasons/S1/HIMPS1.png").
func (f *GameCMSFetcher) fetchBPImage(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	gamecmsPath := strings.TrimLeft(ref.ID, "/")
	url := fmt.Sprintf("%s/hi/images/file/%s", f.baseURL, gamecmsPath)
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: BP image GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: BP image status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return BinaryPayload{
		ContentType: "image/png",
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

// fetchMedalMetadata récupère le JSON des métadonnées de médailles.
func (f *GameCMSFetcher) fetchMedalMetadata(ctx context.Context, ref Ref) (Payload, error) {
	url := fmt.Sprintf("%s/hi/Progression/file/Metadata/Metadata.json", f.baseURL)
	resp, err := f.doGet(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: medal metadata GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: medal metadata status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return JSONPayload{RawJSON: data}, nil
}

// fetchChallengeDefinition récupère la définition JSON d'un défi.
// ref.ID contient le challenge_path.
func (f *GameCMSFetcher) fetchChallengeDefinition(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	challengePath := strings.TrimLeft(ref.ID, "/")
	url := fmt.Sprintf("%s/hi/Progression/file/%s", f.baseURL, challengePath)
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: challenge definition GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: challenge definition status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return JSONPayload{RawJSON: data}, nil
}

// fetchRewardTrackDefinition récupère la définition JSON d'un Reward Track.
// ref.ID contient le reward_track_path.
func (f *GameCMSFetcher) fetchRewardTrackDefinition(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	trackPath := strings.TrimLeft(ref.ID, "/")
	url := fmt.Sprintf("%s/hi/Progression/file/%s", f.baseURL, trackPath)
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: track definition GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: track definition status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return JSONPayload{RawJSON: data}, nil
}

// doGet exécute une requête GET avec les tokens Halo si fournis.
func (f *GameCMSFetcher) doGet(ctx context.Context, url string, tokens *domain.HaloTokens) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if tokens != nil {
		req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
		if tokens.ClearanceToken != "" {
			req.Header.Set("343-clearance", tokens.ClearanceToken)
		}
	}
	return f.httpClient.Do(req)
}

// resolveTokens appelle le TokenProvider si disponible.
func (f *GameCMSFetcher) resolveTokens(ctx context.Context) (*domain.HaloTokens, error) {
	if f.tokens == nil {
		return nil, nil
	}
	return f.tokens(ctx)
}
