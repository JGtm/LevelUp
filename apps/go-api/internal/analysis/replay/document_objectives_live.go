package replay

// document_objectives_live.go — L'OBJECTIF VIVANT : la forme que le DRAPEAU de CTF prend dans
// l'artefact, et ce que la mesure a refuse d'y mettre.
//
// CHRONIQUE — v15 (2026-08-18, plan `.ai/V7.5/replay2d/PLAN_DRAPEAU_OBJET.md`, phase 2). AUCUNE
// CLE NE BOUGE, ET POURTANT LA VERSION MONTE : c'est le CONTENU de `flagCarries` qui change.
// L'OBJET drapeau — le meme archetype `ti=42` que les armes au sol, identifie par le manifeste du
// titre (`[[objective_objects]]`, famille `flag`) — replique sa position quand PERSONNE ne le
// porte. Cette lecture repare deux defauts que le schema 14 declarait explicitement irreparables :
//
//	le LACHER VOLONTAIRE ETAIT NON DATABLE. Un portage que rien ne fermait courait jusqu'a la
//	  fin de l'axe, publie [FlagStateCarriedOpen] — une BORNE HAUTE, pas une mesure. Quand
//	  l'objet REAPPARAIT pendant ce portage AUX PIEDS de son porteur, c'est qu'il ne le porte
//	  plus : le portage se ferme la et devient [FlagStateCarried]. Mesure : 2 portages sur les
//	  trois films du corpus (les deux que `530820e5` portait ouverts).
//	le LACHER ETAIT AU MAUVAIS ENDROIT. [FlagStateDropped] valait la derniere position du
//	  PORTEUR, faute de mieux ; il vaut desormais le dernier point de la piste LIBRE — la ou
//	  l'objet repose apres sa chute. Mesure : 31 / 17 / 4 lachers deplaces.
//
// UN ARTEFACT 14 ET UN 15 DU MEME MATCH PUBLIENT DONC LES MEMES CHAMPS AVEC DES VALEURS ET DES
// INTERVALLES DIFFERENTS — exactement le cas qu'un client ne peut pas distinguer sans la version,
// et la reprise du backfill se fait par `SchemaVersion`.
//
// CE QUI N'EST PAS PUBLIE, ET C'EST LA MOITIE DU RESULTAT : LA PISTE ELLE-MEME. Le controle 3 du
// plan, ecrit AVANT la mesure, exigeait que >= 90 % des vies libres naissent a moins de 1,5 m
// d'un `flag_spawn` ou du porteur qui vient de finir ; la mesure rend 149/197 = 75,6 %. Le temoin
// tient largement (armes ordinaires soumises a la MEME regle : 12,8 %, seuil <= 20 %), donc la
// piste discrimine — d'un facteur six — mais un quart des vies reste inexplique. `flagObjects`
// n'est donc pas publie. LES DEUX CORRECTIONS CI-DESSUS, ELLES, NE TOUCHENT QUE LES VIES NEES AUX
// PIEDS D'UN PORTEUR : la sous-population que ce meme controle VALIDE. Une vie nee a un socle est
// explicitement ecartee, une vie nee ailleurs ne passe pas la distance au porteur.
//
// CE QUE LA MESURE N'A PAS TRANCHE : la cause des 48 vies inexpliquees. Le diagnostic ecarte la
// re-creation sur place (3 cas sur 48) ; le registre des reports porte la condition de reprise.
//
// CHRONIQUE — v14 (2026-08-18, plan `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`,
// phase 1 item 1.3). Le document publie `flagCarries` — LA VIE DE CHAQUE DRAPEAU sur toute la
// partie, en intervalles d'etat — et `coverage.flagCarries`, ses denominateurs. Le champ est
// optionnel, mais la version monte : le drapeau vivant cote client N'EXISTE que si l'artefact le
// porte, et la reprise du backfill se fait par SchemaVersion — un artefact v13 doit se voir
// comme « a re-cuire », pas comme a jour.
//
// LE NUMERO SAUTE DE 11 A 14 POUR CE CALQUE, ET C'EST UNE TRACE DE COORDINATION : la mesure
// (items 1.1 et 1.2) etait prete au schema 12, et la publication a ete REPORTEE parce qu'une
// autre session faisait entrer 12 (`scoreTimeline`) et 13 (`Point.p`) dans la meme branche. Deux
// montees de version concurrentes se seraient marchees dessus.
//
// D'OU VIENT CE QUI EST PUBLIE, ET DE QUOI C'EST FAIT :
//
//	les BORNES        des evenements de statistique NOMMES du statborg, dates a la milliseconde
//	                  (`flag_grabs`, `flag_steals`, `flag_captures`, `flag_returns`) plus le fil
//	                  des morts du film. Aucune estimation, aucune fenetre de tolerance.
//	le PORTEUR        le slot statborg resolu en xuid par le pont par INSTANTS DE MORT
//	                  (`objectiveevents.SlotIdentityResolved`) — le film seul, aucune base.
//	la POSITION       la piste PUBLIEE du porteur a l'instant considere : le drapeau porte EST
//	                  a la position de son porteur. Rien de l'objet n'est decode.
//	le MODE           trois signaux du film qui s'accordent (`objectiveevents.FlagFilmSignals`).
//	le DRAPEAU        le socle `flag_spawn` de la carte, du catalogue versionne d'objectifs,
//	                  joint par `map_id` — jamais par le module ni par le nom public (les deux
//	                  mentent, cf. les decouvertes du plan).
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
//
//	LE RETOUR AUTOMATIQUE d'un drapeau reste au sol. Cherche sur les trois films par l'ecart
//	entre une fin de portage sans reprise et la prise suivante : de 1,3 s a 35,8 s entre p10 et
//	p90 SUR LE MEME FILM, maximum a 111,6 s. Aucune minuterie ne se deduit de cette dispersion,
//	et une minuterie posee la-dessus renverrait a leur base des drapeaux qui sont encore au sol.
//
// Ils vivent dans leur propre fichier pour la meme raison que `document_ground_weapons.go` : la
// FORME publiee ici, la REGLE qui la remplit dans `flag_carries.go`, le CABLAGE dans
// `build_objectives_live.go`.

