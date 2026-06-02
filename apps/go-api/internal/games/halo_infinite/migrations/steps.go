// Package migrations regroupe les migrations DDL APPARTENANT à Halo Infinite
// (Phase 1.5.1 voie B, ADR 0025). Elles vivaient dans internal/migration/
// (couplées au registre global via init()) ; elles en sortent ici, fournies au
// runner via migration.SetTitleStepsProvider(StepsFor).
//
// Construites avec les helpers exportés du package migration (TableExists,
// AddColumnIfMissing, ExecScript, …). L'ordre d'exécution reste imposé par
// migration.canonicalOrder (order.go) — déplacer un step ici ne le réordonne
// pas.
//
// État transition : Steps() se remplit au fur et à mesure des déplacements
// (b3). Vide = aucun step encore migré (le registre global legacy fournit
// tout, comportement inchangé).
package migrations

import "levelup/go-api/internal/migration"

// Steps retourne toutes les migrations title-owned de Halo Infinite, tous
// targets confondus.
func Steps() []migration.Migration {
	return nil // se remplit en b3 (déplacement fichier par fichier)
}

// StepsFor filtre Steps() par target — c'est la fonction enregistrée comme
// provider via migration.SetTitleStepsProvider.
func StepsFor(target migration.TargetDB) []migration.Migration {
	all := Steps()
	if len(all) == 0 {
		return nil
	}
	out := make([]migration.Migration, 0, len(all))
	for _, m := range all {
		if m.TargetDB == target {
			out = append(out, m)
		}
	}
	return out
}
