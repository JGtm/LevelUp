package replay

// document_objectives_live.go — L'OBJECTIF VIVANT : la forme que prendra le DRAPEAU de CTF dans
// l'artefact, et ce que la mesure a refuse d'y mettre.
//
// PAS ENCORE PUBLIE, ET C'EST DELIBERE (2026-08-18). Ces types n'ont AUCUNE etiquette JSON et ne
// sont references par aucun champ de [ReplayDocument] : la mesure (items 1.1 et 1.2 du plan) est
// faite, la PUBLICATION (item 1.3 : champ du document, montee de SchemaVersion, contrat, OpenAPI,
// `generated.ts`, goldens, temoins re-cuits) est reportee APRES le rebasage de cette branche sur
// `feat/v75` — une autre session y fait entrer les schemas 12 et 13, et deux montees de version
// concurrentes se marcheraient dessus. Le calque prendra donc le NUMERO SUIVANT, pas 12.
//
// Ils vivent dans leur propre fichier pour la meme raison que `document_ground_weapons.go` : la
// FORME publiee ici, la REGLE qui la remplit dans `flag_carries.go`.
//
// D'OU VIENT CE QUI SERA PUBLIE, ET DE QUOI C'EST FAIT :
//
//	les BORNES        des evenements de statistique NOMMES du statborg, dates a la milliseconde
//	                  (`flag_grabs`, `flag_steals`, `flag_captures`, `flag_returns`) plus le fil
//	                  des morts du film. Aucune estimation, aucune fenetre de tolerance.
//	le PORTEUR        le slot statborg resolu en xuid par le pont par INSTANTS DE MORT
//	                  (`objectiveevents.SlotIdentityFromDeaths`) — le film seul, aucune base.
//	la POSITION       la piste PUBLIEE du porteur a l'instant considere : le drapeau porte EST
//	                  a la position de son porteur. Rien de l'objet n'est decode.
//	le MODE           trois signaux du film qui s'accordent (`objectiveevents.FlagFilmSignals`).
//	le DRAPEAU        le socle `flag_spawn` de la carte, du catalogue versionne d'objectifs.
//
// CE QUE LA MESURE A REFUSE DE PUBLIER, ET C'EST LA MOITIE DU RESULTAT :
//
//	Le CRANE d'Oddball. Le marqueur de portage du drapeau est TOTALEMENT ABSENT du film Oddball
//	mesure (0 porteur sur 26 images-cles), le statborg ne replique aucun compteur de crane, et la
//	signature structurelle seule laisse 195 motifs candidats. Il n'y a donc ni canal ni oracle :
//	rien n'est publie, et rien n'est devine.
//
//	L'OBJET LUI-MEME. Le marqueur `0x00010005` DIT qu'un joueur porte quelque chose ; il ne le
//	NOMME pas (0 suffixe d'identifiant `weap` sur 83 occurrences). Il sert donc de CONTROLE de ce
//	que les evenements nommes affirment — jamais de source. Le compte des portages qu'il confirme
//	est publie dans la couverture, avec son denominateur.
//
//	LE CANAL DES ARMES TENUES des paquets delta, mesure et REFUTE (0 occurrence du marqueur sur
//	68 284 lectures) : le cache qui le portait a ete retire du decodeur.

// FlagCarry est LA VIE D'UN DRAPEAU sur toute la partie : une suite d'intervalles d'etat.
//
// UN DRAPEAU, PAS UN PORTAGE. Le regroupement est par OBJET : en CTF il y a deux drapeaux, donc
// au plus deux entrees. Publier une entree par portage aurait oblige le client a reconstituer
// lui-meme la continuite entre « lache ici » et « repris la ».
type FlagCarry struct {
	// Team est l'equipe PROPRIETAIRE du drapeau, telle que le fichier de carte la donne sur le
	// socle `flag_spawn` ([TeamNeutral] = inconnue : carte absente du catalogue d'objectifs —
	// 72 cartes couvertes sur la centaine jouee).
	Team int
	// Spans est la vie du drapeau, en intervalles tries par T0, CONTIGUS des lors que le socle de
	// la carte est connu. Carte hors du catalogue d objectifs : les etats `home` sont omis (leur
	// position serait inventee) et la suite peut donc porter des trous.
	Spans []FlagSpan
}

// Les trois etats d'un drapeau. Il n'y en a pas de quatrieme : a tout instant, le drapeau est
// dans une main, par terre, ou sur son socle.
const (
	// FlagStateCarried : un joueur le porte. XUID est renseigne.
	FlagStateCarried = "carried"
	// FlagStateDropped : il est au sol, a l'endroit ou son dernier porteur l'a laisse.
	FlagStateDropped = "dropped"
	// FlagStateHome : il est a sa base.
	FlagStateHome = "home"
)

