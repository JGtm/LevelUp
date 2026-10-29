// Package auth — xbl_exchange.go : façade d'exploitation de l'échange access_token Microsoft ->
// jetons Halo (le protocole vit dans `internal/haloapi/xblauth`).
//
// Ce que la façade ajoute au protocole : le descripteur Halo par défaut, le client HTTP et son
// délai, le créneau de concurrence XBL process-wide, la provenance du jeton portée par le `ctx`
// (posée par le refresh / le pool, mesurée par l'échange) et la conversion des jetons de la brique
// en `domain.HaloTokens`.
package auth

import (
	"context"
	"net/http"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/haloapi"
	"levelup/go-api/internal/haloapi/xblauth"
)

// xblExchangeTimeout borne chaque appel HTTP de la chaîne d'échange (échange Halo, XSTS RTA,
// device-flow SISU).
const xblExchangeTimeout = 20 * time.Second

// xblUserAuthMaxConcurrent borne le nombre d'appels SIMULTANÉS vers user.auth.xboxlive.com
// (endpoint XBL user-token). Xbox Live throttle cet endpoint sur la CONCURRENCE (429
// « currentRequests/maxRequests »). Sans borne, le refresher du pool (jusqu'à N refresh parallèles
// après un cooldown) ET le chemin user-facing (refreshTokensFromDB) tapaient l'endpoint en même
// temps -> 429 auto-infligé. Un sémaphore process-wide sérialise TOUS les appelants de l'échange
// XBL ; la brique le reçoit par `xblauth.Chaine.Creneau`.
const xblUserAuthMaxConcurrent = 2

// xblUserAuthSem est le sémaphore process-wide gardant les appels XBL user-token.
var xblUserAuthSem = make(chan struct{}, xblUserAuthMaxConcurrent)

