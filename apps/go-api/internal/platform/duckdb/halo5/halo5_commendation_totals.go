// Package duckdb — halo5_commendation_totals.go : lecture des TOTAUX à vie des
// commendations natives Halo 5 d'un joueur, depuis le substrat LOCAL synchronisé
// (shared_matches_v2.duckdb du titre h5).
//
// Le total à vie d'une commendation = le `progress` ABSOLU (carnage) du match le PLUS
// RÉCENT du joueur qui la fait progresser. PAS SUM(count) — cela sous-compterait la
// baseline pré-sync (cf. thought_log « persister le Progress absolu »). On lit donc le
// dernier progress par commendation (fenêtre ROW_NUMBER ordonnée par start_time DESC).
//
// Identité : match_commendations est keyé par XUID (l'xuid Xbox résolu au sync), donc
// le filtre se fait sur xuid (≠ match history h5 qui filtre sur gamertag). Name /
// IconURL / Category restent VIDES ici (enrichis par le caller via les définitions).
//
// Satisfait STRUCTURELLEMENT halo_5.CommendationTotalsSource (retour canonical, aucun
// import du package halo_5 → pas de cycle ; parité Halo5MatchHistorySource).
package halo5

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/platform/duckdb"
)

// Halo5CommendationTotalsSource lit les totaux à vie des commendations d'un joueur
// (xuid fixé) depuis le shared via un SharedReader title-aware.
type Halo5CommendationTotalsSource struct {
	shared duckdb.SharedReader
	xuid   string
}

// NewHalo5CommendationTotalsSource construit la source liée à un joueur (xuid) et au
// SharedReader du titre h5.
func NewHalo5CommendationTotalsSource(shared duckdb.SharedReader, xuid string) *Halo5CommendationTotalsSource {
	return &Halo5CommendationTotalsSource{shared: shared, xuid: strings.TrimSpace(xuid)}
}

// h5CommendationTotalsQuery — dernier `progress` (total à vie absolu) par commendation
// pour un xuid. Ignore les lignes progress NULL (écrites avant la colonne / sans
// re-fetch). Tie-break par match_id pour un déterminisme strict.
var h5CommendationTotalsQuery = `
SELECT commendation_id, progress
FROM (
    SELECT mc.commendation_id AS commendation_id,
           mc.progress        AS progress,
           ROW_NUMBER() OVER (
               PARTITION BY mc.commendation_id
               ORDER BY ` + duckdb.StartTimeCanonicalSQL("r") + ` DESC,
                        mc.match_id DESC
           ) AS rn
    FROM match_commendations mc
    JOIN match_registry r ON r.match_id = mc.match_id
    WHERE mc.xuid = ? AND mc.progress IS NOT NULL
) ranked
WHERE rn = 1
ORDER BY progress DESC`

// GetCommendationTotals retourne le total à vie par commendation (Name/IconURL/Category
// vides → enrichis par le caller via CommendationDefSource). xuid vide / reader nil →
// nil neutre.
func (s *Halo5CommendationTotalsSource) GetCommendationTotals(ctx context.Context) ([]canonical.CommendationTotal, error) {
	if s == nil || s.shared == nil || strings.TrimSpace(s.xuid) == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := s.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("h5 commendation totals shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, h5CommendationTotalsQuery, s.xuid)
	if err != nil {
		return nil, fmt.Errorf("h5 commendation totals query: %w", err)
	}
	defer rows.Close()

	var out []canonical.CommendationTotal
	for rows.Next() {
		var id string
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			return nil, fmt.Errorf("h5 commendation totals scan: %w", err)
		}
		out = append(out, canonical.CommendationTotal{ID: id, Total: total})
	}
	return out, rows.Err()
}
