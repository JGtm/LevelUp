// Package duckdb — spartan_identity_repo.go : accès DB pour la table dédiée
// customisation Spartan (PLAN_SPARTAN_IDENTITY_REFACTOR §11, 2026-05-25).
//
// Pourquoi un repo séparé : `career_progression` mélangeait historique
// rank/XP (append-only) et customisation latest-only. Cette table a 1 row
// par xuid, UPSERT atomique via TX (pattern SELECT-then-UPDATE-or-INSERT
// pour rester compatible ADR 0019 — pas de ON CONFLICT DO UPDATE qui
// stresserait l'ART).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SpartanIdentityStatus codifie le résultat du dernier fetch live.
// Permet à la home de diagnostiquer immédiatement pourquoi un joueur
// n'a pas sa bannière, sans avoir à corréler logs/cache mémoire.
type SpartanIdentityStatus string

const (
	// SpartanIdentityStatusOK : fetch API réussi, au moins 1 champ rempli.
	SpartanIdentityStatusOK SpartanIdentityStatus = "ok"
	// SpartanIdentityStatusAPIEmpty : fetch API a répondu sans erreur mais
	// payload vide/non parseable (401/403 silencieux côté Halo Economy).
	SpartanIdentityStatusAPIEmpty SpartanIdentityStatus = "api_empty"
	// SpartanIdentityStatusAuthMissing : pas de tokens disponibles pour ce
	// joueur (sync_meta vide ou refresh impossible).
	SpartanIdentityStatusAuthMissing SpartanIdentityStatus = "auth_missing"
	// SpartanIdentityStatusFailed : erreur réseau / 5xx Halo / decode error.
	SpartanIdentityStatusFailed SpartanIdentityStatus = "failed"
)

// SpartanIdentityRow est la projection raw d'une ligne `spartan_identity`.
// Une seule row par xuid (PK). Les champs URL/SpartanID sont la dernière
// valeur connue ; `last_refreshed_at` indique quand ils ont été remplis
// avec succès. `last_attempt_at` + `last_attempt_status` tracent le dernier
// essai (qu'il ait abouti ou non).
type SpartanIdentityRow struct {
	XUID              string
	SpartanID         string
	BannerImageURL    string
	EmblemImageURL    string
	BackdropImageURL  string
	LastRefreshedAt   time.Time
	LastAttemptAt     time.Time
	LastAttemptStatus SpartanIdentityStatus
}

// IsEmpty retourne true si la row ne porte aucune donnée d'identité
// exploitable (utile pour décider si on retourne nil au caller).
func (r *SpartanIdentityRow) IsEmpty() bool {
	if r == nil {
		return true
	}
	return r.SpartanID == "" &&
		r.BannerImageURL == "" &&
		r.EmblemImageURL == "" &&
		r.BackdropImageURL == ""
}

// SpartanIdentityRepo expose Load/Upsert sur la table spartan_identity.
type SpartanIdentityRepo struct {
	pdb *PlayerDB
}

// NewSpartanIdentityRepo crée un repo lié au PlayerDB du joueur courant.
func NewSpartanIdentityRepo(pdb *PlayerDB) *SpartanIdentityRepo {
	return &SpartanIdentityRepo{pdb: pdb}
}

const qLoadSpartanIdentity = `
SELECT
	spartan_id,
	banner_image_url,
	emblem_image_url,
	backdrop_image_url,
	last_refreshed_at,
	last_attempt_at,
	last_attempt_status
FROM spartan_identity
WHERE xuid = ?`

// Load retourne la row courante pour xuid. (nil, nil) si absente ou si la
// table n'existe pas (migration pas encore tournée).
func (r *SpartanIdentityRepo) Load(ctx context.Context, xuid string) (*SpartanIdentityRow, error) {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil, nil
	}
	if strings.TrimSpace(xuid) == "" {
		return nil, nil
	}
	var (
		spartanID, bannerURL, emblemURL, backdropURL sql.NullString
		lastRefreshed, lastAttempt                   sql.NullTime
		lastStatus                                   sql.NullString
	)
	err := r.pdb.Player.QueryRow(ctx, qLoadSpartanIdentity, xuid).Scan(
		&spartanID,
		&bannerURL,
		&emblemURL,
		&backdropURL,
		&lastRefreshed,
		&lastAttempt,
		&lastStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows || isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("LoadSpartanIdentity: %w", err)
	}
	row := &SpartanIdentityRow{XUID: xuid}
	if spartanID.Valid {
		row.SpartanID = spartanID.String
	}
	if bannerURL.Valid {
		row.BannerImageURL = bannerURL.String
	}
	if emblemURL.Valid {
		row.EmblemImageURL = emblemURL.String
	}
	if backdropURL.Valid {
		row.BackdropImageURL = backdropURL.String
	}
	if lastRefreshed.Valid {
		row.LastRefreshedAt = lastRefreshed.Time
	}
	if lastAttempt.Valid {
		row.LastAttemptAt = lastAttempt.Time
	}
	if lastStatus.Valid {
		row.LastAttemptStatus = SpartanIdentityStatus(lastStatus.String)
	}
	return row, nil
}

