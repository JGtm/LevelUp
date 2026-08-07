// Package duckdb — repositories Prestige cross-joueurs (shared_social.duckdb).
//
// Implémente prestige.PrestigeRepo, prestige.SquadRepo, prestige.SquadChallengeRepo.
//
// Accès DB : lectures via les variantes `*Recovered` + translateSocialLockErr,
// écritures via execCheckpointed. Helpers et détail du bug : voir
// prestige_social_recovery.go (V721-08b).

package prestige

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// ─────────── PrestigeRepo ───────────

// PrestigeSocialRepo implémente prestige.PrestigeRepo.
type PrestigeSocialRepo struct{ db *duckdb.DB }

func NewPrestigeSocialRepo(db *duckdb.DB) *PrestigeSocialRepo { return &PrestigeSocialRepo{db: db} }

var _ prestige.PrestigeRepo = (*PrestigeSocialRepo)(nil)

// EmitEvent journalise un gain de PP ET incrémente le total du joueur de façon
// ATOMIQUE : l'INSERT dans prestige_events (journal) et l'UPSERT dans
// user_prestige (total) sont dans UNE seule transaction — un crash entre les
// deux ne peut plus laisser un événement journalisé sans bump du total. Toute la
// TX est sous WithReopenOnInvalidated (résilience connexion shared_social), suivie
// d'un CHECKPOINT immédiat (durabilité au restart, ADR 0022). Le current_level
// est laissé à 0 ici (recalculé côté service via LevelFromPP).
func (r *PrestigeSocialRepo) EmitEvent(ctx context.Context, ev prestige.PrestigeEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.db.WithReopenOnInvalidated(func() error {
		tx, txErr := r.db.SQLDb().BeginTx(ctx, nil)
		if txErr != nil {
			return txErr
		}
		if _, e := tx.ExecContext(ctx, `
			INSERT INTO prestige_events (id, user_id, title_slug, source_type, source_id, pp_amount, tier, created_at)
			VALUES (?,?,?,?,?,?,?,?)
		`, ev.ID, ev.UserID, ev.TitleSlug, ev.SourceType, duckdb.NullableStr(ev.SourceID),
			ev.PPAmount, duckdb.NullableStr(string(ev.Tier)), ev.CreatedAt); e != nil {
			_ = tx.Rollback()
			return e
		}
		// APPEND-ONLY : INSERT d'un snapshot du nouveau total (accumulation carry-forward
		// = total courant via _latest + delta), au lieu d'un ON CONFLICT DO UPDATE.
		// current_level laissé à 0 (recalculé côté service via LevelFromPP).
		if _, e := tx.ExecContext(ctx, `
			INSERT INTO user_prestige_history (user_id, title_slug, total_pp, current_level, updated_at, written_at)
			SELECT ?, ?,
			       COALESCE((SELECT total_pp FROM user_prestige_latest WHERE user_id = ? AND title_slug = ?), 0) + ?,
			       0, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		`, ev.UserID, ev.TitleSlug, ev.UserID, ev.TitleSlug, ev.PPAmount, ev.CreatedAt); e != nil {
			_ = tx.Rollback()
			return e
		}
		return tx.Commit()
	})
	if err != nil {
		return fmt.Errorf("PrestigeRepo.EmitEvent: %w", err)
	}
	_ = duckdb.CheckpointSharedSocial(ctx, r.db)
	return nil
}

