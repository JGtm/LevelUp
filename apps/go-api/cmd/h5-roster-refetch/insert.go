package main

// insert.go — résolution d'identité LOCALE + INSERT-only ART-safe des participants
// top-up. Isolé de main.go : makeResolver est PUR (testable sans DB/réseau).

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// makeResolver compose le résolveur xuid passé à mapCarnageParticipants :
//
//	(1) idMap : kill-feed LOCAL du match (gamertag→xuid). Autoritatif pour les résidus.
//	(2) universal : fallback résolveur worldenrich (nil si --resolver OFF).
//	(3) "" : non résolu → mapCarnageParticipants SKIP le joueur (resolve-or-skip).
//
// PUR (aucun accès DB/réseau dans la fonction renvoyée au-delà du universal injecté) →
// unit-testable. Un gamertag vide retourne "" sans consulter le fallback.
func makeResolver(idMap map[string]string, universal func(string) string) func(string) string {
	return func(gt string) string {
		if gt == "" {
			return ""
		}
		if x, ok := idMap[gt]; ok && x != "" {
			return x // (1) kill-feed local
		}
		if universal != nil {
			if x := universal(gt); x != "" {
				return x // (2) résolveur universel (worldenrich)
			}
		}
		return "" // (3) skip
	}
}

// insertParticipant — INSERT-only avec garde NOT EXISTS sur (match_id, xuid). La
// sous-requête WHERE NOT EXISTS rend l'INSERT idempotent SANS ON CONFLICT (pas de
// write-path UPDATE sur l'index PK → ART-safe). Liste EXACTE des 41 colonnes de
// match_participants (cf. persist/shared_persister.go::persistParticipants), peuplées
// depuis la carnage via le mapping live (domain.MatchParticipantRow). Retourne le nombre
// de lignes réellement insérées (0 si la garde a sauté la ligne — déjà présente).
//
// Colonnes PEUPLÉES depuis la carnage (mapCarnageParticipants) : match_id, xuid,
// gamertag, team_id, outcome, rank, score, kills, deaths, assists, shots_fired,
// shots_hit, damage_dealt, kda, personal_score, time_played_seconds, avg_life_seconds,
// headshot_kills, grenade_kills, melee_kills, power_weapon_kills, assassination_kills,
// ground_pound_kills, shoulder_bash_kills, created_at=now.
//
// Colonnes laissées NULL (non fournies par l'API carnage h5 — JAMAIS fabriquées) :
// accuracy, damage_taken (absents de l'API h5) ; kills_expected, deaths_expected,
// kills_stddev, deaths_stddev, team_mmr, enemy_mmr (MMR/skill non porté par la carnage
// h5) ; max_killing_spree ; backfill_bits (aucun bit ne signifie « top-up carnage » ; la
// traçabilité de l'origine est portée par xuid_aliases.source).
//
// present_at_beginning / present_at_completion / joined_in_progress / left_in_progress /
// first_joined_time / last_leave_time : laissés NULL (le mapping carnage ne les renseigne
// pas — les participants H5 D'ORIGINE ont aussi ces colonnes à NULL). Cohérent avec les
// originaux → la fonction LUSR concurrentTeamSize retombe uniformément sur len(team) pour
// les deux équipes (mélanger originaux NULL + reconstruits TRUE casserait le compte).
func insertParticipant(ctx context.Context, tx *sql.Tx, r domain.MatchParticipantRow, now time.Time) (int, error) {
	const q = `
		INSERT INTO match_participants (
			match_id, xuid, gamertag,
			team_id, outcome, rank, score,
			kills, deaths, assists,
			shots_fired, shots_hit,
			damage_dealt, damage_taken,
			kda, accuracy, personal_score,
			time_played_seconds, avg_life_seconds,
			kills_expected, deaths_expected, kills_stddev, deaths_stddev,
			team_mmr, enemy_mmr,
			headshot_kills,
			max_killing_spree, grenade_kills, melee_kills, power_weapon_kills,
			assassination_kills, ground_pound_kills, shoulder_bash_kills,
			present_at_beginning, present_at_completion, joined_in_progress, left_in_progress,
			first_joined_time, last_leave_time,
			backfill_bits,
			created_at
		)
		SELECT
			?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?,
			?, NULL,
			?, NULL, ?,
			?, ?,
			NULL, NULL, NULL, NULL,
			NULL, NULL,
			?,
			NULL, ?, ?, ?,
			?, ?, ?,
			NULL, NULL, NULL, NULL,
			NULL, NULL,
			NULL,
			?
		WHERE NOT EXISTS (
			SELECT 1 FROM match_participants WHERE match_id = ? AND xuid = ?
		)`
	res, err := tx.ExecContext(ctx, q,
		r.MatchID, r.XUID, r.Gamertag,
		r.TeamID, r.Outcome, r.Rank, r.Score,
		r.Kills, r.Deaths, r.Assists,
		r.ShotsFired, r.ShotsHit,
		r.DamageDealt,
		r.KDA, r.PersonalScore,
		r.TimePlayedSeconds, r.AvgLifeSeconds,
		r.HeadshotKills,
		r.GrenadeKills, r.MeleeKills, r.PowerWeaponKills,
		r.AssassinationKills, r.GroundPoundKills, r.ShoulderBashKills,
		now,
		r.MatchID, r.XUID,
	)
	if err != nil {
		return 0, fmt.Errorf("INSERT match_participants %s/%s: %w", r.MatchID, r.XUID, err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// insertAlias — INSERT OR IGNORE dans xuid_aliases (xuid PK), source='roster_refetch'
// pour tracer l'origine. INSERT OR IGNORE ne touche pas l'index en UPDATE (insert pur,
// ignoré si la PK existe) — ART-safe.
func insertAlias(ctx context.Context, tx *sql.Tx, r domain.MatchParticipantRow, now time.Time) error {
	if r.XUID == "" {
		return nil
	}
	var gt any
	if r.Gamertag != nil && *r.Gamertag != "" {
		gt = *r.Gamertag
	}
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO xuid_aliases (xuid, gamertag, last_seen, source, updated_at)
		VALUES (?, ?, ?, 'roster_refetch', ?)`,
		r.XUID, gt, now, now)
	if err != nil {
		return fmt.Errorf("INSERT xuid_aliases %s: %w", r.XUID, err)
	}
	return nil
}
