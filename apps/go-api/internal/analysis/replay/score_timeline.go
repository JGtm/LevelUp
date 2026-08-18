package replay

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// score_timeline.go — L'ASSEMBLAGE DE LA COURBE DE SCORE, PUR.
//
// Aucune I/O, aucune base : ce fichier prend des enregistrements d'entite DEJA decodes et les
// pose sur la grille de frames du document. Ce qui vient de la base — les lignes de match
// (pont d'identite) et les scores du registre (identite des camps) — est fourni par
// l'appelant, exactement comme Options.Objectives. Le contrat de forme et les limites du
// calque sont dans document_score.go.

// ScoreInput est ce que l'appelant fournit pour construire la courbe de score.
//
// TOUT Y EST UNE ENTREE, y compris ce qui vient de la base : `analysis/` n'ouvre aucune
// connexion, et `replaybuild` non plus (cf. son en-tete de paquet). Le decodage du film, lui,
// est fait UNE fois par l'appelant et ses enregistrements servent aux trois calques qui en
// dependent (score, identite des slots, actions d'objectif).
type ScoreInput struct {
	// Records sont les enregistrements d'entite du film (objectiveevents.StatRecordsCtx).
	Records []objectiveevents.StatRecord
	// Lines sont les lignes de match des joueurs (`match_participants`) : le triplet
	// (frags, morts, assistances) est la CLE d'appariement du slot d'entite au xuid.
	// Absentes, aucun joueur n'est publie — jamais un slot attribue au hasard.
	Lines []objectiveevents.PlayerLine
	// TeamByXUID donne le camp de chaque joueur (`match_participants.team_id`). Sert a la
	// resolution d'identite (b) : la somme des frags d'un camp contre celle du slot d'equipe.
	TeamByXUID map[string]int
	// TeamScores porte `team_0_score` / `team_1_score` du registre. Nil = absents : la
	// resolution (a) ne s'applique pas, on retombe sur (b) puis (c).
	TeamScores *[2]int
	// Truncated propage le plafond de lecture des enregistrements : les courbes s'arretent
	// alors avant la fin du match, et la couverture le dit.
	Truncated bool
}

// scoreClock convertit un instant du film en frame du document.
type scoreClock struct {
	// intervalMS est le pas de la grille de frames.
	intervalMS int
	// frames borne l'axe : au-dela, l'emission est hors fenetre.
	frames int
	// originMS est l'instant de la frame 0 sur l'horloge du FILM. Les enregistrements sont
	// dates depuis le premier paquet du film, la grille compte depuis le premier paquet de
	// POSITION : sans cette soustraction, toute la courbe glisse de 3,6 s a 50,8 s.
	originMS int
}

// frameOf pose un instant du film sur la grille, ou dit qu'il n'y tient pas.
func (c scoreClock) frameOf(timeMS int) (int, bool) {
	if c.intervalMS <= 0 || c.frames <= 0 || timeMS < c.originMS {
		return 0, false
	}
	t := (timeMS - c.originMS) / c.intervalMS
	if t >= c.frames {
		return 0, false
	}
	return t, true
}

// scoreSeriesSet porte les deux formes d'un meme emplacement, deja decodees pour tous les
// slots d'une famille (equipes ou joueurs) : la courbe par manche et la courbe cumulee.
type scoreSeriesSet struct {
	byRound map[int]map[int][]objectiveevents.ScorePoint
	total   map[int][]objectiveevents.ScorePoint
}

// loadScoreSeries decode les deux formes d'un emplacement.
func loadScoreSeries(recs []objectiveevents.StatRecord, comp objectiveevents.StatComponent, teams bool) scoreSeriesSet {
	return scoreSeriesSet{
		byRound: objectiveevents.SeriesByRound(recs, comp, teams),
		total:   objectiveevents.SeriesTotal(recs, comp, teams),
	}
}

// at rend la serie publiable d'un slot.
func (s scoreSeriesSet) at(slot int, c scoreClock) ScoreSeries {
	return ScoreSeries{Rounds: scoreRoundsOf(s.byRound[slot], c), Total: scoreTicksOf(s.total[slot], c)}
}

