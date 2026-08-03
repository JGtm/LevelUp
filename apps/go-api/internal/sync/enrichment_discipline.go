package sync

import (
	"context"
	"database/sql"
	"fmt"
)

// computeAndPersistH5DisciplineAwards calcule les awards de DISCIPLINE du joueur
// (suicides + trahisons) depuis le SHARED et les écrit dans personal_score_awards
// (player DB). C'est la parité H5 de ce qu'Infinite obtient de l'API PersonalScores :
// la synthèse « fun stats » (TotalSuicides/TotalBetrayals) lit ces award_name
// (self_destruction / betrayed_player) sans aucun changement de surface.
//
// ⚠️ H5-ONLY. Appelé uniquement depuis l'enrich H5 (cmd/h5-enrich + livesync H5,
// les 2 seuls callers de BackfillEnrichmentFromShared). NE JAMAIS câbler pour un
// titre à PSA NATIF (Infinite) : ça DOUBLE-COMPTERAIT les trahisons (PSA de l'API
// + ce calcul). Infinite passe par le sync engine, pas par cette fonction.
//
// Sources SHARED (le carnage H5 n'expose pas de PSA natif) :
//   - suicide  : killer_victim_pairs où killer_xuid = victim_xuid = owner
//     (DeathDisposition 0 ; le kill « soi → soi » forme une paire killer==victim).
//   - trahison : killer_xuid = owner, victim ≠ owner, MÊME équipe (join match_participants).
//     H5 n'a pas de code disposition « trahison » → inférée par l'équipe.
//
// Idempotent + ART-safe : INSERT pur tagué d'une NOUVELLE génération (psa_generation_seq) ;
// la vue personal_score_awards_latest ne lit que la génération MAX par (match_id, xuid),
// donc un re-run supersède proprement — aucun DELETE/UPDATE indexé (ADR 0026).
//
// Self-skip (0, nil) si le schéma PSA append-only est absent (player DB legacy).
// Retourne le nombre de lignes PSA insérées.
func computeAndPersistH5DisciplineAwards(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (int, error) {
	if xuid == "" || !psaAppendOnlyReady(ctx, playerDB) {
		return 0, nil
	}

	// Source canonique depuis le 2026-08-03. `SUM(kill_count)` devient `COUNT(*)` : la table
	// est un JOURNAL (1 ligne = 1 mort), et l'ancienne somme comptait des doublons — un
	// suicide ou une trahison unique pouvait être crédité deux fois. Les jointures sur
	// `match_participants` écartent les bots d'elles-mêmes (ils n'y figurent pas).
	suicides, err := scanMatchCounts(ctx, sharedDB, `
		SELECT match_id, CAST(COUNT(*) AS INTEGER)
		FROM match_kill_events_latest
		WHERE feed_killer_xuid = ? AND victim_xuid = ?
		GROUP BY match_id`, xuid, xuid)
	if err != nil {
		return 0, fmt.Errorf("suicides query: %w", err)
	}
	betrayals, err := scanMatchCounts(ctx, sharedDB, `
		SELECT kvp.match_id, CAST(COUNT(*) AS INTEGER)
		FROM match_kill_events_latest kvp
		JOIN match_participants mk ON mk.match_id = kvp.match_id AND mk.xuid = kvp.feed_killer_xuid
		JOIN match_participants mv ON mv.match_id = kvp.match_id AND mv.xuid = kvp.victim_xuid
		WHERE kvp.feed_killer_xuid = ? AND kvp.victim_xuid <> ? AND mk.team_id = mv.team_id
		GROUP BY kvp.match_id`, xuid, xuid)
	if err != nil {
		return 0, fmt.Errorf("betrayals query: %w", err)
	}
	if len(suicides) == 0 && len(betrayals) == 0 {
		return 0, nil
	}

	// Une génération partagée pour ce run : la vue _latest supersède par (match, xuid).
	var gen int64
	if err := playerDB.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
		return 0, fmt.Errorf("psa generation: %w", err)
	}
	insert := func(matchID, award string, count int) error {
		_, e := playerDB.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
			VALUES (?, ?, ?, 'penalty', ?, 0, ?)`,
			matchID, xuid, award, count, gen)
		return e
	}

	n := 0
	for matchID, c := range suicides {
		if c <= 0 {
			continue
		}
		if err := insert(matchID, "self_destruction", c); err != nil {
			return n, fmt.Errorf("insert self_destruction %s: %w", matchID, err)
		}
		n++
	}
	for matchID, c := range betrayals {
		if c <= 0 {
			continue
		}
		if err := insert(matchID, "betrayed_player", c); err != nil {
			return n, fmt.Errorf("insert betrayed_player %s: %w", matchID, err)
		}
		n++
	}
	return n, nil
}

// psaAppendOnlyReady : true si personal_score_awards existe AVEC la colonne
// generation_id (schéma append-only requis par l'INSERT taggé génération). Une
// player DB legacy sans ce schéma → l'étape se skip proprement.
func psaAppendOnlyReady(ctx context.Context, db *sql.DB) bool {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'personal_score_awards' AND column_name = 'generation_id'`).Scan(&n)
	return err == nil && n > 0
}

// scanMatchCounts exécute une requête projetant (match_id VARCHAR, n INTEGER) et
// retourne la map match_id → n.
func scanMatchCounts(ctx context.Context, db *sql.DB, query string, args ...any) (map[string]int, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var matchID string
		var n int
		if err := rows.Scan(&matchID, &n); err != nil {
			return nil, err
		}
		out[matchID] = n
	}
	return out, rows.Err()
}
