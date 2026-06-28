package duckdb

// halo5_commendation_defs.go — lecture du référentiel commendation_definitions
// (metadata h5) pour résoudre nom + icône d'une commendation NATIVE par UUID.
// Satisfait halo_5.CommendationDefSource structurellement (retour canonical, aucun
// import du package halo_5 → pas de cycle ; parité Halo5MatchHistorySource).

import (
	"context"
	"database/sql"
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// Halo5CommendationDefSource lit commendation_definitions dans la metadata h5 (handle
// read-mostly ; la metadata n'est pas dans le set de B-swap RO/RW → le *sql.DB est
// stable). Read-only : aucune écriture, aucun risque ART.
type Halo5CommendationDefSource struct {
	meta *sql.DB
}

// NewHalo5CommendationDefSource construit la source adossée au handle metadata h5
// (au boot : metaDB.SQLDb()). meta nil → toutes les lookups retournent une map vide
// sans erreur (dégradation).
func NewHalo5CommendationDefSource(meta *sql.DB) *Halo5CommendationDefSource {
	return &Halo5CommendationDefSource{meta: meta}
}

// LookupCommendations résout nom + icône pour les UUID demandés (dédupliqués). Les
// IDs inconnus sont absents de la map retournée. Le nom préfère la traduction FR
// (name_fr, = name_en tant qu'aucun override FR n'est seedé) puis l'EN.
func (s *Halo5CommendationDefSource) LookupCommendations(ctx context.Context, ids []string) (map[string]canonical.CommendationDefinition, error) {
	out := make(map[string]canonical.CommendationDefinition)
	if s == nil || s.meta == nil || len(ids) == 0 {
		return out, nil
	}
	seen := make(map[string]struct{}, len(ids))
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	if len(args) == 0 {
		return out, nil
	}
	q := `SELECT commendation_id,
	             COALESCE(NULLIF(name_fr, ''), name_en) AS name,
	             COALESCE(icon_url, '') AS icon_url,
	             COALESCE(category, '') AS category,
	             COALESCE(tier_targets, '') AS tier_targets
	      FROM commendation_definitions
	      WHERE commendation_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := s.meta.QueryContext(ctx, q, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, icon, category, tierTargets string
		if err := rows.Scan(&id, &name, &icon, &category, &tierTargets); err != nil {
			return out, err
		}
		out[id] = canonical.CommendationDefinition{ID: id, Name: name, IconURL: icon, Category: category, TierTargets: tierTargets}
	}
	return out, rows.Err()
}
