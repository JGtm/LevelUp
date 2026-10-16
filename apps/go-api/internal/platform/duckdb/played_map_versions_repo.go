package duckdb

import (
	"context"
	"errors"
	"fmt"
)

// qPlayedMapVersions : couples (carte, version) des matchs enregistrés dans la base partagée
// d'un titre. match_registry porte une ligne par match (table non append-only) ; la version
// est celle que la vue match met dans l'URL de l'image de carte (Q13MatchMeta, ?v=).
const qPlayedMapVersions = `
SELECT DISTINCT map_id, map_version_id
FROM match_registry
WHERE map_id IS NOT NULL AND map_id <> ''
  AND map_version_id IS NOT NULL AND map_version_id <> ''`

// ForEachPlayedMapVersion parcourt les couples (carte, version) des matchs enregistrés dans la
// base partagée lue par reader (lecteur du provider, jamais un writer) et appelle visit pour
// chacun.
func ForEachPlayedMapVersion(ctx context.Context, reader SharedReader, visit func(mapID, versionID string)) error {
	if reader == nil {
		return errors.New("ForEachPlayedMapVersion: aucune base partagée pour ce titre")
	}
	db, release, err := reader.Get(ctx)
	if err != nil {
		return fmt.Errorf("ForEachPlayedMapVersion: lecteur partagé: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx, qPlayedMapVersions)
	if err != nil {
		return fmt.Errorf("ForEachPlayedMapVersion: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mapID, versionID string
		if err := rows.Scan(&mapID, &versionID); err != nil {
			return fmt.Errorf("ForEachPlayedMapVersion: lecture: %w", err)
		}
		visit(mapID, versionID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ForEachPlayedMapVersion: %w", err)
	}
	return nil
}
