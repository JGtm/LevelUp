// Package service — tactical_service_cellule.go : LE DETAIL D'UNE CELLULE (lien « voir
// dans le rejeu », Tactique S.1, lot M1 du plan d'orchestration 2026-09-07 ; horloge
// exacte, lot M1b du 2026-09-08).
//
// Fichier separe des autres lectures : celle-ci ne rend pas une VALEUR agregee, elle rend
// les CONTRIBUTIONS individuelles d'une cellule deja affichee — {match_id, instant_ms,
// clock, xuid} — pour que le web ouvre le rejeu 2D du bon match au bon instant.
//
// ─── TROIS SOURCES, MEME DISPATCH QUE Raster — ET LEUR HORLOGE (`Clock`) ───────
//
//	morts / kills / gagne   kill_positions_latest x match_kill_events_latest (repo.KillPositions),
//	                        Clock = domain.TacticalClockMatch.
//	isole                   match_death_context_latest (repo.MortsAvecContexte),
//	                        Clock = domain.TacticalClockMatch.
//	temps / routes          sidecars de raster deposes a la cuisson (port.TacticalRasterStore),
//	                        Clock = domain.TacticalClockFilm.
//
// LE SERVICE NE CONVERTIT JAMAIS `InstantMs` LUI-MEME (decision M1b) : il ne lit pas
// l'artefact de rejeu pour ca — un artefact n'est pas toujours cuit — il se contente de
// PUBLIER l'horloge. La conversion en frame exacte (instant + offset si horloge match, puis
// ms -> frame) se fait cote web, quand le document du rejeu est charge.
//
// ─── OWNERSHIP (ADR 0029) ───────────────────────────────────────────────────────
//
// Les trois lectures ci-dessus filtrent DEJA leur univers par `mp.xuid = s.xuid` : leurs
// resultats sont ouvrables PAR CONSTRUCTION. La verification `repo.MatchsOuvrables` porte
// sur le PERIMETRE DEMANDE (`scope.MatchIDs`, le corps de la requete) et sert deux choses :
// la date de tri des contributions, et le compte de ce que le perimetre contenait de non
// ouvrable — un match_id etranger glisse alors dans `MatchsNonOuvrables`, jamais dans
// `Contributions`, meme s'il n'aurait de toute facon produit aucune donnee (defense en
// profondeur : une contribution dont le match n'est pas dans la carte d'ouvrabilite est
// REFUSEE au lieu d'etre triee a une date arbitraire).
package service

