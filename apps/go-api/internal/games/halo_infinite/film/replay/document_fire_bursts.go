package replay

// document_fire_bursts.go — LES RAFALES DE TIR CONTINU, telles que le document les publie (schema
// 71, lot M4b de la campagne « retours rejeu », 2026-09-24).
//
// UNE RAFALE N EST PAS UNE LISTE DE COUPS, et c est la regle de Theater (sonde P1-S3) : le film
// ecrit le DEBUT et la FIN de la gachette tenue, au tick, et rien de l instant de chaque coup — le
// jeu les simule a la cadence du tag. Le document publie donc l intervalle, l arme, le vehicule qui
// la porte et la cadence lue dans le tag ; le client pose les coups (eclairs et sons) a cette
// cadence. Les TROUS de lecture portes par la rafale sont publies : le rejeu s y TAIT (decision de
// l utilisateur du 2026-09-24 — un passage que la lecture n a pas atteint n est jamais « tenu »).

// FireBurst est UNE RAFALE de tir continu publiee.
type FireBurst struct {
	// T0 / T1 : la premiere et la derniere frame de la rafale (axe de `Point.T`). T1 est la frame
	// du lacher lu, du debut du trou ou le lacher a pu avoir lieu, ou de la fin du film ; bornee a
	// la fin de l episode d occupation (arme de vehicule) ou de la vie (arme a pied).
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Slot est le slot du bipede TIREUR : la Track qui le nomme.
	Slot uint32 `json:"slot"`
	// Weapon est la cle de l arme : `VehicleWeaponKey(tag)` pour une arme de vehicule (la cle des
	// tirs de vehicule, celle du registre `vehicleWeapons`), la famille d arme de la dotation
	// (`0x` + 8 chiffres, la cle de `weaponLabels`) pour une arme a pied.
	Weapon string `json:"w"`
	// Vehicle est le slot du VEHICULE qui porte l arme — le PORTEUR pour une piece montee, meme
	// cle que `VehicleTrack.Slot` et que `Shot.Vehicle`. Nil = arme a pied.
	Vehicle *uint32 `json:"v,omitempty"`
	// Rate est la cadence a pleine vitesse, en coups par seconde, tous barillets confondus, lue
	// dans le tag de l arme ; Rate0 et Ramp la montee en cadence d une arme qui en a une (cadence a
	// la pose de la gachette, duree de la montee en secondes) — absentes quand elle est constante.
	Rate  float64 `json:"rate"`
	Rate0 float64 `json:"rate0,omitempty"`
	Ramp  float64 `json:"ramp,omitempty"`
	// Holes sont les passages INTERIEURS que la lecture n a pas atteints, gachette tenue avant et
	// apres : muets.
	Holes []FireBurstHole `json:"holes,omitempty"`
	// StartBound / EndBound disent ce qui borne la rafale : `pressed` (une entree lue pose la
	// gachette apres une entree lue qui ne la posait pas) ou `hole` (la premiere entree lue apres
	// un trou) ; `released`, `hole` (le lacher est dans le trou), `filmEnd`.
	StartBound string `json:"b0"`
	EndBound   string `json:"b1"`
}

// FireBurstHole est un passage muet d une rafale, en frames : [T0, T1).
type FireBurstHole struct {
	T0 int `json:"t0"`
	T1 int `json:"t1"`
}

// ContinuousFireCoverage est la couverture du tir continu : ce que la vue de controle a laisse
// lire (paquets et trous, par cause), et ce que la publication a fait de chaque rafale lue.
//
// LA SOMME EST TENUE : `burstsRead = published + otherInput + noPlayer + ambiguous +
// vehicleNoWeapon + noTrack + weaponUnknown + notContinuous + empty`. Aucune rafale ne disparait
// sans cause.
type ContinuousFireCoverage struct {
	// Packets : paquets delta vus ; Reached : vue C atteinte (vue B close) ; Closed : vue C lue
	// jusqu a son terminateur et paquet clos.
	Packets int `json:"packets"`
	Reached int `json:"reached"`
	Closed  int `json:"closed"`
	// Holes : paquets NON lus, en HoleRuns suites ; puis leurs causes : liste d evenements non
	// localisee, vue B ouverte, debordement, kind 1/2 de la vue C, bloc de 0xbc, plafond
	// d entrees, vue C lue qui ne ferme pas le paquet.
	Holes           int `json:"holes"`
	HoleRuns        int `json:"holeRuns"`
	HolesUnlocated  int `json:"holesUnlocated"`
	HolesOpenViewB  int `json:"holesOpenViewB"`
	HolesOverflow   int `json:"holesOverflow"`
	HolesKind       int `json:"holesKind"`
	HolesBlockBC    int `json:"holesBlockBC"`
	HolesCap        int `json:"holesCap"`
	HolesNotClosing int `json:"holesNotClosing"`
	// Entries : entrees de controle lues ; WithAction : celles qui portent le bloc d action ;
	// Firing : celles qui tiennent une gachette ou un barillet.
	Entries    int `json:"entries"`
	WithAction int `json:"withAction"`
	Firing     int `json:"firing"`
	// BurstsRead : rafales lues (une par bit tenu) ; BurstsWithHole : touchees par un trou ;
	// InnerHoles : passages interieurs muets ; HeldHoleMS : duree de gachette tenue traversee par
	// un trou (inconnue, donc muette).
	BurstsRead     int   `json:"burstsRead"`
	BurstsWithHole int   `json:"burstsWithHole"`
	InnerHoles     int   `json:"innerHoles"`
	HeldHoleMS     int64 `json:"heldHoleMs"`
	// Published : rafales publiees, dont OnVehicle (arme de vehicule) et OnFoot (arme a pied).
	Published int `json:"published"`
	OnVehicle int `json:"onVehicle"`
	OnFoot    int `json:"onFoot"`
	// Les rafales ECARTEES, par cause : un autre bit que la gachette principale de la main 0 ;
	// aucun joueur a cette place ; deux vehicules a la fois ; une monture sans arme a tir continu
	// (klaxon, mortier, siege passager sans arme lue) ; aucune trajectoire publiee du tireur ;
	// l arme en main non lue ; une arme en main qui n est pas a tir continu (charge du pistolet a
	// plasma, du Ravageur) ; une rafale vide une fois bornee a sa monture ou a sa vie.
	OtherInput      int `json:"otherInput"`
	NoPlayer        int `json:"noPlayer"`
	Ambiguous       int `json:"ambiguous"`
	VehicleNoWeapon int `json:"vehicleNoWeapon"`
	NoTrack         int `json:"noTrack"`
	WeaponUnknown   int `json:"weaponUnknown"`
	NotContinuous   int `json:"notContinuous"`
	Empty           int `json:"empty"`
	// ByPlace : rafales dont le tireur a ete trouve par la PLACE (un remplacant, cf.
	// tirs_par_place.go) ; ClippedToMount : bornees a la fin de leur monture ou de leur vie.
	ByPlace        int `json:"byPlace"`
	ClippedToMount int `json:"clippedToMount"`
	// Shots : les coups que le client posera sur les rafales publiees, trous exclus — le CONTROLE
	// du tir continu, a confronter aux sauts du numero de tir des records 36.
	Shots int `json:"shots"`
}

// balanced dit si toute rafale lue a sa case.
func (c ContinuousFireCoverage) balanced() bool {
	return c.BurstsRead == c.Published+c.OtherInput+c.NoPlayer+c.Ambiguous+c.VehicleNoWeapon+
		c.NoTrack+c.WeaponUnknown+c.NotContinuous+c.Empty
}
