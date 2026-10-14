// Package sync — csr_backfill.go : backfill récurrent des CSR par-match.
//
// Le sync nominal écrit déjà les CSR à mesure que les matchs ranked sont
// synchronisés (cf. csr_writes.go + hooks dans engine.go). Ce loader est
// l'outil de rattrapage pour les matchs ranked qui étaient déjà en DB
// AVANT la Phase B (ou pour les cas où l'API skill a renvoyé un payload
// partiel sans RankRecap au moment du sync initial).
//
// Flow :
//  1. Lire les match_ids classés depuis match_registry (sharedDB).
//  2. Si !force : exclure ceux déjà présents en match_skill_rank avec
//     rating_type='CSR' (idempotence).
//  3. Pour chaque match restant : re-fetch GetMatchSkill avec le xuid du
//     joueur, extraire la row CSR via ExtractCSRRowIfRanked, upsert.
//  4. Rate-limiting délégué au client Halo (déjà géré dans doGet).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// CSRBackfillResult résume l'exécution d'un BackfillCSRFromAPI : utile pour
// affichage côté handler /backfill/start et logs structurés.
type CSRBackfillResult struct {
	// RankedMatches : nombre total de matchs classés dans le registry.
	RankedMatches int
	// AlreadyHadCSR : matchs sautés parce qu'une row CSR existait déjà
	// (uniquement pertinent en mode !force).
	AlreadyHadCSR int
	// Fetched : matchs pour lesquels on a appelé GetMatchSkill.
	Fetched int
	// Inserted : rows CSR effectivement insérées/remplacées.
	Inserted int
	// SkippedNoRankRecap : payload skill OK mais RankRecap absent (match
	// classé côté registry mais Microsoft n'a pas retourné de CSR — match
	// custom à playlist ranked-name, ou bug API ponctuel).
	SkippedNoRankRecap int
	// SkillErrors : appels GetMatchSkill qui ont échoué (réseau, 401, 404).
	// Non-bloquants : on continue le batch.
	SkillErrors int
}

// BackfillCSRFromAPI rattrape les CSR par-match manquants pour un joueur.
// playerDB / sharedDB : ouvertes en écriture. xuid : le joueur cible.
// force : si true, ignore le filtre d'idempotence et re-fetche TOUS les
// matchs ranked (utile si le format du payload change ou si on veut tester
// que les valeurs Microsoft n'ont pas drift).
func BackfillCSRFromAPI(
	ctx context.Context,
	client HaloClient,
	playerDB, sharedDB *sql.DB,
	xuid string,
	force bool,
) (CSRBackfillResult, error) {
	var res CSRBackfillResult

	matches, err := loadRankedMatchesForCSRBackfill(ctx, sharedDB)
	if err != nil {
		return res, fmt.Errorf("BackfillCSRFromAPI: load ranked: %w", err)
	}
	res.RankedMatches = len(matches)
	if len(matches) == 0 {
		slog.InfoContext(ctx, "BackfillCSRFromAPI: aucun match classé en registry, rien à faire",
			"xuid", xuid)
		return res, nil
	}

	var existingCSR map[string]bool
	if !force {
		existingCSR, err = loadExistingRatingIDs(ctx, playerDB, "CSR")
		if err != nil {
			return res, fmt.Errorf("BackfillCSRFromAPI: load existing CSR: %w", err)
		}
	}

	slog.InfoContext(ctx, "BackfillCSRFromAPI: démarrage",
		"xuid", xuid,
		"ranked_total", res.RankedMatches,
		"already_csr", len(existingCSR),
		"force", force,
	)

	// Log de progression tous les progressEveryN matchs traités pour suivre
	// un long backfill (~500 matchs × 200ms ≈ 100s) sans noyer les logs.
	const progressEveryN = 50

	for _, m := range matches {
		if !force && existingCSR[m.MatchID] {
			res.AlreadyHadCSR++
			continue
		}

		// Respecter ctx (l'utilisateur peut annuler depuis le handler).
		if err := ctx.Err(); err != nil {
			slog.WarnContext(ctx, "BackfillCSRFromAPI: contexte annulé",
				"xuid", xuid, "progress", res.Fetched, "remaining", res.RankedMatches-res.Fetched-res.AlreadyHadCSR)
			return res, err
		}

		skillData, skillErr := client.GetMatchSkill(ctx, m.MatchID, []string{xuid})
		res.Fetched++
		if skillErr != nil {
			res.SkillErrors++
			slog.WarnContext(ctx, "BackfillCSRFromAPI: GetMatchSkill échoué",
				"match_id", m.MatchID, "xuid", xuid, "err", skillErr)
			continue
		}

		row := ExtractCSRRowIfRanked(&m, skillData[xuid])
		if row == nil {
			// match classé côté registry mais payload skill sans RankRecap
			// (cas rare, possible si playlist renommée ou bug API).
			res.SkippedNoRankRecap++
			slog.DebugContext(ctx, "BackfillCSRFromAPI: RankRecap absent dans le payload skill",
				"match_id", m.MatchID, "xuid", xuid)
			continue
		}

		if err := UpsertCSRRow(ctx, playerDB, row); err != nil {
			res.SkillErrors++
			slog.WarnContext(ctx, "BackfillCSRFromAPI: UpsertCSRRow échoué",
				"match_id", m.MatchID, "err", err)
			continue
		}
		res.Inserted++

		if res.Fetched%progressEveryN == 0 {
			slog.InfoContext(ctx, "BackfillCSRFromAPI: progression",
				"xuid", xuid,
				"fetched", res.Fetched,
				"inserted", res.Inserted,
				"skipped_no_recap", res.SkippedNoRankRecap,
				"errors", res.SkillErrors,
			)
		}
	}

	slog.InfoContext(ctx, "BackfillCSRFromAPI: terminé",
		"xuid", xuid,
		"ranked_total", res.RankedMatches,
		"already_csr", res.AlreadyHadCSR,
		"fetched", res.Fetched,
		"inserted", res.Inserted,
		"skipped_no_recap", res.SkippedNoRankRecap,
		"skill_errors", res.SkillErrors,
	)
	return res, nil
}

