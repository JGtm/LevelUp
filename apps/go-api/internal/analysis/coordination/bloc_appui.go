package coordination

// bloc_appui.go — LE VERSANT APPUI REÇU du bloc, la parité pondérée, et les cases de la
// bande de régularité.
//
// # « COMBIEN DE MES FRAGS M'ONT ÉTÉ PRÉPARÉS »
//
// L'app compte depuis toujours les appuis DONNÉS (la statistique de jeu « assistances »).
// Le côté REÇU est l'autre moitié de la coordination : un joueur qui finit ce que son camp
// entame ne joue pas comme un joueur qui ouvre seul.
//
// # LES APPUIS SONT MESURÉS POUR TOUS LES JOUEURS DU FILM (réserve R2, levée le 2026-09-21)
//
// Aucun filtre « joueur suivi » n'existe sur la chaîne de production. Le dénominateur
// « appuis distribués dans mon camp » est donc COMPLET : il compte le coéquipier de
// rencontre comme le coéquipier suivi, et ne varie pas avec la composition affichée à
// l'écran. C'est la mesure la plus stable, et c'est pour cela qu'elle est retenue.
//
// # LA BASE DE « FRAGS APPUYÉS » EST LA FEUILLE DE MATCH
//
// Règle des bases (domain/relation_assists.go) : le dénominateur est le nombre de frags
// OFFICIELS du joueur sur chaque match mesuré, frags sur des bots compris ; il ne descend
// jamais sous les frags que le film lit pour lui (domain.BaseFragsOfficiels). Une mort dont
// l'assistance n'est pas lue reste dans la base sans entrer au numérateur. Les appuis
// impliquant des BOTS comptent comme les autres.

import "levelup/go-api/internal/domain"

// compterAppuis ventile les lignes d'appui du scope sur les quatre compteurs du joueur, puis
// pose la base officielle de chaque match.
//
// UN APPUI EST DE MON CAMP DÈS QU'UN DE SES ACTEURS L'EST : une assistance est toujours
// portée par un coéquipier du tueur, donc un assistant de mon camp fait le tueur de mon camp,
// et inversement — c'est ce qui range un bot (sans xuid, sans ligne dans la table des
// équipes) du côté de l'acteur nommé de la ligne. Quand ni l'un ni l'autre n'a de xuid, la
// victime tranche (campDeLaVictimeAdverse). Un appui que je porte à mon camp compte au
// dénominateur (c'est ce que le camp a distribué) sans compter au numérateur.
func compterAppuis(in domain.CoordinationEntree, cumuls map[string]*cumulMatch) {
	for _, a := range in.Appuis {
		c := cumuls[a.MatchID]
		if c == nil || a.Nombre <= 0 {
			continue
		}
		if a.KillerXUID == in.MoiXUID {
			c.myFilmKills += a.Nombre
			if a.Assiste() {
				c.myAssisted += a.Nombre
			}
		}
		if !a.Assiste() || !appuiDeMonCamp(in, a) {
			continue
		}
		c.teamAssists += a.Nombre
		if a.KillerXUID == in.MoiXUID {
			c.assistsToMe += a.Nombre
		}
	}
	for matchID, c := range cumuls {
		var officiels *int
		if n, ok := in.FragsOfficiels[matchID]; ok {
			officiels = &n
		}
		c.myKills = domain.BaseFragsOfficiels(officiels, c.myFilmKills)
	}
}

// appuiDeMonCamp : l'appui `a` a-t-il été distribué dans mon camp ?
func appuiDeMonCamp(in domain.CoordinationEntree, a domain.CoordinationAppuiRow) bool {
	if memeCamp(in.Equipes, a.MatchID, in.MoiXUID, a.AssistXUID) ||
		memeCamp(in.Equipes, a.MatchID, in.MoiXUID, a.KillerXUID) {
		return true
	}
	if a.AssistXUID != "" || a.KillerXUID != "" {
		return false
	}
	return campDeLaVictimeAdverse(in.Equipes, a.MatchID, in.MoiXUID, a.VictimXUID)
}

