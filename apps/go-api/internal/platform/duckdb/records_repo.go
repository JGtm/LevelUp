// Package duckdb — records_repo.go : persistance des PB (V2 Ascension).
//
// PersonalRecordsRepo écrit en append-only dans `player_records_history` et lit
// la vue `player_records_latest` (shared_social.duckdb). Fix 2026-05-30 :
// migré de l'ancien INSERT ... ON CONFLICT DO UPDATE sur la table legacy
// `player_records` (anti-pattern ART : pressionnait l'index DuckDB, crashait
// en "Failed to delete all rows from index"). Les écritures passent par
// SocialPersister.AppendPlayerRecord (CHECKPOINT garanti) ; fallback INSERT pur
// _history (tests sans persister wired). Plus aucune écriture sur player_records.
//
// Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §7.2 + ADR 0019/0020/0021.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/progression/records"
)

// PersonalRecordsRepo persiste les PB d'un joueur dans shared_social.duckdb.
type PersonalRecordsRepo struct {
	pdb *PlayerDB
}

// NewPersonalRecordsRepo construit le repo depuis un PlayerDB (qui attache
// shared_social.duckdb).
func NewPersonalRecordsRepo(pdb *PlayerDB) *PersonalRecordsRepo {
	return &PersonalRecordsRepo{pdb: pdb}
}

// Compile-time assertion.
var _ records.PBRepo = (*PersonalRecordsRepo)(nil)

// Lecture via la vue append-only player_records_latest (dernier PB par
// xuid/metric/period). written_at tient lieu d'updated_at pour le scan.
const personalRecordsSelectColumns = `
SELECT xuid, metric, period, value, achieved_at, achieved_match_id,
       previous_value, previous_achieved_at, written_at
FROM player_records_latest`

var errNoSocialPB = fmt.Errorf("PersonalRecordsRepo: shared_social DB not attached")

func (r *PersonalRecordsRepo) sharedSocialPath() string {
	if r.pdb == nil || r.pdb.SharedSocial == nil {
		return ""
	}
	return r.pdb.SharedSocial.Path()
}

// Get retourne le PB pour (xuid, metric, period), ou nil si aucun.
func (r *PersonalRecordsRepo) Get(ctx context.Context, xuid, metric string, period records.RecordPeriod) (*records.PersonalRecord, error) {
	if r.pdb == nil || r.pdb.SharedSocial == nil {
		return nil, errNoSocialPB
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := r.pdb.SharedSocial.QueryRow(ctx, personalRecordsSelectColumns+`
		WHERE xuid = ? AND metric = ? AND period = ?`,
		xuid, metric, string(period),
	)
	pr, err := scanPersonalRecord(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PersonalRecordsRepo.Get: %w", err)
	}
	return &pr, nil
}

// Upsert enregistre un nouveau PB (clé logique : xuid + metric + period) en
// append-only dans player_records_history. Le nom "Upsert" est conservé pour
// l'interface records.PBRepo, mais c'est désormais un INSERT pur (la vue
// player_records_latest dé-doublonne par written_at DESC).
//
// Chemin nominal (prod) : SocialPersister.AppendPlayerRecord (TX + CHECKPOINT
// garanti). Fallback (tests, persister non wired) : INSERT pur dans _history
// via une connexion RW dédiée — toujours append-only, jamais d'ON CONFLICT,
// jamais d'écriture sur la table legacy player_records.
func (r *PersonalRecordsRepo) Upsert(ctx context.Context, pr records.PersonalRecord) error {
	if r.pdb == nil || r.pdb.SharedSocial == nil {
		return errNoSocialPB
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	period := string(pr.Period)
	if period == "" {
		period = "all_time"
	}
	var matchIDPtr *string
	if pr.AchievedMatchID != "" {
		matchIDPtr = &pr.AchievedMatchID
	}

	// Chemin nominal : persister append-only + CHECKPOINT.
	if r.pdb.SocialPersister != nil {
		return r.pdb.SocialPersister.AppendPlayerRecord(ctx,
			pr.XUID, pr.Metric, period, pr.Value,
			pr.AchievedAt, matchIDPtr, pr.PreviousValue, pr.PreviousAchievedAt)
	}
	// ADR 0021 Gap 1 : en prod (RequireSocialPersister=true) on refuse le path
	// legacy silencieux — un persister nil est un bug de wiring boot.
	if RequireSocialPersister {
		return ErrSocialPersisterNotWired
	}

	// Fallback tests : INSERT pur dans _history (append-only, ART-safe).
	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("PersonalRecordsRepo.Upsert: open rw: %w", err)
	}
	defer rwDB.Close()
	if _, err := rwDB.Exec(ctx, `
		INSERT INTO player_records_history (
			xuid, metric, period, value, achieved_at, achieved_match_id,
			previous_value, previous_achieved_at, written_at
		) VALUES (?,?,?,?,?,?,?,?, CURRENT_TIMESTAMP)
	`,
		pr.XUID, pr.Metric, period, pr.Value,
		nullableTime(pr.AchievedAt), NullableStr(pr.AchievedMatchID),
		nullableFloat(pr.PreviousValue), nullableTime(pr.PreviousAchievedAt),
	); err != nil {
		return fmt.Errorf("PersonalRecordsRepo.Upsert: %w", err)
	}
	return nil
}

// ListByXUID retourne tous les PB d'un joueur, triés par period puis metric.
func (r *PersonalRecordsRepo) ListByXUID(ctx context.Context, xuid string) ([]records.PersonalRecord, error) {
	if r.pdb == nil || r.pdb.SharedSocial == nil {
		return nil, errNoSocialPB
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := r.pdb.SharedSocial.Query(ctx, personalRecordsSelectColumns+`
		WHERE xuid = ?
		ORDER BY period ASC, metric ASC`,
		xuid,
	)
	if err != nil {
		return nil, fmt.Errorf("PersonalRecordsRepo.ListByXUID: %w", err)
	}
	defer rows.Close()
	var out []records.PersonalRecord
	for rows.Next() {
		pr, err := scanPersonalRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("PersonalRecordsRepo.ListByXUID scan: %w", err)
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

// scanPersonalRecord parse une ligne de player_records en records.PersonalRecord.
func scanPersonalRecord(row RowScanner) (records.PersonalRecord, error) {
	var (
		pr              records.PersonalRecord
		periodStr       string
		achievedAt      sql.NullTime
		achievedMatchID sql.NullString
		prevValue       sql.NullFloat64
		prevAchievedAt  sql.NullTime
	)
	err := row.Scan(
		&pr.XUID, &pr.Metric, &periodStr, &pr.Value,
		&achievedAt, &achievedMatchID,
		&prevValue, &prevAchievedAt,
		&pr.UpdatedAt,
	)
	if err != nil {
		return records.PersonalRecord{}, err
	}
	pr.Period = records.RecordPeriod(periodStr)
	if achievedAt.Valid {
		pr.AchievedAt = &achievedAt.Time
	}
	if achievedMatchID.Valid {
		pr.AchievedMatchID = achievedMatchID.String
	}
	if prevValue.Valid {
		pr.PreviousValue = &prevValue.Float64
	}
	if prevAchievedAt.Valid {
		pr.PreviousAchievedAt = &prevAchievedAt.Time
	}
	return pr, nil
}