// loadRankedMatchesForCSRBackfill charge match_id + start_time des matchs CANDIDATS au
// rattrapage CSR depuis le registre partagé, triés chronologiquement ASC.
//
// Ce filtre n'est PAS la décision de classement : c'est un pré-filtre pour ne pas re-fetcher
// toute la base. Un match est candidat si is_ranked est posé OU si son nom de playlist ou de
// paire contient « ranked » OU si son playlist_id appartient à la référence des playlists
// classées (rankedCandidatePlaylistIDs : un match dont le nom n'a pas été résolu reste
// repêchable). is_ranked est faux sur la plupart des matchs depuis le 2026-05 (cf. plan
// ajustements 2026-10-14 G5). La DÉCISION appartient au payload skill
// (SkillShowsRankedMatch, via ExtractCSRRowIfRanked) : un candidat dont le RankRecap n'est pas
// celui d'un match classé est compté « sans RankRecap » et ne produit aucune ligne.
func loadRankedMatchesForCSRBackfill(ctx context.Context, sharedDB *sql.DB) ([]MatchRegistryRow, error) {
	ph, args := rankedCandidatePlaylistIDs()
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT match_id, start_time
		FROM match_registry
		WHERE start_time IS NOT NULL
		  AND (
		    COALESCE(is_ranked, FALSE) = TRUE
		    OR STRPOS(LOWER(COALESCE(playlist_name, '')), 'ranked') > 0
		    OR STRPOS(LOWER(COALESCE(pair_name, '')), 'ranked') > 0
		    OR LOWER(COALESCE(playlist_id, '')) IN (`+ph+`)
		  )
		ORDER BY start_time ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MatchRegistryRow
	for rows.Next() {
		var id string
		var start time.Time
		if err := rows.Scan(&id, &start); err != nil {
			return nil, err
		}
		out = append(out, MatchRegistryRow{
			MatchID:   id,
			StartTime: start,
		})
	}
	return out, rows.Err()
}

// rankedCandidatePlaylistIDs rend les identifiants de la référence des playlists classées, sous
// forme de marqueurs « ?, ?, ... » et d'arguments, pour le pré-filtre de CANDIDATURE des
// rattrapages CSR. Elle ne décide JAMAIS du classement d'un match (le payload skill décide,
// SkillShowsRankedMatch) : un candidat sans RankRecap de match classé ne produit aucune ligne.
func rankedCandidatePlaylistIDs() (string, []any) {
	all := rankedplaylists.All()
	args := make([]any, 0, len(all))
	for _, p := range all {
		args = append(args, strings.ToLower(strings.TrimSpace(p.AssetID)))
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(args)), ","), args
}
