package scheduler

// world_leaderboard_failures.go — la synthèse des playlists dont le scrape a échoué pendant une passe.
//
// Une panne du service Halo (5xx) touche toutes les playlists d'une passe : une ligne d'erreur par
// playlist noyait le journal sans rien dire de plus que la première. La passe écrit UNE ligne, au
// niveau WARN quand chaque échec est une panne du service distant, ERROR dès qu'un échec n'en est pas
// une (page illisible, markup changé) : celui-là est à investiguer.

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/util/httpstatus"
)

// scrapeFailures cumule les échecs de scrape d'une passe.
type scrapeFailures struct {
	playlists []string
	statuts   map[int]int // statut HTTP -> nombre de playlists ; 0 = échec sans statut
	exemple   error
	horsPanne bool // au moins un échec qui n'est pas une panne 5xx du service distant
}

func (f *scrapeFailures) add(playlist string, err error) {
	if f.statuts == nil {
		f.statuts = make(map[int]int)
		f.exemple = err
	}
	statut := httpstatus.FromError(err)
	f.statuts[statut]++
	f.playlists = append(f.playlists, playlist)
	if !httpstatus.IsServerError(statut) {
		f.horsPanne = true
	}
}

// log écrit la synthèse de la passe ; rien s'il n'y a eu aucun échec.
func (f *scrapeFailures) log(ctx context.Context, season string) {
	if len(f.playlists) == 0 {
		return
	}
	statuts := make([]int, 0, len(f.statuts))
	for s := range f.statuts {
		statuts = append(statuts, s)
	}
	sort.Ints(statuts)
	level := slog.LevelWarn
	if f.horsPanne {
		level = slog.LevelError
	}
	slog.Log(ctx, level, "world_leaderboard_cron: scrape de playlists échoué",
		"module", logging.ModuleLeaderboard, "season", season,
		"playlists_en_echec", len(f.playlists), "playlists", f.playlists,
		"statuts_http", statuts, "err", f.exemple)
}
