package assets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/netguard"
)

const defaultGameCMSBase = "https://gamecms-hacs.svc.halowaypoint.com"

// gamecmsHostFor résout l'host gamecms pour le titre courant (MT-01 / PMT-1). Si
// le resolver partagé de boot est câblé, route par ctx slug ; sinon retombe sur
// `fallback` (override de construction / const Halo, byte-identique).
func gamecmsHostFor(ctx context.Context, fallback string) string {
	if res := games.DefaultEndpointResolver(); res != nil {
		if host, ok := res.HostFor(ctxkeys.TitleSlug(ctx), games.EndpointGameCMS); ok {
			return strings.TrimRight(host, "/")
		}
	}
	return fallback
}

// base résout l'host gamecms du fetcher (override f.baseURL → resolver title-aware).
func (f *GameCMSFetcher) base(ctx context.Context) string {
	return gamecmsHostFor(ctx, f.baseURL)
}

// gamecmsPrefixFor résout le segment d'URL de jeu du titre courant ("hi"/"h5")
// injecté dans les chemins GameCMS/Waypoint. Free function (miroir de
// gamecmsHostFor) consommée aussi par ChainFetcher ; fallback games.DefaultGamePrefix.
func gamecmsPrefixFor(ctx context.Context) string {
	return games.GamePrefix(ctxkeys.TitleSlug(ctx))
}

// gamePrefix résout le préfixe de jeu du fetcher (title-aware via ctx).
func (f *GameCMSFetcher) gamePrefix(ctx context.Context) string {
	return gamecmsPrefixFor(ctx)
}

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
	case KindMedalImage,
		KindChallengeBadge,
		KindBPTrackImage,
		KindBPBackground,
		KindSpartanEmblem,
		KindSpartanBanner,
		KindSpartanBackdrop,
		KindCareerRankImage,
		KindMedalMetadata, KindChallengeDefinition, KindRewardTrackDefinition, KindBPItemDefinition:
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
	case KindBPTrackImage, KindBPBackground, KindSpartanEmblem, KindSpartanBanner, KindSpartanBackdrop, KindCareerRankImage:
		return f.fetchGameCMSImage(ctx, ref)
	case KindMedalMetadata:
		return f.fetchMedalMetadata(ctx, ref)
	case KindChallengeDefinition:
		return f.fetchChallengeDefinition(ctx, ref)
	case KindRewardTrackDefinition:
		return f.fetchRewardTrackDefinition(ctx, ref)
	case KindBPItemDefinition:
		return f.fetchBPItemDefinition(ctx, ref)
	}
	return nil, ErrUnsupportedKind
}

// fetchMedalImage récupère l'image d'une médaille.
// Retourne URLPayload (redirection) car les images sont publiques sur CDN.
// Le fallback spritesheet est géré par ChainFetcher.
func (f *GameCMSFetcher) fetchMedalImage(ctx context.Context, ref Ref) (Payload, error) {
	url := fmt.Sprintf("%s/%s/Progression/file/medals/%s/%s.png", f.base(ctx), f.gamePrefix(ctx), ref.TitleID, ref.ID)
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
		ContentType: MimeImagePNG,
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
	url := fmt.Sprintf("%s/%s/waypoint/file/images/%s.png", f.base(ctx), f.gamePrefix(ctx), ref.ID)
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
		ContentType: MimeImagePNG,
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

// fetchGameCMSImage récupère une image binaire via GameCMS / Waypoint.
// ref.ID peut contenir un chemin relatif (ex: "Progression/..."), un chemin
// déjà préfixé (ex: "hi/images/file/...", "hi/Waypoint/file/..."), ou une
// URL absolue déjà résolue.
func (f *GameCMSFetcher) fetchGameCMSImage(ctx context.Context, ref Ref) (Payload, error) {
	// Les images sont généralement publiques ; on envoie les tokens si disponibles,
	// sans échouer si le provider n'en a pas sous la main.
	tokens, _ := f.resolveTokens(ctx)
	url := buildGameCMSImageFetchURL(f.base(ctx), f.gamePrefix(ctx), ref.ID)
	if url == "" {
		return nil, ErrNotFound
	}
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: image GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: image status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	ct := http.DetectContentType(data)
	return BinaryPayload{
		ContentType: ct,
		Bytes:       data,
		ETag:        contentHash(data),
	}, nil
}

func buildGameCMSImageFetchURL(baseURL, gamePrefix, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasSuffix(lower, ".json") {
		return ""
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return trimmed
	}

	cleaned := strings.TrimLeft(trimmed, "/")
	lowerCleaned := strings.ToLower(cleaned)
	p := strings.ToLower(gamePrefix)
	switch {
	case strings.HasPrefix(lowerCleaned, p+"/images/file/"),
		strings.HasPrefix(lowerCleaned, p+"/progression/file/"),
		strings.HasPrefix(lowerCleaned, p+"/waypoint/file/"):
		return fmt.Sprintf("%s/%s", baseURL, cleaned)
	case strings.HasPrefix(lowerCleaned, "images/file/"),
		strings.HasPrefix(lowerCleaned, "progression/file/"),
		strings.HasPrefix(lowerCleaned, "waypoint/file/"):
		return fmt.Sprintf("%s/%s/%s", baseURL, gamePrefix, cleaned)
	default:
		return fmt.Sprintf("%s/%s/images/file/%s", baseURL, gamePrefix, cleaned)
	}
}

// fetchMedalMetadata récupère le JSON des métadonnées de médailles.
func (f *GameCMSFetcher) fetchMedalMetadata(ctx context.Context, _ Ref) (Payload, error) {
	url := fmt.Sprintf("%s/%s/Progression/file/Metadata/Metadata.json", f.base(ctx), f.gamePrefix(ctx))
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
	url := fmt.Sprintf("%s/%s/Progression/file/%s", f.base(ctx), f.gamePrefix(ctx), challengePath)
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
	url := fmt.Sprintf("%s/%s/Progression/file/%s", f.base(ctx), f.gamePrefix(ctx), trackPath)
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

// fetchBPItemDefinition récupère la définition JSON d'un item inventaire Battle Pass.
// ref.ID contient l'InventoryItemPath (ex: "Inventory/Armor/Coatings/...").
func (f *GameCMSFetcher) fetchBPItemDefinition(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	itemPath := strings.TrimLeft(ref.ID, "/")
	url := fmt.Sprintf("%s/%s/Progression/file/%s", f.base(ctx), f.gamePrefix(ctx), itemPath)
	resp, err := f.doGet(ctx, url, tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: bp item definition GET: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: bp item definition status %d", ErrUpstreamUnavailable, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}
	return JSONPayload{RawJSON: data}, nil
}

// doGet exécute une requête GET avec les tokens Halo si fournis.
func (f *GameCMSFetcher) doGet(ctx context.Context, url string, tokens *domain.HaloTokens) (*http.Response, error) {
	// Mode démo : aucune sortie tierce (emblèmes, bannières, images de rang).
	// L'appelant remonte ErrUpstreamUnavailable → 502 propre côté API, et le
	// front affiche son placeholder. Sans ce garde, la démo téléchargeait le
	// catalogue d'images de rang carrière depuis gamecms à chaque démarrage.
	if err := netguard.Check(ctx, "gamecms_assets.get"); err != nil {
		return nil, err
	}
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
