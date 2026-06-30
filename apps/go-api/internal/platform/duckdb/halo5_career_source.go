// Package duckdb — halo5_career_source.go : lecture LOCALE (DuckDB) du career Halo 5
// d'un joueur — meilleur CSR à vie (player_csr_snapshots) + Spartan Rank
// (career_progression) — pour projeter un CareerSnapshot hors-ligne quand l'API
// cryptum live est indisponible (démo : aucun token).
//
// Satisfait STRUCTURELLEMENT halo_5.CareerLocalSource (retour domain.H5CareerLocal,
// aucun import du package halo_5 → pas de cycle ; parité Halo5MatchHistorySource).
// La PROJECTION vers canonical.CareerSnapshot (libellés de palier FR, bornes SR)
// reste côté halo_5 (qui détient les référentiels CSR/SR).
package duckdb

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/domain"
)

// Halo5CareerSource lit le career local d'un joueur depuis SA player DB (h5).
type Halo5CareerSource struct {
	pdb *PlayerDB
}

// NewHalo5CareerSource construit la source liée à la player DB h5 d'un joueur.
func NewHalo5CareerSource(pdb *PlayerDB) *Halo5CareerSource {
	return &Halo5CareerSource{pdb: pdb}
}

// h5BestCSRQuery — MEILLEUR CSR à vie (alltime) toutes playlists confondues. Ordonne
// par palier majeur (Bronze<…<Onyx, via CASE — même ordre que halo_5.h5Designations)
// puis sous-palier puis valeur. Ignore les lignes sans palier (alltime_tier vide).
const h5BestCSRQuery = `
SELECT alltime_tier, COALESCE(alltime_sub_tier, 0), COALESCE(alltime_value, 0)
FROM player_csr_snapshots
WHERE NULLIF(TRIM(alltime_tier), '') IS NOT NULL
ORDER BY
  (CASE LOWER(alltime_tier)
     WHEN 'bronze' THEN 0 WHEN 'silver' THEN 1 WHEN 'gold' THEN 2
     WHEN 'platinum' THEN 3 WHEN 'diamond' THEN 4 WHEN 'onyx' THEN 5 ELSE -1 END) DESC,
  COALESCE(alltime_sub_tier, 0) DESC,
  COALESCE(alltime_value, 0) DESC
LIMIT 1`

// h5LatestSRQuery — dernier Spartan Rank VALIDE (rank non NULL). La dernière ligne
// career_progression peut porter un rank NULL (re-fetch sans données) ou être la
// row d'identité empruntée en démo (emblem/banner, rank NULL) → on filtre.
const h5LatestSRQuery = `
SELECT rank, COALESCE(xp_total, 0)
FROM career_progression
WHERE rank IS NOT NULL
ORDER BY recorded_at DESC
LIMIT 1`

// GetLatestCareer retourne le meilleur CSR à vie + le SR courant du joueur. Best-effort
// par bloc : une table/colonne absente (DB legacy) dégrade le bloc concerné sans
// échouer l'ensemble — seule une vraie erreur de lecture remonte (le caller dégrade).
func (s *Halo5CareerSource) GetLatestCareer(ctx context.Context) (*domain.H5CareerLocal, error) {
	out := &domain.H5CareerLocal{}

	var tier sql.NullString
	var sub, val sql.NullInt64
	err := s.pdb.ReadDB().QueryRow(ctx, h5BestCSRQuery).Scan(&tier, &sub, &val)
	switch {
	case err == nil:
		if tier.Valid && tier.String != "" {
			out.HasCSR = true
			out.CSRTier = tier.String
			out.CSRSubTier = int(sub.Int64)
			out.CSRValue = int(val.Int64)
		}
	case err == sql.ErrNoRows:
		// aucun snapshot CSR — joueur non classé, pas une erreur.
	default:
		return nil, err
	}

	var rank, xp sql.NullInt64
	err = s.pdb.ReadDB().QueryRow(ctx, h5LatestSRQuery).Scan(&rank, &xp)
	switch {
	case err == nil:
		if rank.Valid {
			out.SpartanRank = int(rank.Int64)
			out.TotalXP = int(xp.Int64)
		}
	case err == sql.ErrNoRows:
		// aucun SR — pas une erreur.
	default:
		return nil, err
	}

	return out, nil
}

// GetXPHistory retourne l'historique XP (career_progression) du joueur, pour le graphe
// « Historique XP » de la page Carrière. Réutilise la requête partagée Q7
// (CareerRepo.GetXPHistory) : le schéma career_progression est identique inter-titres
// et le sync h5 (livesync / cmd/h5-career-rank-xp) l'alimente. Vide si aucun
// checkpoint (le graphe se masque, dégradation propre).
func (s *Halo5CareerSource) GetXPHistory(ctx context.Context) ([]domain.XPHistoryPoint, error) {
	return NewCareerRepo(s.pdb).GetXPHistory(ctx)
}
