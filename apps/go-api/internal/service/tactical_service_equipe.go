package service

// tactical_service_equipe.go — CE QUE LE DETAIL D'UNE ZONE AJOUTE SUR LES COEQUIPIERS : l'equipe de la
// victime a l'instant de chaque mort, la riposte qui l'a suivie (tueur abattu / abattu en retour), et
// le bilan de la zone.
//
// DEUX SOURCES, BEST-EFFORT COMME LES AUTRES du detail (tactical_service_cellule_enrichir.go) :
//
//	coequipiers   `match_death_mates_latest`, une ligne par coequipier et par mort ;
//	journal       `match_kill_events_latest` x `kill_positions_latest`, les kills publiables des matchs.
//
// Absente ou en echec, une source retire les champs qu'elle nourrit et le journal le dit en la
// nommant ; la liste des contributions est toujours servie. Chacune a sa section de duree (ADR 0036
// I6) et est lue UNE FOIS par requete, bornee aux matchs des contributions (ADR 0036 I2).
//
// L'EQUIPE SE LIT SUR LA CLE EXACTE DE LA LIGNE DE CONTEXTE APPARIEE a la mort (a ± 1,5 s,
// `tactical.ContexteLePlusProche`), jamais sur l'instant de la contribution : les deux tables sont
// ecrites dans la meme passe sous la meme cle. Une mort dont les lignes ne retombent pas sur
// `teammates_total` (match pas encore relu) n'a pas d'equipe, jamais une equipe partielle.

import (
	"context"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

// Noms des sources, tels que le journal les cite.
const (
	sourceCoequipiers = "coequipiers"
	sourceJournal     = "journal"
)

// lectureDeLEquipe : ce que la pose des equipes et des ripostes a lu, pour le bilan de la zone.
type lectureDeLEquipe struct {
	// tueurConnu[i] : la contribution i est une mort dont le journal identifie le tueur (le M de
	// « tueur abattu N sur M »).
	tueurConnu []bool
	// journalLu : le journal a ete lu (le M des frags n'a de sens que dans ce cas).
	journalLu bool
}

// poserEquipeEtRiposte pose l'equipe et la riposte de chaque contribution, puis rend le bilan de la
// zone. `contextes` sont ceux des matchs de ses morts (`lus` faux = source absente).
func (s *TacticalService) poserEquipeEtRiposte(ctx context.Context, mapID string, univers domain.TacticalUnivers,
	retenues []contributionLue,
	out []domain.TacticalContribution, contextes []domain.ContexteDeMort, lus bool) *domain.TacticalBilanZone {
	if len(matchsDesFaces(out)) == 0 {
		return nil
	}
	rayons, _ := s.rayonsParMatch(univers.Matchs)
	apparies := apparierLesContextes(retenues, out, contextes, lus)
	mates := s.lireLesMates(ctx, mapID, apparies)
	journal, journalLu := s.lireLeJournal(ctx, mapID, out)

	lec := lectureDeLEquipe{tueurConnu: make([]bool, len(out)), journalLu: journalLu}
	for i := range out {
		switch out[i].Face {
		case domain.TacticalFaceMort:
			rayon, aUnRayon := rayons[out[i].MatchID]
			if c := apparies[i]; c != nil {
				poserLEquipe(&out[i], *c, mates, rayon, aUnRayon)
			}
			lec.tueurConnu[i] = poserTueurAbattu(&out[i], journal[out[i].MatchID], univers.Equipes[out[i].MatchID])
		case domain.TacticalFaceFrag:
			poserAbattuEnRetour(&out[i], journal[out[i].MatchID])
		}
	}
	return bilanDeZone(out, lec)
}

// matchsDesFaces rend les matchs distincts des contributions de face mort ou frag, tries.
func matchsDesFaces(out []domain.TacticalContribution) []string {
	vus := make(map[string]bool, len(out))
	ids := make([]string, 0, len(out))
	for _, c := range out {
		if (c.Face != domain.TacticalFaceMort && c.Face != domain.TacticalFaceFrag) || vus[c.MatchID] {
			continue
		}
		vus[c.MatchID] = true
		ids = append(ids, c.MatchID)
	}
	return ids
}

// apparierLesContextes rend, pour chaque contribution de face mort, sa ligne de contexte : celle que
// la lecture « isole » porte déjà, sinon la plus proche à ± TolerancePlacementMs parmi `contextes`
// (nil si la source manque ou qu'aucune ligne ne convient).
func apparierLesContextes(retenues []contributionLue, out []domain.TacticalContribution,
	contextes []domain.ContexteDeMort, lus bool) []*domain.ContexteDeMort {
	apparies := make([]*domain.ContexteDeMort, len(out))
	for i := range out {
		switch {
		case out[i].Face != domain.TacticalFaceMort:
		case retenues[i].contexte != nil:
			apparies[i] = retenues[i].contexte
		case lus:
			apparies[i] = tactical.ContexteLePlusProche(contextes, out[i].MatchID, out[i].XUID, out[i].InstantMs)
		}
	}
	return apparies
}

// cleDuContexte : la cle sous laquelle les coequipiers de la mort ont ete ecrits.
func cleDuContexte(c *domain.ContexteDeMort) domain.CleDeMort {
	return domain.CleDeMort{MatchID: c.MatchID, VictimXUID: c.VictimXUID, TimeMs: c.TimeMs}
}

// lireLesMates lit, en une requete bornee aux matchs des morts, les coequipiers des contextes
// apparies qui en ont (`Total` > 0). Source absente ou en echec : table vide, journalisee.
func (s *TacticalService) lireLesMates(ctx context.Context, mapID string,
	apparies []*domain.ContexteDeMort) map[domain.CleDeMort][]domain.MateDeMort {
	var cles []domain.CleDeMort
	matchs := map[string]bool{}
	for _, c := range apparies {
		if c != nil && c.Total > 0 {
			cles = append(cles, cleDuContexte(c))
			matchs[c.MatchID] = true
		}
	}
	if len(cles) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matchs))
	for id := range matchs {
		ids = append(ids, id)
	}
	stop := timing.FromContext(ctx).Section("tactical_cellule_coequipiers")
	lignes, err := s.repo.MatesDesMorts(ctx, domain.TacticalQuery{
		PlayerXUID: s.xuid, MapID: mapID, Matchs: domain.RestreindreAux(ids),
	}, cles)
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceCoequipiers, mapID, err)
		return nil
	}
	par := make(map[domain.CleDeMort][]domain.MateDeMort, len(cles))
	for _, l := range lignes {
		k := domain.CleDeMort{MatchID: l.MatchID, VictimXUID: l.VictimXUID, TimeMs: l.TimeMs}
		par[k] = append(par[k], l)
	}
	return par
}

