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

// LookupByIDs résout les labels et descriptions des IDs donnés dans la langue demandée : les
// colonnes non textuelles viennent de medal_definitions, les textes de medal_translations (langue
// demandée, sinon `en`, source unique : loadMedalLabels). Une langue vide ou inconnue se lit comme
// `en`. Une médaille sans aucun texte retombe sur citation_mappings.
//
// Retourne une map vide si la metadata DB est absente.
func (r *MedalDefinitionsRepo) LookupByIDs(
	ctx context.Context,
	ids []int64,
	locale string,
) (map[int64]port.MedalDefinitionRow, error) {
	result := make(map[int64]port.MedalDefinitionRow, len(ids))
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := r.queryDefinitions(ctx, ids)
	if err != nil {
		return result, err
	}
	labels, err := loadMedalLabels(ctx, r.pdb.Metadata, ids, locale)
	if err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: textes: %w", err)
	}
	for id, row := range result {
		row.Label, row.Description = labels[id].label, labels[id].description
		result[id] = row
	}

	// Fallback citation_mappings pour les IDs absents de medal_definitions OU
	// sans libellé exploitable (parité avec la vue Match — corrige Explorer
	// top_medals + Squad qui affichaient des libellés vides). Source unique :
	// medal_citation_fallback.go.
	missing := make([]int64, 0)
	for _, id := range ids {
		if row, ok := result[id]; !ok || row.Label == "" {
			missing = append(missing, id)
		}
	}
	for id, label := range citationLabelsOrLog(ctx, r.pdb.Metadata, missing, "medal_definitions") {
		result[id] = port.MedalDefinitionRow{MedalID: id, Label: label, Difficulty: "Normal"}
	}
	return result, nil
}

// queryDefinitions lit les colonnes non textuelles de medal_definitions pour `ids` ; les textes
// viennent de medal_translations (loadMedalLabels). Le curseur est fermé au retour : la lecture
// des textes suit sur la même base.
func (r *MedalDefinitionsRepo) queryDefinitions(ctx context.Context, ids []int64) (map[int64]port.MedalDefinitionRow, error) {
	result := make(map[int64]port.MedalDefinitionRow, len(ids))
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty,
		        COALESCE(NULLIF(TRIM(md.medal_type),''), '') AS medal_type,
		        COALESCE(md.personal_score, 0) AS personal_score
		 FROM medal_definitions md
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
		var row port.MedalDefinitionRow
		if err := rows.Scan(&row.MedalID, &row.Difficulty, &row.MedalType, &row.PersonalScore); err != nil {
			return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: scan: %w", err)
		}
		result[row.MedalID] = row
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("MedalDefinitionsRepo.LookupByIDs: rows: %w", err)
	}
	return result, nil
}
