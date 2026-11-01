package handlers

// settings_film_archive.go — l'archive des films dans les réglages (lot L14, ADR 0045 D-12) :
// validation des deux clés du PATCH et occupation exposée par GET /settings/film-archive.
//
// L'OCCUPATION EST MESURÉE UNE FOIS PUIS MISE EN CACHE. Elle parcourt chaque fichier de l'archive
// (des dizaines de milliers pour un parc complet) : la refaire à chaque ouverture de la page des
// réglages serait un disque qui tourne pour rien. La mesure est bornée par un délai (le parcours
// interrompu rend des minorants, marqués `partial`) et valide [filmArchiveMeterTTL].

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/JGtm/LevelUp/libs/film/filmcache"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/ops"
	settings_platform "levelup/go-api/internal/platform/settings"
)

const (
	// filmArchiveMeterTTL : durée de validité d'une mesure d'occupation.
	filmArchiveMeterTTL = 10 * time.Minute
	// filmArchiveMeterBudget : durée maximale d'un parcours de l'archive.
	filmArchiveMeterBudget = 20 * time.Second
)

// Codes d'erreur de la validation (le texte vient du catalogue `settings.film_archive.*`).
const (
	codeFilmArchiveRetention   = "film_archive_retention_conflict"
	codeFilmArchiveNotAbsolute = "film_archive_not_absolute"
	codeFilmArchiveTarget      = "film_archive_target_not_empty"
	codeFilmArchiveMove        = "film_archive_move_required"
)

// filmArchiveMeter garde la dernière mesure d'occupation d'une archive.
type filmArchiveMeter struct {
	mu     sync.Mutex
	dir    string
	at     time.Time
	occ    filmcache.Occupation
	loaded bool
}

// measure rend la mesure en cache, ou en refait une si elle est périmée ou si le dossier a changé.
func (m *filmArchiveMeter) measure(ctx context.Context, dir string, now time.Time) (filmcache.Occupation, time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loaded && m.dir == dir && now.Sub(m.at) < filmArchiveMeterTTL {
		return m.occ, m.at
	}
	mctx, cancel := context.WithTimeout(ctx, filmArchiveMeterBudget)
	defer cancel()
	occ, err := filmcache.MesurerOccupation(mctx, dir)
	if err != nil {
		slog.WarnContext(ctx, "settings: mesure de l'archive des films impossible", "dir", dir, "err", err)
	}
	m.dir, m.at, m.occ, m.loaded = dir, now, occ, true
	return occ, now
}

// filmArchiveRefus construit le refus 400 d'un code, avec son message déjà rédigé dans la langue
// de la requête (`i18n.Msg(i18n.FromContext(ctx), "settings.film_archive.…", …)` à l'appel : la
// clé reste un littéral, tenu par le garde-rail du catalogue).
func filmArchiveRefus(code, message string) error {
	return humacore.NewError(http.StatusBadRequest, code, message)
}

// validateFilmArchive juge le PATCH sur l'archive : réservé à l'administrateur quand les comptes
// sont appliqués, combinaison « sans archive + rétention finie » refusée, et changement
// d'emplacement soumis à la règle du déplacement. `next` est le réglage APRÈS application.
func (h *SettingsHandler) validateFilmArchive(ctx context.Context, req *domain.UpdateSettingsRequest, prev, next *settings_platform.AppSettings) error {
	if req.ArchiveFilms == nil && req.FilmArchiveDir == nil && req.ReplayRetentionMonths == nil {
		return nil
	}
	touched := req.ArchiveFilms != nil || req.FilmArchiveDir != nil
	// Comptes appliqués (capacité du profil, ADR 0044 D-4) : seul un administrateur règle l'archive.
	if touched && h.cfg.Capabilities.UseAccounts && !authz.IsAdmin(middleware.GetSession(ctx), h.users) {
		slog.WarnContext(ctx, "settings: réglage de l'archive des films refusé (admin requis)")
		return humacore.NewError(http.StatusForbidden, "admin_required",
			"Seul un administrateur peut modifier l'archive des films.")
	}
	if err := settings_platform.CheckFilmArchive(next); err != nil {
		return filmArchiveRefus(codeFilmArchiveRetention,
			i18n.Msg(i18n.FromContext(ctx), "settings.film_archive.retention_conflict", nil))
	}
	if req.FilmArchiveDir == nil || strings.TrimSpace(*req.FilmArchiveDir) == strings.TrimSpace(prev.FilmArchiveDir) {
		return nil
	}
	next.FilmArchiveDir = strings.TrimSpace(*req.FilmArchiveDir)
	return h.validateFilmArchiveMove(ctx, next.FilmArchiveDir)
}