// FlagCarry est LA VIE D'UN DRAPEAU sur toute la partie : une suite d'intervalles d'etat.
//
// UN DRAPEAU, PAS UN PORTAGE. Le regroupement est par OBJET : en CTF il y a deux drapeaux, donc
// au plus deux entrees. Publier une entree par portage aurait oblige le client a reconstituer
// lui-meme la continuite entre « lache ici » et « repris la ».
type FlagCarry struct {
	// Team est l'equipe PROPRIETAIRE du drapeau, telle que le fichier de carte la donne sur le
	// socle `flag_spawn` ([TeamNeutral] = inconnue : carte absente du catalogue d'objectifs —
	// 72 cartes couvertes sur la centaine jouee).
	Team int `json:"team"`
	// Spans est la vie du drapeau, en intervalles tries par T0, CONTIGUS des lors que le socle de
	// la carte est connu. Carte hors du catalogue d objectifs : les etats `home` sont omis (leur
	// position serait inventee) et la suite peut donc porter des trous.
	Spans []FlagSpan `json:"spans"`
}

// Les QUATRE etats d'un drapeau. Trois disent OU il est ; le quatrieme dit ce qu'on ne sait pas.
const (
	// FlagStateCarried : un joueur le porte, et un FAIT DATE a mis fin a ce portage (capture,
	// mort du porteur, nouvelle prise, `flag_carriers_killed` sans ambiguite). XUID est
	// renseigne.
	FlagStateCarried = "carried"
	// FlagStateCarriedOpen : un joueur l'a pris, et RIEN dans le film ne dit qu'il l'a lache.
	// L'intervalle court alors jusqu'a la fin de l'axe de temps, et c'est une BORNE HAUTE, pas
	// une mesure.
	//
	// POURQUOI CET ETAT EXISTE, ET POURQUOI IL NE S'APPELLE PAS `carried`. Le LACHER VOLONTAIRE
	// n'est date par aucune chaine (cf. flag_carries.go) : un portage qui en contient un est
	// trop long, et rien dans sa propre chaine ne le dirait. La mesure le CHIFFRE — le controle
	// du marqueur confirme 37/37 des portages FERMES et 0/5 des portages ouverts, sur les trois
	// films CTF du corpus. Les confondre publierait le doute sous le meme nom que la certitude ;
	// le client peut les dessiner differemment, ou taire les seconds.
	FlagStateCarriedOpen = "carried_open"
	// FlagStateDropped : il est au sol, a l'endroit ou son dernier porteur l'a laisse. L'etat
	// court jusqu'a sa reprise, un `flag_returns` ou la fin du match — jamais une minuterie de
	// retour automatique, qui ne se deduit d'aucune mesure (cf. l'en-tete de ce fichier).
	FlagStateDropped = "dropped"
	// FlagStateHome : il est a sa base, le socle `flag_spawn` du catalogue de carte.
	FlagStateHome = "home"
)

