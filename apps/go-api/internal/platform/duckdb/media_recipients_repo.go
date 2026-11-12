package duckdb

// media_recipients_repo.go : les lectures des destinataires d'une notification d'import de média, derrière
// `port.MediaRecipientSource`. Chaque méthode ouvre la base visée par son chemin et la rend en sortant.

import (
	"context"
	"strings"
	"time"

	"levelup/go-api/internal/port"
)

var _ port.MediaRecipientSource = MediaRecipientsRepo{}

// MediaRecipientsRepo lit les matchs associés aux médias et leurs participants.
type MediaRecipientsRepo struct{}

// RecentMediaMatchIDs rend les match_id associés à un média depuis `since` (au plus 50). Plafonné
// implicitement par le nombre d'uploads concomitants.
//
// IMPORTANT : OpenReadWriteShared (et NON OpenReadOnly) — la base sociale partagée est déjà ouverte en mode RW
// partagé par le pool joueur (pool.go) et par prestige_setup au boot. Un OpenReadOnly ici créerait une 2e
// clé cache ("ro:"+path) → 2 duckdb_open() avec des access_mode différents sur le même fichier → erreur
// "Can't open a connection to same database file with a different configuration". Cf. fix similaire commit
// 9c.5 sur shared_matches_v2.
func (MediaRecipientsRepo) RecentMediaMatchIDs(ctx context.Context, sharedSocialDBPath string, since time.Time) ([]string, error) {
	db, err := OpenReadWriteShared(sharedSocialDBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(ctx,
		// APPEND-ONLY : lire la vue _latest + associated_at (immuable) — pas created_at
		// sur la table legacy. Préserve la fenêtre de récence d'upload sans faux positifs.
		`SELECT DISTINCT match_id FROM media_match_associations_latest WHERE associated_at >= ? LIMIT 50`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// MatchParticipantXUIDs rend les xuid distincts (au plus 50) des participants aux matchs donnés.
func (MediaRecipientsRepo) MatchParticipantXUIDs(ctx context.Context, sharedMatchesDBPath string, matchIDs []string) ([]string, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	// OpenReadForQuery réutilise le handle en cache (RO ou RW) s'il existe et ne retombe sur OpenReadOnly que
	// si rien n'est en cache → évite l'échec DuckDB "different configuration" quand le shared est déjà tenu
	// RW ailleurs (incident 2026-06-01, cf. db.go). Fan-out best-effort.
	sqlDB, release, err := OpenReadForQuery(sharedMatchesDBPath)
	if err != nil {
		return nil, err
	}
	defer release()
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	q := `SELECT DISTINCT xuid FROM match_participants WHERE match_id IN (` + strings.Join(placeholders, ",") + `) LIMIT 50`
	rows, err := sqlDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
