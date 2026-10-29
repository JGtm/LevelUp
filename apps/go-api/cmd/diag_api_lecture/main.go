// cmd/diag_api_lecture — LECTURE SEULE de sept surfaces de l'API Halo, sortie sur stdout.
//
// POURQUOI CET OUTIL EXISTE (plan D3, `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D3_2026-10-29.md`, D3-N).
// Le découplage de la brique « API Halo » déplace le code qui parle aux services Halo ; la preuve
// que rien n'a changé est un témoin LIVE pris AVANT et rejoué APRÈS, surface par surface. Aucune
// commande existante ne lisait ces surfaces sans écrire une base ou afficher des jetons. Cet outil
// reste dans le dépôt comme les autres `cmd/diag_*` : il est écrit sur les clients d'aujourd'hui
// (`platform/halo`, `sync/haloclient`) et sera rebranché sur la brique à la fin du chantier.
//
// SORTIES (le témoin est la sérialisation, identique avant et après) :
//   - calendrier : le CORPS JSON REÇU, octet pour octet ;
//   - historique, stats, skill, manifeste, actif, servicerecord : le client rend un type Go, pas
//     les octets ; la sortie est `json.MarshalIndent(valeur, "", "  ")` de ce type (clés de table
//     triées par le codec, donc stable), sans saut de ligne final.
//
// CE QU'IL NE FAIT PAS : aucune ouverture de base, aucune écriture hors stdout/stderr, aucun jeton
// ni en-tête écrit. Le seul effet de bord est celui du chemin d'auth canonique (ADR 0023) : un
// refresh token Microsoft est à USAGE UNIQUE, donc `RefreshHaloTokensViaStoreFirst` en obtient un
// nouveau et le persiste dans le magasin — au plus UNE rotation par exécution. Aucune re-capture :
// si le refresh échoue, l'outil s'arrête en le disant.
//
// Usage (depuis apps/go-api/ d'un worktree sans data/ : LEVELUP_REPO_ROOT vers le dépôt qui les porte) :
//
//	diag_api_lecture [-xuid X] historique
//	diag_api_lecture [-xuid X] stats    <match_id>
//	diag_api_lecture [-xuid X] skill    <match_id>
//	diag_api_lecture [-xuid X] manifeste <match_id>
//	diag_api_lecture [-xuid X] actif    <type> <asset_id> <version_id>   (type : map|playlist|pair|game_variant)
//	diag_api_lecture [-xuid X] calendrier
//	diag_api_lecture [-xuid X] servicerecord
//
// Sans -xuid : le premier joueur de db_profiles.json. Journal slog sur stderr.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/sync/haloclient"
)

// requestsPerSecond : débit du client de l'historique et des stats (le même que les autres diag).
const requestsPerSecond = 3

// session : le contexte authentifié d'une exécution.
type session struct {
	ctx      context.Context
	xuid     string
	gamertag string
	tokens   *domain.HaloTokens
	client   *haloclient.HaloAPIClient
	provider *halo.HaloProvider
	out      io.Writer
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run exécute la ligne de commande et rend le code de sortie. Les arguments sont validés AVANT
// toute authentification : une faute de frappe ne provoque pas de rotation de jeton.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diag_api_lecture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	xuid := fs.String("xuid", "", "xuid du compte lecteur (défaut : premier joueur de db_profiles.json)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s, ok := trouverSurface(fs.Args())
	if !ok {
		fmt.Fprint(stderr, usage())
		return 2
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, nil)))
	ctx := context.Background()
	sess, err := ouvrirSession(ctx, *xuid, stdout)
	if err != nil {
		slog.ErrorContext(ctx, "diag_api_lecture: ouverture de session impossible", "err", err)
		fmt.Fprintln(stderr, "-> diagnostiquer la cause (rotation perdue, mauvais xuid) — ne PAS re-capturer de jeton.")
		return 1
	}
	if err := s.run(sess, fs.Args()[1:]); err != nil {
		slog.ErrorContext(ctx, "diag_api_lecture: lecture échouée", "surface", s.nom, "err", err)
		return 1
	}
	return 0
}

// ouvrirSession résout le joueur, rafraîchit ses jetons par le chemin canonique (magasin d'abord,
// ADR 0023) et construit les deux clients.
func ouvrirSession(ctx context.Context, xuid string, out io.Writer) (*session, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	joueur, err := resoudreJoueur(cfg, xuid)
	if err != nil {
		return nil, err
	}
	store := auth.NewMultiUserTokenStore(titlePkg.NewPathResolver(cfg.RepoRoot).WatcherTokensDir())
	exch, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), joueur.XUID, joueur.Gamertag)
	if err != nil || exch == nil || exch.Tokens == nil {
		return nil, fmt.Errorf("auth %s (xuid %s): %v", joueur.Gamertag, joueur.XUID, err)
	}
	return &session{
		ctx:      ctxkeys.WithHaloAuth(ctx, exch.Tokens, joueur.XUID),
		xuid:     joueur.XUID,
		gamertag: joueur.Gamertag,
		tokens:   exch.Tokens,
		client:   haloclient.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, requestsPerSecond),
		provider: halo.NewHaloProvider().WithTokens(exch.Tokens),
		out:      out,
	}, nil
}

// resoudreJoueur rend le joueur suivi demandé (par xuid) ou le premier de db_profiles.json. Un
// joueur absent du registre n'a pas de profil suivi : aucun jeton n'est cherché pour lui.
func resoudreJoueur(cfg *config.AppConfig, xuid string) (domain.PlayerSummary, error) {
	joueurs, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return domain.PlayerSummary{}, fmt.Errorf("db_profiles.json: %w", err)
	}
	if len(joueurs) == 0 {
		return domain.PlayerSummary{}, fmt.Errorf("db_profiles.json ne déclare aucun joueur")
	}
	if xuid == "" {
		return joueurs[0], nil
	}
	for _, j := range joueurs {
		if j.XUID == xuid {
			return j, nil
		}
	}
	return domain.PlayerSummary{}, fmt.Errorf("xuid %s absent de db_profiles.json", xuid)
}
