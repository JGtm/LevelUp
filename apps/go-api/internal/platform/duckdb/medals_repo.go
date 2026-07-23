// Package duckdb — medals_repo.go : accès DB de la page Médailles (catalogue complet
// du titre + totaux obtenus par joueur).
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Compile-time check : MedalsRepo implémente port.MedalsRepository.
var _ port.MedalsRepository = (*MedalsRepo)(nil)

// MedalsRepo implémente port.MedalsRepository. ListAllMedals lit medal_definitions
// (pdb.Metadata) ; LoadMedalTotals lit medals_earned (SharedReadDB, ADR 0016). Les
// deux DB ne se joignent pas cross-process → 2 lectures + merge en Go côté service.
type MedalsRepo struct {
	pdb *PlayerDB
}

// NewMedalsRepo crée un MedalsRepo lié à un PlayerDB.
func NewMedalsRepo(pdb *PlayerDB) *MedalsRepo {
	return &MedalsRepo{pdb: pdb}
}

// ListAllMedals retourne TOUT le catalogue medal_definitions du titre, labels et
// descriptions résolus locale-aware. Calqué sur MetadataRepo.ListMedalsByTitle mais
// expose en plus difficulty / medal_type / difficulty_index / personal_score (source
// de la catégorie baseline + rareté). Réutilise les helpers COALESCE partagés
// (medal_label_resolve.go) — pas de nouvelle copie de la chaîne FR.
func (r *MedalsRepo) ListAllMedals(ctx context.Context, locale string) ([]domain.MedalCatalogRow, error) {
	if r.pdb == nil || r.pdb.Metadata == nil {
		return nil, fmt.Errorf("ListAllMedals: metadata db indisponible")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	labelExpr, descExpr := medalLabelDescCoalesceSQL(locale)
	query := `
		SELECT md.medal_name_id,
		       ` + labelExpr + ` AS label,
		       ` + descExpr + ` AS description,
		       COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty,
		       COALESCE(NULLIF(TRIM(md.medal_type),''), '')       AS medal_type,
		       COALESCE(md.difficulty_index, 0)                   AS difficulty_index,
		       COALESCE(md.personal_score, 0)                     AS personal_score
		FROM medal_definitions md
		` + medalTranslationJoinsSQL(locale) + `
		ORDER BY md.medal_name_id`

	rows, err := r.pdb.Metadata.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ListAllMedals: %w", err)
	}
	defer rows.Close()

	var out []domain.MedalCatalogRow
	for rows.Next() {
		var row domain.MedalCatalogRow
		if err := rows.Scan(
			&row.MedalID,
			&row.Label,
			&row.Description,
			&row.Difficulty,
			&row.MedalType,
			&row.DifficultyIndex,
			&row.PersonalScore,
		); err != nil {
			return nil, fmt.Errorf("ListAllMedals scan: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// LoadMedalTotals charge les totaux de médailles obtenus par un joueur depuis
// shared.medals_earned (Q36a, réutilisé — même requête que la page Commendations).
// Lecture sur SharedReader (ADR 0016), table à la racine (pas de préfixe `shared.`).
func (r *MedalsRepo) LoadMedalTotals(ctx context.Context, xuid string) ([]domain.MedalEarnedRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalTotals: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q36aMedalTotals, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalTotals: %w", err)
	}
	defer rows.Close()

	var result []domain.MedalEarnedRow
	for rows.Next() {
		var row domain.MedalEarnedRow
		if err := rows.Scan(&row.MedalID, &row.TotalCount); err != nil {
			return nil, fmt.Errorf("LoadMedalTotals scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
