package replay

// tir_continu_armes.go — LES ARMES A TIR CONTINU DU TITRE : quel chassis les porte, et leur CADENCE
// lue dans le tag (lot M4b de la campagne « retours rejeu », 2026-09-24).
//
// # POURQUOI UNE TABLE, ET D OU VIENT CHAQUE VALEUR
//
// Le film n ecrit NI l arme d une rafale NI l instant de ses coups : la vue de controle porte la
// gachette tenue et l index du joueur (sonde P1-S3, `internal/grammar/tir_continu.go`). Theater
// simule les coups a la cadence du TAG de l arme. Le serveur ne lit aucun fichier de jeu (decision
// de cadrage du 2026-09-02, `vehicle_families.go`) : la resolution vit donc ici, valeur par valeur,
// comme les chassis et les pieces montees.
//
// L INSTRUMENT qui a lu chaque valeur, dans les modules installes et en lecture seule :
// `internal/himodule/m4b_barillets_research_test.go` (2026-09-24). Il lit, par tag `weap`, le
// tableau `barrels` (element de 928 octets au champ +3284, le meme sur toutes les armes lues) :
// « rounds per second » (+4, min et max), « acceleration time » (+0x10), le TYPE DE PREDICTION
// (+0x70, court) et le debit d evenements synchronises (+0x74) — les deux champs que
// `FUN_140de87fc` lit pour decider l emission d un record de tir (sonde P1-S3). Et, par tag `vehi`,
// les armes que le CHASSIS declare dans sa table de dependances.
//
// UNE ARME N ENTRE ICI QUE SI SON TIR NE S ECRIT PAS : type de prediction 1 et debit NUL, sur tous
// ses barillets. Son record 36 n est alors jamais emis, et la vue de controle est son seul canal.
// Une arme a debit non nul (MA40 AR : type 1, debit 12/s) ecrit ses coups ; la simuler en plus les
// doublerait.
//
// # CE QUE LA TABLE NE DIT PAS, ET QUI EST COMPTE
//
//   - la CONFIGURATION MULTIJOUEUR du vehicule. Les tables `MPVehicleConfigs` (vcdd -> sofd -> sofa
//     -> uwfa -> weap) declarent d autres tags : le Ghost a trois armes de configuration
//     (`c2a1a279` 2 x 3,75/s — la cadence de la table —, `6a9c28a2` 4,5 + 5,0/s, `d1d1aa79`
//     2 x 1,7/s), la Banshee `a0756835` (2 x 4,0/s, la cadence de la table), le Chopper
//     `7d7d2ebf` (2 x 4,0/s, le DOUBLE de la table). Le film ne dit pas laquelle est montee ; la
//     table retient l arme que le CHASSIS ECRIT DANS LE FILM declare. Le controle par le numero de
//     tir (`m4b_compteur_research_test.go`) borne l ecart par le haut ;
//   - les armes a tir continu qui ne sont PAS un tir : le klaxon du Warthog (`0000a4bc`) et celui du
//     Falcon (`82a69ed0`), 60/s, type 1 — une rafale du conducteur d un Warthog est un coup de
//     klaxon. Leur chassis n est pas dans la table : la rafale est comptee `vehicleNoWeapon`, jamais
//     dessinee en tir ;
//   - les tourelles fixes de la carte (`3a8060e2`) : DEUX armes continues declarees par le meme
//     chassis, sans rien qui dise laquelle tire — absentes, comptees `vehicleNoWeapon`.

// continuousWeapon est ce que le tag d une arme a tir continu dit de sa cadence.
type continuousWeapon struct {
	// rate : coups par seconde a pleine cadence, TOUS barillets confondus (somme des maxima).
	rate float64
	// rate0 : la cadence a la pose de la gachette quand l arme MONTE en cadence (somme des
	// minima) ; zero quand la cadence est constante.
	rate0 float64
	// ramp : la duree de la montee, en secondes (« acceleration time » du barillet).
	ramp float64
}

// continuousWeaponsByTag : les armes a tir continu, par tag `weap` (cf. l en-tete).
var continuousWeaponsByTag = map[uint32]continuousWeapon{
	0x00015435: {rate: 7.5},                     // Ghost : 2 barillets x 3,75/s
	0x0000aa68: {rate: 8},                       // canons de la Banshee : 2 x 4,0/s
	0xb40e9618: {rate: 4},                       // canons du Chopper : 2 x 2,0/s
	0xd3c407ed: {rate: 10},                      // LMG du Wasp : 10/s (acceleration 2 s, min = max)
	0x0c6fd911: {rate: 18, rate0: 5, ramp: 1.6}, // LAAG du Warthog : 5 -> 18/s en 1,6 s
	0xe2066f44: {rate: 13},                      // LMG de la tourelle du Falcon : 13/s
	0x00015cd3: {rate: 13},                      // mitrailleuse du collier du Scorpion : 13/s
	0x001b33e8: {rate: 12, rate0: 6, ramp: 0.4}, // tourelle plasma du Wraith : 6 -> 12/s en 0,4 s
	0xa0955e9e: {rate: 60},                      // Rayon de Sentinelle (a pied) : 60/s
}

// continuousWeaponByChassis : l arme a tir continu que le chassis (ou la piece montee) DECLARE
// dans sa table de dependances — lue par l instrument de l en-tete. N y figurent que les montures
// a UN operateur de l arme : le pilote d un chassis monoplace, l artilleur d une piece montee.
var continuousWeaponByChassis = map[uint32]uint32{
	0x5b80c406: 0x00015435, // Ghost (identifiant observe au parc)
	0x0000d3dc: 0x00015435, // Ghost (identifiant de base)
	0x9af9e693: 0x00015435, // Ghost
	0xc6e79dcc: 0x0000aa68, // Banshee (observe) — declare aussi la bombe 0000aa69, a coup
	0x000026ed: 0x0000aa68, // Banshee (base)
	0x0001530a: 0x0000aa68, // Banshee
	0x3d4a8a5a: 0xb40e9618, // Chopper (observe)
	0x002ba902: 0xb40e9618, // Chopper (base)
	0xb65b3b4a: 0xd3c407ed, // Wasp — declare aussi les missiles 11725dc4, a coup
	0xdd7f9102: 0x0c6fd911, // LAAG, piece montee du Warthog
	0xf4c45d71: 0xe2066f44, // tourelle LMG du Falcon (banque falconlmgturret)
	0x0000d500: 0x00015cd3, // collier de tourelle du Scorpion
	0x001b33fc: 0x001b33e8, // tourelle plasma du Wraith
}

// continuousWeaponOfChassis rend l arme a tir continu d un chassis ecrit en hexadecimal
// (`VehicleTrack.Chassis`), ou faux.
func continuousWeaponOfChassis(chassis string) (uint32, bool) {
	id, ok := parseHex32(chassis)
	if !ok {
		return 0, false
	}
	tag, ok := continuousWeaponByChassis[id]
	return tag, ok
}
