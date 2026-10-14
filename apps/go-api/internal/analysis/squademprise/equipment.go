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
//	familles    le périmètre du bilan privé des deux bonus (ressource à part). Une capacité portée
//	            hors bilan (grappin, propulseur, répulseur) n'a pas de ligne : aucune famille dont
//	            l'usage n'est pas lu n'est publiée.

import (
	"slices"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/equipmentusage"
)

// equipmentLineFamilies — l'ordre des lignes : les familles du bilan, hors bonus, dans l'ordre du
// bilan.
func equipmentLineFamilies() []string {
	bonus := sessionusage.PowerupFamilies()
	var ordre []string
	for _, f := range equipmentusage.EquipmentOutcomeFamilies() {
		if !slices.Contains(bonus, f) {
			ordre = append(ordre, f)
		}
	}
	return ordre
}

// equipementCumul — les comptes d'une famille : moi, le reste de mon camp, le lobby entier.
type equipementCumul struct {
	me, rest, lobby sessionusage.OutcomeCounts
}

// BuildEquipment rend la carte « Équipement ». Nil sans film.
func BuildEquipment(in Input) *domain.EmpriseEquipment {
	if in.Film == nil {
		return nil
	}
	ix := newIndex(&in)
	ordre := equipmentLineFamilies()
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
			cumulerJoueur(cumul, p, cible{moi: moi, monCamp: moi || (ok && team == ours)})
		}
	}
	for _, f := range ordre {
		out.Families = append(out.Families, publierFamille(f, cumul[f]))
	}
	return out
}

// cible — où verser la ligne d'un joueur : toujours au lobby ; à moi, ou au reste de mon camp.
type cible struct{ moi, monCamp bool }

// cumulerJoueur verse une ligne (match, joueur) d'un match mesuré dans les cumuls.
func cumulerJoueur(cumul map[string]*equipementCumul, p *sessionusage.PlayerRow, c cible) {
	for f, cu := range cumul {
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
func publierFamille(family string, c *equipementCumul) domain.EmpriseEquipmentFamily {
	return domain.EmpriseEquipmentFamily{
		Family: family, Me: issuesPubliees(c.me), Rest: issuesPubliees(c.rest), Lobby: issuesPubliees(c.lobby),
	}
}

func issuesPubliees(o sessionusage.OutcomeCounts) domain.EmpriseEquipmentOutcomes {
	return domain.EmpriseEquipmentOutcomes{Taken: o.Taken, Used: o.Used, Kept: o.Kept, Dropped: o.Dropped}
}
