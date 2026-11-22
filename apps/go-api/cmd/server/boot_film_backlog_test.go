// Package main — boot_film_backlog_test.go : l'ordonnanceur lit l'arriéré de films publié par les étapes
// film du post-sync. Sans ce lecteur, la cadence de rattrapage ne suivrait que l'historique et l'écran d'état
// de l'historique dirait « complet » avec des films en retard : ratchet sur la source du câblage.
package main

import (
	"os"
	"strings"
	"testing"
)

func TestBootWiresFilmBacklogReaderOnScheduler(t *testing.T) {
	const src = "boot_tasks.go"
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("lecture %s: %v", src, err)
	}
	if !strings.Contains(string(b), "WithFilmBacklogReader(filmbacklog.Reader{})") {
		t.Error("l'ordonnanceur n'a plus de lecteur de l'arriéré de films : la cadence de 10 minutes ne tiendrait " +
			"plus pendant l'arriéré, et l'historique s'afficherait complet avec des films en retard")
	}
}
