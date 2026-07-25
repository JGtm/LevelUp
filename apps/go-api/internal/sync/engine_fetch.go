// Package sync — engine_fetch.go : pipeline parallèle fetch + insert séquentiel.
//
// Extrait de engine.go (refactor 2026-05-21). Regroupe :
//   - fetchedMatch : container des données extraites d'un GetMatchStats prêtes
//     pour insertion (registry, participants+skill, medals, PSA, highlight chunk).
//   - fetchMatchData : exécute fetch + extraction d'un match (pur, sans DB).
//     Appelé en parallèle par run() via errgroup (Phase 2 — RPS limité par
//     HaloAPIClient).
//   - hasAnyTeamMMR : helper de décision des bits skill, consommé par le collect
//     (collect.go) lors de la construction du MatchBatch.
//
// Le découpage fetch/insert permet de paralléliser les fetches tout en gardant
// les inserts séquentiels (order-preserving, évite races sur les UPSERT shared).
// Comportement INCHANGÉ — pur déplacement.
//
// Voir engine.go (struct SyncEngine + run()) pour le contexte.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/objective"
)

// fetchedMatch contient les données extraites d'un GetMatchStats, prêtes pour
// conversion en MatchBatch (chemin unique Collect→Persist).
//
// Chemin unique : buildBatchFromFetchedMatch(Ctx) convertit ce type en
// *persist.MatchBatch, submit dans la BatchQueue (INSERT-only, anti-ART ADR 0019).
// Le chemin legacy insertFetchedMatch (écriture directe dans les DBs) a été
// supprimé au lot D1b (audits 2026-07) — plus aucun écrivain per-match direct.
type fetchedMatch struct {
	MatchID        string
	Registry       *MatchRegistryRow
	Participants   []ParticipantRow
	Medals         []MedalRow
	PSA            []PersonalScoreAwardRow // PersonalScores du joueur courant (player DB)
	HighlightData  []byte                  // Raw highlight events chunk (ou nil si absent)
	FilmMajorVer   int
	HasHighlights  bool
	HighlightError error // Non-bloquant si présent
	SkillError     error // Non-bloquant si présent
	// CSRRow : ligne CSR à insérer côté player DB. Renseignée uniquement
	// pour les matchs classés dont le payload skill contient RankRecap.
	// Inséré via batch.PlayerData.SkillRank (chemin Collect→Persist).
	CSRRow *MatchCSRRow

	// SharedCSRs : CSR de TOUS les participants ranked du match (lobby
	// context). Produit par ExtractAllSharedCSRRows à partir de skillData,
	// utilisé par batch.Shared.MatchCSRs. Vide si match non-ranked ou
	// skillData absent. Cf. ADR/csr_shared_writes.go.
	SharedCSRs []SharedMatchCSRRow

	// PveStats : stats Firefight pour TOUS les participants du match
	// (1 row par joueur). Produit par ExtractPveStats si le match est
	// firefight (GameVariantCategory 41/42). Vide sinon.
	// Utilisé par batch.PVE.Stats (slice).
	PveStats []PveMatchStatsRow

	// ObjectiveStats : stats objectifs (CTF/Zones/Oddball) pour tous les
	// participants d'un match à objectif (1 row par joueur ayant un bloc). Produit
	// par ExtractObjectiveStats sous opts.WithObjectiveStats. Vide pour Slayer.
	// Utilisé par batch.Shared.ObjectiveStats.
	ObjectiveStats []persist.ObjectiveStatsInsert
}

