// Package sync — engine_batch_path.go : Phase 2.3 du refactor Collect→Persist.
//
// submitMatchAsBatch est le pendant INSERT-only de insertFetchedMatch :
//
//   - Convertit le fetchedMatch en *persist.MatchBatch via
//     buildBatchFromFetchedMatch (logique pure, déjà testée).
//   - Persiste shared et player en transactions INSERT-only via les
//     SharedPersister et PlayerPersister (déjà testés en isolation).
//   - Synchrone : pas de queue ni worker async. Phase 3 ajoutera la couche
//     queue+worker pour découpler la persistance du fetch.
//
// C'est le SEUL chemin d'écriture per-match du live sync : le chemin legacy
// insertFetchedMatch a été supprimé au lot D1b (audits 2026-07). Le point d'entrée
// est persistFetchedMatch : enrichissement registry puis submitMatchAsBatch.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// persistTimeout — timeout par appel Persist (gap C de l'audit safety/guards).
// Évite un hang infini si DuckDB est bloqué (lock interne, IO disque) — la
// TX est annulée via ctx, defer Rollback nettoie. 30s est large : un INSERT
// batch de 10-20 rows en TX DuckDB tourne typiquement en <500ms. Au-delà,
// quelque chose va clairement mal (lock contention, IO saturée).
const persistTimeout = 30 * time.Second