// FlagSpan est UN intervalle d'etat du drapeau.
type FlagSpan struct {
	// State vaut [FlagStateCarried], [FlagStateDropped] ou [FlagStateHome].
	State string
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS.
	T0, T1 int
	// XUID est le PORTEUR, en decimal — renseigne pour `carried`, nil pour les deux autres
	// etats. Un pointeur, pour que « pas de porteur » se distingue de « porteur inconnu ».
	XUID *string
	// X / Y : la position du drapeau en coordonnees monde (memes axes que Point.X/Y).
	//
	// POUR `carried`, C'EST LE POINT DE PRISE, ET LA SUITE SE LIT SUR LA PISTE DU PORTEUR.
	// Republier la trajectoire du drapeau serait republier celle de son porteur : le client
	// joint par XUID et suit la piste deja publiee. Pour `dropped`, c'est le dernier point connu
	// du porteur ; pour `home`, le socle `flag_spawn`.
	X, Y float32
}

// FlagCarriesCoverage porte les denominateurs du calque. Sans eux, « 12 portages » se lirait
// comme une exhaustivite, et un film CTF sans aucun portage publie serait indistinguable d'un
// film qui n'est pas du CTF.
type FlagCarriesCoverage struct {
	// FlagFilm dit si le film a ete RECONNU comme une partie de CTF par l'accord des trois
	// signaux (cf. `objectiveevents.FlagFilmSignals`). Faux : tout le reste vaut zero, et c'est
	// le cas nominal de tous les autres modes.
	FlagFilm bool
	// Bursts / Captures / Steals : les trois signaux qui ont fonde ce verdict, publies pour
	// qu'il se verifie.
	Bursts, Captures, Steals int
	// Openings est le nombre de PRISES de l'oracle (`flag_grabs` + `flag_steals`) une fois les
	// emissions jumelles fusionnees : le denominateur de tout ce qui suit.
	Openings int
	// Carries est le nombre de portages effectivement publies.
	Carries int
	// NoBridge : prises dont le slot statborg n'a pas ete resolu en xuid. Le pont se tait
	// plutot que d'attribuer le drapeau au mauvais joueur.
	NoBridge int
	// NoTrack : le porteur est nomme, mais aucune trajectoire publiee ne couvre l'instant de la
	// prise — le drapeau n'aurait pas de position a dessiner.
	NoTrack int
	// OutOfWindow : la prise tombe hors de l'axe de temps publie (fins de partie que le film
	// prolonge au-dela de la derniere position rendue).
	OutOfWindow int
	// MarkerObserved / MarkerConfirmed : le CONTROLE INDEPENDANT. MarkerObserved compte les
	// portages qui contiennent au moins une image-cle (le denominateur : sans image-cle, le
	// marqueur ne peut rien confirmer) ; MarkerConfirmed ceux dont au moins une image-cle porte
	// le marqueur sur le slot du porteur.
	//
	// LES DEUX CHAINES SONT DISJOINTES : les bornes viennent des compteurs de statistique du
	// statborg, le marqueur d'une suite de bits du record de bipede des images-cles. Leur accord
	// est donc une preuve, pas une tautologie.
	MarkerObserved, MarkerConfirmed int
	// Overlaps compte les prises pour lesquelles PLUS DE DEUX portages sont ouverts a la fois.
	// En CTF il y a deux drapeaux : trois porteurs simultanes est une INCOHERENCE, et elle est
	// publiee plutot que tue.
	Overlaps int
	// AmbiguousCarrierKills : evenements `flag_carriers_killed` qu'aucun portage ouvert UNIQUE
	// ne permet de rattacher a une victime. Ils ne ferment alors aucun portage.
	AmbiguousCarrierKills int
	// AmbiguousReturns : evenements `flag_returns` survenus alors que zero ou plusieurs drapeaux
	// etaient au sol. Ils ne renvoient alors aucun drapeau a sa base.
	AmbiguousReturns int
	// Spawns est le nombre de socles `flag_spawn` connus de la carte. Zero : la carte est hors
	// du catalogue d'objectifs, tous les portages tombent dans UN drapeau d'equipe -1.
	Spawns int
}

// Balanced verifie l'invariant du calque : toute prise de l'oracle est soit publiee, soit
// rejetee sous une cause NOMMEE. Une somme fausse signale une fuite — un chemin de rejet non
// compte.
func (c FlagCarriesCoverage) Balanced() bool {
	return c.Carries+c.NoBridge+c.NoTrack+c.OutOfWindow == c.Openings
}
