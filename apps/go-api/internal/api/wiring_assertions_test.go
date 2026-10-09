// Package api — wiring_assertions_test.go : assertions de compilation sur les
// interfaces que le câblage résout par ASSERTION DE TYPE à l'exécution
// (`daemon.(handlers.TitleWatcher)` dans server_apiv1.go,
// `daemon.(playerdirectory.WatchedReader)` dans server_player_directory.go).
//
// POURQUOI (revue adversariale ronde 2, 2026-09-16) : une assertion de type qui
// échoue rend `ok=false` sans bruit — le suivi live cesserait de suivre les
// pauses/purges, ou l'annuaire perdrait la colonne « suivi », sans qu'aucun
// test ni compilation ne le dise. Ces lignes transforment ce silence en erreur
// de build dès qu'une signature bouge.
package api

import (
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/service/playerdirectory"
	"levelup/go-api/internal/watcher"
)

var (
	_ handlers.TitleWatcher         = (*watcher.Daemon)(nil)
	_ playerdirectory.WatchedReader = (*watcher.Daemon)(nil)
)