import (
	"context"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// Cellule rend le detail d'une cellule de la grille : ses contributions ouvrables et le
// compte de celles ecartees.
func (s *TacticalService) Cellule(ctx context.Context, req domain.TacticalCelluleRequest) (domain.TacticalCelluleReponse, error) {
	var out domain.TacticalCelluleReponse
	scope := domain.TacticalScope{
		MatchIDs:    req.Scope.MatchIDs,
		Coequipiers: compositionNettoyee(req.Scope.Coequipiers),
		Spawn:       req.Scope.Spawn,
	}
	if err := validerLecture(req.MapID, req.Question, req.Qui, scope.Coequipiers); err != nil {
		return out, err
	}
	if s.repo == nil {
		return out, games.ErrCapabilityNotSupported
	}
	req.Scope = scope

	var (
		contributions []domain.TacticalContribution
		err           error
	)
	switch {
	case lectureDArtefact(req.Question):
		contributions, err = s.celluleArtefact(ctx, req, scope)
	case req.Question == domain.TacticalQuestionIsole:
		contributions, err = s.celluleIsole(ctx, req, scope)
	default:
		contributions, err = s.celluleDeKills(ctx, req, scope)
	}
	if err != nil {
		return out, err
	}

	ouvrables, err := s.repo.MatchsOuvrables(ctx, s.xuid, scope.MatchIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: verification d'ouvrabilite en echec",
			"player", s.xuid, "map_id", req.MapID, "err", err)
		return out, err
	}
	out.MatchsNonOuvrables = matchsNonOuvrables(scope.MatchIDs, ouvrables)

	// LE FILTRE D'OUVRABILITE EST LE GARDE-FOU CANONIQUE, PAS DE LA DECORATION : les trois
	// sources ci-dessus sont scopees par construction (mp.xuid = s.xuid), mais une
	// contribution dont le match n'apparaitrait PAS dans `ouvrables` — perimetre incomplet,
	// bug d'un futur appelant — est REFUSEE ici plutot que triee a une date zero qui la
	// glisserait n'importe ou dans la liste.
	filtrees := make([]domain.TacticalContribution, 0, len(contributions))
	for _, c := range contributions {
		debut, ok := ouvrables[c.MatchID]
		if !ok {
			s.logger.WarnContext(ctx, "tactique: contribution d'un match non ouvrable ecartee",
				"player", s.xuid, "map_id", req.MapID, "match_id", c.MatchID)
			continue
		}
		c.MatchStartedAt = debut
		filtrees = append(filtrees, c)
	}
	sort.SliceStable(filtrees, func(i, j int) bool {
		if !filtrees[i].MatchStartedAt.Equal(filtrees[j].MatchStartedAt) {
			return filtrees[i].MatchStartedAt.After(filtrees[j].MatchStartedAt)
		}
		return filtrees[i].InstantMs < filtrees[j].InstantMs
	})
	out.Contributions = filtrees

	s.logger.InfoContext(ctx, "tactique: detail de cellule",
		"player", s.xuid, "map_id", req.MapID, "question", req.Question, "qui", req.Qui,
		"col", req.Col, "lig", req.Lig, "contributions", len(out.Contributions),
		"matchs_non_ouvrables", out.MatchsNonOuvrables)
	return out, nil
}

// matchsNonOuvrables compte les match_id DISTINCTS de `matchIDs` absents de `ouvrables`.
func matchsNonOuvrables(matchIDs []string, ouvrables map[string]time.Time) int {
	vus := make(map[string]bool, len(matchIDs))
	n := 0
	for _, id := range matchIDs {
		if vus[id] {
			continue
		}
		vus[id] = true
		if _, ok := ouvrables[id]; !ok {
			n++
		}
	}
	return n
}

// celluleDeKills sert les trois questions qui se lisent sur les POSITIONS MESUREES :
// ou je meurs, ou je tue, ou je gagne. Meme source que rasterDeKills, projetee sur UNE
// cellule au lieu d'etre sommee sur toute la grille.
func (s *TacticalService) celluleDeKills(ctx context.Context, req domain.TacticalCelluleRequest,
	scope domain.TacticalScope) ([]domain.TacticalContribution, error) {
	if !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID, "question", req.Question)
		return nil, games.ErrCapabilityNotSupported
	}
	lecture, err := s.repo.KillPositions(ctx, requeteDuScope(s.xuid, req.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture des positions en echec (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID, "question", req.Question, "err", err)
		return nil, err
	}
	if len(lecture.Univers.Matchs) == 0 {
		return nil, domain.ErrTacticalCarteInconnue
	}
	dans := cible(lecture.Univers.Equipes, req.Qui, s.xuid, scope.Coequipiers)
	prendVictime, prendTueur := facesDeLaQuestion(req.Question)
	grille := tactical.GrilleParDefaut()

	out := make([]domain.TacticalContribution, 0, 4)
	for _, p := range lecture.Points {
		if prendVictime && dans(p.MatchID, p.VictimXUID) {
			if celluleCorrespond(grille, p.VictimX, p.VictimY, req.Col, req.Lig) {
				out = append(out, domain.TacticalContribution{
					MatchID: p.MatchID, InstantMs: p.TimeMs, XUID: p.VictimXUID,
					Clock: domain.TacticalClockMatch,
				})
			}
		}
		if prendTueur && dans(p.MatchID, p.KillerXUID) {
			if celluleCorrespond(grille, p.KillerX, p.KillerY, req.Col, req.Lig) {
				out = append(out, domain.TacticalContribution{
					MatchID: p.MatchID, InstantMs: p.TimeMs, XUID: p.KillerXUID,
					Clock: domain.TacticalClockMatch,
				})
			}
		}
	}
	return out, nil
}

