package profile

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// activity.go — calendrier d'activité (jours joués) pour l'onglet Réalisations
// (DEC-5/D3). Lecture ART-safe via SharedReader avec le fragment timezone
// canonique + exclusion Campagne (mêmes invariants que les queries Section A/B).

// ActivityDay est un jour où le joueur a joué ≥ 1 match (les jours vides sont
// omis — le front les rend en cases neutres).
type ActivityDay struct {
	Date  string `json:"date"`  // jour UTC (YYYY-MM-DD)
	Count int    `json:"count"` // nb de matchs distincts ce jour-là
}

// ActivityCalendar est la réponse du calendrier d'activité sur [Since, Until].
type ActivityCalendar struct {
	Since string        `json:"since"` // jour UTC (YYYY-MM-DD) inclus
	Until string        `json:"until"` // jour UTC (YYYY-MM-DD) inclus
	Days  []ActivityDay `json:"days"`  // uniquement les jours avec count > 0, triés ASC
}

// dayLayout : format canonique d'un jour UTC servi au front.
const dayLayout = "2006-01-02"

// LoadActivityCalendar agrège le nombre de matchs distincts par jour UTC sur la
// fenêtre [since, until]. Source : match_participants (filtré xuid) JOIN
// match_registry, via SharedReader (ADR 0016). Le bucket jour est calculé en Go
// (`t.UTC().Format`) à partir du timestamp canonique (motif A6 : évite les pièges
// TZ du CAST DATE côté DuckDB). Exclut les matchs Campagne (campaignExcl).
func (s *Service) LoadActivityCalendar(ctx context.Context, userID string, since, until time.Time) (ActivityCalendar, error) {
	out := ActivityCalendar{
		Since: since.UTC().Format(dayLayout),
		Until: until.UTC().Format(dayLayout),
		Days:  []ActivityDay{},
	}
	if s.pdb == nil {
		return out, fmt.Errorf("LoadActivityCalendar: PlayerDB non initialisée")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	sharedDB, release, err := s.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("LoadActivityCalendar: shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, `
		SELECT mp.match_id, `+startTimeCanonicalMR+`
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?`+s.campaignExcl("mr")+`
		  AND `+startTimeCanonicalMR+` >= ? AND `+startTimeCanonicalMR+` <= ?
	`, userID, since, until)
	if err != nil {
		return out, fmt.Errorf("LoadActivityCalendar: %w", err)
	}
	defer rows.Close()

	// Dédup match par jour (robuste à d'éventuelles lignes participant dupliquées).
	perDay := map[string]map[string]struct{}{}
	for rows.Next() {
		var (
			matchID string
			at      sql.NullTime
		)
		if err := rows.Scan(&matchID, &at); err != nil {
			return out, err
		}
		if !at.Valid {
			continue
		}
		day := at.Time.UTC().Format(dayLayout)
		set, ok := perDay[day]
		if !ok {
			set = map[string]struct{}{}
			perDay[day] = set
		}
		set[matchID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	days := make([]ActivityDay, 0, len(perDay))
	for day, set := range perDay {
		days = append(days, ActivityDay{Date: day, Count: len(set)})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	out.Days = days
	return out, nil
}