func (r *PrestigeSocialRepo) GetUserPrestige(ctx context.Context, userID, titleSlug string) (prestige.UserPrestige, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var up prestige.UserPrestige
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT user_id, title_slug, total_pp, current_level, updated_at
		FROM user_prestige_latest WHERE user_id = ? AND title_slug = ?
	`, userID, titleSlug)
	if errors.Is(err, sql.ErrNoRows) {
		// Aucun PP émis (user, titre) : prestige vide, pas une erreur (inchangé).
		return prestige.UserPrestige{UserID: userID, TitleSlug: titleSlug}, nil
	}
	if err != nil {
		return up, translateSocialLockErr(ctx, err)
	}
	defer rows.Close() // curseur DÉJÀ positionné par QueryRowRecovered (pas de Next()).
	err = rows.Scan(&up.UserID, &up.TitleSlug, &up.TotalPP, &up.CurrentLevel, &up.UpdatedAt)
	return up, err
}

func (r *PrestigeSocialRepo) GetUserPrestigeCrossTitle(ctx context.Context, userID string) (prestige.UserPrestige, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var up prestige.UserPrestige
	up.UserID = userID
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT COALESCE(SUM(total_pp), 0), MAX(updated_at)
		FROM user_prestige_latest WHERE user_id = ?
	`, userID)
	if err != nil {
		// Agrégat sans GROUP BY : toujours 1 ligne, donc jamais sql.ErrNoRows ici.
		return up, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	// MAX(updated_at) est NULL quand le joueur n'a AUCUNE ligne de prestige (0 PP
	// émis, tous titres confondus) : l'agrégat rend bien une ligne, mais son 2e
	// champ est NULL. Le scanner directement dans un time.Time échouait
	// (« converting NULL to time.Time is unsupported ») → 500 sur
	// GET /prestige/me appelé sans title_slug pour tout joueur sans PP.
	// COALESCE côté SQL est exclu : il faudrait inventer une date sentinelle.
	// Contrat : joueur sans PP → prestige vide, UpdatedAt zéro, PAS d'erreur —
	// identique à GetUserPrestige (branche sql.ErrNoRows ci-dessus).
	var updatedAt sql.NullTime
	if err := rows.Scan(&up.TotalPP, &updatedAt); err != nil {
		return up, err
	}
	if updatedAt.Valid {
		up.UpdatedAt = updatedAt.Time
	}
	return up, nil
}

func (r *PrestigeSocialRepo) UpsertUserPrestige(ctx context.Context, up prestige.UserPrestige) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : INSERT d'un snapshot complet (plus d'ON CONFLICT DO UPDATE).
	return execCheckpointed(ctx, r.db, `
		INSERT INTO user_prestige_history (user_id, title_slug, total_pp, current_level, updated_at, written_at)
		VALUES (?, ?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
	`, up.UserID, up.TitleSlug, up.TotalPP, up.CurrentLevel, up.UpdatedAt)
}

