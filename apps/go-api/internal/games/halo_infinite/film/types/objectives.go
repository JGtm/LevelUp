package types

// objectives.go — LES TYPES DE CONTRAT DE LA COUCHE DES FAITS, PART `objectives`.
//
// Deplaces de `film/facts/objectives/` au lot 2.6.2 (2026-09-16), SANS reecriture. HUIT des
// quatorze types que la note de preparation §3.2 avait comptes : les six autres
// (`FlagFilmSignals`, `NamedEvent`, `IdentifiedEvent`, `RoundBounds`, `RoundIdentity`,
// `RoundsDecision`, `StatComponent`) portent une METHODE, donc une regle mesuree ou un etat
// non exporte : ils appartiennent a la couche qui les decide. `StatValue` s ajoute a la liste
// de la note — `StatRecord` le nomme dans un champ, il ne pouvait pas rester derriere.

// DeathInstant est une mort DATEE ET NOMMEE, telle que le fil des morts du film la donne.
// L'appelant la fournit : ce paquet ne decode pas le fil des morts (il a un seul proprietaire
// dans le depot, `games/halo_infinite/film/replay`) et n'ouvre aucune base.
type DeathInstant struct {
	// XUID de la victime, en decimal — meme ecriture que [PlayerLine.XUID].
	XUID string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH, la meme que celle des
	// [StatRecord].
	TimeMS int
}

// FlagSpan est UN intervalle d etat d un drapeau, borne en millisecondes.
type FlagSpan struct {
	// State : l une des quatre constantes ci-dessus. Une valeur inconnue est IGNOREE — elle
	// n est ni un portage ni un retour au socle, et l inventer serait pire que la taire.
	State string
	// StartMS / EndMS bornent l intervalle. EndMS est INCLUS.
	StartMS, EndMS int
	// XUID est le porteur, en decimal. Vide pour les etats non portes — et pour un portage
	// que le pont n a pas nomme : une prise sans proprietaire ne se compte a personne.
	XUID string
}

// FlagTrack est LA VIE D UN DRAPEAU sur toute la partie. Le regroupement est par OBJET : en CTF
// il y a deux drapeaux, donc au plus deux pistes. C est LUI qui porte l identite « meme
// drapeau » de la definition — deux prises de deux drapeaux differents ne se replient jamais
// l une l autre, meme a une milliseconde d ecart.
type FlagTrack struct {
	// Team est l equipe PROPRIETAIRE du drapeau (-1 = inconnue). Publiee pour se lire ; la
	// regle n en depend pas, c est la piste qui fait l identite.
	Team int
	// Spans est la vie du drapeau. L ordre d entree n a pas d importance : la fonction trie.
	Spans []FlagSpan
}

// FlagGrabsNetPlayer porte les deux comptes d UN joueur sur UN match.
type FlagGrabsNetPlayer struct {
	// XUID du joueur, en decimal.
	XUID string
	// Raw : les prises BRUTES lues sur les pistes — le compteur officiel, jonglage compris.
	Raw int
	// Net : les prises NETTES, jonglage replie.
	Net int
}

// PlayerLine est la ligne de match d'un joueur, telle que `match_participants` la donne.
// L'appelant la fournit : ce paquet ne touche jamais la base.
type PlayerLine struct {
	// XUID en decimal, meme forme que la base et que le rejeu 2D.
	XUID string
	// Kills, Deaths, Assists sont les compteurs du match.
	Kills, Deaths, Assists int
}

// ScorePoint est une emission de score par une entite du match.
type ScorePoint struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film.
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Value est le score a cet instant.
	Value int64
}

// StatValue porte les valeurs d'un composant. Le sens de chaque canal depend du composant :
// le score de mode est en A, le score personnel en B (cf. score.go).
//
// # QUATRE CANAUX, PAS DEUX — et les deux derniers n'ont jamais ete lus (2026-08-31)
//
// La grammaire du composant porte DEUX valeurs inconditionnelles (A, B) puis DEUX DRAPEAUX,
// chacun commandant une valeur CONDITIONNELLE. Jusqu'ici [decodeStatComponent] lisait ces deux
// dernieres pour AVANCER le curseur, et les jetait. A 28 composants et 2 canaux perdus, ce sont
// **56 emplacements que rien dans le depot n'avait jamais regardes** — et la question ouverte
// de l'Assaut (« ou est l'ARMEMENT de la bombe ? », dont le releve A0.3 dit qu'il n'a aucun
// increment dans les canaux A et B) tombe exactement dans cet angle mort.
//
// C et D ne sont donc pas une commodite : ce sont les deux tiroirs qu'on n'avait pas ouverts.
// `HasC` / `HasD` disent si le drapeau etait pose — un canal ABSENT et un canal a ZERO sont
// deux choses differentes, et les confondre ferait lire un compteur la ou il n'y a rien.
type StatValue struct {
	A, B       int64
	C, D       int64
	HasC, HasD bool
}

// StatRecord est un enregistrement d'entite decode d'un paquet FRAME.
type StatRecord struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film (meme base que le
	// TimeMS des ObjectiveEvent).
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Round est la MANCHE que decrit cet enregistrement, 0-based (0 = premiere manche). Elle
	// est lue dans les en-tetes de 5 bits du premier composant. Un compteur repart de zero a
	// chaque manche : le total d'un match est la SOMME des manches, jamais la derniere valeur.
	Round int
	// Comps porte les composants effectivement transportes par cet enregistrement.
	Comps map[int]StatValue
}
