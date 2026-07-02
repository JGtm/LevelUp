// Package sync — engine_v2bridge.go : wrappers exportés pour le pipeline
// V2 (cf. internal/sync/v2/, ADR 0027).
//
// Ce fichier est l'UNIQUE point de contact entre V1 et V2 côté V1. Il
// expose 2 fonctions wrapper qui délèguent à la logique interne sans
// la modifier. Conçu pour être supprimé en D8 cleanup (rm engine_v2bridge.go +
// internal/sync/v2/).
//
// Règle : pas de logique métier ici. Uniquement de la délégation pure.
// Si un wrapper a besoin de plus de 3 instructions, c'est un signe qu'il
// faut refactorer (extraire un helper réutilisable + faire le wrapper en
// 1 ligne) plutôt que d'ajouter de la complexité ici.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// RunPostSyncForV2 expose runPostSyncPipeline (méthode interne) au pipeline V2.
//
// Wrapper trivial : aucune logique ajoutée. La SyncEngine doit avoir été
// configurée via NewSyncEngine + WithCSRSeasonID + WithFriendsLoader +
// WithResolver + WithMediaScanHook (tous exportés).
//
// À supprimer en D8 cleanup quand V1 est désactivé.
func (e *SyncEngine) RunPostSyncForV2(
	ctx context.Context,
	playerDB, sharedDB *sql.DB,
	client HaloClient,
	insertedIDs []string,
) domain.PostSyncResult {
	// Compat V2 actuelle : le runner tient déjà le writer → pinned (byte-identique).
	// L'incrément 5 basculera le runner sur RunPostSyncForV2Shared (bursts).
	return e.runPostSyncPipeline(ctx, playerDB, NewPinnedSharedAccess(sharedDB), client, insertedIDs)
}

// RunPostSyncForV2Shared est la variante bursts paresseux (étape 1 contention) :
// le caller ne tient PAS de writer — le pipeline acquiert des bursts courts par
// écriture shared via le SharedAccess fourni. Cible du flip V2 (incrément 5).
func (e *SyncEngine) RunPostSyncForV2Shared(
	ctx context.Context,
	playerDB *sql.DB,
	shared *SharedAccess,
	client HaloClient,
	insertedIDs []string,
) domain.PostSyncResult {
	return e.runPostSyncPipeline(ctx, playerDB, shared, client, insertedIDs)
}

// BuildBatchFromRawForV2 construit un *persist.MatchBatch à partir des
// JSON raw stats + skill data déjà fetchés par V2. Délègue aux fonctions
// Extract* exportées + buildBatchFromFetchedMatchCtx interne.
//
// V2 a déjà fait l'I/O HTTP (Phase 3), cette fonction fait uniquement la
// transformation pure raw → MatchBatch. Inclut le sanitize NaN/Inf
// (Phase 4 plan reliability 2026-05-24).
//
// Paramètres :
//   - titleSlug, gamertag, xuid : contexte du joueur (canonical fetcher en V2)
//   - matchID : identifiant unique du match
//   - statsJSON : payload raw GetMatchStats
//   - skillData : payload raw GetMatchSkill (peut être nil/vide)
//   - highlightChunk, filmMajorVer, hasHighlights : payload raw
//     GetHighlightEventsChunk (peut être nil/0/false si non-fetché par V2)
//
// À supprimer en D8 cleanup.
func BuildBatchFromRawForV2(
	ctx context.Context,
	titleSlug, gamertag, xuid, matchID string,
	statsJSON map[string]any,
	skillData map[string]*MatchSkillData,
	highlightChunk []byte,
	filmMajorVer int,
	hasHighlights bool,
) (*persist.MatchBatch, error) {
	return BuildBatchFromRawForV2WithMeta(ctx, nil, titleSlug, gamertag, xuid, matchID,
		statsJSON, skillData, highlightChunk, filmMajorVer, hasHighlights)
}

// BuildBatchFromRawForV2WithMeta est identique à BuildBatchFromRawForV2 mais
// ENRICHIT le registry depuis metadata.asset_translations (résolution des noms
// d'assets restés en UUID brut) AVANT la construction du batch — le chemin V2
// n'enrichissait pas, écrivant donc l'UUID tel quel dans match_registry. metaDB
// est le handle metadata RW partagé ; nil → enrich no-op (parité legacy).
func BuildBatchFromRawForV2WithMeta(
	ctx context.Context,
	metaDB *sql.DB,
	titleSlug, gamertag, xuid, matchID string,
	statsJSON map[string]any,
	skillData map[string]*MatchSkillData,
	highlightChunk []byte,
	filmMajorVer int,
	hasHighlights bool,
) (*persist.MatchBatch, error) {
	fm := &fetchedMatch{MatchID: matchID}

	reg, err := ExtractRegistry(statsJSON, gamertag)
	if err != nil {
		return nil, fmt.Errorf("BuildBatchFromRawForV2 ExtractRegistry: %w", err)
	}
	// Enrich registry (UUID → nom) depuis asset_translations au primary write V2.
	// Best-effort : metaDB nil ou asset absent → fallback historique conservé.
	if eerr := EnrichRegistryFromMetadata(ctx, metaDB, reg); eerr != nil {
		slog.WarnContext(ctx, "BuildBatchFromRawForV2: enrich registry non-bloquant",
			"match_id", matchID, "err", eerr)
	}
	fm.Registry = reg

	fm.Participants = ExtractParticipants(statsJSON)
	ensureGamertagForSelf(fm.Participants, xuid, gamertag)

	if len(skillData) > 0 {
		fm.Participants = MergeSkillIntoParticipants(fm.Participants, skillData)
		fm.CSRRow = ExtractCSRRowIfRanked(fm.Registry, skillData[xuid])
		fm.SharedCSRs = ExtractAllSharedCSRRows(fm.Registry, skillData)
	}

	fm.Medals = ExtractMedals(statsJSON)
	if fm.Registry != nil && fm.Registry.IsFirefight {
		fm.PveStats = ExtractPveStats(matchID, statsJSON)
	}
	fm.PSA = ExtractPersonalScoreAwards(statsJSON, matchID, xuid)
	fm.HighlightData = highlightChunk
	fm.FilmMajorVer = filmMajorVer
	fm.HasHighlights = hasHighlights

	batch, parseErr := buildBatchFromFetchedMatchCtx(ctx, fm, titleSlug, gamertag, xuid)
	if parseErr != nil {
		// Non-fatal : parse highlight events warning, batch produit quand même.
		slog.WarnContext(ctx, "BuildBatchFromRawForV2: parse highlight events warning",
			"match_id", matchID, "err", parseErr)
	}
	if batch != nil {
		persist.SanitizeBatch(batch)
	}
	return batch, nil
}
