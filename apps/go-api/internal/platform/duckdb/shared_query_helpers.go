package duckdb

import "strings"

// Package-level helpers pour la migration B1 (sprint sharedprovider, commit
// 8k.0). Utilisés par les repos qui exécutent leurs queries shared via
// `pdb.SharedReadDB().Get(ctx)` au lieu de l'ancien pattern ATTACH +
// `r.pdb.Player.Query(... "JOIN shared.X ...")`.
//
// Ces helpers factorisent les patterns récurrents de la migration :
//   1. Charger un set d'IDs depuis player (ex: match_ids du joueur)
//   2. Charger les rows shared.X correspondantes via une IN clause
//   3. Merger en Go via une map[ID]Row
//
// L'utilisation typique :
//
//	matchIDs := loadPlayerMatchIDs(...)
//	ph := Placeholders(len(matchIDs))
//	q := fmt.Sprintf("SELECT match_id, x FROM match_registry WHERE match_id IN (%s)", ph)
//
//	db, release, err := r.pdb.SharedReadDB().Get(ctx)
//	if err != nil { return nil, err }
//	defer release()
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
//	...

// Placeholders construit la chaîne "?, ?, ?, ..." pour une IN clause SQL.
// Retourne "" si n <= 0 (le caller doit gérer le cas de slice vide en
// amont — exécuter une query avec IN vide cause une erreur SQL).
//
// Exemples :
//
//	Placeholders(0) → ""
//	Placeholders(1) → "?"
//	Placeholders(3) → "?, ?, ?"
//
// Pattern d'usage :
//
//	if len(ids) == 0 {
//	    return nil, nil // ou retour neutre selon le cas métier
//	}
//	q := fmt.Sprintf("SELECT * FROM X WHERE id IN (%s)", Placeholders(len(ids)))
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(ids)...)
func Placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?, ", n), ", ")
}

// ToAnySlice convertit une slice typée en []any pour passage variadique à
// `db.QueryContext(ctx, q, args...)`. Évite la boucle manuelle de conversion
// que chaque caller devrait écrire sinon.
//
// Pattern d'usage :
//
//	matchIDs := []string{"m1", "m2", "m3"}
//	q := fmt.Sprintf("... WHERE match_id IN (%s)", Placeholders(len(matchIDs)))
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
//
// Utiliser de préférence ce helper plutôt que :
//
//	args := make([]any, len(matchIDs))
//	for i, id := range matchIDs { args[i] = id }
func ToAnySlice[T any](s []T) []any {
	if len(s) == 0 {
		return nil
	}
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
