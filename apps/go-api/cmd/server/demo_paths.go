// demo_paths.go — résolution des chemins de DB en mode démo.
//
// Isolé de main.go pour être TESTABLE : la règle qu'il porte a été enfreinte
// pendant des mois sans que rien ne le signale (cf. demo_paths_test.go).
package main

import "path/filepath"

// demoWarehouseDBPath traduit un chemin de warehouse de PRODUCTION en son
// équivalent dans la fixture démo.
//
// INVARIANT : la traduction est INCONDITIONNELLE. Elle ne doit JAMAIS dépendre
// de l'existence du fichier de production ni de celle du fichier de fixture.
//
// L'implémentation précédente ne basculait sur la fixture que si la DB de
// production était ABSENTE. Sur toute machine possédant des données réelles —
// c'est-à-dire tout poste de développement — la démo servait donc la PRODUCTION :
// le joueur `demo-player` n'y ayant aucun match, les pages sortaient vides
// (healthz/home 503, `teammates … top_rows_count=0`) et le harnais visuel
// skippait 6 pages sur 7 en annonçant « données absentes ». Les migrations de
// boot s'appliquaient de surcroît aux bases de production, qu'un simple
// lancement de démo modifiait.
//
// Le nom de fichier est conservé (shared_matches_v2.duckdb, metadata.duckdb, …) :
// la fixture produite par `seed-demo --synthetic` utilise les mêmes noms sous
// `{fixtures}/warehouse/`.
func demoWarehouseDBPath(demoFixturesDir, prodPath string) string {
	return filepath.Join(demoFixturesDir, "warehouse", filepath.Base(prodPath))
}
