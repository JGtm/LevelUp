package duckdb

// commendation_defs_metadata.go — définitions des commendations natives Halo 5 : colonnes non
// textuelles de commendation_definitions (icône, paliers) et noms par langue de
// asset_translations (type « commendation », ADR 0040 D-5).

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/i18n"
)

// commendationDef contient nom localisé + icône + paliers d'une commendation native
// par UUID.
type commendationDef struct {
	name        string
	iconURL     string
	tierTargets string // CSV croissant des seuils de paliers (vide si Meta/Daily ou non seedé)
}

// loadCommendationDefs résout nom (langue de la requête, repli sur la langue de repli) + icône
// CDN + tier_targets pour un ensemble d'UUID de commendations depuis commendation_definitions
// (metadata h5). IDs inconnus absents de la map. Dégradation : metadata nil / table absente →
// map vide.
func (r *HomeRepo) loadCommendationDefs(ctx context.Context, ids []string) map[string]commendationDef {
	if r.pdb == nil {
		return map[string]commendationDef{}
	}
	return loadCommendationDefsFromMetadata(ctx, r.pdb.Metadata, ids)
}

// loadCommendationDefsFromMetadata résout nom + icône CDN + tier_targets pour un ensemble
// d'UUID de commendations depuis commendation_definitions (metadata h5). Fonction libre
// partagée par HomeRepo (slot TopCitations de la MatchCard) et CitationsRepo (onglet citations
// de la Match View). IDs inconnus absents de la map. Dégradation : metadata nil / table absente
// → map vide.
//
// Le nom se lit dans asset_translations et se choisit par `Pick(i18n.FromContext(ctx))` : la
// langue demandée, puis la langue de repli, sinon vide (le nom reste alors celui que l'appelant
// substitue). Une définition sans aucun nom reste servie avec son icône et ses paliers.
//
// tier_targets est COALESCE'é vide : nullable + colonne possiblement absente sur une
// DB pré-migration. Pour absorber le cas « colonne absente » (metadata h5 provisionnée
// avant l'ajout de tier_targets), on retombe sur une requête sans tier_targets si le
// SELECT échoue (table présente, colonne manquante).
func loadCommendationDefsFromMetadata(ctx context.Context, metadata *DB, ids []string) map[string]commendationDef {
	out := make(map[string]commendationDef, len(ids))
	if len(ids) == 0 || metadata == nil {
		return out
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	in := strings.Join(placeholders, ", ")
	query := `SELECT commendation_id,
	                 COALESCE(icon_url, '') AS icon_url,
	                 COALESCE(tier_targets, '') AS tier_targets
	          FROM commendation_definitions
	          WHERE commendation_id IN (` + in + `)`
	rows, err := metadata.Query(ctx, query, args...)
	if err != nil {
		// Colonne tier_targets absente (DB pré-migration) → retry sans elle (le
		// progrès dégradera proprement en anneau vide).
		queryNoTiers := `SELECT commendation_id,
		                        COALESCE(icon_url, '') AS icon_url,
		                        '' AS tier_targets
		                 FROM commendation_definitions
		                 WHERE commendation_id IN (` + in + `)`
		rows, err = metadata.Query(ctx, queryNoTiers, args...)
		if err != nil {
			return out // dégradation : titre sans table commendation_definitions
		}
	}
	defer rows.Close()
	for rows.Next() {
		var id, icon, tiers string
		if err := rows.Scan(&id, &icon, &tiers); err != nil {
			continue
		}
		out[id] = commendationDef{iconURL: icon, tierTargets: tiers}
	}
	applyCommendationNames(ctx, metadata, out)
	return out
}

// applyCommendationNames pose, dans chaque définition de `defs`, le nom de la commendation dans la
// langue de la requête (repli sur la langue de repli). Un échec de lecture des textes est journalisé
// et laisse les noms vides : l'appelant affiche alors l'identifiant.
func applyCommendationNames(ctx context.Context, metadata *DB, defs map[string]commendationDef) {
	if len(defs) == 0 {
		return
	}
	ids := make([]string, 0, len(defs))
	for id := range defs {
		ids = append(ids, id)
	}
	texts, err := LoadAssetTexts(ctx, metadata, AssetTypeCommendation, ids)
	if err != nil {
		slog.WarnContext(ctx, "commendations: noms par langue illisibles, identifiants affichés", "err", err)
		return
	}
	loc := i18n.FromContext(ctx)
	for id, def := range defs {
		def.name = texts[id].Name.Pick(loc)
		defs[id] = def
	}
}
