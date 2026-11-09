package wakeup

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/sync/deadtoken"
)

// TokenOutcome : l'issue de la vérification du jeton d'un profil.
type TokenOutcome string

const (
	// TokenAlive : le refresh token a répondu ; le jeton tourné est écrit au magasin (écriture
	// atomique) et l'éventuel état « mort » est levé.
	TokenAlive TokenOutcome = "vivant"
	// TokenDead : Microsoft refuse le jeton (invalid_grant) ; reauth_required est posé, ce qui lève
	// la bannière de reconnexion. Le jeton n'est PAS re-capturé (ADR 0023).
	TokenDead TokenOutcome = "mort"
	// TokenAlreadyDead : le magasin porte déjà l'état « mort » pour ce jeton ; il n'est pas
	// présenté une seconde fois à Microsoft.
	TokenAlreadyDead TokenOutcome = "deja_mort"
	// TokenRefused : refus déterministe qui n'est pas une révocation (application ou client mal
	// configuré) ; consigné au magasin comme dernier échec, sans bannière de reconnexion.
	TokenRefused TokenOutcome = "refuse"
	// TokenUnreachable : aucune réponse exploitable (réseau absent, délai, 5xx). Rien n'est marqué ;
	// la vérification se refait plus tard.
	TokenUnreachable TokenOutcome = "injoignable"
	// TokenAbsent : le profil n'a aucun jeton à vérifier (jamais connecté par le SSO).
	TokenAbsent TokenOutcome = "absent"
)

// TokenResult : l'issue pour un joueur.
type TokenResult struct {
	XUID     string
	Gamertag string
	Outcome  TokenOutcome
	// Err : la cause quand le jeton n'est ni vivant ni absent. Ne contient jamais de jeton.
	Err error
}

// TokenReport : les issues de toute une vérification, dans l'ordre des profils.
type TokenReport struct {
	Results []TokenResult
}

// count rend le nombre de joueurs dont l'issue est outcome.
func (r TokenReport) count(outcome TokenOutcome) int {
	n := 0
	for _, res := range r.Results {
		if res.Outcome == outcome {
			n++
		}
	}
	return n
}

// Unreachable rend les xuid dont le jeton n'a pas pu être vérifié faute de réponse.
func (r TokenReport) Unreachable() []string {
	var xuids []string
	for _, res := range r.Results {
		if res.Outcome == TokenUnreachable {
			xuids = append(xuids, res.XUID)
		}
	}
	return xuids
}

// NetworkAbsent dit si la vérification n'a obtenu AUCUNE réponse de Microsoft alors qu'elle en
// demandait : au moins un joueur injoignable, aucun vivant, mort ni refusé. Dans ce cas la suite de la
// séquence (la synchronisation) attend elle aussi le retour du réseau.
func (r TokenReport) NetworkAbsent() bool {
	if r.count(TokenUnreachable) == 0 {
		return false
	}
	return r.count(TokenAlive)+r.count(TokenDead)+r.count(TokenRefused) == 0
}

// TokenChecker vérifie les jetons des profils suivis. only non vide : seulement ces xuid (reprise
// après un réseau absent) ; vide : tous les profils suivis.
type TokenChecker interface {
	Check(ctx context.Context, only []string) TokenReport
}

// perPlayerTimeout borne la vérification d'UN joueur (deux appels OAuth de 15 s au plus, plus
// l'écriture du magasin) : un joueur qui ne répond pas ne retient pas les suivants plus que cela.
const perPlayerTimeout = 45 * time.Second

// TokenCheck est la vérification de production : le rafraîchissement OAuth par joueur de
// deadtoken.Gate.Resolve (RT lu du magasin, jeton tourné écrit atomiquement, reauth_required posé
// sur un jeton révoqué et levé sur un succès, rien sur un échec transitoire), sans échange Halo :
// vérifier qu'un jeton vit n'a pas besoin de Spartan.
type TokenCheck struct {
	// Players rend les profils déclarés (db_profiles.json) ; la porte de l'ADR 0035 (non auth_only,
	// sync actif) s'applique ici.
	Players func() ([]domain.PlayerSummary, error)
	// Store est le magasin de jetons par joueur (source unique, ADR 0023).
	Store *auth.MultiUserTokenStore
	// Provider fait le rafraîchissement OAuth.
	Provider auth.TokenProvider
	// Gate retient les jetons constatés morts ; une instance par processus.
	Gate *deadtoken.Gate
}