// FlagReturnZone est LA REGLE DE RETOUR du drapeau, telle que le manifeste du titre la donne
// (schema 29). Elle ne decrit PAS ce match-ci : elle decrit le MODE, et c'est pour cela qu'elle
// est publiee une fois et non par lacher.
//
// LE MODELE, ET D'OU IL VIENT. Le jeu remplit une jauge de retour au taux `1/reset + H(n)/solo`,
// ou `n` est le nombre de defenseurs dans la zone et `H` la SERIE HARMONIQUE — son propre script
// nomme la fonction `CalculateReturnRateHarmonic`. Deux defenseurs valent donc 1 + 1/2, trois
// 1 + 1/2 + 1/3 : le rendement decroit, il n'est jamais lineaire.
//
// CE QUE LE CLIENT EN FAIT, ET POURQUOI CE N'EST PAS CALCULE ICI. Le modele donne la FORME de la
// jauge ; ses BORNES viennent de l'observation (le lacher, puis le retour date). Mais compter les
// defenseurs exige de savoir a quelle equipe appartient chaque joueur — et l'equipe N'EST PAS
// DANS LE FILM (cf. Track.Team). Le constructeur du rejeu est hors ligne et n'ouvre aucune base ;
// le client, lui, a deja joint le tableau de bord pour colorer les camps. C'est donc lui qui
// compte, et cette table est ce qu'il lui faut pour le faire.
// LA CONTESTATION EN FAIT PARTIE, et elle a son propre rayon. Le jeu surveille DEUX cylindres
// autour du drapeau tombe : l'INTERIEUR, ou les coequipiers du proprietaire le renvoient, et
// l'EXTERIEUR, ou un ENNEMI du proprietaire le CONTESTE (`GetAnyEnemyTeamInOuterArea`, etats
// `Contested` / `ContestedRefilling`). Les deux rayons sont egaux dans la configuration du jeu ;
// ce sont les HAUTEURS des cylindres qui different (`cylinderInnerHeight` / `cylinderOuterHeight`)
// — un ennemi sur une plateforme au-dessus du drapeau conteste sans pouvoir le prendre. Le rejeu
// est en 2D : il publie les deux rayons separement quand meme, parce que ce sont deux REGLES, et
// qu'une valeur unique se ferait passer pour une coincidence necessaire.
type FlagReturnZone struct {
	// RadiusM est le rayon de la zone de RETOUR, dans les MEMES coordonnees que `FlagSpan.X/Y`.
	RadiusM float32 `json:"radiusM"`
	// ContestRadiusM est le rayon de la zone de CONTESTATION.
	ContestRadiusM float32 `json:"contestRadiusM"`
	// ResetSeconds est la duree qu'un drapeau au sol met a rentrer TOUT SEUL.
	ResetSeconds float32 `json:"resetSeconds"`
	// SoloSeconds est la duree qu'il met avec UN defenseur dans la zone.
	SoloSeconds float32 `json:"soloSeconds"`
}

