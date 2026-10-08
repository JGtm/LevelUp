// Package scheduler — spartan_customization_bearer.go : choix du token qui lit
// l'apparence d'un joueur suivi pour le cron de personnalisation Spartan.
//
// Le token du joueur lui-même passe en premier : il ouvre la vue privée de sa
// personnalisation (/customization/appearance). Quand ce token n'est pas utilisable
// (refresh token refusé, donc aucun créneau dans le pool ; créneau malsain ou en
// pause de débit), la lecture passe par le token d'un AUTRE compte du parc, le
// porteur. Les endpoints lus ciblent le joueur dans l'URL et ne dérivent jamais
// l'identité du token appelant : careerranks (xuid), customization et sa vue
// publique `?view=public` (xuid, repli de haloclient.fetchCustomizationAppearance
// sur un 403), profils Halo 5 (gamertag).
//
// Le porteur ne fait que LIRE. Aucun token n'est capturé ni rafraîchi ici : le pool
// s'en charge (ADR 0023). La ligne écrite va dans la base du joueur lu, sous son
// xuid, par le chemin d'écriture habituel du refresher du titre.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth/pool"
)

// errNoBearer : ni le token du joueur ni celui d'un autre compte du parc n'est
// utilisable. Rien n'est lu ni écrit, et le cycle compte un échec.
var errNoBearer = errors.New("spartan_cron: aucun token utilisable, ni propre ni porteur")

// errNoOwnToken : le joueur n'a pas de créneau dans le pool (refresh token absent,
// ou refusé quand le pool l'a résolu).
var errNoOwnToken = errors.New("aucun token propre dans le pool")

// Compteurs expvar des lectures portées par un autre compte.
const (
	metricBearerReads = "spartan_cron_bearer_reads_total"
	metricNoBearer    = "spartan_cron_no_bearer_total"
)

// AccountsReader lit les comptes de connexion de l'instance (`data/auth/users.json`).
// Implémenté par *userstore.Store, comme playerdirectory.AccountsReader : le cron en
// tire le xuid lié des comptes de rôle admin, ceux de l'utilisateur de l'instance.
type AccountsReader interface {
	List() ([]domain.AdminUserSummary, error)
}

// bearerCandidate est un compte déclaré dont le token peut porter la lecture de
// l'apparence d'un autre joueur. admin : son xuid est lié à un compte de rôle admin.
type bearerCandidate struct {
	gamertag string
	xuid     string
	admin    bool
}

// orderBearerCandidates rend les comptes déclarés dans l'ordre où un porteur est
// cherché : ceux dont le xuid est lié à un compte de rôle admin d'abord (adminXUIDs),
// puis les autres ; dans chaque groupe, par gamertag insensible à la casse. L'ordre
// est le même à chaque cycle, quel que soit l'ordre de lecture des fichiers.
//
// La jointure se fait par xuid, seule clé commune aux comptes et aux profils (ADR
// 0035 D1) : le gamertag est la clé du créneau dans le pool, pas une identité.
// Tous les profils déclarés comptent, y compris les comptes auth_only (qui n'existent
// que pour prêter leur token au parc) et ceux en pause : porter une lecture n'est ni
// un sync ni un suivi. Un gamertag déclaré pour plusieurs titres n'apparaît qu'une
// fois. Un profil sans xuid est écarté : le porteur doit être nommé dans les journaux
// et le budget de débit se compte par xuid.
func orderBearerCandidates(players []domain.PlayerSummary, adminXUIDs map[string]bool) []bearerCandidate {
	seen := make(map[string]bool, len(players))
	out := make([]bearerCandidate, 0, len(players))
	for _, p := range players {
		if p.Gamertag == "" || p.XUID == "" || seen[p.Gamertag] {
			continue
		}
		seen[p.Gamertag] = true
		out = append(out, bearerCandidate{gamertag: p.Gamertag, xuid: p.XUID, admin: adminXUIDs[p.XUID]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].admin != out[j].admin {
			return out[i].admin
		}
		li, lj := strings.ToLower(out[i].gamertag), strings.ToLower(out[j].gamertag)
		if li != lj {
			return li < lj
		}
		return out[i].gamertag < out[j].gamertag
	})
	return out
}

// errAccountsUnreadable : les comptes de l'instance n'ont pas pu être lus, donc le
// compte de l'utilisateur ne peut pas être préféré.
var errAccountsUnreadable = errors.New("comptes de l'instance illisibles")

