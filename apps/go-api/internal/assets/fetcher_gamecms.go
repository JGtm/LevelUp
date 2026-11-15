package assets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/halo-api/infinite"
)

// surfaceGameCMS nomme la sortie des fichiers GameCMS dans le journal de netguard.
const surfaceGameCMS = "gamecms_assets.get"

// hoteGameCMSParDefaut : l'hôte GameCMS par défaut, celui des adresses de la brique.
func hoteGameCMSParDefaut() string {
	return infinite.DefaultEndpoints().Hote(infinite.CleGameCMS)
}

// gamecmsHostFor résout l'host gamecms pour le titre courant (MT-01 / PMT-1). Si
// le resolver partagé de boot est câblé, route par ctx slug ; sinon retombe sur
// `fallback` (override de construction / hôte par défaut de la brique).
func gamecmsHostFor(ctx context.Context, fallback string) string {
	if res := games.DefaultEndpointResolver(); res != nil {
		if host, ok := res.HostFor(ctxkeys.TitleSlug(ctx), games.EndpointGameCMS); ok {
			return strings.TrimRight(host, "/")
		}
	}
	return fallback
}

// gamecmsPrefixFor résout le segment d'URL de jeu du titre courant ("hi"/"h5")
// injecté dans les chemins GameCMS/Waypoint. Free function (miroir de
// gamecmsHostFor) consommée aussi par ChainFetcher ; fallback games.DefaultGamePrefix.
func gamecmsPrefixFor(ctx context.Context) string {
	return games.GamePrefix(ctxkeys.TitleSlug(ctx))
}

// GameCMSFetcher implémente Fetcher pour tous les assets GameCMS :
// images de médailles, badges de défis, images BP, métadonnées médailles,
// définitions de défis et de tracks Battle Pass. Le GET est celui de la brique
// (`infinite.Client.TelechargerFichierGameCMS`) ; le fetcher garde la résolution
// d'hôte par titre, le choix des jetons et le classement des erreurs.
type GameCMSFetcher struct {
	api     *infinite.Client
	tokens  TokenProvider
	baseURL string // défaut : hoteGameCMSParDefaut()
}

// NewGameCMSFetcher crée un GameCMSFetcher.
// tokens peut être nil pour les assets publics (médailles, maps). Le client de la
// brique émet par un netguard.Doer : en mode démo, aucune sortie tierce (emblèmes,
// bannières, images de rang) ; l'appelant remonte ErrUpstreamUnavailable → 502
// propre côté API, et le front affiche son placeholder.
func NewGameCMSFetcher(client *http.Client, tokens TokenProvider, baseURL string) *GameCMSFetcher {
	if baseURL == "" {
		baseURL = hoteGameCMSParDefaut()
	}
	if client == nil {
		client = http.DefaultClient
	}
	api := infinite.New(netguard.Doer{Surface: surfaceGameCMS, Next: client}, infinite.RetryAucune())
	return &GameCMSFetcher{api: api, tokens: tokens, baseURL: strings.TrimRight(baseURL, "/")}
}

// adresses rend les adresses GameCMS du titre courant (hôte et segment de jeu).
func (f *GameCMSFetcher) adresses(ctx context.Context) infinite.Endpoints {
	return infinite.DefaultEndpoints().
		Avec(infinite.CleGameCMS, gamecmsHostFor(ctx, f.baseURL)).
		AvecPrefixeJeu(gamecmsPrefixFor(ctx))
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

// fetchMedalImage récupère l'image d'une médaille, sans jeton.
// Le fallback spritesheet est géré par SpritesheetFallbackFetcher.
func (f *GameCMSFetcher) fetchMedalImage(ctx context.Context, ref Ref) (Payload, error) {
	url := infinite.URLFichierGameCMS(f.adresses(ctx), fmt.Sprintf("Progression/file/medals/%s/%s.png", ref.TitleID, ref.ID))
	data, err := f.telecharger(ctx, url, nil, "medal image", true)
	if err != nil {
		return nil, err
	}
	return BinaryPayload{ContentType: MimeImagePNG, Bytes: data, ETag: contentHash(data)}, nil
}

// fetchChallengeBadge récupère l'image d'un badge de défi.
// ref.ID contient le stem (ex: "combat/EnemiesKilledMelee").
func (f *GameCMSFetcher) fetchChallengeBadge(ctx context.Context, ref Ref) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	url := infinite.URLFichierGameCMS(f.adresses(ctx), "waypoint/file/images/"+ref.ID+".png")
	data, err := f.telecharger(ctx, url, tokens, "challenge badge", true)
	if err != nil {
		return nil, err
	}
	return BinaryPayload{ContentType: MimeImagePNG, Bytes: data, ETag: contentHash(data)}, nil
}

