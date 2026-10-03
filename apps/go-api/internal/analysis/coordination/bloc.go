package coordination

// bloc.go — LE BLOC « COORDINATION » D'UN SCOPE : riposte et appui reçu, au même endroit.
//
// # CE FICHIER NE DÉFINIT AUCUNE MÉCANIQUE NEUVE
//
// « Qui venge qui, dans la fenêtre » est `Echanges` (trade.go), et rien d'autre. Ce fichier
// PROJETTE ce bilan sur DEUX axes que les pages Sessions et Séries temporelles demandent —
// les morts de MON CAMP, et les ripostes que JE porte — puis y ajoute le versant appui,
// compté sur les mêmes matchs mesurés. Une seconde recherche de vengeur aurait donné deux
// vengeurs pour la même mort au premier ajustement (CLAUDE.md n 6).
//
// # LES DEUX DÉNOMINATEURS DE LA RIPOSTE NE SONT PAS LES MÊMES
//
//	JE SUIS COUVERT   mes morts vengées / MES morts vengeables ;
//	JE RIPOSTE        mes ripostes / les morts vengeables DE MON CAMP.
//
// Normaliser la seconde par mes morts ferait gonfler ma part dans tout match où le camp
// meurt peu — ce serait une mesure de la mortalité des autres, pas de ce que je fais.
//
// # UN MATCH NON MESURÉ NE FOURNIT NI NUMÉRATEUR NI DÉNOMINATEUR
//
// Le drapeau `Mesure` vient du lecteur (au moins une ligne publiable dans
// `match_kill_events_latest`) — la MÊME définition que l'onglet Tactique et la page
// Escouade. Compter ses matchs au dénominateur « par match » ferait varier la grandeur avec
// la COUVERTURE DE FILM au lieu du jeu (correction G2).

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// cumulMatch — les comptes bruts d'UN match mesuré, avant mise en forme.
type cumulMatch struct {
	teamSize *int

	teamDeaths, teamAvenged  int
	myDeaths, myAvenged      int
	myRipostes               int
	delais                   []int64
	myKills, myAssisted      int
	teamAssists, assistsToMe int
}

// Bloc rend le bloc de coordination d'un scope de matchs.
//
// Available=false avec une RAISON MACHINE dans deux cas, et aucun des deux n'est une panne :
// pas de sujet (xuid vide) et aucun match mesuré (films expirés, titre sans décodeur).
// Jamais un bloc de zéros : « aucune donnée » n'est pas « zéro pour cent ».
func Bloc(in domain.CoordinationEntree) domain.CoordinationBlock {
	out := domain.CoordinationBlock{
		FenetreMs:    FenetreEchangeMs,
		MatchesTotal: len(in.Matchs),
	}
	cumuls, ordre := cumulsMesures(in.Matchs)
	out.MatchesMeasured = len(ordre)
	if in.MoiXUID == "" || out.MatchesMeasured == 0 {
		out.UnavailableReason = domain.CoordinationNoMeasuredMatch
		return out
	}

	compterRipostes(in, cumuls)
	compterAppuis(in, cumuls)

	out.Available = true
	out.Riposte = agregerRiposte(cumuls, ordre, out.MatchesMeasured)
	out.Appui = agregerAppui(cumuls, ordre, out.MatchesMeasured)
	out.PerMatch = casesParMatch(cumuls, ordre)
	return out
}

// Restreindre découpe une entrée sur un sous-ensemble de matchs, pour la maille SOIRÉE de
// la frise temporelle.
//
// L'UNIVERS EST DÉCOUPÉ AVEC LES ÉVÉNEMENTS : un match retenu qui ne porte aucune mort
// compte au dénominateur « par match », et le déduire des événements l'effacerait (même
// défaut, mêmes conséquences, que la phase 1 du plan tactique). Les équipes suivent : un
// match sans sa table d'équipes n'a plus aucune mort vengeable.
func Restreindre(in domain.CoordinationEntree, matchIDs []string) domain.CoordinationEntree {
	garde := make(map[string]struct{}, len(matchIDs))
	for _, id := range matchIDs {
		garde[id] = struct{}{}
	}
	out := domain.CoordinationEntree{MoiXUID: in.MoiXUID, Equipes: domain.EquipesParMatch{}}
	for _, m := range in.Matchs {
		if _, ok := garde[m.MatchID]; ok {
			out.Matchs = append(out.Matchs, m)
		}
	}
	for matchID, equipes := range in.Equipes {
		if _, ok := garde[matchID]; ok {
			out.Equipes[matchID] = equipes
		}
	}
	for _, e := range in.Kills {
		if _, ok := garde[e.MatchID]; ok {
			out.Kills = append(out.Kills, e)
		}
	}
	for _, a := range in.Appuis {
		if _, ok := garde[a.MatchID]; ok {
			out.Appuis = append(out.Appuis, a)
		}
	}
	return out
}

