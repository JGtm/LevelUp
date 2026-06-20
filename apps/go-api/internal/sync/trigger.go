// Package sync — trigger.go : déclencheur de sync in-process.
//
// Adapte le SyncEngine (conçu pour être jetable, 1 instance = 1 sync)
// pour être utilisé par le Coordinator du watcher.
//
// Implémente l'interface SyncRunner du coordinator.
package sync

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// TokenProvider fournit les tokens Halo nécessaires au sync.
type TokenProvider interface {
	GetTokens(ctx context.Context) (*domain.HaloTokens, error)
}

// LiveTitleRunner exécute une sync delta pour un titre live-only (Halo 5+).
// Implémenté par *livesync.Runner. Déclaré ici (et non importé de livesync) pour
// éviter le cycle sync → livesync (livesync importe déjà sync).
type LiveTitleRunner interface {
	RunDelta(ctx context.Context, opts domain.SyncOptions) (domain.SyncResult, error)
}

// LiveTitleRunnerFactory résout le runner d'un titre live-only pour (slug,
// gamertag, xuid). Sémantique du retour :
//   - handled == false                 → le slug N'EST PAS un titre live → path engine.
//   - handled == true, err == nil      → runner prêt ; runCtx porte l'auth.
//   - handled == true, err != nil      → titre live mais résolution échouée → le
//     caller ABANDONNE le tick (JAMAIS de fallback engine : fetch Infinite dans un
//     store h5 = corruption). release est toujours non-nil (no-op si rien à libérer).
//
// Câblée par cmd/server/main.go via livesync.HandlesTitle + livesync.AcquireRunner
// (auth pinnée depuis le pool). Nil → le Trigger reste 100 % path engine (legacy).
type LiveTitleRunnerFactory func(ctx context.Context, slug, gamertag, xuid string) (runner LiveTitleRunner, runCtx context.Context, release func(), handled bool, err error)

// EngineFactory construit un *SyncEngine fully-configured pour un (gamertag,
// xuid). C'est l'extension point qui permet au caller (cmd/server/main.go)
// de partager la MÊME logique de wiring entre le path watcher (Trigger) et le
// path scheduler (auto_sync.defaultRunnerFactory). Sans factory, le Trigger
// fait un NewSyncEngine direct (mode legacy — pas de SharedProvider, pas de
// PostSyncRunner, etc., utile uniquement pour des tests étroits).
//
// Cf. incident 2026-05-26 : trigger.go faisait NewSyncEngine sans
// .WithSharedProvider, tous les syncs déclenchés par le watcher échouaient
// avec "Can't open a connection to same database file with a different
// configuration than existing connections" car le path legacy contournait
// le swap RO↔RW orchestré par sharedprovider.Manager.
type EngineFactory func(ctx context.Context, gamertag, xuid string) *SyncEngine

// Trigger déclenche des syncs in-process via SyncEngine.
// Il crée un SyncEngine jetable à chaque appel.
type Trigger struct {
	repoRoot      string
	tokenProvider TokenProvider
	defaultOpts   domain.SyncOptions
	// engineFactory construit le SyncEngine. Si nil, fallback NewSyncEngine
	// direct (legacy — risque conflit "different configuration" en présence
	// d'un sharedprovider.Manager global). À câbler en production via
	// WithEngineFactory(autoScheduler.BuildEngine).
	engineFactory EngineFactory
	// liveRunnerFactory route les titres live-only (Halo 5+) vers leur pipeline
	// dédié au lieu de l'engine Infinite. Nil → 100 % path engine. Câblée par
	// main.go (cf. LiveTitleRunnerFactory).
	liveRunnerFactory LiveTitleRunnerFactory
}

// NewTrigger crée un trigger de sync in-process.
//
// Pour la production, chaîner avec WithEngineFactory(autoScheduler.BuildEngine)
// afin de partager le wiring (SharedProvider, PostSyncRunner, FriendsLoader,
// PooledClient, batch mode, etc.) avec l'auto-sync scheduler.
func NewTrigger(repoRoot string, tp TokenProvider, defaultOpts domain.SyncOptions) *Trigger {
	return &Trigger{
		repoRoot:      repoRoot,
		tokenProvider: tp,
		defaultOpts:   defaultOpts,
	}
}

// WithEngineFactory injecte la factory qui construit le SyncEngine fully-
// configured (avec SharedProvider, PostSyncRunner, etc.). C'est l'unique
// point d'extension du Trigger : il permet de garantir la parité runtime
// entre le path watcher et le path scheduler en réutilisant la MÊME closure
// de construction (cf. scheduler.BuildEngine).
//
// Si f est nil, le Trigger reste en mode legacy (NewSyncEngine direct).
// Le retour est le même Trigger pour permettre le chaînage style builder.
func (t *Trigger) WithEngineFactory(f EngineFactory) *Trigger {
	t.engineFactory = f
	return t
}