// fetchGameCMSImage récupère une image binaire via GameCMS / Waypoint.
// ref.ID peut contenir un chemin relatif (ex: "Progression/..."), un chemin
// déjà préfixé (ex: "hi/images/file/...", "hi/Waypoint/file/..."), ou une
// URL absolue déjà résolue (voir infinite.URLImageGameCMS).
func (f *GameCMSFetcher) fetchGameCMSImage(ctx context.Context, ref Ref) (Payload, error) {
	// Les images sont généralement publiques ; on envoie les tokens si disponibles,
	// sans échouer si le provider n'en a pas sous la main.
	tokens, _ := f.resolveTokens(ctx)
	url := infinite.URLImageGameCMS(f.adresses(ctx), ref.ID)
	if url == "" {
		return nil, ErrNotFound
	}
	data, err := f.telecharger(ctx, url, tokens, "image", true)
	if err != nil {
		return nil, err
	}
	return BinaryPayload{ContentType: http.DetectContentType(data), Bytes: data, ETag: contentHash(data)}, nil
}

// fetchMedalMetadata récupère le JSON des métadonnées de médailles, sans jeton. Un 404
// n'y est pas un ErrNotFound : tout non-200 est ErrUpstreamUnavailable.
func (f *GameCMSFetcher) fetchMedalMetadata(ctx context.Context, _ Ref) (Payload, error) {
	url := infinite.URLFichierGameCMS(f.adresses(ctx), "Progression/file/Metadata/Metadata.json")
	data, err := f.telecharger(ctx, url, nil, "medal metadata", false)
	if err != nil {
		return nil, err
	}
	return JSONPayload{RawJSON: data}, nil
}

// fetchChallengeDefinition récupère la définition JSON d'un défi.
// ref.ID contient le challenge_path.
func (f *GameCMSFetcher) fetchChallengeDefinition(ctx context.Context, ref Ref) (Payload, error) {
	return f.fetchProgressionJSON(ctx, ref, "challenge definition")
}

// fetchRewardTrackDefinition récupère la définition JSON d'un Reward Track.
// ref.ID contient le reward_track_path.
func (f *GameCMSFetcher) fetchRewardTrackDefinition(ctx context.Context, ref Ref) (Payload, error) {
	return f.fetchProgressionJSON(ctx, ref, "track definition")
}

// fetchBPItemDefinition récupère la définition JSON d'un item inventaire Battle Pass.
// ref.ID contient l'InventoryItemPath (ex: "Inventory/Armor/Coatings/...").
func (f *GameCMSFetcher) fetchBPItemDefinition(ctx context.Context, ref Ref) (Payload, error) {
	return f.fetchProgressionJSON(ctx, ref, "bp item definition")
}

// fetchProgressionJSON récupère un document JSON authentifié sous `Progression/file/`
// (barres de tête de ref.ID retirées). Les jetons sont obligatoires ; `libelle` nomme
// le document dans les erreurs.
func (f *GameCMSFetcher) fetchProgressionJSON(ctx context.Context, ref Ref, libelle string) (Payload, error) {
	tokens, err := f.resolveTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: tokens unavailable: %v", ErrUpstreamUnavailable, err)
	}
	url := infinite.URLFichierGameCMS(f.adresses(ctx), "Progression/file/"+strings.TrimLeft(ref.ID, "/"))
	data, err := f.telecharger(ctx, url, tokens, libelle, true)
	if err != nil {
		return nil, err
	}
	return JSONPayload{RawJSON: data}, nil
}

// telecharger exécute le GET de la brique et classe son issue : 404 → ErrNotFound si
// `avec404`, autre non-200 → ErrUpstreamUnavailable « `libelle` status N », corps coupé →
// « read body », échec d'émission (refus netguard compris) → « `libelle` GET ».
func (f *GameCMSFetcher) telecharger(ctx context.Context, url string, tokens *domain.HaloTokens, libelle string, avec404 bool) ([]byte, error) {
	data, err := f.api.TelechargerFichierGameCMS(ctx, authDe(tokens), url)
	if err == nil {
		return data, nil
	}
	var he *infinite.HTTPError
	if errors.As(err, &he) {
		if avec404 && he.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: %s status %d", ErrUpstreamUnavailable, libelle, he.StatusCode)
	}
	var le *infinite.ErreurLecture
	if errors.As(err, &le) {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, le.Err)
	}
	return nil, fmt.Errorf("%w: %s GET: %v", ErrUpstreamUnavailable, libelle, err)
}

// authDe convertit les jetons Halo en en-têtes de la brique (nil : aucun en-tête).
func authDe(t *domain.HaloTokens) infinite.Auth {
	if t == nil {
		return infinite.Auth{}
	}
	return infinite.Auth{Spartan: t.SpartanToken, Clearance: t.ClearanceToken}
}

// resolveTokens appelle le TokenProvider si disponible.
func (f *GameCMSFetcher) resolveTokens(ctx context.Context) (*domain.HaloTokens, error) {
	if f.tokens == nil {
		return nil, nil
	}
	return f.tokens(ctx)
}