// cumulsMesures ouvre un cumul par match MESURÉ et fige l'ordre d'affichage (celui du
// scope). Le parcours d'une map n'est pas un ordre : la bande de régularité doit rendre
// les mêmes cases dans le même sens à chaque appel.
func cumulsMesures(matchs []domain.CoordinationMatch) (map[string]*cumulMatch, []string) {
	cumuls := make(map[string]*cumulMatch, len(matchs))
	ordre := make([]string, 0, len(matchs))
	for _, m := range matchs {
		if !m.Mesure {
			continue
		}
		if _, deja := cumuls[m.MatchID]; deja {
			continue
		}
		cumuls[m.MatchID] = &cumulMatch{teamSize: m.TeamSize}
		ordre = append(ordre, m.MatchID)
	}
	return cumuls, ordre
}

// compterRipostes projette le bilan d'échange sur les deux axes du joueur consulté.
//
// Seules les morts VENGEABLES entrent : compter comme un échec une mort que personne ne
// pouvait venger (tueur inconnu, camps inconnus) fausserait les deux taux.
func compterRipostes(in domain.CoordinationEntree, cumuls map[string]*cumulMatch) {
	for _, m := range Echanges(in.Kills, in.Equipes).Morts {
		c := cumuls[m.MatchID]
		if c == nil || !m.Vengeable {
			continue
		}
		if m.Vengee && m.VengeurXUID == in.MoiXUID {
			c.myRipostes++
		}
		if !memeCamp(in.Equipes, m.MatchID, in.MoiXUID, m.VictimeXUID) {
			continue
		}
		c.teamDeaths++
		if m.VictimeXUID == in.MoiXUID {
			c.myDeaths++
		}
		if !m.Vengee {
			continue
		}
		c.teamAvenged++
		c.delais = append(c.delais, m.DelaiMs)
		if m.VictimeXUID == in.MoiXUID {
			c.myAvenged++
		}
	}
}

// memeCamp : ce joueur est-il de MON camp sur ce match ? Moi-même en fais partie.
//
// Une identité vide (bot non rattaché, environnement) ou absente de la table des équipes
// n'a pas de camp : elle n'est d'aucun côté, et lui en deviner un serait une invention.
func memeCamp(equipes domain.EquipesParMatch, matchID, moi, autre string) bool {
	if autre == "" || moi == "" {
		return false
	}
	if autre == moi {
		return true
	}
	duMatch := equipes[matchID]
	monEquipe, jeSuisLa := duMatch[moi]
	son, ilEstLa := duMatch[autre]
	return jeSuisLa && ilEstLa && son == monEquipe
}

// agregerRiposte somme les cumuls et rend les deux couvertures, le délai médian et la
// parité pondérée.
func agregerRiposte(cumuls map[string]*cumulMatch, ordre []string, mesures int) domain.CoordinationRiposte {
	var out domain.CoordinationRiposte
	myDeaths, myAvenged, myRipostes := 0, 0, 0
	delais := make([]int64, 0, 64)
	parite := nouveauCumulParite()
	for _, id := range ordre {
		c := cumuls[id]
		out.TeamDeaths += c.teamDeaths
		out.TeamDeathsAvenged += c.teamAvenged
		myDeaths += c.myDeaths
		myAvenged += c.myAvenged
		myRipostes += c.myRipostes
		delais = append(delais, c.delais...)
		parite.ajouter(c.teamSize, c.teamDeaths)
	}
	out.JeSuisCouvert = Mesurer(myAvenged, myDeaths, mesures)
	out.JeRiposte = Mesurer(myRipostes, out.TeamDeaths, mesures)
	out.DelaiMedianMs = medianeMs(delais)
	out.ParityPct = parite.pct()
	return out
}

// agregerAppui somme le versant appui reçu et sa parité.
func agregerAppui(cumuls map[string]*cumulMatch, ordre []string, mesures int) domain.CoordinationAppui {
	myKills, myAssisted, teamAssists, toMe := 0, 0, 0, 0
	parite := nouveauCumulParite()
	for _, id := range ordre {
		c := cumuls[id]
		myKills += c.myKills
		myAssisted += c.myAssisted
		teamAssists += c.teamAssists
		toMe += c.assistsToMe
		parite.ajouter(c.teamSize, c.teamAssists)
	}
	return domain.CoordinationAppui{
		OnMePrepare:     Mesurer(myAssisted, myKills, mesures),
		MaPartDesAppuis: Mesurer(toMe, teamAssists, mesures),
		ParityPct:       parite.pct(),
	}
}

// medianeMs rend la médiane d'une suite de délais, ou nil si elle est vide.
//
// Convention du dépôt : sur un effectif PAIR, la moyenne des deux valeurs centrales,
// arrondie à la milliseconde. Nil plutôt que 0 — un zéro se lirait « riposte instantanée »
// là où il n'y a eu aucune riposte.
func medianeMs(delais []int64) *int64 {
	if len(delais) == 0 {
		return nil
	}
	sort.Slice(delais, func(i, j int) bool { return delais[i] < delais[j] })
	milieu := len(delais) / 2
	m := delais[milieu]
	if len(delais)%2 == 0 {
		m = (delais[milieu-1] + delais[milieu]) / 2
	}
	return &m
}
