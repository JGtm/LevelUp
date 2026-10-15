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
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
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

// viesDuFilm : les pistes de l'artefact rangees par porteur (xuid, ou nom de bot en minuscules), UNE
// ENTREE PAR VIE et triees par premier instant. Les vies d'un porteur ne se concatenent JAMAIS : entre
// deux vies il n'a pas de position (cf. `tactical.positionA`).
type viesDuFilm struct {
	parXUID map[string][][]domain.TacticalPistePoint
	parBot  map[string][][]domain.TacticalPistePoint
}

func viesDesPistes(pistes *domain.TacticalPistes) viesDuFilm {
	v := viesDuFilm{
		parXUID: make(map[string][][]domain.TacticalPistePoint, len(pistes.Tracks)),
		parBot:  make(map[string][][]domain.TacticalPistePoint),
	}
	for _, p := range pistes.Tracks {
		if len(p.Points) == 0 {
			continue
		}
		if p.XUID != "" {
			v.parXUID[p.XUID] = append(v.parXUID[p.XUID], p.Points)
		} else if p.Bot != "" {
			cle := strings.ToLower(p.Bot)
			v.parBot[cle] = append(v.parBot[cle], p.Points)
		}
	}
	for _, vies := range v.parXUID {
		trierLesVies(vies)
	}
	for _, vies := range v.parBot {
		trierLesVies(vies)
	}
	return v
}

// trierLesVies range les vies par premier instant : l'ordre du fichier suit les sieges, pas le temps.
func trierLesVies(vies [][]domain.TacticalPistePoint) {
	sort.SliceStable(vies, func(i, j int) bool { return vies[i][0].T < vies[j][0].T })
}

// acteursDeLaSequence compose les acteurs d'une mort depuis la BASE (journal, coequipiers de la
// mort) et leur associe leurs vies dans le film.
func (s *TacticalService) acteursDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest,
	pistes *domain.TacticalPistes) tactical.EntreeSequence {
	vies := viesDesPistes(pistes)
	in := tactical.EntreeSequence{
		Victime: tactical.ActeurSequence{Role: domain.TacticalRoleVictime, XUID: req.VictimXUID,
			Vies: vies.parXUID[req.VictimXUID]},
	}
	journal := s.lireLeJournal(ctx, req.MapID, req.MatchID)
	if mort := tactical.KillDeLaMort(journal, req.VictimXUID, req.TimeMs); mort != nil {
		in.Victime.Gamertag = mort.VictimGamertag
		in.Tueur = tueurDeLaSequence(*mort, vies)
	}
	in.Coequipiers = s.coequipiersDeLaSequence(ctx, req, vies)
	return in
}

// tueurDeLaSequence rend le tueur du journal avec ses vies : par xuid, a defaut par le nom du bot.
// nil sans identite.
func tueurDeLaSequence(mort domain.KillDuJournal, vies viesDuFilm) *tactical.ActeurSequence {
	if mort.KillerXUID == "" && mort.KillerGamertag == "" {
		return nil
	}
	a := &tactical.ActeurSequence{Role: domain.TacticalRoleTueur, XUID: mort.KillerXUID, Gamertag: mort.KillerGamertag}
	if mort.KillerXUID != "" {
		a.Vies = vies.parXUID[mort.KillerXUID]
	} else {
		a.Vies = vies.parBot[strings.ToLower(mort.KillerGamertag)]
	}
	return a
}

// coequipiersDeLaSequence rend les coequipiers de la mort (lignes de `match_death_mates` de la ligne
// de contexte appariee), chacun avec ses vies. Source absente ou lignes incoherentes : aucun coequipier.
func (s *TacticalService) coequipiersDeLaSequence(ctx context.Context, req domain.TacticalSequenceRequest,
	vies viesDuFilm) []tactical.ActeurSequence {
	contextes, lus := s.lireLesContextes(ctx, req.MapID, []string{req.MatchID})
	if !lus {
		return nil
	}
	c := tactical.ContexteLePlusProche(contextes, req.MatchID, req.VictimXUID, req.TimeMs)
	if c == nil || c.Total == 0 {
		return nil
	}
	mates := s.lireLesMates(ctx, req.MapID, []*domain.ContexteDeMort{c})
	lignes, ok := s.lignesCompletes(ctx, req.MapID, c, mates)
	if !ok {
		return nil
	}
	out := make([]tactical.ActeurSequence, 0, len(lignes))
	for _, l := range lignes {
		out = append(out, tactical.ActeurSequence{Role: domain.TacticalRoleCoequipier, XUID: l.MateXUID,
			Gamertag: l.Gamertag, Vies: vies.parXUID[l.MateXUID]})
	}
	return out
}

// sourceJournal : le nom de la source du journal, tel que le journal applicatif la cite.
const sourceJournal = "journal"

// lireLeJournal lit les kills publiables d'UN match (liste blanche d'un match) : la vue de sequence y
// retrouve le kill de la mort. Source best-effort : en echec, elle est journalisee et la sequence
// se compose sans tueur.
func (s *TacticalService) lireLeJournal(ctx context.Context, mapID, matchID string) []domain.KillDuJournal {
	stop := timing.FromContext(ctx).Section("tactical_sequence_journal")
	kills, err := s.repo.JournalDesKills(ctx, domain.TacticalQuery{
		PlayerXUID: s.xuid, MapID: mapID, Matchs: domain.RestreindreAux([]string{matchID}),
	})
	stop()
	if err != nil {
		s.sourceDegradee(ctx, sourceJournal, mapID, err)
		return nil
	}
	return kills
}