// acquireXBLUserAuthSlot prend un slot du sémaphore XBL user-token (bloquant, mais annulable par
// ctx). Retourne la fonction de libération (à defer).
func acquireXBLUserAuthSlot(ctx context.Context) (func(), error) {
	select {
	case xblUserAuthSem <- struct{}{}:
		return func() { <-xblUserAuthSem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExchangeResult regroupe les tokens Halo ET l'identité extraite de la réponse XSTS.
type ExchangeResult struct {
	Tokens   *domain.HaloTokens
	Gamertag string // DisplayClaims.xui[0].gtg
	XUID     string // DisplayClaims.xui[0].xid
}

// ExchangeAccessToken échange un access_token Microsoft contre des tokens Halo Infinite.
// Implémente la chaîne : access_token -> XBL user -> XSTS Halo -> Spartan -> Clearance.
// Retourne aussi le gamertag et le XUID extraits de la réponse XSTS.
func ExchangeAccessToken(ctx context.Context, accessToken string) (*ExchangeResult, error) {
	return ExchangeAccessTokenWithDescriptor(ctx, accessToken, title.DefaultHaloAuthDescriptor())
}

// ExchangeAccessTokenWithDescriptor échange un access_token Microsoft contre des tokens d'un titre
// donné (MT-02). Le descripteur porte l'audience XSTS, l'audience + endpoint spartan, et
// l'endpoint clearance. ExchangeAccessToken délègue avec le défaut Halo -> byte-identique.
func ExchangeAccessTokenWithDescriptor(ctx context.Context, accessToken string, d title.AuthDescriptor) (*ExchangeResult, error) {
	return exchangeAccessTokenWithClient(ctx, &http.Client{Timeout: xblExchangeTimeout}, accessToken, d)
}

// exchangeAccessTokenWithClient exécute la chaîne avec le Doer fourni (injectable en test). Le
// préfixe RpsTicket suit la provenance posée en ctx (WithTokenClientFamily) ; la provenance
// mesurée est rapportée à l'observateur du ctx (WithTokenFamilyObserver).
func exchangeAccessTokenWithClient(ctx context.Context, doer haloapi.Doer, accessToken string, d title.AuthDescriptor) (*ExchangeResult, error) {
	res, err := xblauth.ExchangeAccessToken(ctx, chaineDepuisCtx(ctx, doer), accessToken, descripteurXBL(d))
	if err != nil {
		return nil, err
	}
	return &ExchangeResult{Tokens: haloTokensDepuisXBL(res.Tokens), Gamertag: res.Gamertag, XUID: res.XUID}, nil
}

// ExchangeXSTSForHaloTokens échange un XSTS Token Halo déjà obtenu contre un Spartan Token
// et un Clearance Token. Utile pour les CLIs batch qui chargent le token depuis tokens.json
// sans refaire la chaîne OAuth complète.
func ExchangeXSTSForHaloTokens(ctx context.Context, xstsToken string) (*domain.HaloTokens, error) {
	return ExchangeXSTSForHaloTokensWithDescriptor(ctx, xstsToken, title.DefaultHaloAuthDescriptor())
}

// ExchangeXSTSForHaloTokensWithDescriptor échange un XSTS Token contre Spartan + Clearance pour
// un titre donné (MT-02). ExchangeXSTSForHaloTokens délègue Halo.
func ExchangeXSTSForHaloTokensWithDescriptor(ctx context.Context, xstsToken string, d title.AuthDescriptor) (*domain.HaloTokens, error) {
	toks, err := xblauth.ExchangeXSTSForTokens(ctx, &http.Client{Timeout: xblExchangeTimeout}, xstsToken, descripteurXBL(d))
	if err != nil {
		return nil, err
	}
	return haloTokensDepuisXBL(toks), nil
}

// chaineDepuisCtx assemble ce que la brique reçoit pour un échange XBL : le Doer, le créneau
// process-wide, la provenance connue (ctx) et l'observateur de la provenance mesurée (ctx).
func chaineDepuisCtx(ctx context.Context, doer haloapi.Doer) xblauth.Chaine {
	return xblauth.Chaine{
		Doer:    doer,
		Creneau: acquireXBLUserAuthSlot,
		Famille: TokenClientFamilyFromContext(ctx),
		Observe: func(family string) { RecordObservedTokenFamily(ctx, family) },
	}
}

// descripteurXBL rend la part du descripteur de titre que la chaîne d'échange lit.
func descripteurXBL(d title.AuthDescriptor) xblauth.Descriptor {
	return xblauth.Descriptor{
		XSTSAudience:    d.XSTSAudience,
		SpartanAudience: d.SpartanAudience,
		SpartanTokenURL: d.SpartanTokenURL,
		ClearanceURL:    d.ClearanceURL,
	}
}

// haloTokensDepuisXBL convertit les jetons de la brique en `domain.HaloTokens` (mêmes champs :
// TestXblauthTokens_PariteAvecDomainHaloTokens). nil rend nil.
func haloTokensDepuisXBL(t *xblauth.Tokens) *domain.HaloTokens {
	if t == nil {
		return nil
	}
	return &domain.HaloTokens{
		SpartanToken:     t.SpartanToken,
		ClearanceToken:   t.ClearanceToken,
		SpartanExpiresAt: t.SpartanExpiresAt,
	}
}

// =============================================================================
// Provenance du jeton le long de l'échange (AU4/F12)
// =============================================================================

// tokenClientFamilyCtxKey porte la provenance du token (AU4/F12) le long de la chaîne d'échange,
// SANS changer les signatures publiques (même patron que WithHaloAuth). Le caller qui connaît la
// famille (refresh) la pose ; la façade la passe à la brique, qui choisit le préfixe RpsTicket.
type tokenClientFamilyCtxKey struct{}

// WithTokenClientFamily attache la famille de client OAuth (TokenFamilyAzure /
// TokenFamilyXboxNative) au contexte. Une famille vide laisse le ctx inchangé.
// Exportée : le pool de tokens (paquet auth/pool) pose la famille persistée du
// joueur avant l'échange.
func WithTokenClientFamily(ctx context.Context, family string) context.Context {
	if family == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenClientFamilyCtxKey{}, family)
}

// TokenClientFamilyFromContext lit la famille posée par WithTokenClientFamily
// ("" = provenance inconnue).
func TokenClientFamilyFromContext(ctx context.Context) string {
	f, _ := ctx.Value(tokenClientFamilyCtxKey{}).(string)
	return f
}

// tokenFamilyObserverCtxKey porte l'observateur de provenance MESURÉE.
type tokenFamilyObserverCtxKey struct{}

// TokenFamilyObserver recueille la provenance RÉELLEMENT acceptée par l'endpoint
// XBL user-token : le préfixe RpsTicket qui a rendu un token.
//
// POURQUOI UNE MESURE ET PAS LA FAMILLE DU CLIENT OAUTH (constat du 2026-09-20).
// La famille déduite du endpoint de refresh (Azure vs MSA natif) NE PRÉDIT PAS le
// préfixe accepté : sur les 13 comptes du poste, les 13 refresh passent par l'app
// Azure (zéro repli MSA dans 87 Mo de auth.log) et pourtant 5 d'entre eux se font
// refuser « d= » en 401 et ne passent qu'en « t= ». Persister la famille du client
// aurait donc reconduit le mauvais préfixe. La seule donnée fiable est le résultat
// de l'échange précédent — d'où cet observateur : la brique rapporte ce qui a
// marché, le caller le persiste, l'échange suivant commence par là.
type TokenFamilyObserver struct {
	mu     sync.Mutex
	family string
}

// Observed rend la famille mesurée au dernier échange XBL user-token
// ("" si aucun échange n'a eu lieu — cache, échec amont).
func (o *TokenFamilyObserver) Observed() string {
	if o == nil {
		return ""
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.family
}

func (o *TokenFamilyObserver) record(family string) {
	if o == nil || family == "" {
		return
	}
	o.mu.Lock()
	o.family = family
	o.mu.Unlock()
}

// WithTokenFamilyObserver attache un observateur de provenance au contexte et le
// rend. Le caller lit Observed() APRÈS l'échange et persiste la valeur si elle
// diffère de celle qu'il connaissait.
func WithTokenFamilyObserver(ctx context.Context) (context.Context, *TokenFamilyObserver) {
	o := &TokenFamilyObserver{}
	return context.WithValue(ctx, tokenFamilyObserverCtxKey{}, o), o
}

// RecordObservedTokenFamily écrit la provenance mesurée dans l'observateur du ctx
// (no-op si le caller n'en a pas installé). Appelée par l'échange XBL (via
// `xblauth.Chaine.Observe`) ; exportée pour qu'un échangeur substitué (double de
// test du pool, futur fournisseur) renseigne la même mesure.
func RecordObservedTokenFamily(ctx context.Context, family string) {
	o, _ := ctx.Value(tokenFamilyObserverCtxKey{}).(*TokenFamilyObserver)
	o.record(family)
}