func (r *PrestigeSocialRepo) ListEvents(ctx context.Context, userID, titleSlug string, since time.Time) ([]prestige.PrestigeEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT id, user_id, title_slug, source_type, COALESCE(source_id, ''),
		       pp_amount, COALESCE(tier, ''), created_at
		FROM prestige_events
		WHERE user_id = ? AND title_slug = ? AND created_at >= ?
		ORDER BY created_at DESC
	`, userID, titleSlug, since)
	if err != nil {
		return nil, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	var out []prestige.PrestigeEvent
	for rows.Next() {
		var ev prestige.PrestigeEvent
		var tier string
		if err := rows.Scan(&ev.ID, &ev.UserID, &ev.TitleSlug, &ev.SourceType,
			&ev.SourceID, &ev.PPAmount, &tier, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Tier = prestige.Tier(tier)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// GetLeaderboard rend le classement PP d'un ENSEMBLE de joueurs pour UN titre.
//
// titleSlug est OBLIGATOIRE (V721-14a / D-08) : la branche `titleSlug == nil`
// qui sommait `total_pp` tous titres confondus (`SUM … GROUP BY user_id`) a été
// supprimée le 2026-07-25. Elle n'avait aucun appelant de production et
// contredisait la décision produit du 2026-07-18 (arcs, défis et PP strictement
// indépendants par titre — cf. internal/prestige/no_cross_title_aggregation_test.go).
// Le type du paramètre (string, plus *string) rend la voie cross-titre
// inexprimable à la compilation. Un slug vide est refusé explicitement plutôt
// que dégradé en « tous titres ».
func (r *PrestigeSocialRepo) GetLeaderboard(
	ctx context.Context,
	userIDs []string,
	titleSlug string,
	since time.Time,
) ([]prestige.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if len(userIDs) == 0 {
		return nil, nil
	}
	if titleSlug == "" {
		return nil, fmt.Errorf("GetLeaderboard: title_slug requis (pas de classement tous titres confondus): %w",
			prestige.ErrInvalidInput)
	}
	// Construire les placeholders ?,?,?,...
	placeholders := strings.Repeat("?,", len(userIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(userIDs)+1)
	for _, uid := range userIDs {
		args = append(args, uid)
	}

	q := fmt.Sprintf(`
		SELECT user_id, title_slug, total_pp
		FROM user_prestige_latest
		WHERE user_id IN (%s) AND title_slug = ?
		ORDER BY total_pp DESC
	`, placeholders)
	args = append(args, titleSlug)

	rows, err := r.db.QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetLeaderboard: %w", translateSocialLockErr(ctx, err))
	}
	defer rows.Close()
	var out []prestige.LeaderboardEntry
	for rows.Next() {
		var e prestige.LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.TitleSlug, &e.TotalPP); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─────────── SquadRepo ───────────

// PrestigeSquadRepo implémente prestige.SquadRepo.
type PrestigeSquadRepo struct{ db *duckdb.DB }

func NewPrestigeSquadRepo(db *duckdb.DB) *PrestigeSquadRepo { return &PrestigeSquadRepo{db: db} }

var _ prestige.SquadRepo = (*PrestigeSquadRepo)(nil)

func (r *PrestigeSquadRepo) Create(ctx context.Context, s prestige.Squad) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return execCheckpointed(ctx, r.db,
		`INSERT INTO squad (id, name, created_by, created_at) VALUES (?, ?, ?, ?)`,
		s.ID, s.Name, s.CreatedBy, s.CreatedAt)
}

func (r *PrestigeSquadRepo) Get(ctx context.Context, id string) (prestige.Squad, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var s prestige.Squad
	rows, err := r.db.QueryRowRecovered(ctx,
		`SELECT id, name, created_by, created_at FROM squad WHERE id = ?`, id)
	if err != nil {
		// sql.ErrNoRows rendu TEL QUEL (mapping « introuvable » du service).
		return s, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	err = rows.Scan(&s.ID, &s.Name, &s.CreatedBy, &s.CreatedAt)
	return s, err
}

func (r *PrestigeSquadRepo) Rename(ctx context.Context, id, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// UPDATE basse fréquence (HTTP) sur la table create-only squad : pas de
	// pression concurrente → toléré (cf. doctrine append-only, sites HTTP). Sous
	// lease KindSharedSocial + CHECKPOINT (durabilité, ADR 0022).
	return execCheckpointed(ctx, r.db,
		`UPDATE squad SET name = ? WHERE id = ?`, name, id)
}

func (r *PrestigeSquadRepo) AddMember(ctx context.Context, m prestige.SquadMember) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : join = INSERT event is_member=TRUE dans squad_member_history.
	// gamertag = snapshot d'affichage (peut être vide pour un membre sans GT connu).
	return execCheckpointed(ctx, r.db,
		`INSERT INTO squad_member_history (squad_id, xuid, user_id, gamertag, is_member, joined_at)
		 VALUES (?, ?, ?, ?, TRUE, ?)`,
		m.SquadID, m.Xuid, m.UserID, duckdb.NullableStr(m.Gamertag), m.JoinedAt)
}

func (r *PrestigeSquadRepo) RemoveMember(ctx context.Context, squadID, xuid string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : leave = INSERT event is_member=FALSE (plus de DELETE).
	return execCheckpointed(ctx, r.db,
		`INSERT INTO squad_member_history (squad_id, xuid, is_member, joined_at)
		 VALUES (?, ?, FALSE, NULL)`, squadID, xuid)
}

func (r *PrestigeSquadRepo) ListMembers(ctx context.Context, squadID string) ([]prestige.SquadMember, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx,
		`SELECT squad_id, xuid, COALESCE(user_id, ''), COALESCE(gamertag, ''), joined_at
		 FROM squad_member_latest
		 WHERE squad_id = ? AND is_member = TRUE`, squadID)
	if err != nil {
		return nil, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	var out []prestige.SquadMember
	for rows.Next() {
		var m prestige.SquadMember
		if err := rows.Scan(&m.SquadID, &m.Xuid, &m.UserID, &m.Gamertag, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PrestigeSquadRepo) ListSquadsForUser(ctx context.Context, userID string) ([]prestige.Squad, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT s.id, s.name, s.created_by, s.created_at
		FROM squad s JOIN squad_member_latest sm ON sm.squad_id = s.id
		WHERE sm.user_id = ? AND sm.is_member = TRUE
		ORDER BY s.created_at DESC
	`, userID)
	if err != nil {
		return nil, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	var out []prestige.Squad
	for rows.Next() {
		var s prestige.Squad
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ─────────── SquadChallengeRepo ───────────

// PrestigeSquadChallengeRepo implémente prestige.SquadChallengeRepo.
type PrestigeSquadChallengeRepo struct{ db *duckdb.DB }

func NewPrestigeSquadChallengeRepo(db *duckdb.DB) *PrestigeSquadChallengeRepo {
	return &PrestigeSquadChallengeRepo{db: db}
}

var _ prestige.SquadChallengeRepo = (*PrestigeSquadChallengeRepo)(nil)

func (r *PrestigeSquadChallengeRepo) Create(ctx context.Context, sc prestige.SquadChallenge) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return execCheckpointed(ctx, r.db, `
		INSERT INTO squad_challenge (
			id, squad_id, template_id, title_slug, mode, eval_type,
			window_type, window_value, target_per_member, expires_at, created_by, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`, sc.ID, sc.SquadID, duckdb.NullableStr(sc.TemplateID), sc.TitleSlug,
		string(sc.Mode), string(sc.EvalType),
		string(sc.WindowType), sc.WindowValue, sc.TargetPerMember,
		sc.ExpiresAt, sc.CreatedBy, sc.CreatedAt)
}

func (r *PrestigeSquadChallengeRepo) Get(ctx context.Context, id string) (prestige.SquadChallenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var sc prestige.SquadChallenge
	var mode, evalType, windowType string
	rows, err := r.db.QueryRowRecovered(ctx, `
		SELECT id, squad_id, COALESCE(template_id, ''), title_slug, mode, eval_type,
		       window_type, COALESCE(window_value, ''), COALESCE(target_per_member, 0),
		       expires_at, created_by, created_at
		FROM squad_challenge WHERE id = ?
	`, id)
	if err != nil {
		// sql.ErrNoRows rendu TEL QUEL (mapping « introuvable » du service).
		return sc, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	if err := rows.Scan(&sc.ID, &sc.SquadID, &sc.TemplateID, &sc.TitleSlug, &mode, &evalType,
		&windowType, &sc.WindowValue, &sc.TargetPerMember,
		&sc.ExpiresAt, &sc.CreatedBy, &sc.CreatedAt); err != nil {
		return sc, err
	}
	sc.Mode = prestige.SquadMode(mode)
	sc.EvalType = prestige.EvalType(evalType)
	sc.WindowType = prestige.WindowType(windowType)
	return sc, nil
}

func (r *PrestigeSquadChallengeRepo) ListBySquad(ctx context.Context, squadID string) ([]prestige.SquadChallenge, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Défis ACTIFS uniquement (archived_at IS NULL) : un défi abandonné/archivé
	// sort de la liste (cycle de vie, Lot 3). L'archivage est un UPDATE de la
	// colonne non indexée archived_at → aucun risque ART.
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT id, squad_id, COALESCE(template_id, ''), title_slug, mode, eval_type,
		       window_type, COALESCE(window_value, ''), COALESCE(target_per_member, 0),
		       expires_at, created_by, created_at
		FROM squad_challenge WHERE squad_id = ? AND archived_at IS NULL ORDER BY created_at DESC
	`, squadID)
	if err != nil {
		return nil, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	var out []prestige.SquadChallenge
	for rows.Next() {
		var sc prestige.SquadChallenge
		var mode, evalType, windowType string
		if err := rows.Scan(&sc.ID, &sc.SquadID, &sc.TemplateID, &sc.TitleSlug,
			&mode, &evalType, &windowType, &sc.WindowValue, &sc.TargetPerMember,
			&sc.ExpiresAt, &sc.CreatedBy, &sc.CreatedAt); err != nil {
			return nil, err
		}
		sc.Mode = prestige.SquadMode(mode)
		sc.EvalType = prestige.EvalType(evalType)
		sc.WindowType = prestige.WindowType(windowType)
		out = append(out, sc)
	}
	return out, rows.Err()
}

// Archive marque un défi d'escouade comme archivé (abandon). UPDATE basse
// fréquence (HTTP) sur la colonne NON indexée archived_at de la table create-only
// squad_challenge : pas de pression concurrente ni de risque ART (mêmes garanties
// que Rename sur squad). Sous lease KindSharedSocial + CHECKPOINT (ADR 0022).
// Idempotent : ré-archiver un défi déjà archivé est un UPDATE sans effet.
func (r *PrestigeSquadChallengeRepo) Archive(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return execCheckpointed(ctx, r.db,
		`UPDATE squad_challenge SET archived_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
}

func (r *PrestigeSquadChallengeRepo) AddParticipant(ctx context.Context, p prestige.SquadChallengeParticipant) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : INSERT event idempotent. Si le participant existe déjà dans
	// _latest, on n'insère PAS (équivalent ON CONFLICT DO NOTHING) → un re-join ne
	// réinitialise pas la progression carry-forward.
	return execCheckpointed(ctx, r.db, `
		INSERT INTO squad_challenge_participant_history (
			squad_challenge_id, user_id, chosen_tier, data_tier,
			current_value, completed_at, is_private, joined_at, written_at
		)
		SELECT ?,?,?,?,?,?,?,?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		WHERE NOT EXISTS (
			SELECT 1 FROM squad_challenge_participant_latest
			WHERE squad_challenge_id = ? AND user_id = ?
		)
	`, p.SquadChallengeID, p.UserID, duckdb.NullableStr(string(p.ChosenTier)), string(p.DataTier),
		p.CurrentValue, p.CompletedAt, p.IsPrivate, p.JoinedAt,
		p.SquadChallengeID, p.UserID)
}

func (r *PrestigeSquadChallengeRepo) UpdateParticipantProgress(
	ctx context.Context,
	challengeID, userID string,
	value float64,
	completedAt *time.Time,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : INSERT…SELECT carry-forward des champs immuables (chosen_tier,
	// data_tier, is_private, joined_at) depuis _latest, current_value/completed_at
	// mis à jour. No-op si le participant n'existe pas (0 ligne) — comme l'UPDATE.
	return execCheckpointed(ctx, r.db, `
		INSERT INTO squad_challenge_participant_history (
			squad_challenge_id, user_id, chosen_tier, data_tier,
			current_value, completed_at, is_private, joined_at, written_at
		)
		SELECT squad_challenge_id, user_id, chosen_tier, data_tier,
		       ?, ?, is_private, joined_at, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		FROM squad_challenge_participant_latest
		WHERE squad_challenge_id = ? AND user_id = ?
	`, value, completedAt, challengeID, userID)
}

func (r *PrestigeSquadChallengeRepo) ListParticipants(ctx context.Context, challengeID string) ([]prestige.SquadChallengeParticipant, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT squad_challenge_id, user_id, COALESCE(chosen_tier, ''), data_tier,
		       current_value, completed_at, is_private, joined_at
		FROM squad_challenge_participant_latest WHERE squad_challenge_id = ?
	`, challengeID)
	if err != nil {
		return nil, translateSocialLockErr(ctx, err)
	}
	defer rows.Close()
	var out []prestige.SquadChallengeParticipant
	for rows.Next() {
		var p prestige.SquadChallengeParticipant
		var tier, dataTier string
		if err := rows.Scan(&p.SquadChallengeID, &p.UserID, &tier, &dataTier,
			&p.CurrentValue, &p.CompletedAt, &p.IsPrivate, &p.JoinedAt); err != nil {
			return nil, err
		}
		p.ChosenTier = prestige.Tier(tier)
		p.DataTier = prestige.DataTier(dataTier)
		out = append(out, p)
	}
	return out, rows.Err()
}
