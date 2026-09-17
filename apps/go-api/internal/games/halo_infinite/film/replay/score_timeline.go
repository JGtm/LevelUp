package replay

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
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
	// Records sont les enregistrements d'entite du film (objectives.StatRecordsCtx).
	Records []types.StatRecord
	// Lines sont les lignes de match des joueurs (`match_participants`) : le triplet
	// (frags, morts, assistances) est la CLE d'appariement du slot d'entite au xuid.
	// Absentes, aucun joueur n'est publie — jamais un slot attribue au hasard.
	Lines []types.PlayerLine
	// TeamByXUID donne le camp de chaque joueur (`match_participants.team_id`). Sert a la
	// resolution d'identite (b) : la somme des frags d'un camp contre celle du slot d'equipe.
	TeamByXUID map[string]int
	// TeamScores porte `team_0_score` / `team_1_score` du registre. Nil = absents : la
	// resolution (a) ne s'applique pas, on retombe sur (b) puis (c).
	TeamScores *[2]int
	// TargetScore est la cible de victoire de la variante (regulation.toml [score_target]).
	// 0 = inconnue : rien n'est publie, le client retombe sur son repli.
	TargetScore int
	// HoldTicksPerPoint est le nombre de TICS de garde qui valent un point sur la variante
	// (regulation.toml [hold_ticks_per_point]). 0 = inconnu : ni serie ni denominateur ne sont
	// publies, et le client n'affiche aucune progression de garde.
	HoldTicksPerPoint int
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
	byRound map[int]map[int][]types.ScorePoint
	total   map[int][]types.ScorePoint
}

