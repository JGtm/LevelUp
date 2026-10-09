package sync

// achievements_token_gate.go — le chemin des succès Xbox ne réessaie pas un jeton mort.
//
// Le refresh token d'un joueur sert d'abord au pool d'auto-sync, qui suspend lui-même un jeton
// mort (cache négatif par jeton, `platform/auth/pool`). Le chemin secondaire des succès Xbox,
// lui, rafraîchissait à CHAQUE passe : un jeton refusé par Microsoft (`invalid_grant`, par
// exemple AADSTS70000) produisait un avertissement par joueur et par passe.
//
// LA RÈGLE : un jeton dont l'entrée du store est marquée morte (`reauth_required`, ou dernier
// échec de classe `revoked` / `config`) n'est plus présenté à Microsoft par ce chemin tant que le
// refresh token n'a pas CHANGÉ (import, SSO, rotation) ou que l'entrée n'a pas été démarquée
// (refresh réussi ailleurs). L'empreinte (SHA-256) du jeton constaté mort est retenue en mémoire
// du processus, par xuid ; le jeton lui-même n'est jamais conservé ni journalisé.
//
// Une trace par CHANGEMENT d'état (jeton constaté mort, jeton remplacé ou redevenu valide), un
// Debug par passe sautée. Jamais de re-capture : l'état ne se lève que par un nouveau jeton ou un
// refresh réussi. La décision vit ici (paquet sync) ; `auth` ne fournit que le store et la
// classification d'erreur (ADR 0023).

import (
	"context"
	"crypto/sha256"
	"log/slog"
	gosync "sync"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
)

// deadTokenGate retient, par xuid, l'empreinte du refresh token constaté mort.
type deadTokenGate struct {
	mu   gosync.Mutex
	dead map[string][sha256.Size]byte
}

// achievementsDeadTokens : la mémoire du processus (les SyncEngine sont créés par passe).
var achievementsDeadTokens = newDeadTokenGate()

func newDeadTokenGate() *deadTokenGate {
	return &deadTokenGate{dead: map[string][sha256.Size]byte{}}
}

func rtFingerprint(rt string) [sha256.Size]byte { return sha256.Sum256([]byte(rt)) }

// entryMarkedDead : l'entrée du store porte un état « jeton mort » (bannière de
// reconnexion, ou dernier échec permanent mémorisé par un autre chemin).
func entryMarkedDead(u *auth.UserTokens) bool {
	return u.ReauthRequired ||
		u.LastAuthErrorClass == string(auth.AuthErrorRevoked) ||
		u.LastAuthErrorClass == string(auth.AuthErrorConfig)
}

// skip dit si la passe doit sauter le refresh pour l'entrée `u` (refresh token non vide).
func (g *deadTokenGate) skip(ctx context.Context, xuid, gamertag string, u *auth.UserTokens) bool {
	fp := rtFingerprint(u.OAuthRefreshToken)
	marked := entryMarkedDead(u)
	g.mu.Lock()
	defer g.mu.Unlock()
	known, ok := g.dead[xuid]
	switch {
	case ok && known == fp && marked:
		slog.DebugContext(ctx, "achievements: jeton toujours marqué mort — passe sautée", "gamertag", gamertag)
		return true
	case ok:
		delete(g.dead, xuid)
		slog.InfoContext(ctx, "achievements: jeton remplacé ou démarqué — nouvelle tentative",
			"gamertag", gamertag, "jeton_change", known != fp)
		return false
	case marked:
		g.dead[xuid] = fp
		slog.WarnContext(ctx, "achievements: jeton marqué mort dans le store — succès Xbox suspendus "+
			"jusqu'à un nouveau jeton", "gamertag", gamertag, "reauth_required", u.ReauthRequired,
			"last_auth_error_class", u.LastAuthErrorClass)
		return true
	}
	return false
}

