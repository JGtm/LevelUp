package settings

// film_archive.go — les règles des réglages de l'archive des films (lot L14, ADR 0045 D-12).
//
// UN INVARIANT, POSÉ À LA LECTURE : sans archive, la rétention des rejeux est ILLIMITÉE. La purge
// des rejeux (scheduler/replay_purge_cron.go) supprime des artefacts qu'elle suppose pouvoir
// recuire depuis l'archive ; sans archive, un artefact purgé serait perdu. Le PATCH refuse la
// combinaison (CheckFilmArchive) ; la lecture la REPLIE sur illimité (un fichier édité à la main
// ne doit pas armer une purge irréversible) en le journalisant une fois.

import (
	"errors"
	"log/slog"
	"sync/atomic"

	"levelup/go-api/internal/domain"
)

// ErrNoArchiveFiniteRetention : « ne pas archiver » et une rétention de rejeux finie.
var ErrNoArchiveFiniteRetention = errors.New("settings: archive_films=false requires replay_retention_months=0")

// applyFilmArchive applique les deux clés de l'archive d'une écriture partielle.
func applyFilmArchive(cfg *AppSettings, req *domain.FilmArchiveUpdate) {
	if req.ArchiveFilms != nil {
		cfg.ArchiveFilms = *req.ArchiveFilms
	}
	if req.FilmArchiveDir != nil {
		cfg.FilmArchiveDir = *req.FilmArchiveDir
	}
}

// CheckFilmArchive refuse la combinaison « ne pas archiver » + rétention finie des rejeux.
func CheckFilmArchive(cfg *AppSettings) error {
	if !cfg.ArchiveFilms && cfg.ReplayRetentionMonths != 0 {
		return ErrNoArchiveFiniteRetention
	}
	return nil
}

// foldWarned : le repli a déjà été journalisé dans ce processus (Load est appelée à chaque
// requête, un WARN par appel noierait le journal).
var foldWarned atomic.Bool

// enforceFilmArchiveInvariant replie la rétention sur illimitée quand l'archive est désactivée.
func enforceFilmArchiveInvariant(cfg *AppSettings) {
	if cfg.ArchiveFilms || cfg.ReplayRetentionMonths == 0 {
		return
	}
	if foldWarned.CompareAndSwap(false, true) {
		slog.Warn("settings: archive_films=false avec une rétention de rejeux finie — rétention forcée illimitée",
			"replay_retention_months", cfg.ReplayRetentionMonths)
	}
	cfg.ReplayRetentionMonths = 0
}
