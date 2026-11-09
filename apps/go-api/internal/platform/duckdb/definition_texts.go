package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// DefinitionTexts porte le nom et la description d'une définition (médaille, commendation,
// couleur d'équipe, succès, jalon) dans chaque langue pour laquelle une ligne de traduction existe.
// La langue demandée se choisit par `Name.Pick(loc)` : langue, puis langue de repli, jamais une
// traduction inventée (ADR 0040 D-4).
type DefinitionTexts struct {
	Name        i18n.Text
	Description i18n.Text
}

// metaQuerier est ce que ces lecteurs demandent à la base metadata (`*DB` le satisfait).
type metaQuerier interface {
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Types d'actifs de asset_translations qui portent les définitions migrées par
// metadata_definition_texts_to_translations_v1 (le texte d'un succès verrouillé est rangé dans
// `name` de AssetTypeAchievementLocked).
const (
	AssetTypeCommendation        = "commendation"
	AssetTypeTeamColor           = "team_color"
	AssetTypeAchievement         = "achievement"
	AssetTypeAchievementLocked   = "achievement_locked"
	AssetTypeMilestone           = "milestone"
	AssetTypeCitation            = "citation"
	AssetTypeChallengeTemplate   = "challenge_template"
	AssetTypePresetArc           = "preset_arc"
	definitionTextsInChunkLength = 500
)

// LoadAssetTexts lit les textes par langue des actifs `ids` du type `assetType`. Un identifiant
// sans ligne de traduction est absent du résultat. Une étiquette qui ne correspond à aucune langue
// de la liste est ignorée ; quand plusieurs étiquettes d'une même langue coexistent, la première
// étiquette Halo de la langue l'emporte, puis l'ordre alphabétique.
func LoadAssetTexts(ctx context.Context, q metaQuerier, assetType string, ids []string) (map[string]DefinitionTexts, error) {
	out := make(map[string]DefinitionTexts, len(ids))
	for start := 0; start < len(ids); start += definitionTextsInChunkLength {
		end := min(start+definitionTextsInChunkLength, len(ids))
		chunk := ids[start:end]
		args := append([]any{assetType}, ToAnySlice(chunk)...)
		err := scanDefinitionTexts(ctx, q,
			`SELECT asset_id, lang, name, COALESCE(description, '') FROM asset_translations
			 WHERE asset_type = ? AND asset_id IN (`+Placeholders(len(chunk))+`) ORDER BY asset_id, lang`,
			args, out)
		if err != nil {
			return nil, fmt.Errorf("LoadAssetTexts(%s): %w", assetType, err)
		}
	}
	return out, nil
}

// LoadAssetNames lit les NOMS par langue des actifs `ids` du type `assetType` (sans la description) :
// les lecteurs d'affichage de cartes, playlists, paires et variantes n'ont pas besoin de la
// colonne `description`. Mêmes règles que LoadAssetTexts ; `Description` du résultat est vide.
func LoadAssetNames(ctx context.Context, q metaQuerier, assetType string, ids []string) (map[string]DefinitionTexts, error) {
	out := make(map[string]DefinitionTexts, len(ids))
	for start := 0; start < len(ids); start += definitionTextsInChunkLength {
		end := min(start+definitionTextsInChunkLength, len(ids))
		chunk := ids[start:end]
		args := append([]any{assetType}, ToAnySlice(chunk)...)
		err := scanDefinitionTexts(ctx, q,
			`SELECT asset_id, lang, name, '' FROM asset_translations
			 WHERE asset_type = ? AND asset_id IN (`+Placeholders(len(chunk))+`) ORDER BY asset_id, lang`,
			args, out)
		if err != nil {
			return nil, fmt.Errorf("LoadAssetNames(%s): %w", assetType, err)
		}
	}
	return out, nil
}

// LoadAllAssetTexts lit les textes par langue de tous les actifs d'un type (tables de quelques
// dizaines de lignes : jalons, couleurs d'équipe).
func LoadAllAssetTexts(ctx context.Context, q metaQuerier, assetType string) (map[string]DefinitionTexts, error) {
	out := map[string]DefinitionTexts{}
	err := scanDefinitionTexts(ctx, q,
		`SELECT asset_id, lang, name, COALESCE(description, '') FROM asset_translations
		 WHERE asset_type = ? ORDER BY asset_id, lang`,
		[]any{assetType}, out)
	if err != nil {
		return nil, fmt.Errorf("LoadAllAssetTexts(%s): %w", assetType, err)
	}
	return out, nil
}

// LoadMedalTexts lit les textes par langue des médailles `ids` (medal_translations).
func LoadMedalTexts(ctx context.Context, q metaQuerier, ids []int64) (map[int64]DefinitionTexts, error) {
	byID := make(map[string]DefinitionTexts, len(ids))
	for start := 0; start < len(ids); start += definitionTextsInChunkLength {
		end := min(start+definitionTextsInChunkLength, len(ids))
		chunk := ids[start:end]
		err := scanDefinitionTexts(ctx, q,
			`SELECT CAST(medal_name_id AS VARCHAR), lang, name, COALESCE(description, '') FROM medal_translations
			 WHERE medal_name_id IN (`+Placeholders(len(chunk))+`) ORDER BY medal_name_id, lang`,
			ToAnySlice(chunk), byID)
		if err != nil {
			return nil, fmt.Errorf("LoadMedalTexts: %w", err)
		}
	}
	out := make(map[int64]DefinitionTexts, len(byID))
	for _, id := range ids {
		if t, ok := byID[fmt.Sprint(id)]; ok {
			out[id] = t
		}
	}
	return out, nil
}

// scanDefinitionTexts exécute `query` (colonnes : identifiant, étiquette, nom, description) et
// range chaque ligne dans `out` sous la langue de son étiquette.
func scanDefinitionTexts(ctx context.Context, q metaQuerier, query string, args []any, out map[string]DefinitionTexts) error {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, tag, name, desc string
		if err := rows.Scan(&id, &tag, &name, &desc); err != nil {
			return err
		}
		loc, ok := i18n.FromHaloTag(tag)
		if !ok {
			continue
		}
		t, seen := out[id]
		if !seen {
			t = DefinitionTexts{Name: i18n.Text{}, Description: i18n.Text{}}
		}
		// Plusieurs étiquettes pour une langue : l'étiquette principale l'emporte sur les autres.
		primary := strings.EqualFold(tag, i18n.HaloTags(loc)[0])
		if _, has := t.Name[loc]; !has || primary {
			if strings.TrimSpace(name) != "" {
				t.Name[loc] = name
			}
			if strings.TrimSpace(desc) != "" {
				t.Description[loc] = desc
			}
		}
		out[id] = t
	}
	return rows.Err()
}

// UpsertAssetTexts écrit le nom et la description d'une définition par langue dans
// asset_translations (migration.UpsertAssetTexts : SELECT-puis-UPDATE-ou-INSERT, jamais d'ON
// CONFLICT), avec la réouverture d'un handle invalidé. Pour les dépôts de référentiels (modèles de
// défi, arcs préréglés) dont les textes ne sont plus des colonnes.
func (db *DB) UpsertAssetTexts(ctx context.Context, assetType, assetID string, names, descriptions i18n.Text) error {
	return db.WithReopenOnInvalidated(func() error {
		return migration.UpsertAssetTexts(ctx, db.loadSQL(), assetType, assetID, names, descriptions)
	})
}