// loadScoreSeries decode les deux formes d'un emplacement.
func loadScoreSeries(recs []types.StatRecord, comp objectives.StatComponent, teams bool) scoreSeriesSet {
	return scoreSeriesSet{
		byRound: objectives.SeriesByRound(recs, comp, teams),
		total:   objectives.SeriesTotal(recs, comp, teams),
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

// scoreTicksOf pose une suite d'emissions sur la grille de frames, AUX CHANGEMENTS SEULEMENT.
//
// DEUX FILTRES, ET LE PREMIER MANQUAIT (correctif de revue R1, 2026-08-18) :
//
//	PAR VALEUR   une emission qui REPETE la valeur precedente n'est pas un changement, et le
//	             contrat du champ dit « aux CHANGEMENTS seulement ». Le cas n'est pas marginal :
//	             un composant porte DEUX valeurs (le composant 2 porte les frags en A et les
//	             morts en B) et il est reemis des que l'UNE des deux bouge — chaque frag creait
//	             donc aussi un point de morts a valeur inchangee. Mesure sur les 5 temoins :
//	             44,7 % des points `kills` et 46,3 % des points `deaths` etaient des repetitions
//	             (23 points pour 11 frags chez un joueur de `530820e5`). On garde la PREMIERE
//	             emission d'un palier : c'est l'instant ou la valeur a ete atteinte.
//	PAR FRAME    la derniere valeur d'une frame gagne — plusieurs changements peuvent tomber
//	             dans le meme pas de 100 ms, et ce que le client dessine a cette frame est
//	             l'etat a sa fin.
//
// La deduplication ne se fait JAMAIS sur `T` seul : deux valeurs differentes dans la meme frame
// sont un vrai changement, c'est la VALEUR qui tranche.
//
// Les emissions hors fenetre sont ecartees sans bruit — elles sont le cas nominal (le film
// continue apres la derniere position publiee, et commence avant la premiere).
func scoreTicksOf(pts []types.ScorePoint, c scoreClock) []ScoreTick {
	out := make([]ScoreTick, 0, len(pts))
	for _, p := range pts {
		t, ok := c.frameOf(p.TimeMS)
		if !ok {
			continue
		}
		v, n := int(p.Value), len(out)
		switch {
		case n > 0 && out[n-1].V == v:
			continue // palier : la valeur n'a pas change, l'emission n'est pas un point
		case n > 0 && out[n-1].T == t:
			out[n-1].V = v // meme frame, valeur differente : l'etat de fin de frame gagne
		default:
			out = append(out, ScoreTick{T: t, V: v})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// scoreRoundsOf pose les manches d'un slot sur la grille, dans l'ordre des manches.
func scoreRoundsOf(byRound map[int][]types.ScorePoint, c scoreClock) []ScoreRound {
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
func buildScoreTimeline(in *ScoreInput, deaths []Death, c scoreClock,
	fb *fallback.Compteur) (*ScoreTimeline, *ScoreCoverage) {
	if in == nil {
		return nil, nil
	}
	recs := in.Records
	teamScore := loadScoreSeries(recs, objectives.ModeScoreComponent, true)
	teamFrags := loadScoreSeries(recs, objectives.KillsComponent, true)
	playerFrags := loadScoreSeries(recs, objectives.KillsComponent, false)
	// L'identite PLATE par TOTAUX reste la source de la preuve (b) d'identite des CAMPS
	// (`resolveTeamIdentity`) : les courbes d'equipe ne bougent pas. En MONO-MANCHE elle nomme
	// aussi les joueurs (le slot n'est pas reattribue) ; en MULTI-MANCHE, `buildPlayerScores`
	// passe par l'identite PAR MANCHE (les instants de mort). Cf. buildPlayerScores.
	identity := objectives.SlotIdentityFrom(recs, in.Lines)

	slots := teamSlotsOf(teamScore, teamFrags)
	teamID, method := resolveTeamIdentity(in, slots, teamScore, teamFrags, playerFrags, identity)

	tl := &ScoreTimeline{
		Teams:   buildTeamScores(slots, teamScore, teamID, c),
		Players: buildPlayerScores(recs, identity, in.Lines, deaths, c),
	}
	tl.TargetScore = publishableTarget(in.TargetScore, slots, teamScore, len(tl.Teams))
	if in.HoldTicksPerPoint > 0 {
		// GARDE DE MODE : la serie n'est construite que sur une variante DECLAREE — `comp 23 A`
		// existe sur tous les modes, il ne porte des tics de colline que sur un mode a colline.
		tl.HoldTicks = buildHoldTicks(recs, identity, in.TeamByXUID, c)
		tl.HoldTicksPerPoint = publishableHold(in.HoldTicksPerPoint, len(tl.HoldTicks))
	}
	cov := &ScoreCoverage{
		TeamIdentity:  method,
		ModeSupported: len(teamScore.total) > 0,
		Truncated:     in.Truncated,
		Oracle:        ScoreOracleDisplayed,
		Points:        countScorePoints(tl),
	}
	attachRoundsCoverage(cov, objectives.ResolveRounds(recs), fb)
	if len(tl.Teams) == 0 && len(tl.Players) == 0 {
		tl = nil
	}
	return tl, cov
}

// attachRoundsCoverage porte le verdict de la lecture des designateurs de manche dans la
// couverture, et compte le seul REPLI de cette chaine.
//
// LES TROIS FORMES DE D14 (c) EN UN SEUL ENDROIT : la grammaire (`RoundsWritten`), la
// contradiction (`RoundsContradicted` et ses enregistrements), le repli (`RoundsDecreed`, qui
// se compte AUSSI au registre pour apparaitre en `coverage.fallbacks[]`). `Rounds` reste le
// COMPTE des manches retenues, inchange — c'est la grandeur que le rejeu consomme.
//
// `RoundsWritten` n'est publie qu'a partir de DEUX designateurs : sur un film mono-manche il
// rendrait `[0]`, ce que `Rounds = 1` dit deja, au prix d'un champ sur les 1 300 films du parc.
func attachRoundsCoverage(cov *ScoreCoverage, d objectives.RoundsDecision, fb *fallback.Compteur) {
	cov.Rounds = len(d.Real)
	if len(d.Written) > 1 {
		cov.RoundsWritten = d.Written
	}
	cov.RoundsContradicted = d.Contradicted
	cov.RoundsContradictedRecords = d.ContradictedRecords
	cov.RoundsDecreed = d.Decreed
	if d.Decreed {
		fb.Declenche(fallback.NomReplayMancheZeroDecretee)
	}
}

// publishableTarget applique la garde de la cible de victoire (cf. ScoreTimeline.TargetScore) :
// elle n'est publiee que si elle est connue (> 0), si le calque porte des courbes d'equipe (une
// cible sans courbe ne norme rien), et si AUCUN score final du film ne la depasse — un final
// au-dessus prouve que la table est perimee pour cette variante, et une cible fausse se lirait
// comme une mesure.
func publishableTarget(target int, slots []int, score scoreSeriesSet, teams int) *int {
	if target <= 0 || teams == 0 {
		return nil
	}
	for _, slot := range slots {
		if final, ok := score.final(slot); ok && final > int64(target) {
			return nil
		}
	}
	return &target
}

// publishableHold applique la garde du denominateur de garde (cf.
// ScoreTimeline.HoldTicksPerPoint) : il n'est publie que s'il est connu (> 0) et si le calque
// porte au moins une serie de garde.
//
// POURQUOI LA SECONDE CONDITION. Un denominateur sans numerateur ne norme rien : le client
// aurait de quoi diviser mais rien a diviser, et une jauge a zero se lirait comme une mesure.
//
// AUCUNE GARDE « la valeur depasse ce que le film montre », contrairement a publishableTarget :
// le film ne porte pas de total de reference pour la garde. La mesure qui fonde le 35 est en
// tete de hill_hold_ticks.go.
func publishableHold(ticks, series int) *int {
	if ticks <= 0 || series == 0 {
		return nil
	}
	return &ticks
}

// teamSlotsOf rend les slots d'entite d'equipe vus par le film, dans l'ordre.
func teamSlotsOf(score, frags scoreSeriesSet) []int {
	seen := map[int]bool{}
	for _, m := range []map[int][]types.ScorePoint{score.total, frags.total} {
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