// poserLEquipe pose l'equipe d'une mort quand ses lignes retombent sur le total de son contexte.
func poserLEquipe(c *domain.TacticalContribution, ctx domain.ContexteDeMort,
	mates map[domain.CleDeMort][]domain.MateDeMort, rayon float64, aUnRayon bool) {
	lignes := mates[cleDuContexte(&ctx)]
	if len(lignes) != ctx.Total {
		return
	}
	eq := tactical.EquipeDeLaMort(ctx, lignes, rayon, aUnRayon, c.AutrePoint)
	c.Equipe = &eq
}

// lireLeJournal lit, en une requete bornee aux matchs des contributions, les kills publiables,
// indexes par match. `lu` est faux quand la source manque ou echoue (journalisee).
func (s *TacticalService) lireLeJournal(ctx context.Context, mapID string,
	out []domain.TacticalContribution) (map[string][]domain.KillDuJournal, bool) {
	ids := matchsDesFaces(out)
	stop := timing.FromContext(ctx).Section("tactical_cellule_journal")
	kills, err := s.repo.JournalDesKills(ctx, domain.TacticalQuery{
		PlayerXUID: s.xuid, MapID: mapID, Matchs: domain.RestreindreAux(ids),
	})
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceJournal, mapID, err)
		return nil, false
	}
	par := make(map[string][]domain.KillDuJournal, len(ids))
	for _, k := range kills {
		par[k.MatchID] = append(par[k.MatchID], k)
	}
	return par, true
}

// poserTueurAbattu cherche si un coequipier de la victime a abattu le tueur de la mort ; rend vrai
// quand le journal identifie ce tueur (la mort entre alors au denominateur du bilan).
func poserTueurAbattu(c *domain.TacticalContribution, journal []domain.KillDuJournal,
	equipes map[string]int) bool {
	mort := tactical.KillDeLaMort(journal, c.XUID, c.InstantMs)
	if mort == nil || !coordination.TueurConnu(*mort) {
		return false
	}
	equipeVictime, connue := equipes[c.XUID]
	coequipier := func(xuid string) bool {
		e, ok := equipes[xuid]
		return connue && ok && e == equipeVictime && xuid != c.XUID
	}
	if r, ok := coordination.TueurAbattu(journal, *mort, coequipier); ok {
		c.Riposte = &r
	}
	return true
}

// poserAbattuEnRetour cherche si l'auteur d'un frag a ete abattu dans la foulee.
func poserAbattuEnRetour(c *domain.TacticalContribution, journal []domain.KillDuJournal) {
	if r, ok := coordination.AbattuEnRetour(journal, c.MatchID, c.XUID, c.InstantMs); ok {
		c.Riposte = &r
	}
}
