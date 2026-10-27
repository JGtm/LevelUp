package duckdb

// pair_names_locale.go — le nom d'une paire (mode d'un match, `pair_name` du registre) dans la langue
// de la requête, pour les lectures qui ne portent que le nom brut du registre (celui de la langue de
// repli) : duels des rivalités, diagnostic LUSR de l'administration. La règle est celle de
// l'historique (applyPairNameTexts : mode_name_tr, puis traduction de l'actif de la paire, puis nom
// brut), choisie par la langue de la requête (i18n.Text.Pick).

import (
	"context"

	"levelup/go-api/internal/i18n"
)

// PairRef désigne une paire à nommer : l'identifiant de son actif (vide s'il est inconnu) et son nom
// au registre (langue de repli, vide s'il est absent).
type PairRef struct {
	ID   string
	Name string
}

// PairNamesIn rend, pour chaque paire de refs et dans le même ordre, son nom dans la langue du
// contexte. Dans la langue de repli, ou sans metadata, le nom du registre est rendu tel quel. Une
// paire sans traduction connue garde son nom du registre (jamais un libellé vide). Best-effort : un
// échec de lecture est journalisé et consigné par les lecteurs de noms, le registre nomme alors la
// paire.
func PairNamesIn(ctx context.Context, meta *DB, refs []PairRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Name
	}
	loc := i18n.FromContext(ctx)
	if loc == i18n.Fallback || meta == nil || len(refs) == 0 {
		return out
	}
	ids := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, r := range refs {
		if _, ok := seen[r.ID]; r.ID != "" && !ok {
			seen[r.ID] = struct{}{}
			ids = append(ids, r.ID)
		}
	}
	assets := assetNameTexts(ctx, meta, assetTypePair, ids)
	keys := modeKeySet{}
	for _, r := range refs {
		keys.addLabel(r.Name)
		keys.addTexts(assets[r.ID])
	}
	modes := modeTextsByLocale(ctx, meta, keys.list())
	for i, r := range refs {
		var name i18n.Text
		if r.Name != "" {
			name = i18n.Text{i18n.Fallback: r.Name}
		}
		applyPairNameTexts(&name, assets[r.ID], modes)
		if picked := name.Pick(loc); picked != "" {
			out[i] = picked
		}
	}
	return out
}
