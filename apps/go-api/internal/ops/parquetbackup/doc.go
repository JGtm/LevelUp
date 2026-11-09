// Package parquetbackup : la sauvegarde périodique des bases DuckDB en Parquet vers un dossier choisi par
// l'utilisateur (profil desktop, ADR 0045).
//
// UN JEU DE SAUVEGARDE est un dossier `levelup-<AAAAMMJJ-HHMM>/` du dossier choisi :
//
//	db/<clé>/<table>_<horodatage>.parquet   un Parquet zstd par table de chaque base (clé : ops.BackupDatabases)
//	db/<clé>/_ddl_<horodatage>.json         le schéma de la base à l'instant de l'export
//	config/...                              db_profiles.json, app_settings.json, comptes, groupes, jetons (tels quels)
//	manifest.json                           version de l'application, tables, lignes, durée, sommes SHA-256
//
// Le jeu se construit dans un dossier `.partial-*` renommé à la fin : un dossier `levelup-*` porte
// toujours son manifeste, un jeu interrompu ne laisse que du `.partial-*`, effacé au passage suivant.
//
// RÈGLES :
//   - lecture seule : une base se lit par [duckdb.WithHeldRead] (handle du cache process-wide ou lecture
//     autonome, sous le bail d'écriture de la base) ; aucune instruction d'écriture ni CHECKPOINT ici
//     (garde-rail run_readonly_test.go) ;
//   - une base dont le bail est tenu est sautée au premier passage et reprise à la fin ; une base encore
//     occupée rend le jeu PARTIEL (dit par le manifeste), jamais un échec ;
//   - les films archivés ne sont PAS copiés (taille) : le manifeste le dit ;
//   - la rotation ne supprime que des dossiers `levelup-*` du dossier choisi, direct enfants.
package parquetbackup
