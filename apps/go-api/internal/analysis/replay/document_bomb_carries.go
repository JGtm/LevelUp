package replay

// document_bomb_carries.go — LE PORTEUR DE LA BOMBE D'ASSAUT : la forme qu'il prend dans
// l'artefact.
//
// CHRONIQUE — v30 (2026-09-01, lot BOMBE VISIBLE). Le document publie `bombCarries` — LES
// PERIODES DE PORTAGE DE LA BOMBE, en intervalles de frames, chacun nomme par le xuid du
// porteur. Un champ de document, un schema de plus (`BombCarry`) et un bloc de couverture
// (`BombCarriesCoverage`) — le patron exact de `skullCarries` (v23).
//
// SOURCE, TOUTE DANS LE FILM : la bombe est un OBJET TENU — le moteur la replique dans le
// composant weapon-state-type-info du bipede, exactement comme une arme
// (`filmdec.ScanFilmHeldWeaponChanges`, le MEME balayage que `weaponChanges`). Sa famille est
// `0x3fee4fcf` (B1 2026-09-01 : unique candidate hors catalogue d'armes des 9 films d'Assaut,
// prise et lachee sur chacun ; l'atlas HUD la nomme independamment « ball | bomb »,
// sprite contour-34). PRISE = transition VERS la famille ; LACHER = transition DEPUIS ; la
// MORT du porteur ferme SANS emission (le canal ne lache rien quand la vie du bipede
// s'arrete — piege mesure, ferme par le fil des morts via `BuildHeldObjectCarry`).
//
// CE QUI A ETE MESURE (bombe_b2_chronologie_test.go, 2026-09-01), ecrit ici parce que c'est
// ce qu'un lecteur doit trouver sur place : temoin Oddball 46/46 heartbeats de possession
// dans une periode pontee du meme joueur (100 %) ; porteur a la pose = detonateur du statborg
// 13/17 (les 4 desaccords instruits par la position, B3 : 3 penchent CANAL, 1 indecis) ;
// bombe posee portee par personne 27/28 intervalles (96,4 %). Le delai median lacher ->
// explosion vaut 4 804 ms : le LACHER du canal EST le geste de pose, a ~130 ms de la meche.
//
// LA GARDE DE MODE COUVRE TOUTES LES VARIANTES D'ASSAUT, One Bomb comprise — et c'est LA
// DIFFERENCE avec `bombArmings` (v29) : ce que One Bomb refute est le canal de l'ANNEAU
// D'ARMEMENT (CV 0,725), pas le canal des armes tenues, qui reste le meme composant de
// bipede dans toutes les variantes. La garde est chez l'APPELANT (`replaybuild`,
// `ObjectiveTypeOf` == bomb), comme le crane et la couronne : ce paquet ne devine aucun mode.
//
// LA BOMBE AU SOL N'EST PAS PUBLIEE, et c'est une decision de mesure, pas un oubli. Le crane
// libre a son canal MESURE (`objectiveObjects`, archétype ti=42) ; l'objet bombe au sol n'a
// pas ete etabli dans ce canal. Entre un lacher et la prise suivante, la bombe est immobile
// au DERNIER POINT de son lacheur — le client le derive des periodes publiees et des pistes
// deja publiees (meme regle que le drapeau `dropped`), sans qu'aucune position inventee
// n'entre dans l'artefact.
//
// POURQUOI LA VERSION MONTE ALORS QUE LE CHAMP EST OPTIONNEL : la reprise du backfill se fait
// par `SchemaVersion`, et un artefact 29 doit se lire « a re-cuire », pas « a jour » — sans
// quoi aucun rejeu d'Assaut deja cuit ne montrerait jamais la bombe. Meme raison que les
// montees v14 (drapeau), v22 (couronne), v23 (crane) et v29 (armement).

// BombCarry est UNE periode de portage de la bombe : un joueur, un intervalle de frames.
//
// PAS DE POSITION. La bombe portee est TOUJOURS sur le joueur qui la porte : le client joint
// par `xuid` et la pose sur la piste deja publiee, exactement comme le crane d'Oddball et la
// couronne VIP. Republier une trajectoire de bombe serait republier celle de son porteur.
type BombCarry struct {
	// XUID est le porteur de la bombe, en decimal.
	XUID string `json:"xuid"`
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS. Ils datent
	// la PRISE et le premier de : lacher, mort du porteur, fin du film — le canal date les
	// transitions a la milliseconde du paquet, sans le retard d'un train de tics.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Closed dit qu'un FAIT a mis fin au portage AVANT la fin du rejeu (un lacher — souvent la
	// POSE elle-meme — ou la mort du porteur). Faux : rien ne ferme la periode avant la fin de
	// l'axe — le film s'arrete pendant le portage, une BORNE HAUTE.
	Closed bool `json:"closed"`
}

// BombCarriesCoverage porte les denominateurs du calque. Sans eux, « 5 portages » se lirait
// comme une exhaustivite, et un film d'Assaut sans aucun portage publie serait indistinguable
// d'un film d'un autre mode. ELLE EST PUBLIEE MEME QUAND AUCUN PORTAGE NE L'EST ; son ABSENCE
// dit encore autre chose : l'appelant n'a pas reconnu un film d'Assaut.
type BombCarriesCoverage struct {
	// BombFilm dit que l'appelant a reconnu un film d'Assaut et fourni de quoi lire. Toujours
	// vrai quand ce bloc existe (l'absence du bloc EST le « pas un film d'Assaut »).
	BombFilm bool `json:"bombFilm"`
	// Events est le nombre de transitions de la famille bombe lues dans le canal des armes
	// tenues (prises + lachers confondus), avant toute reconstruction.
	Events int `json:"events"`
	// Periods est le nombre de periodes de portage reconstruites, avant rejet — le
	// denominateur de `Carries`, `NoBridge` et `OutOfWindow`.
	Periods int `json:"periods"`
	// Carries est le nombre de portages effectivement publies.
	Carries int `json:"carries"`
	// Closed / Open partagent ces portages : ceux qu'un fait a fermes (lacher ou mort), et
	// ceux que rien ne ferme (borne haute a la fin de l'axe).
	Closed int `json:"closed"`
	Open   int `json:"open"`
	// ByDeath compte, parmi les fermes, ceux que la MORT du porteur a fermes : le canal
	// n'emet aucun lacher a la mort, la fermeture vient du fil des morts. Publie parce que
	// c'est la moitie mesuree du protocole — sans lui, un lecteur croirait le canal exhaustif.
	ByDeath int `json:"byDeath"`
	// NoBridge : periodes dont le slot de bipede n'a pas ete resolu en xuid. Le pont se tait
	// plutot que de poser la bombe sur le mauvais joueur.
	NoBridge int `json:"noBridge"`
	// OutOfWindow : periodes dont le debut tombe hors de l'axe de frames publie.
	OutOfWindow int `json:"outOfWindow"`
	// CarrierAbsent : periodes dont le porteur ponte n'est PAS present sur la carte (aucune
	// vie bipede publiee ne couvre l'intervalle). Un tel portage n'a AUCUNE position ou poser
	// la bombe : on l'ecarte plutot que de faire disparaitre l'icone — meme regle que le crane.
	CarrierAbsent int `json:"carrierAbsent"`
}

// Balanced verifie l'invariant : toute periode est publiee ou rejetee sous une cause NOMMEE,
// et tout portage publie est ferme ou ouvert.
func (c BombCarriesCoverage) Balanced() bool {
	return c.Carries+c.NoBridge+c.OutOfWindow+c.CarrierAbsent == c.Periods &&
		c.Closed+c.Open == c.Carries
}
