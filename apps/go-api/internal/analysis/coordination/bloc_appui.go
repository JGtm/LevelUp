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
// # UN FRAG DONT L'ASSISTANCE EST INCONNUE N'ENTRE DANS AUCUN DÉNOMINATEUR
//
// La porte est celle du lecteur (`publishable AND assist_known`). Un assistant VIDE sur une
// ligne mesurée est un fait — « personne n'a assisté » — et compte au dénominateur ; une
// mort dont l'assistance n'est pas mesurée n'a pas de ligne du tout. Confondre les deux
// écrirait « aucune assistance » là où la mesure dit « on ne sait pas ».

import "levelup/go-api/internal/domain"

// compterAppuis ventile les lignes d'appui du scope sur les quatre compteurs du joueur.
//
// LES DEUX CÔTÉS DU COUPLE SONT TESTÉS SÉPARÉMENT : « le tueur est de mon camp » décide de
// l'appartenance de l'appui à mon camp, « l'assistant est de mon camp » évite de compter
// un appui adverse ou d'un joueur sans camp. Un appui que je porte à mon camp compte au
// dénominateur (c'est ce que le camp a distribué) sans compter au numérateur.
func compterAppuis(in domain.CoordinationEntree, cumuls map[string]*cumulMatch) {
	for _, a := range in.Appuis {
		c := cumuls[a.MatchID]
		if c == nil || a.Nombre <= 0 {
			continue
		}
		if a.KillerXUID == in.MoiXUID {
			c.myKills += a.Nombre
			if a.AssistXUID != "" {
				c.myAssisted += a.Nombre
			}
		}
		if a.AssistXUID == "" {
			continue
		}
		if !memeCamp(in.Equipes, a.MatchID, in.MoiXUID, a.AssistXUID) ||
			!memeCamp(in.Equipes, a.MatchID, in.MoiXUID, a.KillerXUID) {
			continue
		}
		c.teamAssists += a.Nombre
		if a.KillerXUID == in.MoiXUID {
			c.assistsToMe += a.Nombre
		}
	}
}

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
// Les parts sont NIL quand leur dénominateur est vide : aucune mort vengeable de camp,
// aucun frag mesuré, aucun appui dans le camp. La case reste grise — « non mesuré n'est pas
// zéro », et un zéro peint se lirait comme une contre-performance.
func casesParMatch(cumuls map[string]*cumulMatch, ordre []string) []domain.CoordinationMatchPoint {
	out := make([]domain.CoordinationMatchPoint, 0, len(ordre))
	for _, id := range ordre {
		c := cumuls[id]
		point := domain.CoordinationMatchPoint{
			MatchID:              id,
			TeamSize:             c.teamSize,
			ParityPct:            pariteDuMatch(c.teamSize),
			TeamDeaths:           c.teamDeaths,
			TeamDeathsAvenged:    c.teamAvenged,
			MyDeaths:             c.myDeaths,
			MyDeathsAvenged:      c.myAvenged,
			MyRipostes:           c.myRipostes,
			RiposteSharePct:      partPct(c.myRipostes, c.teamDeaths),
			CoveredSharePct:      partPct(c.myAvenged, c.myDeaths),
			MyMeasuredKills:      c.myKills,
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
