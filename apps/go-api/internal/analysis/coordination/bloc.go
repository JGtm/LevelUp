package coordination

// bloc.go — LE BLOC « COORDINATION » D'UN SCOPE : l'appui reçu (bloc_appui.go), compté sur les
// matchs MESURÉS du scope, avec la parité pondérée et une case par match.
//
// # UN MATCH NON MESURÉ NE FOURNIT NI NUMÉRATEUR NI DÉNOMINATEUR
//
// Le drapeau `Mesure` vient du lecteur (au moins une ligne publiable dans
// `match_kill_events_latest`) — la MÊME définition que l'onglet Tactique et la page
// Escouade. Compter ses matchs au dénominateur « par match » ferait varier la grandeur avec
// la COUVERTURE DE FILM au lieu du jeu (correction G2).

import "levelup/go-api/internal/domain"

// cumulMatch — les comptes bruts d'UN match mesuré, avant mise en forme.
type cumulMatch struct {
	teamSize *int

	myKills, myAssisted      int
	teamAssists, assistsToMe int
}

// Bloc rend le bloc de coordination d'un scope de matchs.
//
// Available=false avec une RAISON MACHINE dans deux cas, et aucun des deux n'est une panne :
// pas de sujet (xuid vide) et aucun match mesuré (films expirés, titre sans décodeur).
// Jamais un bloc de zéros : « aucune donnée » n'est pas « zéro pour cent ».
func Bloc(in domain.CoordinationEntree) domain.CoordinationBlock {
	out := domain.CoordinationBlock{MatchesTotal: len(in.Matchs)}
	cumuls, ordre := cumulsMesures(in.Matchs)
	out.MatchesMeasured = len(ordre)
	if in.MoiXUID == "" || out.MatchesMeasured == 0 {
		out.UnavailableReason = domain.CoordinationNoMeasuredMatch
		return out
	}

	compterAppuis(in, cumuls)

	out.Available = true
	out.Appui = agregerAppui(cumuls, ordre, out.MatchesMeasured)
	out.PerMatch = casesParMatch(cumuls, ordre)
	return out
}

// Restreindre découpe une entrée sur un sous-ensemble de matchs, pour la maille SOIRÉE de
// la frise temporelle.
//
// L'UNIVERS EST DÉCOUPÉ AVEC LES APPUIS : un match retenu qui ne porte aucun appui
// compte au dénominateur « par match », et le déduire des appuis l'effacerait (même
// défaut, mêmes conséquences, que la phase 1 du plan tactique). Les équipes suivent : un
// match sans sa table d'équipes n'a plus aucun appui de camp.
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
