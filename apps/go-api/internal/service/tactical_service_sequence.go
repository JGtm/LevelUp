package service

// tactical_service_sequence.go — LA SEQUENCE D'UNE MORT (vue D) : les traces de la victime, de son
// tueur et de ses coequipiers autour de l'instant de la mort, et les huit etapes du cercle de portee.
//
// TROIS SOURCES, ET CE QUE CHACUNE FAIT FOI :
//
//	la BASE      qui sont les acteurs (le tueur du journal, les coequipiers de `match_death_mates`),
//	             le rayon du match, le coup d'envoi (T0 du registre). L'equipe du film n'est jamais lue.
//	l'ARTEFACT   OU ils etaient : les trajectoires, lues a la demande, sans cache (store dedie).
//
// LA CONVERSION D'HORLOGE EST CELLE DU DEPOT, pas une copie : la mort est un instant du MATCH
// (`match_kill_events.time_ms`), son instant dans le film est `matchMs − originMs` (la soustraction
// unique de `matchClock.filmMsOfMatchMs`, cote web), et l'horloge AFFICHEE des etapes est
// `instantAffiche` (tactical_service_cellule_horloge.go), le T0 du registre comme pour les tuiles.
//
// SANS ARTEFACT, SANS `originMs`, SANS RAYON DE RADAR OU SANS POSITION DE LA VICTIME A SA MORT, LA
// SEQUENCE EST INDISPONIBLE (`port.ErrReplayNotAvailable`, 404) SANS AUTRE MESSAGE : le web ne rend
// la vue D que si la requete reussit. Un match que l'appelant n'a pas joue est indiscernable d'un
// match sans rejeu (ADR 0029) : meme erreur, meme statut.

import (
	"context"
	"errors"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// Sequence rend la sequence d'une mort du joueur de la page.
func (s *TacticalService) Sequence(ctx context.Context, req domain.TacticalSequenceRequest) (domain.TacticalSequence, error) {
	var out domain.TacticalSequence
	debut := time.Now()
	if s.repo == nil || s.sequences == nil || !s.caps.Has(games.CapFilmReplayArtifact) {
		s.logger.WarnContext(ctx, "tactique: sequence indisponible (lecteur de pistes ou capability absents)",
			"player", s.xuid, "map_id", req.MapID)
		return out, games.ErrCapabilityNotSupported
	}
	if req.MatchID == "" || req.VictimXUID == "" {
		return out, port.ErrReplayNotAvailable
	}
	ouvrables, err := s.repo.MatchsOuvrables(ctx, s.xuid, []string{req.MatchID})
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: verification d'ouvrabilite en echec (sequence)",
			"player", s.xuid, "match_id", req.MatchID, "err", err)
		return out, err
	}
	if _, ok := ouvrables[req.MatchID]; !ok {
		s.logger.WarnContext(ctx, "tactique: sequence d'un match non ouvrable refusee",
			"player", s.xuid, "match_id", req.MatchID)
		return out, port.ErrReplayNotAvailable
	}
	match, rayon, ok := s.matchDeLaSequence(ctx, req)
	if !ok {
		return out, port.ErrReplayNotAvailable
	}
	pistes, err := s.sequences.Pistes(ctx, req.MatchID)
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: pistes de l'artefact illisibles (sequence)",
			"player", s.xuid, "match_id", req.MatchID, "err", err)
		return out, err
	}
	if pistes == nil || pistes.OriginMs == nil || pistes.FrameIntervalMS <= 0 {
		return out, port.ErrReplayNotAvailable
	}
	entree := s.acteursDeLaSequence(ctx, req, pistes)
	entree.FrameIntervalMs = pistes.FrameIntervalMS
	entree.MortFilmMs = req.TimeMs - *pistes.OriginMs
	entree.MortMatchMs, entree.T0Ms, entree.RayonM = req.TimeMs, match.T0Ms, rayon
	seq, err := tactical.Sequence(entree)
	if errors.Is(err, tactical.ErrSequenceSansMort) {
		s.logger.InfoContext(ctx, "tactique: sequence sans position de la victime a sa mort",
			"player", s.xuid, "match_id", req.MatchID)
		return out, port.ErrReplayNotAvailable
	}
	s.logger.InfoContext(ctx, "tactique: sequence d'une mort",
		"player", s.xuid, "match_id", req.MatchID, "acteurs", len(seq.Acteurs),
		"pistes", len(pistes.Tracks), "duration", time.Since(debut))
	return seq, err
}