// WithLiveRunnerFactory injecte la fabrique de runner des titres live-only
// (Halo 5+). Sans elle, le Trigger route TOUS les titres vers l'engine Infinite —
// ce qui corromprait un store h5 (fetch Infinite). Cf. LiveTitleRunnerFactory.
// Retour = même Trigger pour le chaînage builder.
func (t *Trigger) WithLiveRunnerFactory(f LiveTitleRunnerFactory) *Trigger {
	t.liveRunnerFactory = f
	return t
}

// RunSync implémente SyncRunner pour le Coordinator.
// Crée un SyncEngine via engineFactory (ou NewSyncEngine direct en fallback)
// et lance un RunDelta.
func (t *Trigger) RunSync(ctx context.Context, gamertag, xuid string, matchIDs []string) error {
	slug := ctxkeys.TitleSlug(ctx)
	slog.InfoContext(ctx, "trigger: démarrage sync",
		"gamertag", gamertag,
		"xuid", xuid,
		"title_slug", slug,
		"match_ids_hint", len(matchIDs),
		"engine_factory_wired", t.engineFactory != nil,
	)

	// Titres live-only (Halo 5+) : pipeline dédié, JAMAIS l'engine Infinite (qui
	// fetcherait des matchs Infinite dans le store du titre). Branche registry-
	// driven via la factory injectée ; auth pinnée depuis le pool (runCtx).
	if t.liveRunnerFactory != nil {
		runner, runCtx, release, handled, lerr := t.liveRunnerFactory(ctx, slug, gamertag, xuid)
		if handled {
			defer release()
			if lerr != nil {
				// Titre live mais résolution échouée → on abandonne (pas de fallback
				// engine : corromprait le store). Re-tenté au prochain trigger.
				return fmt.Errorf("trigger: runner live %s indisponible: %w", slug, lerr)
			}
			return t.runLiveDelta(runCtx, runner, gamertag, slug, matchIDs)
		}
	}

	tokens, err := t.tokenProvider.GetTokens(ctx)
	if err != nil {
		return fmt.Errorf("trigger: get tokens: %w", err)
	}

	var engine *SyncEngine
	if t.engineFactory != nil {
		engine = t.engineFactory(ctx, gamertag, xuid)
		if engine == nil {
			return fmt.Errorf("trigger: engineFactory returned nil for %s", gamertag)
		}
		// Tokens passés par staticTokenProvider — on les pousse sur le moteur
		// au cas où la factory n'aurait pas (re)résolu d'auth (cas typique :
		// auto_sync.BuildEngine met un HaloTokens vide car le pool joueur fait
		// l'auth via PooledClient).
		_ = tokens
	} else {
		// Path legacy : aucune factory câblée. Le moteur ne sera PAS configuré
		// avec SharedProvider, donc OpenSharedDB direct → potentiel conflit
		// "different configuration" si un Manager global tient déjà shared en
		// RO. Conservé uniquement pour la rétrocompat des tests existants.
		engine = NewSyncEngine(t.repoRoot, gamertag, xuid, tokens, nil)
	}

	opts := t.defaultOpts
	// Le watcher détecte les matchs mais le RunDelta va re-fetch l'historique API
	// pour récupérer les stats complètes. On limite au nombre de matchs détectés + marge.
	if len(matchIDs) > 0 && opts.MaxMatches == 0 {
		opts.MaxMatches = len(matchIDs) + 5
	}

	result, err := engine.RunDelta(ctx, opts)
	if err != nil {
		slog.ErrorContext(ctx, "trigger: sync échoué",
			"gamertag", gamertag,
			"err", err,
		)
		return fmt.Errorf("trigger: sync %s: %w", gamertag, err)
	}

	slog.InfoContext(ctx, "trigger: sync terminé",
		"gamertag", gamertag,
		"matches_inserted", result.MatchesInserted,
		"participants_done", result.ParticipantsDone,
		"medals_inserted", result.MedalsInserted,
	)

	return nil
}

// runLiveDelta exécute le RunDelta d'un titre live-only (Halo 5+). Même cadrage
// de MaxMatches que le path engine (matchs détectés + marge) ; auth déjà posée
// dans runCtx par la factory. Pas de tokenProvider ni d'engine ici : le runner
// live fait son propre fetch authentifié via le ctx.
func (t *Trigger) runLiveDelta(runCtx context.Context, runner LiveTitleRunner, gamertag, slug string, matchIDs []string) error {
	opts := t.defaultOpts
	if len(matchIDs) > 0 && opts.MaxMatches == 0 {
		opts.MaxMatches = len(matchIDs) + 5
	}
	result, err := runner.RunDelta(runCtx, opts)
	if err != nil {
		slog.ErrorContext(runCtx, "trigger: sync live échoué",
			"gamertag", gamertag, "title_slug", slug, "err", err)
		return fmt.Errorf("trigger: sync live %s (%s): %w", gamertag, slug, err)
	}
	slog.InfoContext(runCtx, "trigger: sync live terminé",
		"gamertag", gamertag,
		"title_slug", slug,
		"matches_inserted", result.MatchesInserted,
		"participants_done", result.ParticipantsDone,
		"medals_inserted", result.MedalsInserted,
	)
	return nil
}
