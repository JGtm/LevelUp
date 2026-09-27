package killcollector

// postsync_travail.go — LA LISTE DE TRAVAIL D UN CYCLE, SANS LES MATCHS SANS CARTE (2026-09-27).
//
// Sorti de `postsync.go` (482 lignes) pour tenir le seuil de 500 : la pagination est le seul ajout,
// et la lecture du backlog ([backlogAJour], [pageDuBacklog]) l a rejointe a la revue du correctif.
//
// POURQUOI UNE PAGINATION. Retirer les matchs sans carte d UNE page de backlog ne suffit pas : ils
// ne quittent jamais le backlog, et le jour ou plus de `horizon` d entre eux sont en tete, la page
// n en contient plus aucun avec carte — le cycle ne traiterait RIEN et l affamement reviendrait,
// simplement plus tard. Le cycle lit donc les pages suivantes jusqu a remplir `perCycle`, dans la
// limite de [PostSyncBacklogPagesMax] pages.
//
// CE QU UNE PAGE COUTE, DEPUIS LA REVUE DU CORRECTIF (2026-09-27). La taille du backlog (le
// `COUNT(*)` sans borne) se lit UNE fois par cycle, avec la premiere page ; les pages suivantes ne
// lisent que leurs identifiants. Les matchs deja constates sans carte sous le catalogue courant
// sont sautes sans relire leur carte (registre de `postsync_sans_carte.go`).

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/matchflags"
)

// PostSyncBacklogPagesMax borne le nombre de pages de backlog lues par cycle : 8 x 64 = 512
// candidats au plus. C est la borne du COUT de lecture (une page d identifiants, et une resolution
// de carte par match sans carte rencontre POUR LA PREMIERE FOIS) ; au-dela de 512 matchs sans
// carte en tete de backlog, un cycle peut encore rester vide, et la jauge
// `killsource_postsync_backlog_sans_carte` le dit.
const PostSyncBacklogPagesMax = 8

// sourceDuBacklog : la premiere page (lue avec la taille du backlog) et la lecture des suivantes.
// `lire` rend `ok == false` quand la page n a pas pu etre lue.
type sourceDuBacklog struct {
	premiere []string
	lire     func(offset int) (ids []string, ok bool)
}

// travailDuCycle rend la liste de travail du cycle — au plus `perCycle` matchs, TOUS avec carte,
// dans l ordre d `ordonnancer` (les inseres d abord, puis du plus recent au plus vieux). Les
// matchs sans carte sont retires, leur place allant au suivant ; aucun n est telecharge. Ceux que
// `reg` connait deja sont sautes sans relecture ; les nouveaux constats y sont inscrits. `reg`
// nil : aucun registre (carte non cablee ce cycle).
func (h *PostSyncHook) travailDuCycle(
	ctx context.Context, col *KillSourceCollector, reg *registreSansCarte, src sourceDuBacklog, insertedIDs []string,
) []string {
	lus := make(map[string]bool)
	var (
		travail []string
		sautes  int
	)
	page := src.premiere
	for n := 0; ; n++ {
		prioritaires := insertedIDs
		if n > 0 {
			prioritaires = nil // les inseres ne sont prioritaires que dans la PREMIERE page
		}
		ordre, _ := ordonnancer(page, prioritaires, len(page)+len(prioritaires))
		nouveaux := ordre[:0:0]
		for _, id := range ordre {
			if !lus[id] {
				lus[id] = true
				nouveaux = append(nouveaux, id)
			}
		}
		aResoudre, s := reg.filtrer(nouveaux)
		sautes += s
		retenus, ecartes := col.RetenirLesMatchsAvecCarte(ctx, aResoudre, h.perCycle-len(travail))
		reg.noter(ecartes)
		travail = append(travail, retenus...)
		finDuBacklog := len(page) < h.horizon
		if finDuBacklog {
			reg.elaguer(lus) // le backlog a ete lu jusqu a sa fin : ce qui n y est plus en sort
		}
		if len(travail) >= h.perCycle || finDuBacklog || n+1 >= PostSyncBacklogPagesMax {
			break
		}
		suivante, ok := src.lire((n + 1) * h.horizon)
		if !ok {
			break
		}
		page = suivante
	}
	if sautes > 0 {
		observability.AddInt(metricSansCarteDejaConstates, int64(sautes))
	}
	return travail
}

// lectureDesPages : la lecture des pages suivantes par un segment de LECTURE court.
func lectureDesPages(ctx context.Context, d PostSyncDeps, horizon int) func(offset int) ([]string, bool) {
	return func(offset int) ([]string, bool) {
		var (
			ids []string
			ok  bool
		)
		d.WithRead(ctx, "killsource_select", func(sharedDB *sql.DB) {
			ids, ok = pageDuBacklog(ctx, sharedDB, horizon, offset)
		})
		return ids, ok
	}
}

// argsBacklog : les parametres de [conditionBacklog], une seule copie pour la jauge et la liste.
func argsBacklog() []any {
	return []any{matchflags.MBitFilmAbsent, decfilm.Rev, killscope.ReadPathCreditBackfill}
}

// backlogAJour rend la premiere page de travail (bornee) et la taille TOTALE du backlog — le seul
// `COUNT(*)` du cycle.
//
// ⚠ LECTURE PAR LA VUE `_latest` (ADR 0026) : une lecture brute servirait des passes perimees
// et ferait sauter des matchs a redecoder.
func backlogAJour(ctx context.Context, db *sql.DB, horizon, offset int) (ids []string, total int) {
	total = tailleDuBacklog(ctx, db)
	ids, _ = pageDuBacklog(ctx, db, horizon, offset)
	return ids, total
}

// tailleDuBacklog : le `COUNT(*)` SANS borne, le plus cher des deux lectures. UNE fois par cycle,
// jamais par page. C est une variable de paquet pour une seule raison : la couture qui laisse un
// test COMPTER ses appels sur un cycle pagine (postsync_compte_integration_test.go).
var tailleDuBacklog = func(ctx context.Context, db *sql.DB) (total int) {
	if err := db.QueryRowContext(ctx, requeteBacklogTaille, argsBacklog()...).Scan(&total); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource taille du backlog illisible", "err", err)
		// On continue : une jauge absente ne doit pas empecher le travail.
	}
	return total
}

// pageDuBacklog lit UNE page d identifiants, sans la taille. `ok == false` : page illisible.
func pageDuBacklog(ctx context.Context, db *sql.DB, horizon, offset int) (ids []string, ok bool) {
	rows, err := db.QueryContext(ctx, requeteBacklog, append(argsBacklog(), horizon, offset)...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: killsource backlog illisible", "err", err)
		return nil, false
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.WarnContext(ctx, "post-sync: killsource backlog (scan)", "err", err)
			return ids, false
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: killsource backlog (rows)", "err", err)
		return ids, false
	}
	return ids, true
}
