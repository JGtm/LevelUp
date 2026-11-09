package ops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
)

// UpsertAssetTexts écrit le nom et la description d'une définition dans asset_translations, une
// ligne par langue qui porte un texte non vide (étiquette Halo principale de la langue). Écriture
// SELECT-puis-UPDATE-ou-INSERT, sans ON CONFLICT (metadata, ADR 0019) ; une langue sans texte
// n'écrit rien et n'efface rien. Idempotente.
func UpsertAssetTexts(ctx context.Context, metaDB *sql.DB, assetType, assetID string, names, descriptions i18n.Text) error {
	assetType, assetID = strings.TrimSpace(assetType), strings.TrimSpace(assetID)
	if metaDB == nil {
		return errors.New("asset texts: metadata DB nil")
	}
	if assetType == "" || assetID == "" {
		return errors.New("asset texts: type ou identifiant vide")
	}
	for _, l := range i18n.Locales() {
		name, desc := strings.TrimSpace(names[l]), strings.TrimSpace(descriptions[l])
		if name == "" && desc == "" {
			continue
		}
		tags := i18n.HaloTags(l)
		if len(tags) == 0 {
			return fmt.Errorf("asset texts %s/%s: langue %s sans étiquette Halo", assetType, assetID, l)
		}
		if err := upsertTextRow(ctx, metaDB, textRowSpec{
			selectQ: `SELECT 1 FROM asset_translations WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
			updateQ: `UPDATE asset_translations SET name = ?, description = ? WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
			insertQ: `INSERT INTO asset_translations (asset_id, asset_type, lang, name, description) VALUES (?, ?, ?, ?, ?)`,
			keys:    []any{assetID, assetType, tags[0]},
			name:    name,
			desc:    nullIfEmpty(desc),
		}); err != nil {
			return fmt.Errorf("asset texts %s/%s (%s): %w", assetType, assetID, l, err)
		}
	}
	return nil
}

// UpsertMedalTexts écrit le nom et la description d'une médaille dans medal_translations, comme
// UpsertAssetTexts.
func UpsertMedalTexts(ctx context.Context, metaDB *sql.DB, medalNameID int64, names, descriptions i18n.Text) error {
	if metaDB == nil {
		return errors.New("medal texts: metadata DB nil")
	}
	for _, l := range i18n.Locales() {
		name, desc := strings.TrimSpace(names[l]), strings.TrimSpace(descriptions[l])
		if name == "" && desc == "" {
			continue
		}
		tags := i18n.HaloTags(l)
		if len(tags) == 0 {
			return fmt.Errorf("medal texts %d: langue %s sans étiquette Halo", medalNameID, l)
		}
		if err := upsertTextRow(ctx, metaDB, textRowSpec{
			selectQ: `SELECT 1 FROM medal_translations WHERE medal_name_id = ? AND lang = ?`,
			updateQ: `UPDATE medal_translations SET name = ?, description = ? WHERE medal_name_id = ? AND lang = ?`,
			insertQ: `INSERT INTO medal_translations (medal_name_id, lang, name, description) VALUES (?, ?, ?, ?)`,
			keys:    []any{medalNameID, tags[0]},
			name:    name,
			desc:    nullIfEmpty(desc),
		}); err != nil {
			return fmt.Errorf("medal texts %d (%s): %w", medalNameID, l, err)
		}
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type textRowSpec struct {
	selectQ, updateQ, insertQ string
	keys                      []any
	name                      string
	desc                      any
}

// upsertTextRow : SELECT puis UPDATE ou INSERT dans une transaction courte.
func upsertTextRow(ctx context.Context, db *sql.DB, spec textRowSpec) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // sans effet après Commit

	var one int
	err = tx.QueryRowContext(ctx, spec.selectQ, spec.keys...).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		args := append(append([]any{}, spec.keys...), spec.name, spec.desc)
		// L'ordre des colonnes de l'INSERT est (clés..., nom, description).
		if _, err := tx.ExecContext(ctx, spec.insertQ, args...); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	case err != nil:
		return fmt.Errorf("select: %w", err)
	default:
		args := append([]any{spec.name, spec.desc}, spec.keys...)
		if _, err := tx.ExecContext(ctx, spec.updateQ, args...); err != nil {
			return fmt.Errorf("update: %w", err)
		}
	}
	return tx.Commit()
}
