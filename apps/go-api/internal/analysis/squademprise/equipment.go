package squademprise

// equipment.go — LA CARTE « ÉQUIPEMENT PRIS, ET CE QUE J'EN AI FAIT » (Séries temporelles › Usages,
// décision D4 du plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05).
//
//	périmètre   les matchs MESURÉS (filmés à camp connu), les mêmes pour moi, le reste de mon camp
//	            et le lobby : deux comptes comparés sur deux périmètres différents ne se lisent pas ;
//	lobby       tous les joueurs de ces matchs, adversaire et joueurs sans camp connu compris : dit
//	            si une famille a été tenue par quelqu'un (le web ne liste que celles-là) ;
//	issues      sessionusage.PlayerOutcomeCounts, donc la bascule unique « utilisé » (mur = posé,
//	            autres déployables = charge consommée) ;
//	familles    le périmètre du bilan privé des deux bonus (ressource à part), encadré par les
//	            familles « non mesurées » (grappin, propulseur : mes lâchers seulement). Le répulseur
//	            n'a pas de ligne.

import (
	"slices"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/equipmentusage"
)

// equipmentLineFamilies — l'ordre des lignes : la première famille non mesurée (grappin), les
// familles mesurées dans l'ordre du bilan, puis les autres non mesurées (propulseur).
func equipmentLineFamilies() (ordre []string, mesuree map[string]bool) {
	bonus := sessionusage.PowerupFamilies()
	nonMesurees := equipmentusage.EquipmentUnmeasuredLineFamilies()
	mesuree = map[string]bool{}
	ordre = append(ordre, nonMesurees[:1]...)
	for _, f := range equipmentusage.EquipmentOutcomeFamilies() {
		if !slices.Contains(bonus, f) {
			mesuree[f] = true
			ordre = append(ordre, f)
		}
	}
	return append(ordre, nonMesurees[1:]...), mesuree
}

// equipementCumul — les comptes d'une famille : moi, le reste de mon camp, le lobby entier.
type equipementCumul struct {
	me, rest, lobby         sessionusage.OutcomeCounts
	droppedMe, droppedLobby int
}

// BuildEquipment rend la carte « Équipement ». Nil sans film.
func BuildEquipment(in Input) *domain.EmpriseEquipment {
	if in.Film == nil {
		return nil
	}
	ix := newIndex(&in)
	ordre, mesuree := equipmentLineFamilies()
	cumul := make(map[string]*equipementCumul, len(ordre))
	for _, f := range ordre {
		cumul[f] = &equipementCumul{}
	}
	out := &domain.EmpriseEquipment{Families: make([]domain.EmpriseEquipmentFamily, 0, len(ordre))}
	for _, m := range in.Current {
		_, filme := ix.films[m.MatchID]
		ours, campConnu := ix.tc.PlayerTeam[m.MatchID]
		if !filme || !campConnu {
			continue
		}
		out.MatchesMeasured++
		teamOf := ix.tc.TeamOf[m.MatchID]
		for i := range ix.players[m.MatchID] {
			p := &ix.players[m.MatchID][i]
			moi := p.XUID == in.PlayerXUID
			team, ok := teamOf[p.XUID]
			// Le lobby : tout joueur du match mesuré, l'adversaire et les joueurs sans camp compris.
			cumulerJoueur(cumul, mesuree, p, cible{moi: moi, monCamp: moi || (ok && team == ours)})
		}
	}
	for _, f := range ordre {
		out.Families = append(out.Families, publierFamille(f, mesuree[f], cumul[f]))
	}
	return out
}

// cible — où verser la ligne d'un joueur : toujours au lobby ; à moi, ou au reste de mon camp.
type cible struct{ moi, monCamp bool }

// cumulerJoueur verse une ligne (match, joueur) d'un match mesuré dans les cumuls.
func cumulerJoueur(cumul map[string]*equipementCumul, mesuree map[string]bool, p *sessionusage.PlayerRow, c cible) {
	for f, cu := range cumul {
		if !mesuree[f] {
			cu.droppedLobby += p.DroppedByFamily[f]
			if c.moi {
				cu.droppedMe += p.DroppedByFamily[f]
			}
			continue
		}
		oc := sessionusage.PlayerOutcomeCounts(p, []string{f})
		cu.lobby.Add(oc)
		switch {
		case c.moi:
			cu.me.Add(oc)
		case c.monCamp:
			cu.rest.Add(oc)
		}
	}
}

// publierFamille projette une famille au contrat.
func publierFamille(family string, mesuree bool, c *equipementCumul) domain.EmpriseEquipmentFamily {
	if !mesuree {
		return domain.EmpriseEquipmentFamily{Family: family, DroppedMe: c.droppedMe, DroppedLobby: c.droppedLobby}
	}
	return domain.EmpriseEquipmentFamily{
		Family: family, Measured: true, Me: issuesPubliees(c.me), Rest: issuesPubliees(c.rest),
		Lobby: issuesPubliees(c.lobby),
	}
}

func issuesPubliees(o sessionusage.OutcomeCounts) *domain.EmpriseEquipmentOutcomes {
	return &domain.EmpriseEquipmentOutcomes{Taken: o.Taken, Used: o.Used, Kept: o.Kept, Dropped: o.Dropped}
}
