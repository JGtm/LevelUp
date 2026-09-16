//go:build research

// Package filmre est un INSTRUMENT DE RECHERCHE, hors production.
//
// CE QUE C'EST : le releve, sous forme de donnee typee, des ecrivains de composants ECS
// trouves dans `HaloInfinite.exe` par Ghidra en lecture seule pendant la preparation du lot 3.6
// du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`. Les notes qui l'expliquent vivent dans
// `.ai/V7.5/film_re/NOTE_3_6_*.md` ; ce paquet en est la forme MACHINE, pour que le lot 3.6
// n'ait pas a recopier cinquante-sept adresses a la main dans `ecs_table.tsv`.
//
// CE QUE CE N'EST PAS : du code de production. Tous les fichiers portent le tag de compilation
// `research` : `go build ./...`, `go vet ./...` et la CI ne les voient pas. Aucun paquet de
// `internal/` ni de `cmd/` ne l'importe, et aucun ne doit l'importer — le jour ou une de ces
// grammaires est portee, elle vit dans `internal/games/halo_infinite/film/filmdec/` et sa ligne
// de `ecs_table.tsv` est mise a jour DANS LE MEME COMMIT (garde-rails `ecs_table_guard_test.go`).
// Ce paquet, lui, ne bouge plus : il est le releve date, pas la verite courante.
//
// COMMENT S'EN SERVIR :
//
//	cd apps/go-api && go vet -tags=research ./tools/film_re/
//	go run -tags=research ./tools/film_re/cmd/releve   (si un jour on lui ajoute une sortie)
//
// PROVENANCE DE CHAQUE LIGNE : `HaloInfinite.exe`, image base 0x140000000, projet Ghidra du
// 2026-06-04, releve du 2026-09-16. La chaine de resolution (chaine de caracteres -> accesseur
// de nom -> descripteur -> ecrivain a `descripteur + 0x40`) est decrite et VERIFIEE sur quatre
// composants deja portes dans `.ai/V7.5/film_re/NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md`.
package filmre
