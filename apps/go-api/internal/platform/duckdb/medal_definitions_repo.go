package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/port"
)

// MedalDefinitionsRepo implémente port.MedalDefinitionsRepository.
// Requête sur pdb.Metadata (medal_definitions + medal_translations).
type MedalDefinitionsRepo struct {
	pdb *PlayerDB
}

// NewMedalDefinitionsRepo crée un MedalDefinitionsRepo lié à un PlayerDB.
func NewMedalDefinitionsRepo(pdb *PlayerDB) *MedalDefinitionsRepo {
	return &MedalDefinitionsRepo{pdb: pdb}
}

// LookupByIDs résout les labels et descriptions anglaises pour les IDs donnés.
// Ordre de priorité pour le label : medal_translations en-US → medal_definitions.name_en.
// Retourne une map vide si la metadata DB est absente.
func (r *MedalDefinitionsRepo) LookupByIDs(
	ctx context.Context,
	ids []int64,
) (map[int64]port.MedalDefinitionRow, error) {
	result := make(map[int64]port.MedalDefinitionRow, len(ids))
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        COALESCE(NULLIF(TRIM(mt_en.name),''), NULLIF(TRIM(md.name_en),''), '') AS label,
		        COALESCE(NULLIF(TRIM(md.description_en),''), '') AS description,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty,
		        COALESCE(NULLIF(TRIM(md.medal_type),''), '') AS medal_type
		 FROM medal_definitions md
		 LEFT JOIN medal_translations mt_en
		     ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = 'en-US'
		 WHERE md.medal_name_id IN (%s)`,
		ids,
	)
	if !ok {
		return result, nil
	}

	rows, err := r.pdb.Metadata.Query(ctx, q, args...)
	if err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label, description, difficulty, medalType string
		if err := rows.Scan(&id, &label, &description, &difficulty, &medalType); err != nil {
			return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: scan: %w", err)
		}
		result[id] = port.MedalDefinitionRow{
			MedalID:     id,
			Label:       label,
			Description: description,
			Difficulty:  difficulty,
			MedalType:   medalType,
		}
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: rows: %w", err)
	}
	return result, nil
}