// recordFailure retient le jeton `rt` comme mort si l'échec est permanent, et le consigne
// au store (mêmes écritures que le pool : dernier échec, et reconnexion requise pour un jeton
// révoqué) pour que l'état survive au redémarrage. Échec transitoire : rien.
func (g *deadTokenGate) recordFailure(ctx context.Context, store *auth.MultiUserTokenStore,
	xuid, gamertag, rt string, err error) {
	class := auth.ClassifyAuthError(err)
	if class != auth.AuthErrorRevoked && class != auth.AuthErrorConfig {
		return
	}
	g.mu.Lock()
	g.dead[xuid] = rtFingerprint(rt)
	g.mu.Unlock()
	slog.WarnContext(ctx, "achievements: jeton refusé (échec permanent) — succès Xbox suspendus "+
		"jusqu'à un nouveau jeton", "gamertag", gamertag, "class", string(class))
	if werr := store.RecordAuthError(xuid, gamertag, string(class), err.Error()); werr != nil {
		slog.WarnContext(ctx, "achievements: écriture du dernier échec au store échouée",
			"gamertag", gamertag, "err", werr)
	}
	if class == auth.AuthErrorRevoked {
		if _, werr := store.MarkReauthRequired(xuid, gamertag); werr != nil {
			slog.WarnContext(ctx, "achievements: marquage reauth_required échoué",
				"gamertag", gamertag, "err", werr)
		}
	}
}

// recordSuccess lève l'état mort après un refresh réussi : le jeton vit, la bannière et le
// dernier échec sont obsolètes (même effacement que le pool sur un succès).
func (g *deadTokenGate) recordSuccess(ctx context.Context, store *auth.MultiUserTokenStore,
	xuid, gamertag string, u *auth.UserTokens) {
	g.mu.Lock()
	delete(g.dead, xuid)
	g.mu.Unlock()
	if !entryMarkedDead(u) {
		return
	}
	slog.InfoContext(ctx, "achievements: jeton de nouveau valide — état mort levé", "gamertag", gamertag)
	if err := store.ClearReauthRequired(xuid); err != nil {
		slog.WarnContext(ctx, "achievements: effacement reauth_required échoué", "gamertag", gamertag, "err", err)
	}
	if err := store.ClearAuthError(xuid); err != nil {
		slog.WarnContext(ctx, "achievements: effacement du dernier échec échoué", "gamertag", gamertag, "err", err)
	}
}

// resolveAchievementsAccessToken résout l'access_token Xbox Live (succès) depuis le
// MultiUserTokenStore (source unique, ADR 0023) via auth.ResolveMSAccessTokenStoreFirst,
// derrière la porte des jetons morts. Rend skipped=true (sans erreur) quand la porte saute la
// passe ; ("", false, nil) quand aucun jeton n'est disponible.
func (e *SyncEngine) resolveAchievementsAccessToken(ctx context.Context) (token string, skipped bool, err error) {
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(e.repoRoot).WatcherTokensDir())
	return resolveGatedAccessToken(ctx, achievementsDeadTokens, store, e.provider, e.xuid, e.gamertag)
}

// resolveGatedAccessToken : le corps de resolveAchievementsAccessToken, porte et store injectés.
func resolveGatedAccessToken(ctx context.Context, gate *deadTokenGate, store *auth.MultiUserTokenStore,
	provider auth.TokenProvider, xuid, gamertag string) (string, bool, error) {
	var entry *auth.UserTokens
	if xuid != "" {
		// Lecture d'aiguillage seulement : un échec de lecture est journalisé et tranché par
		// auth.ResolveMSAccessTokenStoreFirst, qui relit le store.
		if u, lerr := store.Load(xuid); lerr == nil && u != nil && u.OAuthRefreshToken != "" {
			entry = u
		}
	}
	if entry != nil && gate.skip(ctx, xuid, gamertag, entry) {
		return "", true, nil
	}
	at, err := auth.ResolveMSAccessTokenStoreFirst(ctx, provider, store, xuid, gamertag)
	if entry == nil {
		return at, false, err
	}
	if err != nil {
		gate.recordFailure(ctx, store, xuid, gamertag, entry.OAuthRefreshToken, err)
		return "", false, err
	}
	if at != "" {
		gate.recordSuccess(ctx, store, xuid, gamertag, entry)
	}
	return at, false, nil
}