// persistFetchedMatch est le point d'entrée d'écriture per-match du live sync :
// enrichissement registry (résolution des noms d'assets) puis submitMatchAsBatch
// (INSERT-only). Appelé depuis la boucle d'insertion de SyncEngine.run().
func (e *SyncEngine) persistFetchedMatch(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	// Résolution des noms d'assets (playlist/map/pair/game_variant) depuis
	// metadata.asset_translations AVANT l'écriture registry. Placé dans la phase
	// d'insert séquentielle (et non dans fetchMatchData parallèle) pour bénéficier
	// du dictionnaire fraîchement peuplé par le pré-pass de résolution d'assets.
	// Best-effort : metaDB nil ou asset absent → fallback historique (no-op).
	if fm != nil && fm.Registry != nil {
		if err := EnrichRegistryFromMetadata(ctx, e.metaDB, fm.Registry); err != nil {
			slog.WarnContext(ctx, "sync: EnrichRegistryFromMetadata non-bloquant",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
		}
	}
	return e.submitMatchAsBatch(ctx, sharedDB, playerDB, result, fm)
}

// submitMatchAsBatch est le chemin Phase 2.3 INSERT-only.
//
// Étapes :
//  1. buildBatchFromFetchedMatch → *persist.MatchBatch
//  2. SharedPersister.Persist (1 TX INSERT-only sur sharedDB)
//  3. PlayerPersister.Persist (1 TX INSERT-only sur playerDB)
//  4. Met à jour result.MatchesInserted + InsertedMatchIDs identique au legacy.
//
// PVE/Metadata hors scope Phase 2.3 (le live sync ne les écrit pas) :
// le batch contient les rows si fetchedMatch les a remplies, mais elles
// ne sont pas persistées ici — Phase 2.3.b ajoutera les Persisters PVE +
// Metadata avec leurs propres connexions DB.
func (e *SyncEngine) submitMatchAsBatch(
	ctx context.Context,
	sharedDB, playerDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	batch, parseErr := buildBatchFromFetchedMatchCtx(ctx, fm, e.titleSlug, e.gamertag, e.xuid)
	if parseErr != nil {
		// Parse highlight events échoué — non-bloquant, le batch est
		// quand même produit avec les autres rows.
		slog.WarnContext(ctx, "submitMatchAsBatch: parse highlight events warning",
			"match_id", fm.MatchID, "err", parseErr)
		result.AddWarning(fmt.Sprintf("highlight_events %s: %v", fm.MatchID, parseErr))
	}

	// Phase 4 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : sanitize NaN/Inf au
	// point de production. Defensif (queue.Submit re-sanitize aussi, mais
	// ici on garantit que le path SYNC sans queue est aussi protege).
	persist.SanitizeBatch(batch)

	// Phase 3 — parité legacy insertFetchedMatch : une erreur skill (GetMatchSkill
	// KO) devient un warning. Les colonnes skill restent NULL et le skill heal les
	// complétera ; les bits skill ne sont pas posés au collect dans ce cas
	// (buildBatchFromFetchedMatch : skillOK == false).
	if fm.SkillError != nil {
		result.AddWarning(fmt.Sprintf("skill %s: %v", fm.MatchID, fm.SkillError))
	}

	// Phase 5 PLAN_FIX_SYNC_RELIABILITY_2026-05-24 : pre-check idempotence
	// match_registry. Le SharedPersister no-op silently si le match_id
	// existe deja (cf. shared_persister.go:78), mais le compteur en aval
	// (result.MatchesInserted) etait incrementé optimistement → inflation
	// observee en prod 2026-05-24 (XxDaemonGamerxX 9 matchs inserted/cycle
	// alors qu'il n'a pas joue depuis 1 mois). Cascade : post-sync trigge
	// sur InsertedMatchIDs, re-download films → 285s perdues.
	//
	// On determine ICI si le match est reellement nouveau, et on
	// repercutera l'info plus bas (MatchesInserted vs MatchesSkipped +
	// InsertedMatchIDs filtre).
	matchAlreadyExists := false
	if sharedDB != nil && fm.MatchID != "" {
		var exists bool
		if err := sharedDB.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM match_registry WHERE match_id = ?)`,
			fm.MatchID,
		).Scan(&exists); err == nil {
			matchAlreadyExists = exists
		}
		// Erreur ignoree (tests minimaux : table absente) → on garde le
		// comportement legacy (compte comme insert).
	}

	// Chemin ASYNC (queue + worker) si batchQueue non-nil. Sinon SYNC direct.
	if e.batchQueue != nil {
		if err := e.batchQueue.Submit(batch); err != nil {
			observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_batch_submit_error")
			slog.ErrorContext(ctx, "submitMatchAsBatch: queue.Submit échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
			return fmt.Errorf("submitMatchAsBatch queue: %w", err)
		}
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_batch_submitted_total")
		// Le worker async persiste + ACK plus tard. Drain à la fin du cycle.
	} else {
		// Chemin SYNC (Phase 2.3) — pas de WAL, pas de worker, juste les
		// Persisters directs sur les connexions DB ouvertes par run().
		// Timeout par Persist : évite un hang infini si DuckDB est bloqué.
		sharedP := persist.NewSharedPersister(sharedDB)
		sharedCtx, sharedCancel := context.WithTimeout(ctx, persistTimeout)
		err := sharedP.Persist(sharedCtx, batch)
		sharedCancel()
		if err != nil {
			observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_shared_total_error")
			if ctxErr := sharedCtx.Err(); ctxErr != nil {
				observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_shared_total_timeout")
			}
			slog.ErrorContext(ctx, "submitMatchAsBatch: SharedPersister.Persist échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
			return fmt.Errorf("submitMatchAsBatch shared: %w", err)
		}
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_shared_total_ok")

		playerP := persist.NewPlayerPersister(playerDB)
		playerCtx, playerCancel := context.WithTimeout(ctx, persistTimeout)
		err = playerP.Persist(playerCtx, batch)
		playerCancel()
		if err != nil {
			observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_player_total_error")
			if ctxErr := playerCtx.Err(); ctxErr != nil {
				observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_player_total_timeout")
			}
			slog.ErrorContext(ctx, "submitMatchAsBatch: PlayerPersister.Persist échoué",
				"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
			return fmt.Errorf("submitMatchAsBatch player: %w", err)
		}
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_player_total_ok")
	}

	// Alias xuid→gamertag : plus d'écriture vers le store global xbox_aliases
	// (consolidé dans shared 2026-06-19). Les gamertags des participants sont
	// persistés en shared.match_participants par le SharedPersister ci-dessus
	// (lus par v_gamertag_lookup) ; le chemin convergent upserte shared.xuid_aliases.

	// Métriques : aligne avec le legacy pour ne pas casser les compteurs.
	// Phase 5 : MatchesInserted ne compte que les matchs REELLEMENT nouveaux
	// (pre-check match_registry plus haut). InsertedMatchIDs idem — c'est
	// cette liste qui trigge le post-sync (films, perf scores, etc.) donc
	// elle DOIT refleter les vrais inserts pour ne pas gaspiller 285s/cycle
	// sur des dupes.
	if len(fm.Participants) > 0 {
		result.ParticipantsDone += len(fm.Participants)
	}
	if len(fm.Medals) > 0 {
		result.MedalsInserted += len(fm.Medals)
	}
	if matchAlreadyExists {
		result.MatchesSkipped++
		slog.DebugContext(ctx, "submitMatchAsBatch: match deja en registry — Submit pour idempotence, pas compte en inserted",
			"gamertag", e.gamertag, "match_id", fm.MatchID)
	} else {
		result.MatchesInserted++
		result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)
		observability.IncCounterT(ctxkeys.TitleSlug(ctx), "persist_batch_committed_total")
	}

	slog.InfoContext(ctx, "submitMatchAsBatch: match persisté",
		"gamertag", e.gamertag, "match_id", fm.MatchID,
		"participants", len(fm.Participants), "medals", len(fm.Medals),
	)
	return nil
}