// campDeLaVictimeAdverse : la victime est-elle de l'AUTRE camp d'un match à deux camps ? Le
// tueur est alors de mon camp. Un match à plus de deux camps, ou une victime sans camp
// connu, ne permet pas de conclure : l'appui n'est pas compté.
func campDeLaVictimeAdverse(equipes domain.EquipesParMatch, matchID, moi, victime string) bool {
	duMatch := equipes[matchID]
	monEquipe, jeSuisLa := duMatch[moi]
	saEquipe, elleEstLa := duMatch[victime]
	if victime == "" || !jeSuisLa || !elleEstLa || saEquipe == monEquipe {
		return false
	}
	camps := make(map[int]struct{}, campsParMatchADeuxCamps)
	for _, equipe := range duMatch {
		camps[equipe] = struct{}{}
	}
	return len(camps) == campsParMatchADeuxCamps
}

// campsParMatchADeuxCamps : un match où la victime d'un camp désigne le tueur de l'autre.
const campsParMatchADeuxCamps = 2

// cumulParite accumule la part ÉQUITABLE d'un scope, PONDÉRÉE par le volume de chaque match.
//
// POURQUOI PONDÉRER. Un scope qui mêle du 4v4 et du BTB n'a pas une parité unique : 25 %
// d'un côté, 8,3 % de l'autre. La moyenne simple des deux donnerait une référence que
// AUCUN match n'a connue, et la moyenne des EFFECTIFS (100/2,5) est pire encore — elle
// n'est même pas la moyenne des parités. La bonne référence est la part attendue du joueur
// si les événements s'étaient répartis également : chaque match pèse ce qu'il a produit.
//
// Un match sans effectif de camp (FFA) ne pèse RIEN et n'ajoute rien : il n'a pas de
// parité, pas une parité nulle.
type cumulParite struct {
	somme float64
	poids float64
}

func nouveauCumulParite() *cumulParite { return &cumulParite{} }

// ajouter pèse la parité 100/n de ce match par son volume d'événements.
func (p *cumulParite) ajouter(teamSize *int, volume int) {
	if teamSize == nil || *teamSize <= 0 || volume <= 0 {
		return
	}
	p.somme += float64(volume) * 100 / float64(*teamSize)
	p.poids += float64(volume)
}

// pct rend la parité du scope, ou nil quand aucun match mesuré n'a d'effectif de camp.
func (p *cumulParite) pct() *float64 {
	if p.poids <= 0 {
		return nil
	}
	v := p.somme / p.poids
	return &v
}

// casesParMatch rend UNE case par match mesuré, dans l'ordre du scope.
//
// Les parts sont NIL quand leur dénominateur est vide : aucun frag, aucun appui dans le camp.
// La case reste grise — un zéro peint se lirait comme une contre-performance.
func casesParMatch(cumuls map[string]*cumulMatch, ordre []string) []domain.CoordinationMatchPoint {
	out := make([]domain.CoordinationMatchPoint, 0, len(ordre))
	for _, id := range ordre {
		c := cumuls[id]
		point := domain.CoordinationMatchPoint{
			MatchID:              id,
			TeamSize:             c.teamSize,
			ParityPct:            pariteDuMatch(c.teamSize),
			MyKills:              c.myKills,
			MyAssistedKills:      c.myAssisted,
			TeamAssists:          c.teamAssists,
			AssistsToMe:          c.assistsToMe,
			AssistShareOfTeamPct: partPct(c.assistsToMe, c.teamAssists),
			AssistedSharePct:     partPct(c.myAssisted, c.myKills),
		}
		out = append(out, point)
	}
	return out
}

// pariteDuMatch rend 100/n, ou nil quand le camp du match est inconnu (FFA, réserve R1) —
// jamais une parité de 100 % fabriquée sur un effectif de 1 inventé.
func pariteDuMatch(teamSize *int) *float64 {
	if teamSize == nil || *teamSize <= 0 {
		return nil
	}
	v := 100 / float64(*teamSize)
	return &v
}

// partPct rend un pourcentage, ou nil sur dénominateur vide.
func partPct(brut, n int) *float64 {
	if n <= 0 {
		return nil
	}
	v := float64(brut) * 100 / float64(n)
	return &v
}