// fetchMatchData exécute le fetch et l'extraction pour un match (pur, sans DB).
// Retourne les données extraites prêtes pour insertion séquentielle.
func (e *SyncEngine) fetchMatchData(
	ctx context.Context,
	client HaloClient,
	matchID string,
	opts domain.SyncOptions,
) (*fetchedMatch, error) {
	matchJSON, err := client.GetMatchStats(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "sync: GetMatchStats échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("GetMatchStats: %w", err)
	}

	fm := &fetchedMatch{
		MatchID: matchID,
	}

	// Extract registry (obligatoire).
	reg, err := ExtractRegistry(matchJSON, e.gamertag)
	if err != nil {
		slog.WarnContext(ctx, "sync: ExtractRegistry échoué",
			"gamertag", e.gamertag, "match_id", matchID, "err", err,
		)
		return nil, fmt.Errorf("ExtractRegistry: %w", err)
	}
	fm.Registry = reg

	// Extract optionnels.
	if opts.WithParticipants {
		fm.Participants = ExtractParticipants(matchJSON)

		// Garantir gamertag sur la row du joueur synchronisé : l'API renvoie
		// parfois Gamertag/PlayerName vide pour le joueur appelant.
		ensureGamertagForSelf(fm.Participants, e.xuid, e.gamertag)

		// Skill API : team_mmr, enemy_mmr, kills/deaths_expected.
		// Endpoint séparé du stats — non-bloquant : un échec produit un warning.
		if xuids := ParticipantXUIDs(fm.Participants); len(xuids) > 0 {
			skillData, skillErr := client.GetMatchSkill(ctx, matchID, xuids)
			if skillErr != nil {
				fm.SkillError = fmt.Errorf("GetMatchSkill: %w", skillErr)
			} else if len(skillData) > 0 {
				fm.Participants = MergeSkillIntoParticipants(fm.Participants, skillData)
				// CSR par-match (player DB) : extraction depuis RankRecap si
				// match classé. L'écriture en player DB est différée au batch
				// (batch.PlayerData.SkillRank, chemin Collect→Persist).
				fm.CSRRow = ExtractCSRRowIfRanked(fm.Registry, skillData[e.xuid])
				// CSR de tous les participants ranked (shared.match_csrs)
				// — lobby context, utilisé par batch.Shared.MatchCSRs.
				fm.SharedCSRs = ExtractAllSharedCSRRows(fm.Registry, skillData)
			}
		}
	}
	if opts.WithMedals {
		fm.Medals = ExtractMedals(matchJSON)
	}
	// PVE Firefight stats — extraits si le match est firefight (déterminé
	// dans la registry). Tous les participants ; le batch écrit dans
	// shared_pve.pve_match_stats PK (match_id, xuid). Cf. batch.PVE.Stats.
	if fm.Registry != nil && fm.Registry.IsFirefight {
		fm.PveStats = ExtractPveStats(matchID, matchJSON)
	}
	// Objectif (CTF/Zones/Oddball) — extraits du même payload participants (aucun
	// appel réseau). ExtractObjectiveStats ne renvoie que les joueurs d'un match à
	// objectif ; vide pour Slayer/Firefight. Écrit dans shared.match_objective_stats.
	if opts.WithObjectiveStats {
		fm.ObjectiveStats = objective.ExtractObjectiveStats(matchID, matchJSON)
	}
	// PersonalScores du joueur courant — toujours extraits (pas de flag dédié,
	// même cycle de vie que les participants). La table n'est pas dans shared :
	// l'insertion se fera côté playerDB via le batch (batch.PlayerData).
	fm.PSA = ExtractPersonalScoreAwards(matchJSON, matchID, e.xuid)
	if opts.WithHighlightEvents {
		// Attente bornée du film pour un match FRAIS (cf. fetchHighlightChunkResilient) :
		// garantit que les events (→ killer_victim → frags par arme) sont récupérés
		// dans le sync qui découvre le match, plutôt qu'un affichage à moitié rattrapé
		// par la convergence un cycle plus tard. startTime via la registry déjà extraite.
		var startTime time.Time
		if fm.Registry != nil {
			startTime = fm.Registry.StartTime
		}
		data, filmMajorVer, found, err := fetchHighlightChunkResilient(ctx, client, matchID, startTime)
		fm.HasHighlights = found
		fm.FilmMajorVer = filmMajorVer
		if err != nil {
			fm.HighlightError = fmt.Errorf("GetHighlightEventsChunk: %w", err)
		} else if found {
			fm.HighlightData = data
		}
	}

	return fm, nil
}

// hasAnyTeamMMR retourne true si au moins un participant a team_mmr renseigné.
// Utilisé par le collect (collect.go) pour décider si les bits skill doivent être
// posés (skillOK) au moment de construire le MatchBatch.
func hasAnyTeamMMR(parts []ParticipantRow) bool {
	for _, p := range parts {
		if p.TeamMMR != nil {
			return true
		}
	}
	return false
}