// final rend la derniere valeur cumulee d'un slot (le total du match), et si elle existe.
func (s scoreSeriesSet) final(slot int) (int64, bool) {
	pts := s.total[slot]
	if len(pts) == 0 {
		return 0, false
	}
	return pts[len(pts)-1].Value, true
}

// scoreTicksOf pose une suite d'emissions sur la grille de frames, une valeur par frame.
//
// LA DERNIERE VALEUR D'UNE FRAME GAGNE : plusieurs emissions peuvent tomber dans le meme pas
// de 100 ms, et ce que le client dessine a cette frame est l'etat a sa fin. Les emissions
// hors fenetre sont ecartees sans bruit — elles sont le cas nominal (le film continue apres
// la derniere position publiee, et commence avant la premiere).
func scoreTicksOf(pts []objectiveevents.ScorePoint, c scoreClock) []ScoreTick {
	out := make([]ScoreTick, 0, len(pts))
	for _, p := range pts {
		t, ok := c.frameOf(p.TimeMS)
		if !ok {
			continue
		}
		if n := len(out); n > 0 && out[n-1].T == t {
			out[n-1].V = int(p.Value)
			continue
		}
		out = append(out, ScoreTick{T: t, V: int(p.Value)})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// scoreRoundsOf pose les manches d'un slot sur la grille, dans l'ordre des manches.
func scoreRoundsOf(byRound map[int][]objectiveevents.ScorePoint, c scoreClock) []ScoreRound {
	rounds := make([]int, 0, len(byRound))
	for r := range byRound {
		rounds = append(rounds, r)
	}
	sort.Ints(rounds)
	out := make([]ScoreRound, 0, len(rounds))
	for _, r := range rounds {
		ticks := scoreTicksOf(byRound[r], c)
		if len(ticks) == 0 {
			continue
		}
		out = append(out, ScoreRound{Round: r, Points: ticks})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildScoreTimeline assemble le calque du score et sa couverture.
//
// Rend (nil, nil) quand l'appelant n'a rien fourni : un artefact construit sans acces aux
// enregistrements du film ne porte AUCUNE couverture de score, ce qui le distingue d'un film
// dont la lecture n'a rien donne (couverture presente, courbes vides).
func buildScoreTimeline(in *ScoreInput, c scoreClock) (*ScoreTimeline, *ScoreCoverage) {
	if in == nil {
		return nil, nil
	}
	recs := in.Records
	teamScore := loadScoreSeries(recs, objectiveevents.ModeScoreComponent, true)
	teamFrags := loadScoreSeries(recs, objectiveevents.KillsComponent, true)
	playerFrags := loadScoreSeries(recs, objectiveevents.KillsComponent, false)
	identity := objectiveevents.SlotIdentityFrom(recs, in.Lines)

	slots := teamSlotsOf(teamScore, teamFrags)
	teamID, method := resolveTeamIdentity(in, slots, teamScore, teamFrags, playerFrags, identity)

	tl := &ScoreTimeline{
		Teams:   buildTeamScores(slots, teamScore, teamID, c),
		Players: buildPlayerScores(recs, identity, c),
	}
	cov := &ScoreCoverage{
		TeamIdentity:  method,
		Rounds:        len(objectiveevents.RealRounds(recs)),
		ModeSupported: len(teamScore.total) > 0,
		Truncated:     in.Truncated,
		Oracle:        ScoreOracleDisplayed,
		Points:        countScorePoints(tl),
	}
	if len(tl.Teams) == 0 && len(tl.Players) == 0 {
		tl = nil
	}
	return tl, cov
}

// teamSlotsOf rend les slots d'entite d'equipe vus par le film, dans l'ordre.
func teamSlotsOf(score, frags scoreSeriesSet) []int {
	seen := map[int]bool{}
	for _, m := range []map[int][]objectiveevents.ScorePoint{score.total, frags.total} {
		for slot := range m {
			seen[slot] = true
		}
	}
	out := make([]int, 0, len(seen))
	for slot := range seen {
		out = append(out, slot)
	}
	sort.Ints(out)
	return out
}

// buildTeamScores rend une courbe par slot d'equipe, avec son camp quand il est resolu.
func buildTeamScores(slots []int, score scoreSeriesSet, teamID map[int]int, c scoreClock) []TeamScore {
	out := make([]TeamScore, 0, len(slots))
	for _, slot := range slots {
		s := score.at(slot, c)
		if len(s.Rounds) == 0 && len(s.Total) == 0 {
			continue
		}
		team := TeamScore{Rounds: s.Rounds, Total: s.Total}
		if id, ok := teamID[slot]; ok {
			team.TeamID = &id
		}
		out = append(out, team)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildPlayerScores rend les compteurs vivants des joueurs APPARIES, tries par xuid.
//
// Un slot que le triplet n'a pas apparie sans ambiguite n'apparait pas : c'est la meme regle
// que pour les actions d'objectif, et pour la meme raison — attribuer les compteurs d'un
// joueur a un autre serait indetectable a l'ecran.
func buildPlayerScores(recs []objectiveevents.StatRecord, identity map[int]string, c scoreClock) []PlayerScore {
	if len(identity) == 0 {
		return nil
	}
	personal := loadScoreSeries(recs, objectiveevents.PersonalScoreComponent, false)
	kills := loadScoreSeries(recs, objectiveevents.KillsComponent, false)
	deaths := loadScoreSeries(recs, objectiveevents.DeathsComponent, false)
	assists := loadScoreSeries(recs, objectiveevents.AssistsComponent, false)

	slots := make([]int, 0, len(identity))
	for slot := range identity {
		slots = append(slots, slot)
	}
	sort.Ints(slots)

	out := make([]PlayerScore, 0, len(slots))
	for _, slot := range slots {
		p := PlayerScore{
			XUID:    identity[slot],
			Score:   personal.at(slot, c),
			Kills:   kills.at(slot, c),
			Deaths:  deaths.at(slot, c),
			Assists: assists.at(slot, c),
		}
		if p.empty() {
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	if len(out) == 0 {
		return nil
	}
	return out
}

// empty dit qu'aucun des quatre compteurs d'un joueur n'a rien a publier.
func (p PlayerScore) empty() bool {
	for _, s := range []ScoreSeries{p.Score, p.Kills, p.Deaths, p.Assists} {
		if len(s.Rounds) > 0 || len(s.Total) > 0 {
			return false
		}
	}
	return true
}

// countScorePoints compte les points publies, equipes et joueurs confondus.
func countScorePoints(tl *ScoreTimeline) int {
	n := 0
	for _, t := range tl.Teams {
		n += countSeriesPoints(ScoreSeries{Rounds: t.Rounds, Total: t.Total})
	}
	for _, p := range tl.Players {
		for _, s := range []ScoreSeries{p.Score, p.Kills, p.Deaths, p.Assists} {
			n += countSeriesPoints(s)
		}
	}
	return n
}

// countSeriesPoints compte les points d'une serie, manches et total compris.
func countSeriesPoints(s ScoreSeries) int {
	n := len(s.Total)
	for _, r := range s.Rounds {
		n += len(r.Points)
	}
	return n
}

// logScoreCoverage journalise ce que le calque a publie — et ce qu'il n'a pas resolu.
func logScoreCoverage(matchID string, cov *ScoreCoverage, tl *ScoreTimeline) {
	if cov == nil {
		return
	}
	teams, players := 0, 0
	if tl != nil {
		teams, players = len(tl.Teams), len(tl.Players)
	}
	if cov.TeamIdentity == ScoreIdentityUnresolved {
		slog.Warn("rejeu : identite des camps NON RESOLUE — courbes de score publiees sans equipe",
			"match_id", matchID, "equipes", teams, "joueurs", players, "manches", cov.Rounds)
	}
	if cov.Truncated {
		slog.Warn("rejeu : lecture des enregistrements TRONQUEE — courbes de score incompletes",
			"match_id", matchID, "points", cov.Points)
	}
	slog.Info("rejeu : courbe de score",
		"match_id", matchID, "identiteEquipes", cov.TeamIdentity, "manches", cov.Rounds,
		"modePorte", cov.ModeSupported, "equipes", teams, "joueurs", players, "points", cov.Points)
}
