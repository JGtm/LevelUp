package domain

// Types de coordination — mesures d'equipe PARTAGEES par l'onglet Tactique et la page
// Escouade (l'echange, la couverture). Structs purs : aucune I/O, aucun SQL.

// Couverture est LE type de retour d'un taux dans ce domaine. Il n'y a pas de variante qui
// rendrait un float64 nu, et c'est une regle, pas une preference.
//
// POURQUOI. Un taux seul ment de deux facons. Il ment sur la TAILLE : « 100 % de morts
// vengees » sur huit morts n'est pas une performance, c'est un echantillon. Il ment sur le
// VOLUME : deux joueurs a 40 % ne se comparent pas si l'un joue deux fois plus. Le type
// force donc les trois grandeurs a voyager ensemble — le taux, le compte brut, et la
// quantite par match — et porte le drapeau d'echantillon faible avec elles (doctrine
// SquadAssistPairsTable, plan tactique 2026-09-05).
type Couverture struct {
	// Taux est en unite 0..1 (ADR 0006), jamais en pourcentage : la mise en forme est un
	// choix d'affichage, pas une propriete de la mesure.
	Taux float64

	// Brut est le numerateur : le nombre d'evenements comptes (morts vengees, ...).
	Brut int

	// ParMatch est la quantite brute ramenee au match. C'est ce qui rend deux joueurs
	// comparables quand ils n'ont pas joue le meme nombre de matchs.
	ParMatch float64

	// N est le denominateur : la taille de l'echantillon (morts examinees, ...).
	N int

	// EchantillonFaible dit que N est sous le plancher (cf.
	// coordination.SeuilEchantillonFaible). L'affichage doit alors poser la reserve
	// `explorer.briefing.low_sample` — et ne classer personne.
	EchantillonFaible bool
}
