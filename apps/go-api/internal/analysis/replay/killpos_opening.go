package replay

// killpos_opening.go — OÙ L'ENGAGEMENT A COMMENCÉ, ET POURQUOI CE N'EST QU'UN PROXY.
//
// # LA QUESTION
//
// La position d'une mort dit où le coup fatal est tombé. Elle ne dit pas où l'engagement a
// COMMENCÉ — et c'est cette distance-là qui a un sens tactique : on ouvre à 20 m au fusil de
// précision et on finit au contact, ou l'inverse.
//
// # POURQUOI UN PROXY ET PAS LE VRAI PREMIER DÉGÂT
//
// La vraie ouverture exigerait le PREMIER dégât de l'échange. Mesuré le 2026-09-06 (sonde
// `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`) : le film n'émet que 91 à 428
// enregistrements de dégât pour 90 à 117 morts, et le dégât fatal lui-même n'est capturé que
// dans 10 à 50 % des morts selon le film. Le flux de dégâts complet N'EST PAS dans le film ;
// aucune amélioration de décodage n'y changera rien (le sous-type 0xC0 non décodé n'ouvre
// aucun réservoir). Il n'y a donc PAS d'événement d'ouverture à lire.
//
// Les trajectoires, elles, sont denses et continues. D'où le proxy : la position des deux
// joueurs UN TEMPS-POUR-TUER avant le coup fatal. C'est un décalage d'horloge, pas un
// événement — et c'est précisément pour ça qu'il se valide (voir plus bas).
//
// # IL N'Y A PAS DE SECONDE FONCTION DE PLACEMENT
//
// La position d'entame se compose, elle ne se réécrit pas :
//
//	BuildKillPositions(pos, slotXUID, ShiftKillRefs(kills, -OpeningLeadMS), off)
//
// `BuildKillPositions` est PURE et prend l'instant en paramètre ; lui donner des couples
// décalés rend l'entame. Un second producteur de position divergerait du premier à la
// première correction portée d'un seul côté — c'est la règle « deux décodeurs du même fait
// divergeraient » qui gouverne déjà killpos.go et killpos_bridge.go.

// OpeningLeadMS : l'avance, en millisecondes, du proxy d'entame sur le coup fatal.
//
// D'OÙ VIENT LA VALEUR : c'est UN TEMPS-POUR-TUER, pas un réglage. La sonde du 2026-09-06 le
// mesure à 1,2-1,5 s au fusil de combat sur les quatre films — c'est la durée pendant
// laquelle les deux joueurs se sont vus. Un décalage plus court retomberait sur la position
// du coup fatal (donc ne dirait rien de neuf) ; un décalage plus long remonterait avant le
// contact visuel, là où les deux trajectoires ne parlent plus du même engagement.
//
// LA VALEUR EST VALIDÉE, PAS SUPPOSÉE : `duels_ouverture_research_test.go` compare, sur les
// morts dont le premier dégât de l'échange EST capturé, la distance à cet instant et la
// distance à T − OpeningLeadMS. Gate écrit avant la mesure : écart médian <= 2 m.
const OpeningLeadMS int64 = 1_500

// ShiftKillRefs rend une COPIE des couples, tous décalés de deltaMS sur l'horloge du match.
//
// PURE : l'entrée n'est ni mutée ni réordonnée. Pour remonter à l'entame, l'appelant passe
// `-OpeningLeadMS` — le signe est à lui, cette fonction ne présume d'aucun sens.
//
// UN INSTANT QUI DEVIENDRAIT NÉGATIF N'EST PAS ÉCARTÉ ICI, ET C'EST VOULU. Une mort survenue
// dans la première seconde et demie d'un match n'a pas d'entame lisible ; le décalage la
// place avant l'origine du film, `BuildKillPositions` n'y trouve aucun échantillon dans la
// tolérance et rend une position ABSENTE. C'est le résultat correct — mieux vaut pas
// d'entame qu'une position de réapparition présentée comme une entame. Le comportement est
// épinglé par `TestShiftKillRefsInstantNegatifNeDonneAucunePosition` : le jour où la porte
// de `positionOf` changerait, ce test le dirait plutôt que de laisser passer un zéro.
func ShiftKillRefs(kills []KillRef, deltaMS int64) []KillRef {
	if len(kills) == 0 {
		return nil
	}
	out := make([]KillRef, len(kills))
	for i, k := range kills {
		k.TimeMS += deltaMS
		out[i] = k
	}
	return out
}