// matchDeLaSequence lit l'univers du seul match demande : son coup d'envoi et le rayon
// de son radar. Faux quand le match n'est pas de cette carte ou n'a pas de rayon connu.
func (s *TacticalService) matchDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest,
) (domain.TacticalMatch, float64, bool) {
	univers, err := s.repo.Univers(ctx, domain.TacticalQuery{
		PlayerXUID: s.xuid, MapID: req.MapID, Matchs: domain.RestreindreAux([]string{req.MatchID}),
	})
	if err != nil || len(univers.Matchs) == 0 {
		if err != nil {
			s.logger.ErrorContext(ctx, "tactique: univers de la sequence en echec",
				"player", s.xuid, "match_id", req.MatchID, "err", err)
		}
		return domain.TacticalMatch{}, 0, false
	}
	rayons, _ := s.rayonsParMatch(univers.Matchs)
	rayon, ok := rayons[req.MatchID]
	return univers.Matchs[0], rayon, ok
}

// acteursDeLaSequence compose les acteurs d'une mort depuis la BASE (journal, coequipiers de la
// mort) et leur associe leur piste dans le film.
func (s *TacticalService) acteursDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest,
	pistes *domain.TacticalPistes) tactical.EntreeSequence {
	parXUID := make(map[string][]domain.TacticalPistePoint, len(pistes.Tracks))
	parBot := make(map[string][]domain.TacticalPistePoint)
	for _, p := range pistes.Tracks {
		// Un porteur peut avoir plusieurs vies : leurs points s'ajoutent, dans l'ordre du fichier.
		if p.XUID != "" {
			parXUID[p.XUID] = append(parXUID[p.XUID], p.Points...)
		} else if p.Bot != "" {
			parBot[strings.ToLower(p.Bot)] = append(parBot[strings.ToLower(p.Bot)], p.Points...)
		}
	}
	in := tactical.EntreeSequence{
		Victime: tactical.ActeurSequence{Role: domain.TacticalRoleVictime, XUID: req.VictimXUID,
			Points: parXUID[req.VictimXUID]},
	}
	journal, _ := s.lireLeJournal(ctx, req.MapID, []domain.TacticalContribution{
		{MatchID: req.MatchID, Face: domain.TacticalFaceMort},
	})
	if mort := tactical.KillDeLaMort(journal[req.MatchID], req.VictimXUID, req.TimeMs); mort != nil {
		in.Victime.Gamertag = mort.VictimGamertag
		in.Tueur = tueurDeLaSequence(*mort, parXUID, parBot)
	}
	in.Coequipiers = s.coequipiersDeLaSequence(ctx, req, parXUID)
	return in
}

// tueurDeLaSequence rend le tueur du journal avec sa piste : par xuid, a defaut par le nom du bot.
// nil sans identite.
func tueurDeLaSequence(mort domain.KillDuJournal, parXUID, parBot map[string][]domain.TacticalPistePoint) *tactical.ActeurSequence {
	if mort.KillerXUID == "" && mort.KillerGamertag == "" {
		return nil
	}
	a := &tactical.ActeurSequence{Role: domain.TacticalRoleTueur, XUID: mort.KillerXUID, Gamertag: mort.KillerGamertag}
	if mort.KillerXUID != "" {
		a.Points = parXUID[mort.KillerXUID]
	} else {
		a.Points = parBot[strings.ToLower(mort.KillerGamertag)]
	}
	return a
}

// coequipiersDeLaSequence rend les coequipiers de la mort (lignes de `match_death_mates` de la ligne
// de contexte appariee), chacun avec sa piste. Source absente : aucun coequipier.
func (s *TacticalService) coequipiersDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest,
	parXUID map[string][]domain.TacticalPistePoint) []tactical.ActeurSequence {
	contextes, lus := s.lireLesContextes(ctx, req.MapID, []string{req.MatchID})
	if !lus {
		return nil
	}
	c := tactical.ContexteLePlusProche(contextes, req.MatchID, req.VictimXUID, req.TimeMs)
	if c == nil || c.Total == 0 {
		return nil
	}
	mates := s.lireLesMates(ctx, req.MapID, []*domain.ContexteDeMort{c})
	lignes := mates[cleDuContexte(c)]
	if len(lignes) != c.Total {
		return nil
	}
	out := make([]tactical.ActeurSequence, 0, len(lignes))
	for _, l := range lignes {
		out = append(out, tactical.ActeurSequence{Role: domain.TacticalRoleCoequipier, XUID: l.MateXUID,
			Gamertag: l.Gamertag, Points: parXUID[l.MateXUID]})
	}
	return out
}