// adminXUIDs rend les xuid liés aux comptes de rôle admin. Sans lecteur de comptes
// câblé (tests, process sans comptes), aucun compte n'est préféré.
func (c *SpartanCustomizationCron) adminXUIDs() (map[string]bool, error) {
	if c.accounts == nil {
		return nil, nil
	}
	accounts, err := c.accounts.List()
	if err != nil {
		return nil, fmt.Errorf("%w : %w", errAccountsUnreadable, err)
	}
	out := make(map[string]bool)
	for _, a := range accounts {
		if a.Role == domain.RoleAdmin && a.XUID != "" {
			out[a.XUID] = true
		}
	}
	return out, nil
}

// bearerCandidates charge les comptes déclarés de TOUS les titres (le pool est
// indexé par gamertag, un compte peut prêter son token pour n'importe quel titre)
// dans l'ordre de orderBearerCandidates.
//
// Une lecture en échec est journalisée et rend une liste vide : seul le token propre
// reste possible pendant ce cycle. Pour les comptes, c'est voulu : sans eux, le
// compte de l'utilisateur ne se distingue pas des autres, et la lecture n'est
// jamais confiée au compte d'un autre utilisateur faute de savoir lequel préférer.
func (c *SpartanCustomizationCron) bearerCandidates(ctx context.Context) []bearerCandidate {
	adminXUIDs, err := c.adminXUIDs()
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: comptes de l'instance illisibles — seul le token propre sera essayé",
			"err", err)
		return nil
	}
	players, err := c.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: lecture des comptes porteurs impossible — seul le token propre sera essayé",
			"err", err)
		return nil
	}
	return orderBearerCandidates(players, adminXUIDs)
}

// acquireReaderToken rend le lease dont les tokens liront l'apparence de p et le
// xuid du compte qui les porte : le token de p s'il est utilisable (xuid == p.XUID),
// sinon celui du premier candidat utilisable. errNoBearer si aucun ne l'est.
//
// Les deux recherches passent par PolicyPinnedPlayer : elle désigne le token d'un
// compte précis, ce qui rend le choix du porteur déterministe (PolicyAnyPublic
// tourne en rond sur le parc).
func (c *SpartanCustomizationCron) acquireReaderToken(
	ctx context.Context,
	p domain.PlayerSummary,
	candidates []bearerCandidate,
) (*pool.Lease, string, error) {
	ownErr := errNoOwnToken
	if c.pool.HasPlayer(p.Gamertag) {
		lease, err := c.pool.Acquire(ctx, pool.PolicyPinnedPlayer, p.Gamertag)
		if err == nil && lease != nil {
			return lease, p.XUID, nil
		}
		ownErr = leaseError(err)
	}
	for _, cand := range candidates {
		if cand.xuid == p.XUID || cand.gamertag == p.Gamertag || !c.pool.HasPlayer(cand.gamertag) {
			continue
		}
		lease, err := c.pool.Acquire(ctx, pool.PolicyPinnedPlayer, cand.gamertag)
		if err != nil || lease == nil {
			slog.DebugContext(ctx, "spartan_cron: porteur indisponible, candidat suivant",
				"gamertag", p.Gamertag, "bearer", cand.gamertag, "err", leaseError(err))
			continue
		}
		observability.AddInt(metricBearerReads, 1)
		slog.InfoContext(ctx, "spartan_cron: apparence lue avec le token d'un porteur",
			"gamertag", p.Gamertag, "xuid", p.XUID,
			"bearer", cand.gamertag, "bearer_xuid", cand.xuid, "bearer_admin", cand.admin,
			"own_token_err", ownErr)
		return lease, cand.xuid, nil
	}
	observability.AddInt(metricNoBearer, 1)
	return nil, "", fmt.Errorf("%w (token propre : %v)", errNoBearer, ownErr)
}

// leaseError nomme l'échec d'un Acquire, y compris le cas d'un lease nil sans erreur.
func leaseError(err error) error {
	if err != nil {
		return err
	}
	return errors.New("lease nil")
}

// readerContext pose dans ctx les tokens retenus et le SUJET de la lecture. Token
// propre : porteur et sujet sont le joueur (WithHaloAuth seul). Porteur : le budget
// de débit est imputé au porteur (TokensOwnerXUID) et le sujet reste le joueur lu
// (HaloXUID). Le chemin d'identité Spartan persiste quand le sujet demandé est
// HaloXUID (subjectIsOwner) : la ligne va donc à la base du joueur lu.
func readerContext(ctx context.Context, tokens *domain.HaloTokens, bearerXUID, subjectXUID string) context.Context {
	rctx := ctxkeys.WithHaloAuth(ctx, tokens, bearerXUID)
	if bearerXUID != subjectXUID {
		rctx = ctxkeys.WithHaloXUID(rctx, subjectXUID)
	}
	return rctx
}
