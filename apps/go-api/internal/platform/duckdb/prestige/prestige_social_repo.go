// Package duckdb — repositories Prestige cross-joueurs (shared_social.duckdb).
//
// Implémente prestige.PrestigeRepo, prestige.SquadRepo, prestige.SquadChallengeRepo.

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

// execCheckpointed exécute une écriture mutative sur shared_social (avec reopen
// auto sur invalidation) PUIS flushe le WAL via CHECKPOINT immédiat (non-fatal).
//
// Les writes Prestige tournent déjà sous le lease KindSharedSocial (acquis par
// LazyPrestigeService pour le HTTP, tenu par le sync engine pour le post-sync) :
// le CHECKPOINT s'exécute donc sous le lease, sérialisé avec le sync engine.
// Sans lui, l'INSERT/UPDATE/DELETE reste dans le WAL et est perdu si la recovery
// #7659 quarantine un WAL orphelin au restart — même classe de bug que les
// mutations notifications (ADR 0022). CHECKPOINT non-fatal : la donnée est déjà
// commit, le scheduler 5 min fera fallback.
func execCheckpointed(ctx context.Context, db *duckdb.DB, query string, args ...any) error {
	if _, err := db.ExecRecovered(ctx, query, args...); err != nil {
		return err
	}
	_ = duckdb.CheckpointSharedSocial(ctx, db)
	return nil
}

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
			       0, ?, CURRENT_TIMESTAMP
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
	err := r.db.QueryRow(ctx, `
		SELECT user_id, title_slug, total_pp, current_level, updated_at
		FROM user_prestige_latest WHERE user_id = ? AND title_slug = ?
	`, userID, titleSlug).Scan(&up.UserID, &up.TitleSlug, &up.TotalPP, &up.CurrentLevel, &up.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return prestige.UserPrestige{UserID: userID, TitleSlug: titleSlug}, nil
	}
	return up, err
}

func (r *PrestigeSocialRepo) GetUserPrestigeCrossTitle(ctx context.Context, userID string) (prestige.UserPrestige, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var up prestige.UserPrestige
	up.UserID = userID
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_pp), 0), MAX(updated_at)
		FROM user_prestige_latest WHERE user_id = ?
	`, userID).Scan(&up.TotalPP, &up.UpdatedAt)
	return up, err
}

func (r *PrestigeSocialRepo) UpsertUserPrestige(ctx context.Context, up prestige.UserPrestige) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// APPEND-ONLY : INSERT d'un snapshot complet (plus d'ON CONFLICT DO UPDATE).
	return execCheckpointed(ctx, r.db, `
		INSERT INTO user_prestige_history (user_id, title_slug, total_pp, current_level, updated_at, written_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
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
		return nil, err
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

func (r *PrestigeSocialRepo) GetLeaderboard(
	ctx context.Context,
	userIDs []string,
	titleSlug *string,
	since time.Time,
) ([]prestige.LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if len(userIDs) == 0 {
		return nil, nil
	}
	// Construire les placeholders ?,?,?,...
	placeholders := strings.Repeat("?,", len(userIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]any, 0, len(userIDs)+1)
	for _, uid := range userIDs {
		args = append(args, uid)
	}

	var q string
	if titleSlug != nil && *titleSlug != "" {
		q = fmt.Sprintf(`
			SELECT user_id, title_slug, total_pp
			FROM user_prestige_latest
			WHERE user_id IN (%s) AND title_slug = ?
			ORDER BY total_pp DESC
		`, placeholders)
		args = append(args, *titleSlug)
	} else {
		// Cross-titre : SUM par user
		q = fmt.Sprintf(`
			SELECT user_id, '' AS title_slug, COALESCE(SUM(total_pp), 0) AS total_pp
			FROM user_prestige_latest
			WHERE user_id IN (%s)
			GROUP BY user_id
			ORDER BY total_pp DESC
		`, placeholders)
	}

	rows, err := r.db.QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("GetLeaderboard: %w", err)
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
	err := r.db.QueryRow(ctx,
		`SELECT id, name, created_by, created_at FROM squad WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.CreatedBy, &s.CreatedAt)
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
		return nil, err
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
		return nil, err
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
	err := r.db.QueryRow(ctx, `
		SELECT id, squad_id, COALESCE(template_id, ''), title_slug, mode, eval_type,
		       window_type, COALESCE(window_value, ''), COALESCE(target_per_member, 0),
		       expires_at, created_by, created_at
		FROM squad_challenge WHERE id = ?
	`, id).Scan(&sc.ID, &sc.SquadID, &sc.TemplateID, &sc.TitleSlug, &mode, &evalType,
		&windowType, &sc.WindowValue, &sc.TargetPerMember,
		&sc.ExpiresAt, &sc.CreatedBy, &sc.CreatedAt)
	if err != nil {
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
	rows, err := r.db.QueryRecovered(ctx, `
		SELECT id, squad_id, COALESCE(template_id, ''), title_slug, mode, eval_type,
		       window_type, COALESCE(window_value, ''), COALESCE(target_per_member, 0),
		       expires_at, created_by, created_at
		FROM squad_challenge WHERE squad_id = ? ORDER BY created_at DESC
	`, squadID)
	if err != nil {
		return nil, err
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
		SELECT ?,?,?,?,?,?,?,?, CURRENT_TIMESTAMP
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
		       ?, ?, is_private, joined_at, CURRENT_TIMESTAMP
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
		return nil, err
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

func (r *PrestigeSquadChallengeRepo) CountActiveParticipants(ctx context.Context, challengeID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM squad_challenge_participant_latest
		WHERE squad_challenge_id = ? AND completed_at IS NULL
	`, challengeID).Scan(&n)
	return n, err
}
