package weapons

// label_names.go — écriture d'un nom d'arme dans weapon_label_names (weapon_id, lang, name), la
// table des noms par langue posés par l'API Halo ou le seed (ADR 0040 D-5, plan i18n 8.4).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
)

// SetLabelName pose le nom de l'arme `weaponID` dans la langue `lang` : SELECT puis UPDATE ou
// INSERT (jamais ON CONFLICT DO UPDATE), donc une langue posée ailleurs n'est jamais touchée. Un
// nom vide est ignoré (une langue sans nom reste absente, jamais inventée). Rend true quand une
// ligne a été écrite (nouvelle ou modifiée).
//
// L'identifiant est écrit en littéral décimal : database/sql ne lie pas un uint64 dont le bit 63
// est posé (cas des identifiants du film), et weapon_id n'est jamais une saisie utilisateur.
func SetLabelName(ctx context.Context, db *sql.DB, weaponID uint64, lang i18n.Locale, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	var cur string
	err := db.QueryRowContext(ctx, //nolint:gosec
		fmt.Sprintf("SELECT name FROM weapon_label_names WHERE weapon_id = %d AND lang = ?", weaponID),
		string(lang)).Scan(&cur)
	switch {
	case err == nil && cur == name:
		return false, nil
	case err == nil:
		_, err = db.ExecContext(ctx, //nolint:gosec
			fmt.Sprintf("UPDATE weapon_label_names SET name = ? WHERE weapon_id = %d AND lang = ?", weaponID),
			name, string(lang))
	case errors.Is(err, sql.ErrNoRows):
		_, err = db.ExecContext(ctx, //nolint:gosec
			fmt.Sprintf("INSERT INTO weapon_label_names (weapon_id, lang, name) VALUES (%d, ?, ?)", weaponID),
			string(lang), name)
	}
	if err != nil {
		return false, fmt.Errorf("weapon_label_names %d/%s: %w", weaponID, lang, err)
	}
	return true, nil
}