// Upsert écrit/remplace la row pour xuid avec les valeurs fournies + le
// status du dernier essai. Pattern SELECT-then-UPDATE-or-INSERT atomique
// via TX — évite le `INSERT … ON CONFLICT DO UPDATE` qui stresse l'ART
// DuckDB (cf. ADR 0019).
//
// `data` peut être nil : on enregistre alors juste le timestamp + status
// (utile pour tracer un échec "api_empty" / "auth_missing" sans écraser
// les URLs précédemment réussies).
func (r *SpartanIdentityRepo) Upsert(
	ctx context.Context,
	xuid string,
	data *SpartanIdentityRow,
	status SpartanIdentityStatus,
) error {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return fmt.Errorf("spartan_identity_repo: pdb non initialisé")
	}
	if strings.TrimSpace(xuid) == "" {
		return fmt.Errorf("spartan_identity_repo: xuid vide")
	}

	now := time.Now().UTC()
	hasFresh := data != nil && !data.IsEmpty()

	sqlDB := r.pdb.Player.SQLDb()
	if sqlDB == nil {
		return fmt.Errorf("spartan_identity_repo: sqlDB nil")
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("spartan_identity_repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existing int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM spartan_identity WHERE xuid = ?`, xuid).Scan(&existing)
	switch {
	case err == sql.ErrNoRows:
		return r.insertNewRow(ctx, tx, xuid, data, status, now, hasFresh)
	case err != nil:
		return fmt.Errorf("spartan_identity_repo: select: %w", err)
	}
	return r.updateExistingRow(ctx, tx, xuid, data, status, now, hasFresh)
}

func (r *SpartanIdentityRepo) insertNewRow(
	ctx context.Context,
	tx *sql.Tx,
	xuid string,
	data *SpartanIdentityRow,
	status SpartanIdentityStatus,
	now time.Time,
	hasFresh bool,
) error {
	var (
		spartanID, bannerURL, emblemURL, backdropURL string
		lastRefreshed                                sql.NullTime
	)
	if hasFresh {
		spartanID = data.SpartanID
		bannerURL = data.BannerImageURL
		emblemURL = data.EmblemImageURL
		backdropURL = data.BackdropImageURL
		lastRefreshed = sql.NullTime{Time: now, Valid: true}
	}
	const insertSQL = `
INSERT INTO spartan_identity (
	xuid,
	spartan_id,
	banner_image_url,
	emblem_image_url,
	backdrop_image_url,
	last_refreshed_at,
	last_attempt_at,
	last_attempt_status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertSQL,
		xuid,
		spartanID,
		bannerURL,
		emblemURL,
		backdropURL,
		lastRefreshed,
		now,
		string(status),
	); err != nil {
		return fmt.Errorf("spartan_identity_repo: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("spartan_identity_repo: commit: %w", err)
	}
	return nil
}

func (r *SpartanIdentityRepo) updateExistingRow(
	ctx context.Context,
	tx *sql.Tx,
	xuid string,
	data *SpartanIdentityRow,
	status SpartanIdentityStatus,
	now time.Time,
	hasFresh bool,
) error {
	if hasFresh {
		// UPDATE complet : on remplace toutes les valeurs + last_refreshed_at.
		const updateFullSQL = `
UPDATE spartan_identity SET
	spartan_id          = ?,
	banner_image_url    = ?,
	emblem_image_url    = ?,
	backdrop_image_url  = ?,
	last_refreshed_at   = ?,
	last_attempt_at     = ?,
	last_attempt_status = ?
WHERE xuid = ?`
		if _, err := tx.ExecContext(ctx, updateFullSQL,
			data.SpartanID,
			data.BannerImageURL,
			data.EmblemImageURL,
			data.BackdropImageURL,
			now,
			now,
			string(status),
			xuid,
		); err != nil {
			return fmt.Errorf("spartan_identity_repo: update full: %w", err)
		}
	} else {
		// UPDATE status-only : on préserve les URLs/SpartanID précédents.
		// Utile pour tracer un échec "api_empty" sans perdre la dernière
		// valeur connue (cf. contrat UI-first du CareerLiveService).
		const updateStatusSQL = `
UPDATE spartan_identity SET
	last_attempt_at     = ?,
	last_attempt_status = ?
WHERE xuid = ?`
		if _, err := tx.ExecContext(ctx, updateStatusSQL, now, string(status), xuid); err != nil {
			return fmt.Errorf("spartan_identity_repo: update status: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("spartan_identity_repo: commit: %w", err)
	}
	return nil
}