// Check vérifie, un joueur à la fois, le jeton de chaque profil suivi. Un même xuid porté par
// plusieurs titres n'est vérifié qu'une fois : deux rafraîchissements du même jeton à la suite en
// perdraient la rotation.
func (c *TokenCheck) Check(ctx context.Context, only []string) TokenReport {
	players, err := c.Players()
	if err != nil {
		slog.ErrorContext(ctx, "wakeup: profils suivis illisibles — jetons non vérifiés", "err", err)
		return TokenReport{}
	}
	wanted := map[string]bool{}
	for _, x := range only {
		wanted[x] = true
	}
	seen := map[string]bool{}
	var report TokenReport
	for _, p := range domain.SyncablePlayers(players) {
		xuid := c.xuidOf(p)
		if xuid == "" || seen[xuid] || (len(wanted) > 0 && !wanted[xuid]) {
			continue
		}
		seen[xuid] = true
		if ctx.Err() != nil {
			break // arrêt demandé : on ne démarre plus d'échange
		}
		report.Results = append(report.Results, c.checkOne(ctx, xuid, p.Gamertag))
	}
	return report
}

// xuidOf rend le xuid du profil : celui du fichier de profils, sinon celui que le magasin porte pour
// ce gamertag (un profil ancien peut ne pas porter son xuid). "" : aucun jeton connu pour lui.
func (c *TokenCheck) xuidOf(p domain.PlayerSummary) string {
	if p.XUID != "" {
		return p.XUID
	}
	if p.Gamertag == "" {
		return ""
	}
	u, err := c.Store.LoadByGamertag(p.Gamertag)
	if err != nil || u == nil {
		return ""
	}
	return u.XUID
}

// checkOne vérifie un joueur et journalise l'issue.
func (c *TokenCheck) checkOne(ctx context.Context, xuid, gamertag string) TokenResult {
	ctx, cancel := context.WithTimeout(ctx, perPlayerTimeout)
	defer cancel()
	at, skipped, err := c.Gate.Resolve(ctx, c.Store, c.Provider, xuid, gamertag)
	res := TokenResult{XUID: xuid, Gamertag: gamertag, Err: err}
	switch {
	case skipped:
		res.Outcome = TokenAlreadyDead
		slog.InfoContext(ctx, "wakeup: jeton déjà marqué mort — reconnexion Xbox nécessaire", "gamertag", gamertag)
	case err != nil:
		res.Outcome = outcomeOfFailure(err)
		logFailure(ctx, gamertag, res.Outcome, err)
	case at == "":
		res.Outcome = TokenAbsent
		slog.InfoContext(ctx, "wakeup: aucun jeton à vérifier pour ce profil", "gamertag", gamertag)
	default:
		res.Outcome = TokenAlive
		slog.InfoContext(ctx, "wakeup: jeton vivant — rotation écrite au magasin", "gamertag", gamertag)
	}
	return res
}

// outcomeOfFailure range un échec de rafraîchissement selon la classe d'échec d'authentification
// du code existant : révoqué = mort, configuration = refusé, tout le reste (transport, délai, 5xx,
// erreur non typée) = injoignable.
func outcomeOfFailure(err error) TokenOutcome {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return TokenUnreachable
	}
	switch auth.ClassifyAuthError(err) {
	case auth.AuthErrorRevoked:
		return TokenDead
	case auth.AuthErrorConfig:
		return TokenRefused
	default:
		return TokenUnreachable
	}
}

// logFailure journalise l'échec d'un joueur au niveau de sa gravité : un jeton mort ou refusé est
// un avertissement (le magasin le porte), un réseau absent une simple information.
func logFailure(ctx context.Context, gamertag string, outcome TokenOutcome, err error) {
	switch outcome {
	case TokenDead:
		slog.WarnContext(ctx, "wakeup: jeton refusé par Microsoft — reauth_required posé, reconnexion Xbox nécessaire",
			"gamertag", gamertag, "err", bounded(err))
	case TokenRefused:
		slog.WarnContext(ctx, "wakeup: jeton refusé pour une cause de configuration — pas de bannière de reconnexion",
			"gamertag", gamertag, "err", bounded(err))
	default:
		slog.InfoContext(ctx, "wakeup: jeton non vérifié faute de réponse — rien n'est marqué, nouvel essai plus tard",
			"gamertag", gamertag, "err", bounded(err))
	}
}

// bounded rend le texte de err borné : une réponse serveur peut être longue.
func bounded(err error) string {
	const maxLen = 200
	msg := strings.TrimSpace(err.Error())
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "..."
	}
	return msg
}