// FlagSpan est UN intervalle d'etat du drapeau.
type FlagSpan struct {
	// State vaut [FlagStateCarried], [FlagStateCarriedOpen], [FlagStateDropped] ou
	// [FlagStateHome].
	State string `json:"state"`
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// XUID est le PORTEUR, en decimal — renseigne pour les deux etats portes, `null` pour les
	// deux autres. POINTEUR ET SANS `omitempty` : le champ doit se VOIR a `null`, sinon « pas de
	// porteur » et « artefact plus ancien » se confondent (meme regle que `PadPickup.XUID`).
	XUID *string `json:"xuid"`
	// X / Y : la position du drapeau en coordonnees monde (memes axes que Point.X/Y).
	//
	// POUR UN ETAT PORTE, C'EST LE POINT DE PRISE, ET LA SUITE SE LIT SUR LA PISTE DU PORTEUR.
	// Republier la trajectoire du drapeau serait republier celle de son porteur : le client
	// joint par XUID et suit la piste deja publiee. Pour `dropped`, c'est le dernier point connu
	// du porteur ; pour `home`, le socle `flag_spawn`.
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// FlagCarriesCoverage porte les denominateurs du calque. Sans eux, « 12 portages » se lirait
// comme une exhaustivite, et un film CTF sans aucun portage publie serait indistinguable d'un
// film qui n'est pas du CTF.
//
// ELLE EST PUBLIEE MEME QUAND AUCUN DRAPEAU NE L'EST, pour la meme raison que `placements` et
// `groundWeapons` : un film d'un autre mode, un film CTF ou personne ne capture et un film dont
// le pont n'a nomme personne rendent tous trois zero portage — seuls ces compteurs les
// distinguent. Son ABSENCE dit encore autre chose : l'appelant n'a rien fourni a lire.
type FlagCarriesCoverage struct {
	// FlagFilm dit si le film a ete RECONNU comme une partie de CTF par l'accord des trois
	// signaux (cf. `objectiveevents.FlagFilmSignals`). Faux : tout le reste vaut zero, et c'est
	// le cas nominal de tous les autres modes.
	FlagFilm bool `json:"flagFilm"`
	// Bursts / Captures / Steals : les trois signaux qui ont fonde ce verdict, publies pour
	// qu'il se verifie.
	Bursts   int `json:"bursts"`
	Captures int `json:"captures"`
	Steals   int `json:"steals"`
	// Openings est le nombre de PRISES de l'oracle (`flag_grabs` + `flag_steals`) une fois les
	// emissions jumelles fusionnees : le denominateur de tout ce qui suit.
	Openings int `json:"openings"`
	// Carries est le nombre de portages effectivement publies.
	Carries int `json:"carries"`
	// Closed / Open partagent ces portages en deux populations qui ne valent PAS la meme chose :
	// ceux qu'un fait date a fermes, et ceux que rien ne ferme — publies en
	// [FlagStateCarriedOpen], borne haute a la fin de l'axe.
	Closed int `json:"closed"`
	Open   int `json:"open"`
	// NoBridge : prises dont le slot statborg n'a pas ete resolu en xuid. Le pont se tait
	// plutot que d'attribuer le drapeau au mauvais joueur.
	NoBridge int `json:"noBridge"`
	// NoTrack : le porteur est nomme, mais aucune trajectoire publiee ne couvre l'instant de la
	// prise — le drapeau n'aurait pas de position a dessiner.
	NoTrack int `json:"noTrack"`
	// OutOfWindow : la prise tombe hors de l'axe de temps publie (fins de partie que le film
	// prolonge au-dela de la derniere position rendue).
	OutOfWindow int `json:"outOfWindow"`
	// MarkerObserved / MarkerConfirmed : le CONTROLE INDEPENDANT, SUR LES SEULS PORTAGES FERMES.
	// MarkerObserved compte ceux qui contiennent au moins une image-cle (le denominateur : sans
	// image-cle, le marqueur ne peut rien confirmer) ; MarkerConfirmed ceux dont au moins une
	// image-cle porte le marqueur sur le slot du porteur.
	//
	// POURQUOI LES FERMES SEULS. Un portage ouvert est trop long PAR CONSTRUCTION (le lacher
	// volontaire n'est date par rien) : ses images-cles tardives tombent apres que le drapeau a
	// ete lache, et aucune ne porte le marqueur. Les melanger ferait baisser un taux qui mesure
	// la justesse des bornes — la mesure du 2026-08-18 le chiffre exactement : 37/37 sur les
	// fermes, 37/42 en melangeant.
	//
	// LES DEUX CHAINES SONT DISJOINTES : les bornes viennent des compteurs de statistique du
	// statborg, le marqueur d'une suite de bits du record de bipede des images-cles. Leur accord
	// est donc une preuve, pas une tautologie.
	MarkerObserved  int `json:"markerObserved"`
	MarkerConfirmed int `json:"markerConfirmed"`
	// OpenObserved / OpenConfirmed : les MEMES deux comptes sur les portages OUVERTS. Ils sont
	// publies pour que rien ne soit tu : le taux « tous portages confondus » reste calculable,
	// et l'ecart entre les deux populations se voit.
	OpenObserved  int `json:"openObserved"`
	OpenConfirmed int `json:"openConfirmed"`
	// Overlaps compte les prises pour lesquelles PLUS DE DEUX portages sont ouverts a la fois.
	// En CTF il y a deux drapeaux : trois porteurs simultanes est une INCOHERENCE, et elle est
	// publiee plutot que tue.
	Overlaps int `json:"overlaps"`
	// ClosedOverlaps compte les memes depassements EN NE REGARDANT QUE LES PORTAGES FERMES.
	//
	// C'EST LUI QUI JUGE, ET LA DISTINCTION EST LE RESULTAT D'UNE MESURE. Le plan attendait que
	// le pont par instants de mort leve les depassements de `64e8adfa` ; il ne les a pas leves
	// (12 avec la regle de production, sur un film ou plus AUCUNE prise n'est sans pont). La
	// cause n'etait donc pas l'identite mais la DUREE des portages que rien ne ferme. Un
	// `Overlaps` non nul avec `ClosedOverlaps` a zero est donc explique — c'est l'incertitude
	// deja publiee comme telle ; un `ClosedOverlaps` non nul serait une contradiction entre
	// faits dates, et il se lit ici.
	ClosedOverlaps int `json:"closedOverlaps"`
	// AmbiguousCarrierKills : evenements `flag_carriers_killed` qu'aucun portage ouvert UNIQUE
	// ne permet de rattacher a une victime. Ils ne ferment alors aucun portage.
	AmbiguousCarrierKills int `json:"ambiguousCarrierKills"`
	// AmbiguousReturns : evenements `flag_returns` survenus alors que zero ou plusieurs drapeaux
	// etaient au sol. Ils ne renvoient alors aucun drapeau a sa base.
	AmbiguousReturns int `json:"ambiguousReturns"`
	// HomeByObject compte les drapeaux ramenes chez eux par la RENTREE DE L'OBJET — une vie libre
	// nee a leur socle alors qu'ils etaient au sol (cf. `flagObjectHomecomings`).
	//
	// C'EST LE COMPTE DES RETOURS AUTOMATIQUES, ceux que personne ne provoque et qu'aucun
	// compteur du statborg ne credite. Avant le schema 29 ils n'existaient pas et les laches
	// couraient jusqu'a la reprise ou la fin de l'axe — des etats `dropped` de plus de deux
	// minutes, qui n'ont jamais existe a l'ecran. Un retour DEJA credite ne s'y compte pas : la
	// rentree ne fait alors rien.
	HomeByObject int `json:"homeByObject"`
	// AmbiguousHomecomings : rentrees ecartees parce qu'un AUTRE drapeau gisait au point de
	// naissance — rien ne dit lequel des deux vient d'etre recree.
	AmbiguousHomecomings int `json:"ambiguousHomecomings"`
	// NeutralFlag dit que la partie a ete reconnue « DRAPEAU NEUTRE » : un seul drapeau, au socle
	// du centre, que les deux camps se disputent. Le mode n'est PAS dans le film — c'est l'OBJET
	// qui tranche, par le socle ou il renait (cf. flag_neutral.go).
	NeutralFlag bool `json:"neutralFlag"`
	// NeutralBirths / TeamBirths sont les deux comptes qui FONDENT ce verdict : les naissances de
	// l'objet au socle neutre, et celles aux socles d'equipe. Publies pour que le verdict se
	// verifie au lieu de se croire.
	NeutralBirths int `json:"neutralBirths"`
	TeamBirths    int `json:"teamBirths"`
	// Spawns est le nombre de socles `flag_spawn` connus de la carte. Zero : la carte est hors
	// du catalogue d'objectifs, tous les portages tombent dans UN drapeau d'equipe -1.
	Spawns int `json:"spawns"`
	// ObjectLives est le nombre de VIES LIBRES de l'objet drapeau LUES sur ce film (schema 15).
	// C'est le DENOMINATEUR des deux compteurs suivants : sans lui, « 2 portages fermes » ne se
	// juge pas. La PISTE elle-meme n'est pas publiee — son controle de provenance l'a refusee
	// (149/197 = 75,6 % contre 90 % exiges) ; seules les vies nees AUX PIEDS D'UN PORTEUR, la
	// sous-population que ce controle valide, servent aux corrections ci-dessous.
	ObjectLives int `json:"objectLives"`
	// ClosedByObject compte les portages que RIEN NE FERMAIT et qu'une vie libre a fermes — le
	// LACHER VOLONTAIRE, enfin date.
	//
	// C'EST LA MESURE DU LOT, ET ELLE SE LIT CONTRE `Open` : ces portages-la etaient publies
	// [FlagStateCarriedOpen], c'est-a-dire trop longs par construction. Chacun qui bascule est un
	// drapeau qu'on cesse de dessiner dans une main qui ne le tient plus.
	ClosedByObject int `json:"closedByObject"`
	// DropsRepositioned compte les etats [FlagStateDropped] dont la position vient desormais de
	// la piste LIBRE et non plus de la derniere position du porteur. L'ecart n'est pas
	// cosmetique : un drapeau tombe rebondit, et le porteur meurt rarement la ou l'objet se pose.
	DropsRepositioned int `json:"dropsRepositioned"`
}

// Balanced verifie les DEUX invariants du calque : toute prise de l'oracle est soit publiee, soit
// rejetee sous une cause NOMMEE, et tout portage publie est soit ferme, soit ouvert. Une somme
// fausse signale une fuite — un chemin de rejet non compte, ou une population qui echappe au
// partage.
func (c FlagCarriesCoverage) Balanced() bool {
	return c.Carries+c.NoBridge+c.NoTrack+c.OutOfWindow == c.Openings &&
		c.Closed+c.Open == c.Carries
}
