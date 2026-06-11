// Package ops — data_quality_resolve.go : écritures de résolution metadata
// du dashboard admin (traductions de modes + traductions d'assets).
//
// Pattern d'écriture : transaction courte + SELECT-then-UPDATE-or-INSERT —
// JAMAIS d'ON CONFLICT (piège : CREATE TABLE IF NOT EXISTS n'ajoute pas la PK
// à une table préexistante du prebuilt → ON CONFLICT échouerait). Basse
// fréquence (action admin manuelle), aucun risque ART.
//
// Effets immédiats : mode_name_tr et asset_translations sont lus LIVE par les
// repos d'affichage (home_repo, match_history_fr_translations) — aucune
// invalidation de cache nécessaire côté serveur.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ResolveActionCreated / Updated : valeur de retour des upserts (affichée par
// le front dans le toast).
const (
	ResolveActionCreated = "created"
	ResolveActionUpdated = "updated"
)

// UpsertModeTranslation insère ou met à jour mode_name_tr (clé mode_en+lang).
// modeEN doit déjà être normalisé (analysis.NormalizeModeLabel) par le caller.
func UpsertModeTranslation(ctx context.Context, metaDB *sql.DB, modeEN, lang, name string) (string, error) {
	modeEN, lang, name = strings.TrimSpace(modeEN), strings.TrimSpace(lang), strings.TrimSpace(name)
	if metaDB == nil {
		return "", fmt.Errorf("mode translation: metadata DB nil")
	}
	if modeEN == "" || lang == "" || name == "" {
		return "", fmt.Errorf("mode translation: champs vides")
	}
	return upsertTwoKeyRow(ctx, metaDB, upsertSpec{
		selectQ: `SELECT 1 FROM mode_name_tr WHERE mode_en = ? AND lang = ?`,
		updateQ: `UPDATE mode_name_tr SET name = ? WHERE mode_en = ? AND lang = ?`,
		insertQ: `INSERT INTO mode_name_tr (mode_en, lang, name) VALUES (?, ?, ?)`,
		keys:    []any{modeEN, lang},
		name:    name,
	})
}

// UpsertAssetTranslation insère ou met à jour asset_translations (clé
// asset_id+asset_type+lang). C'est LA table lue par toute la chaîne
// d'affichage (résolution UUID → nom) et par le backfill registry names.
func UpsertAssetTranslation(ctx context.Context, metaDB *sql.DB, assetType, assetID, lang, name string) (string, error) {
	assetType = strings.TrimSpace(assetType)
	assetID = strings.TrimSpace(assetID)
	lang = strings.TrimSpace(lang)
	name = strings.TrimSpace(name)
	if metaDB == nil {
		return "", fmt.Errorf("asset translation: metadata DB nil")
	}
	if assetType == "" || assetID == "" || lang == "" || name == "" {
		return "", fmt.Errorf("asset translation: champs vides")
	}
	return upsertTwoKeyRow(ctx, metaDB, upsertSpec{
		selectQ: `SELECT 1 FROM asset_translations WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
		updateQ: `UPDATE asset_translations SET name = ? WHERE asset_id = ? AND asset_type = ? AND lang = ?`,
		insertQ: `INSERT INTO asset_translations (asset_id, asset_type, lang, name) VALUES (?, ?, ?, ?)`,
		keys:    []any{assetID, assetType, lang},
		name:    name,
	})
}

type upsertSpec struct {
	selectQ, updateQ, insertQ string
	keys                      []any
	name                      string
}

// upsertTwoKeyRow exécute le SELECT-then-UPDATE-or-INSERT en transaction
// courte. Retourne ResolveActionCreated ou ResolveActionUpdated.
func upsertTwoKeyRow(ctx context.Context, db *sql.DB, spec upsertSpec) (string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op après Commit

	var one int
	err = tx.QueryRowContext(ctx, spec.selectQ, spec.keys...).Scan(&one)
	switch {
	case err == sql.ErrNoRows:
		args := append([]any{}, spec.keys...)
		args = append(args, spec.name)
		if _, err := tx.ExecContext(ctx, spec.insertQ, args...); err != nil {
			return "", fmt.Errorf("insert: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return ResolveActionCreated, nil
	case err != nil:
		return "", fmt.Errorf("select: %w", err)
	default:
		args := append([]any{spec.name}, spec.keys...)
		if _, err := tx.ExecContext(ctx, spec.updateQ, args...); err != nil {
			return "", fmt.Errorf("update: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return ResolveActionUpdated, nil
	}
}
