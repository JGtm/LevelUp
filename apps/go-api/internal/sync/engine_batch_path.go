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
// **Activation** : via WithBatchPersistMode(true) sur le SyncEngine —
// l'orchestrateur cmd/server / scheduler peut activer le flag selon
// LEVELUP_PERSIST_BATCH=1 ou tout autre critère.
//
// **Coexistence** : tant que batchMode=false (défaut), le chemin legacy
// insertFetchedMatch reste utilisé. submitOrInsertMatch est le point de
// branchement unique.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// submitOrInsertMatch est le point de branchement unique entre le chemin
// legacy (insertFetchedMatch) et le chemin Collect→Persist (submitMatchAsBatch).
// Appelé depuis la boucle d'insertion de SyncEngine.run().
func (e *SyncEngine) submitOrInsertMatch(
	ctx context.Context,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	if e.batchMode {
		return e.submitMatchAsBatch(ctx, sharedDB, playerDB, globalDB, result, fm)
	}
	return e.insertFetchedMatch(ctx, sharedDB, playerDB, globalDB, result, fm)
}

// submitMatchAsBatch est le chemin Phase 2.3 INSERT-only.
//
// Étapes :
//  1. buildBatchFromFetchedMatch → *persist.MatchBatch
//  2. SharedPersister.Persist (1 TX INSERT-only sur sharedDB)
//  3. PlayerPersister.Persist (1 TX INSERT-only sur playerDB)
//  4. UpsertXUIDAlias dans globalDB (legacy, hors batch — la global DB
//     n'est pas couverte par les Persisters pour l'instant)
//  5. Met à jour result.MatchesInserted + InsertedMatchIDs identique au legacy.
//
// PVE/Metadata hors scope Phase 2.3 (le live sync ne les écrit pas) :
// le batch contient les rows si fetchedMatch les a remplies, mais elles
// ne sont pas persistées ici — Phase 2.3.b ajoutera les Persisters PVE +
// Metadata avec leurs propres connexions DB.
func (e *SyncEngine) submitMatchAsBatch(
	ctx context.Context,
	sharedDB, playerDB, globalDB *sql.DB,
	result *domain.SyncResult,
	fm *fetchedMatch,
) error {
	batch, parseErr := buildBatchFromFetchedMatch(fm, e.titleSlug, e.gamertag, e.xuid)
	if parseErr != nil {
		// Parse highlight events échoué — non-bloquant, le batch est
		// quand même produit avec les autres rows.
		slog.WarnContext(ctx, "submitMatchAsBatch: parse highlight events warning",
			"match_id", fm.MatchID, "err", parseErr)
		result.AddWarning(fmt.Sprintf("highlight_events %s: %v", fm.MatchID, parseErr))
	}

	sharedP := persist.NewSharedPersister(sharedDB)
	if err := sharedP.Persist(ctx, batch); err != nil {
		slog.ErrorContext(ctx, "submitMatchAsBatch: SharedPersister.Persist échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
		return fmt.Errorf("submitMatchAsBatch shared: %w", err)
	}

	playerP := persist.NewPlayerPersister(playerDB)
	if err := playerP.Persist(ctx, batch); err != nil {
		slog.ErrorContext(ctx, "submitMatchAsBatch: PlayerPersister.Persist échoué",
			"gamertag", e.gamertag, "match_id", fm.MatchID, "err", err)
		return fmt.Errorf("submitMatchAsBatch player: %w", err)
	}

	// xuid_aliases global DB — non couvert par les Persisters (DB séparée).
	// Comportement identique au legacy : best-effort upsert pour chaque
	// participant ayant un gamertag.
	if globalDB != nil {
		for _, p := range fm.Participants {
			if p.Gamertag != nil && *p.Gamertag != "" {
				_ = UpsertXUIDAlias(ctx, globalDB, p.XUID, *p.Gamertag)
			}
		}
	}

	// Métriques : aligne avec le legacy pour ne pas casser les compteurs.
	if len(fm.Participants) > 0 {
		result.ParticipantsDone += len(fm.Participants)
	}
	if len(fm.Medals) > 0 {
		result.MedalsInserted += len(fm.Medals)
	}
	result.MatchesInserted++
	result.InsertedMatchIDs = append(result.InsertedMatchIDs, fm.MatchID)

	slog.InfoContext(ctx, "submitMatchAsBatch: match persisté",
		"gamertag", e.gamertag, "match_id", fm.MatchID,
		"participants", len(fm.Participants), "medals", len(fm.Medals),
	)
	return nil
}