// validateFilmArchiveMove applique la règle du déplacement à un nouveau dossier (vide = défaut).
//
// L'ARCHIVE ACTUELLE CONTIENT DES FILMS : refus, la commande `levelup film archive-move` les range
// serveur arrêté (dizaines de Go, irremplaçables : ni dans une requête HTTP, ni sous un serveur qui
// les lit). Sinon la cible doit être vide ou inexistante : on ne mélange pas deux archives.
func (h *SettingsHandler) validateFilmArchiveMove(ctx context.Context, target string) error {
	pr := h.cfg.PathResolver()
	if target != "" && !filepath.IsAbs(target) {
		return filmArchiveRefus(codeFilmArchiveNotAbsolute,
			i18n.Msg(i18n.FromContext(ctx), "settings.film_archive.not_absolute", nil))
	}
	effective := pr.FilmArchiveDir()
	resolved := target
	if resolved == "" {
		resolved = pr.Roots().DefaultFilmArchiveDir()
	}
	if films, err := filmcache.ListShortIDs(effective); err != nil {
		slog.WarnContext(ctx, "settings: archive des films illisible, déplacement refusé", "dir", effective, "err", err)
		return humacore.NewErrorFrom(ctx, http.StatusInternalServerError, "film_archive_unreadable", err)
	} else if len(films) > 0 && !samePath(effective, resolved) {
		return filmArchiveRefus(codeFilmArchiveMove,
			i18n.Msg(i18n.FromContext(ctx), "settings.film_archive.move_required", i18n.Params{"films": len(films)}))
	}
	if err := checkFilmArchiveTarget(resolved, target == ""); err != nil {
		return filmArchiveRefus(codeFilmArchiveTarget,
			i18n.Msg(i18n.FromContext(ctx), "settings.film_archive.target_not_empty", i18n.Params{"dir": resolved}))
	}
	return nil
}

// checkFilmArchiveTarget : un dossier explicite doit être vide ou absent ; le dossier par défaut
// (le cache applicatif, jamais vide) ne doit pas contenir de film.
func checkFilmArchiveTarget(dir string, isDefault bool) error {
	if !isDefault {
		return filmcache.ValiderCible(dir)
	}
	films, err := filmcache.ListShortIDs(dir)
	if err != nil {
		return err
	}
	if len(films) > 0 {
		return filmcache.ErrCibleNonVide
	}
	return nil
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// filmArchiveStatusOutput : le corps de GET /settings/film-archive.
type filmArchiveStatusOutput struct {
	Body *domain.FilmArchiveStatus
}

// handleGetFilmArchive expose l'emplacement de l'archive, son occupation (mesure en cache) et
// l'espace de son volume.
// GET /settings/film-archive
func (h *SettingsHandler) handleGetFilmArchive(ctx context.Context, _ *struct{}) (*filmArchiveStatusOutput, error) {
	cfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}
	dir := h.cfg.PathResolver().FilmArchiveDir()
	occ, at := h.filmMeter.measure(ctx, dir, time.Now())
	configured := strings.TrimSpace(cfg.FilmArchiveDir)
	out := &domain.FilmArchiveStatus{
		Dir:           dir,
		ConfiguredDir: configured,
		ArchiveFilms:  cfg.ArchiveFilms,
		Films:         occ.Films,
		Bytes:         occ.Octets,
		Partial:       occ.Partielle,
		MeasuredAt:    at,
	}
	out.RestartRequired = !samePath(filmArchiveWanted(h, configured), dir)
	if vol, verr := ops.MeasureVolume(dir); verr != nil {
		slog.WarnContext(ctx, "settings: espace du volume de l'archive indisponible", "dir", dir, "err", verr)
	} else {
		out.Volume = &domain.FilmArchiveVolume{
			FreeBytes: vol.FreeBytes, TotalBytes: vol.TotalBytes, Status: vol.Status,
			UsedPercent: int(ops.DiskUsedPercent(vol.FreeBytes, vol.TotalBytes)),
		}
	}
	return &filmArchiveStatusOutput{Body: out}, nil
}

// filmArchiveWanted rend le dossier que le réglage enregistré désigne (vide = défaut).
func filmArchiveWanted(h *SettingsHandler, configured string) string {
	if configured == "" {
		return h.cfg.PathResolver().Roots().DefaultFilmArchiveDir()
	}
	return configured
}
