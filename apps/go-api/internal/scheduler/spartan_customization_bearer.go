// Package scheduler — spartan_customization_bearer.go : choix du token qui lit
// l'apparence d'un joueur suivi pour le cron de personnalisation Spartan.
//
// Le token du joueur lui-même passe en premier : il ouvre la vue privée de sa
// personnalisation (/customization/appearance). Quand ce token n'est pas utilisable
// (refresh token refusé, donc aucun créneau dans le pool ; créneau malsain ou en
// pause de débit), la lecture passe par le token du compte de l'utilisateur de
// l'instance — le xuid lié à un compte de rôle admin —, le porteur, et par AUCUN
// autre : le token d'un autre utilisateur ne lit jamais l'apparence d'un tiers sans
// son accord. Sans porteur utilisable, rien n'est lu ni écrit.
//
// Les endpoints lus ciblent le joueur dans l'URL et ne dérivent jamais l'identité du
// token appelant : careerranks (xuid), customization et sa vue publique
// `?view=public` (xuid, repli de haloclient.fetchCustomizationAppearance sur un 403),
// profils Halo 5 (gamertag). Le porteur ne fait que LIRE. Aucun token n'est capturé
// ni rafraîchi ici : le pool s'en charge (ADR 0023). La ligne écrite va dans la base
// du joueur lu, sous son xuid, par le chemin d'écriture habituel du refresher du titre.
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

// errNoBearer : ni le token du joueur ni celui du compte admin n'est utilisable.
// Rien n'est lu ni écrit, et le cycle compte un échec.
var errNoBearer = errors.New("spartan_cron: aucun token utilisable, ni propre ni du compte admin")

// errNoOwnToken : le joueur n'a pas de créneau dans le pool (refresh token absent,
// ou refusé quand le pool l'a résolu).
var errNoOwnToken = errors.New("aucun token propre dans le pool")

// Compteurs expvar des lectures portées par le compte admin, et des lectures
// abandonnées faute de tout token utilisable.
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

// bearerCandidate est un profil déclaré dont le xuid est lié à un compte de rôle
// admin : son token peut porter la lecture de l'apparence d'un autre joueur.
type bearerCandidate struct {
	gamertag string
	xuid     string
}

// adminBearerCandidates rend les profils déclarés dont le xuid est lié à un compte de
// rôle admin (adminXUIDs), par gamertag insensible à la casse : l'ordre est le même à
// chaque cycle, quel que soit l'ordre de lecture des fichiers. Aucun autre profil
// n'est candidat, quel que soit l'état de son token.
//
// La jointure se fait par xuid, seule clé commune aux comptes et aux profils (ADR
// 0035 D1) : le gamertag est la clé du créneau dans le pool, pas une identité. Un
// profil en pause ou auth_only compte : porter une lecture n'est ni un sync ni un
// suivi. Un gamertag déclaré pour plusieurs titres n'apparaît qu'une fois.
func adminBearerCandidates(players []domain.PlayerSummary, adminXUIDs map[string]bool) []bearerCandidate {
	seen := make(map[string]bool, len(players))
	out := make([]bearerCandidate, 0, len(adminXUIDs))
	for _, p := range players {
		if p.Gamertag == "" || !adminXUIDs[p.XUID] || seen[p.Gamertag] {
			continue
		}
		seen[p.Gamertag] = true
		out = append(out, bearerCandidate{gamertag: p.Gamertag, xuid: p.XUID})
	}
	sort.SliceStable(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i].gamertag), strings.ToLower(out[j].gamertag)
		if li != lj {
			return li < lj
		}
		return out[i].gamertag < out[j].gamertag
	})
	return out
}

// errAccountsUnreadable : les comptes de l'instance n'ont pas pu être lus, donc le
// compte de l'utilisateur ne peut pas être reconnu.
var errAccountsUnreadable = errors.New("comptes de l'instance illisibles")

// adminXUIDs rend les xuid liés aux comptes de rôle admin. Sans lecteur de comptes
// câblé (tests, process sans comptes), aucun : seul le token propre reste possible.
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

// bearerCandidates charge les profils déclarés de TOUS les titres (le pool est indexé
// par gamertag, un compte peut porter la lecture pour n'importe quel titre) et en
// garde ceux du compte admin (adminBearerCandidates). Une lecture des comptes ou des
// profils en échec est journalisée et rend une liste vide : seul le token propre reste
// possible pendant ce cycle.
func (c *SpartanCustomizationCron) bearerCandidates(ctx context.Context) []bearerCandidate {
	adminXUIDs, err := c.adminXUIDs()
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: comptes de l'instance illisibles — seul le token propre sera essayé",
			"err", err)
		return nil
	}
	if len(adminXUIDs) == 0 {
		return nil
	}
	players, err := c.cfg.LoadPlayers()
	if err != nil {
		slog.ErrorContext(ctx, "spartan_cron: lecture des profils impossible — seul le token propre sera essayé",
			"err", err)
		return nil
	}
	return adminBearerCandidates(players, adminXUIDs)
}

// acquireReaderToken rend le lease dont les tokens liront l'apparence de p et le
// xuid du compte qui les porte : le token de p s'il est utilisable (xuid == p.XUID),
// sinon celui du premier candidat admin utilisable. errNoBearer si aucun ne l'est,
// compté dans spartan_cron_no_bearer_total.
//
// Les deux recherches passent par PolicyPinnedPlayer : elle désigne le token d'un
// compte précis, quand PolicyAnyPublic tournerait sur tout le parc.
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
			slog.DebugContext(ctx, "spartan_cron: token du compte admin indisponible",
				"gamertag", p.Gamertag, "bearer", cand.gamertag, "err", leaseError(err))
			continue
		}
		observability.AddInt(metricBearerReads, 1)
		slog.InfoContext(ctx, "spartan_cron: apparence lue avec le token du compte admin",
			"gamertag", p.Gamertag, "xuid", p.XUID,
			"bearer", cand.gamertag, "bearer_xuid", cand.xuid, "own_token_err", ownErr)
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
