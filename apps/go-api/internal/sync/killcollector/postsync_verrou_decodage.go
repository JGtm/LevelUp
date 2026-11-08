package killcollector

// postsync_verrou_decodage.go — LE VERROU SOLO DE DECODAGE, PRIS PAR L ETAPE 1.57 DANS LE PROCESSUS DU SERVEUR.
//
// Cette etape DECODE des films dans le processus du serveur (kill-feed, tirs, positions) — c est le seul decodage de
// film qui s y fasse : la cuisson des rejeux part dans un enfant, et son parent prend `filmrun.AcquireSolo` avant
// de le lancer. Regle du projet : JAMAIS DEUX DECODAGES DE FILM A LA FOIS sur la machine (pic memoire cumule). Le
// meme verrou borne donc l etape 1.57, la cuisson apres synchronisation, l action admin « construire le rejeu » et la
// file de regeneration des rejeux de l application de bureau.
//
// OCCUPE = REPORTE. Comme la cuisson apres synchronisation (`replaychild.Spawn`) et comme la porte du mode jeu, l etape
// ne fait pas la queue : le backlog se relit en base a chaque cycle (`decoder_rev` fait foi), le premier cycle qui
// trouve le verrou libre le rattrape. Ce n est ni une erreur ni un echec : le cycle continue, la passe rend 0, un
// compteur et une ligne INFO disent pourquoi.

import (
	"context"
	"errors"
	"log/slog"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmrun"
	"levelup/go-api/internal/observability"
)

// CompteurPostSyncReporteeVerrou : passes reportees parce qu un autre decodage de film tenait le verrou solo.
const CompteurPostSyncReporteeVerrou = "killsource_postsync_reportee_verrou_decodage"

// outilPostSync : le nom sous lequel l etape se declare dans le verrou (un operateur qui se voit refuser le verrou lit
// QUI decode).
const outilPostSync = "post-sync killsource"

// prendreLeVerrouDeDecodage prend le verrou solo de la machine, en refus immediat, avant toute passe de decodage.
// Rend (release, true) quand l etape peut decoder ; (nil, false) quand elle doit se reporter au cycle suivant. Un
// verrou qui ne se prend pas pour une autre raison que l occupation (disque) reporte aussi — jamais de decodage sans
// le verrou — mais se journalise en WARN.
func (h *PostSyncHook) prendreLeVerrouDeDecodage(ctx context.Context, d PostSyncDeps) (func(), bool) {
	lock, err := filmrun.AcquireSolo(titlePkg.NewPathResolver(h.repoRoot).CacheRootDir(), outilPostSync, "")
	if err == nil {
		return lock.Release, true
	}
	if errors.Is(err, filmrun.ErrDecodeBusy) {
		observability.IncCounter(CompteurPostSyncReporteeVerrou)
		slog.InfoContext(ctx, "post-sync: kill source reportee — un autre decodage de film tient la machine, "+
			"reprise au cycle suivant", "title", d.TitleSlug, "gamertag", d.Gamertag, "detenteur", err)
		return nil, false
	}
	slog.WarnContext(ctx, "post-sync: kill source reportee — verrou de decodage indisponible",
		"title", d.TitleSlug, "gamertag", d.Gamertag, "err", err)
	return nil, false
}
