// Package duckdb — records_repo.go : persistance des PB (V2 Ascension).
//
// PersonalRecordsRepo  → table `player_records` dans `shared_social.duckdb`.
//   (xuid en PK avec metric + period — cf. migration extend_player_records_with_window)
//
// Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §7.2.
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

const personalRecordsSelectColumns = `
SELECT xuid, metric, period, value, achieved_at, achieved_match_id,
       previous_value, previous_achieved_at, updated_at
FROM player_records`

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

// Upsert insère ou met à jour un PB (clé : xuid + metric + period).
//
// L'écriture passe par OpenReadWrite pour respecter le contrat partagé du
// shared_social.duckdb (la base est attachée en lecture seule depuis le
// PlayerDB ; les écritures ouvrent leur propre connexion read-write).
func (r *PersonalRecordsRepo) Upsert(ctx context.Context, pr records.PersonalRecord) error {
	if r.sharedSocialPath() == "" {
		return errNoSocialPB
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rwDB, err := OpenReadWrite(r.sharedSocialPath())
	if err != nil {
		return fmt.Errorf("PersonalRecordsRepo.Upsert: open rw: %w", err)
	}
	defer rwDB.Close()

	_, err = rwDB.Exec(ctx, `
		INSERT INTO player_records (
			xuid, metric, period, value, achieved_at, achieved_match_id,
			previous_value, previous_achieved_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT (xuid, metric, period) DO UPDATE SET
			value                = excluded.value,
			achieved_at          = excluded.achieved_at,
			achieved_match_id    = excluded.achieved_match_id,
			previous_value       = excluded.previous_value,
			previous_achieved_at = excluded.previous_achieved_at,
			updated_at           = excluded.updated_at
	`,
		pr.XUID, pr.Metric, string(pr.Period), pr.Value,
		nullableTime(pr.AchievedAt), nullableStr(pr.AchievedMatchID),
		nullableFloat(pr.PreviousValue), nullableTime(pr.PreviousAchievedAt),
		pr.UpdatedAt,
	)
	if err != nil {
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
func scanPersonalRecord(row rowScanner) (records.PersonalRecord, error) {
	var (
		pr                records.PersonalRecord
		periodStr         string
		achievedAt        sql.NullTime
		achievedMatchID   sql.NullString
		prevValue         sql.NullFloat64
		prevAchievedAt    sql.NullTime
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
