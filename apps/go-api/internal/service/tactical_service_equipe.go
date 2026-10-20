package service

// tactical_service_equipe.go — CE QUE LE DETAIL D'UNE ZONE AJOUTE SUR LES COEQUIPIERS : l'equipe de la
// victime a l'instant de chaque mort, et le bilan de la zone.
//
// UNE SOURCE, BEST-EFFORT COMME LES AUTRES du detail (tactical_service_cellule_enrichir.go) :
// `match_death_mates_latest`, une ligne par coequipier et par mort. Absente ou en echec, elle retire
// les champs qu'elle nourrit et le journal le dit en la nommant ; la liste des contributions est
// toujours servie. Elle a sa section de duree (ADR 0036 I6) et est lue UNE FOIS par requete, bornee
// aux matchs des contributions (ADR 0036 I2).
// I6) et est lue UNE FOIS par requete, bornee aux matchs des contributions (ADR 0036 I2).
//
// L'EQUIPE SE LIT SUR LA CLE EXACTE DE LA LIGNE DE CONTEXTE APPARIEE a la mort (a ± 1,5 s,
// `tactical.ContexteLePlusProche`), jamais sur l'instant de la contribution : les deux tables sont
// ecrites dans la meme passe sous la meme cle. Une mort dont les lignes ne retombent pas sur
// `teammates_total` (match pas encore relu) n'a pas d'equipe, jamais une equipe partielle.

import (
	"context"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"
)

// Nom de la source, tel que le journal la cite.
const (
	sourceCoequipiers = "coequipiers"
)

// poserEquipe pose l'equipe de chaque contribution de face mort, puis rend le bilan de la
// zone. `contextes` sont ceux des matchs de ses morts (`lus` faux = source absente).
func (s *TacticalService) poserEquipe(ctx context.Context, mapID string, univers domain.TacticalUnivers,
	retenues []contributionLue,
	out []domain.TacticalContribution, contextes []domain.ContexteDeMort, lus bool) *domain.TacticalBilanZone {
	if len(matchsDesFaces(out)) == 0 {
		return nil
	}
	rayons, _ := s.rayonsParMatch(univers.Matchs)
	apparies := apparierLesContextes(retenues, out, contextes, lus)
	mates := s.lireLesMates(ctx, mapID, apparies)
	for i := range out {
		if out[i].Face != domain.TacticalFaceMort {
			continue
		}
		rayon, aUnRayon := rayons[out[i].MatchID]
		if c := apparies[i]; c != nil {
			if lignes, ok := s.lignesCompletes(ctx, mapID, c, mates); ok {
				eq := tactical.EquipeDeLaMort(*c, lignes, rayon, aUnRayon, out[i].AutrePoint)
				out[i].Equipe = &eq
			}
		}
	}
	return bilanDeZone(out)
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
	// Une ligne par coequipier et par mort : un doublon de cle du journal (deux morts de meme victime au
	// meme instant) a ecrit les coequipiers deux fois, la lecture les dedoublonne par `mate_xuid`.
	par := make(map[domain.CleDeMort][]domain.MateDeMort, len(cles))
	vus := make(map[domain.CleDeMort]map[string]bool, len(cles))
	for _, l := range lignes {
		k := domain.CleDeMort{MatchID: l.MatchID, VictimXUID: l.VictimXUID, TimeMs: l.TimeMs}
		if vus[k] == nil {
			vus[k] = map[string]bool{}
		}
		if vus[k][l.MateXUID] {
			continue
		}
		vus[k][l.MateXUID] = true
		par[k] = append(par[k], l)
	}
	return par
}

// metricEquipeLignesIncoherentes compte (expvar, ADR 0009) les morts dont les lignes de coequipiers ne
// retombent pas sur `teammates_total` alors qu'elles existent : un producteur ou une relecture a laisse
// un etat incoherent. Le cas « aucune ligne » (match pas encore relu) n'est pas une incoherence.
const metricEquipeLignesIncoherentes = "tactique_equipe_lignes_incoherentes"

// lignesCompletes rend les coequipiers d'une mort quand leurs lignes retombent sur le total de son
// contexte. Sinon faux, ET JAMAIS EN SILENCE : « aucune ligne » = match pas encore relu (DEBUG) ; un
// autre ecart (plus ou moins de lignes que de coequipiers) est journalise et compte.
func (s *TacticalService) lignesCompletes(ctx context.Context, mapID string, c *domain.ContexteDeMort,
	mates map[domain.CleDeMort][]domain.MateDeMort) ([]domain.MateDeMort, bool) {
	lignes := mates[cleDuContexte(c)]
	if len(lignes) == c.Total {
		return lignes, true
	}
	if len(lignes) == 0 {
		s.logger.DebugContext(ctx, "tactique: coequipiers de la mort pas encore lus (match a relire)",
			"player", s.xuid, "map_id", mapID, "match_id", c.MatchID, "total", c.Total)
		return nil, false
	}
	observability.AddInt(metricEquipeLignesIncoherentes, 1)
	s.logger.DebugContext(ctx, "tactique: lignes de coequipiers incoherentes avec le contexte, equipe ecartee",
		"player", s.xuid, "map_id", mapID, "match_id", c.MatchID, "time_ms", c.TimeMs, "lignes", len(lignes), "total", c.Total)
	return nil, false
}