// celluleIsole sert « ou je meurs isole ». MEME REGLE QUE coordination.Isolement
// (rasterIsole, tactical_service_isolement.go) : une mort est isolee quand personne ne
// pouvait accompagner OU quand personne accompagnant n'etait a portee du rayon radar du
// match. Reecrite ici (et non appelee via coordination.Isolement) parce que ce paquet-la
// ne rend que X/Y/MatchID (domain.MortAExaminer) — il n'a pas besoin du xuid ni de
// l'instant, quand ce detail de cellule a besoin des deux pour construire un lien de rejeu.
func (s *TacticalService) celluleIsole(ctx context.Context, req domain.TacticalCelluleRequest,
	scope domain.TacticalScope) ([]domain.TacticalContribution, error) {
	if !positionsDeKillLisibles(s.caps) {
		s.logger.WarnContext(ctx, "tactique: aucune position de kill lisible pour ce titre (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID)
		return nil, games.ErrCapabilityNotSupported
	}
	lecture, err := s.repo.MortsAvecContexte(ctx, requeteDuScope(s.xuid, req.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: lecture d'isolement en echec (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID, "err", err)
		return nil, err
	}
	if len(lecture.Univers.Matchs) == 0 {
		return nil, domain.ErrTacticalCarteInconnue
	}
	rayons, _ := s.rayonsParMatch(lecture.Univers.Matchs)
	dans := cible(lecture.Univers.Equipes, req.Qui, s.xuid, scope.Coequipiers)
	grille := tactical.GrilleParDefaut()

	out := make([]domain.TacticalContribution, 0, 2)
	for _, m := range lecture.Morts {
		if !dans(m.MatchID, m.VictimXUID) {
			continue
		}
		rayon, ok := rayons[m.MatchID]
		if !ok {
			// Match sans portee de radar mesuree : hors univers de cette lecture (correction
			// G2), meme regle que rasterIsole.
			continue
		}
		if m.Visibles+m.HorsDeVue == 0 {
			// Equipe a terre : personne ne pouvait accompagner, la mort ne dit rien du
			// placement (meme exclusion que coordination.Isolement).
			continue
		}
		accompagnee := m.PlusProcheM != nil && *m.PlusProcheM <= rayon
		if accompagnee {
			continue
		}
		if !celluleCorrespond(grille, m.X, m.Y, req.Col, req.Lig) {
			continue
		}
		out = append(out, domain.TacticalContribution{
			MatchID: m.MatchID, InstantMs: m.TimeMs, XUID: m.VictimXUID,
			Clock: domain.TacticalClockMatch,
		})
	}
	return out, nil
}

// celluleArtefact sert « ou je passe mon temps » et « par ou je sors du spawn », sur les
// sidecars de raster deposes a la cuisson. LE VERDICT DU LOT M1 (item 1 du brief) :
//
//	temps    l'instant contributeur est la frame de PREMIERE ENTREE dans la cellule
//	         (TacticalRasterEntree.Frame, sidecar.PremieresEntrees) ;
//	routes   l'instant contributeur est le DEBUT DE VIE de la route qui traverse la
//	         cellule (TacticalRasterRoute.DebutFrame) — la MEME route peut contribuer a
//	         plusieurs cellules avec le MEME instant, c'est le depart qui compte, pas le
//	         passage.
//
// Les deux sont en FRAMES, converties en millisecondes par `sidecar.FrameIntervalMs` — un
// champ TOUJOURS present au schema 6 (ecrit a la cuisson) : aucun cas de « temps sans
// instant » n'a ete rencontre sur le corpus courant (verdict consigne au journal du lot).
func (s *TacticalService) celluleArtefact(ctx context.Context, req domain.TacticalCelluleRequest,
	scope domain.TacticalScope) ([]domain.TacticalContribution, error) {
	if !s.caps.Has(games.CapFilmReplayArtifact) {
		s.logger.WarnContext(ctx, "tactique: lecture d'artefact indisponible (detail de cellule)",
			"player", s.xuid, "map_id", req.MapID, "question", req.Question)
		return nil, games.ErrCapabilityNotSupported
	}
	if s.rasters == nil {
		s.logger.ErrorContext(ctx, "tactique: detail de cellule d'artefact demande sans lecteur de sidecars cable",
			"player", s.xuid, "map_id", req.MapID, "question", req.Question)
		return nil, games.ErrCapabilityNotSupported
	}
	univers, err := s.repo.Univers(ctx, s.requeteAvecRetention(req.MapID, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers du detail de cellule en echec",
			"player", s.xuid, "map_id", req.MapID, "err", err)
		return nil, err
	}
	if len(univers.Matchs) == 0 {
		return nil, domain.ErrTacticalCarteInconnue
	}
	dans := cible(univers.Equipes, req.Qui, s.xuid, scope.Coequipiers)

	out := make([]domain.TacticalContribution, 0, 2)
	for _, m := range univers.Matchs {
		sc, err := s.rasters.Charger(ctx, m.MatchID)
		if err != nil {
			s.logger.ErrorContext(ctx, "tactique: sidecar de raster illisible (detail de cellule)",
				"player", s.xuid, "map_id", req.MapID, "match_id", m.MatchID, "err", err)
			continue
		}
		if !s.sidecarExploitable(ctx, sc, m.MatchID) {
			continue
		}
		out = append(out, contributionsDuSidecar(sc, m.MatchID, req.Question, req.Col, req.Lig, dans)...)
	}
	return out, nil
}

// contributionsDuSidecar rend les contributions d'UN sidecar pour UNE cellule.
func contributionsDuSidecar(sc *domain.TacticalRasterSidecar, matchID, question string,
	col, lig int, dans predicatQui) []domain.TacticalContribution {
	out := make([]domain.TacticalContribution, 0, 2)
	for _, j := range sc.Joueurs {
		if !dans(matchID, j.XUID) {
			continue
		}
		switch question {
		case domain.TacticalQuestionRoutes:
			for _, route := range j.Routes {
				if !routeTraverseCellule(route, col, lig) {
					continue
				}
				out = append(out, domain.TacticalContribution{
					MatchID:   matchID,
					InstantMs: int64(route.DebutFrame) * int64(sc.FrameIntervalMs),
					XUID:      j.XUID,
					Clock:     domain.TacticalClockFilm,
				})
			}
		default: // domain.TacticalQuestionTemps
			for _, e := range j.PremieresEntrees {
				if e.Col != col || e.Lig != lig {
					continue
				}
				out = append(out, domain.TacticalContribution{
					MatchID:   matchID,
					InstantMs: int64(e.Frame) * int64(sc.FrameIntervalMs),
					XUID:      j.XUID,
					Clock:     domain.TacticalClockFilm,
				})
			}
		}
	}
	return out
}

// routeTraverseCellule dit si une route de spawn passe par (col, lig).
func routeTraverseCellule(r domain.TacticalRasterRoute, col, lig int) bool {
	for _, c := range r.Cases {
		if c.Col == col && c.Lig == lig {
			return true
		}
	}
	return false
}

// celluleCorrespond projette (x, y) sur la grille par defaut et dit si elle tombe dans la
// cellule (col, lig) demandee. Un point non fini (NaN/Inf) ne correspond a aucune cellule —
// meme regle que le rasterisage agrege (tactical.Grille.Cellule).
func celluleCorrespond(g tactical.Grille, x, y float64, col, lig int) bool {
	c, ok := g.Cellule(x, y)
	return ok && c.Col == col && c.Lig == lig
}
