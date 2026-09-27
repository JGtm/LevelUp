package killcollector

// postsync_travail.go — LA LISTE DE TRAVAIL D UN CYCLE, SANS LES MATCHS SANS CARTE (2026-09-27).
//
// Sorti de `postsync.go` (482 lignes) pour tenir le seuil de 500 : la pagination est le seul ajout.
//
// POURQUOI UNE PAGINATION. Retirer les matchs sans carte d UNE page de backlog ne suffit pas : ils
// ne quittent jamais le backlog, et le jour ou plus de `horizon` d entre eux sont en tete, la page
// n en contient plus aucun avec carte — le cycle ne traiterait RIEN et l affamement reviendrait,
// simplement plus tard. Le cycle lit donc les pages suivantes jusqu a remplir `perCycle`, dans la
// limite de [PostSyncBacklogPagesMax] pages.

import (
	"context"
	"database/sql"
)

// PostSyncBacklogPagesMax borne le nombre de pages de backlog lues par cycle : 8 x 64 = 512
// candidats au plus. C est la borne du COUT de lecture (une resolution de carte par match sans
// carte rencontre) ; au-dela de 512 matchs sans carte en tete de backlog, un cycle peut encore
// rester vide, et le compteur `killsource_ecartes_carte_avant_telechargement` le dit.
const PostSyncBacklogPagesMax = 8

// travailDuCycle rend la liste de travail du cycle — au plus `perCycle` matchs, TOUS avec carte,
// dans l ordre d `ordonnancer` (les inseres d abord, puis du plus recent au plus vieux) — et la
// matchs sans carte retires, leur place allant au suivant ; aucun n est telecharge.
func (h *PostSyncHook) travailDuCycle(
	ctx context.Context, d PostSyncDeps, col *KillSourceCollector, premiere []string, insertedIDs []string,
) []string {
	vus := make(map[string]bool)
	var travail []string
	page := premiere
	for n := 0; ; n++ {
		prioritaires := insertedIDs
		if n > 0 {
			prioritaires = nil // les inseres ne sont prioritaires que dans la PREMIERE page
		}
		ordre, _ := ordonnancer(page, prioritaires, len(page)+len(prioritaires))
		nouveaux := ordre[:0:0]
		for _, id := range ordre {
			if !vus[id] {
				vus[id] = true
				nouveaux = append(nouveaux, id)
			}
		}
		retenus, _ := col.RetenirLesMatchsAvecCarte(ctx, nouveaux, h.perCycle-len(travail))
		travail = append(travail, retenus...)
		if len(travail) >= h.perCycle || len(page) < h.horizon || n+1 >= PostSyncBacklogPagesMax {
			return travail
		}
		offset := (n + 1) * h.horizon
		d.WithRead(ctx, "killsource_select", func(sharedDB *sql.DB) {
			page, _ = backlogAJour(ctx, sharedDB, h.horizon, offset)
		})
	}
}
