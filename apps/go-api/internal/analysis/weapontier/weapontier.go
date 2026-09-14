// Package weapontier décide À QUEL NIVEAU se lit une arme prise sur un socle : arme de
// BASE, arme de TERRAIN, arme de PUISSANCE, ou non classée.
//
// POURQUOI CE PAQUET EXISTE. Le bloc « contrôle des armes » comptait jusqu'ici des prises de
// socle toutes égales entre elles : ramasser un fusil d'assaut posé sur un râtelier et
// ramasser le lance-roquettes du socle central y pesaient pareil. Ce n'est pas ce qu'un match
// raconte. Le niveau remet chaque prise à sa place, et il se MESURE — il ne s'écrit pas dans
// une liste d'armes.
//
// LES TROIS SOURCES, ET AUCUNE N'EST UN NOM D'ARME :
//
//	BASE       l'arme est dans l'équipement de DÉPART d'une vie du match (canal `loadouts`
//	           du film, PREMIÈRE émission de chaque slot). Donc BR75/Bandit en classé,
//	           AR+Sidekick en partie rapide, sans qu'aucune liste ne le déclare.
//	TERRAIN    l'emplacement de la carte qui confirme le socle est un RÂTELIER (`rack`).
//	PUISSANCE  cet emplacement est un SOCLE DE PUISSANCE (`power`).
//	NON CLASSÉ aucun emplacement ne confirme le socle (carte hors référence, ou socle hors
//	           du rayon d'un mètre). Il reste VISIBLE avec son compte — jamais fondu dans un
//	           autre niveau (décision D4 du plan).
//
// LA NATURE D'UN EMPLACEMENT VIENT DE LA CARTE, JAMAIS DE L'ARME (décision D1). Mesure de
// l'étape 0 sur 76 artefacts : 70 socles sur 669 portent une arme de rôle « lourd » (Hydra,
// Needler, Sentinel Beam, Shock Rifle) sur un RÂTELIER, et c'est nominal. Brancher le niveau
// sur le rôle du registre canonique produirait donc 10 % de faux niveaux.
//
// PUR : aucune base, aucun HTTP, aucun log, aucune langue. Le paquet compte et classe ; c'est
// l'appelant qui journalise (cf. [CrossCheck]) et l'écran qui nomme.
package weapontier

// Tier est le niveau de lecture d'une arme de socle.
type Tier string

const (
	// TierBase — arme présente dans un équipement de départ du match.
	TierBase Tier = "base"
	// TierGround — arme apparue sur un râtelier de la carte.
	TierGround Tier = "ground"
	// TierPower — arme apparue sur un socle de puissance.
	TierPower Tier = "power"
	// TierPowerup — socle de BONUS (surbouclier, camouflage). Ce n'est pas une arme, et ce
	// niveau existe pour que le bloc puisse l'écarter SANS tester un préfixe de nom : le
	// compte des bonus vit à part depuis toujours (`powerupOccupations`), et ce paquet ne
	// change pas cette frontière, il la nomme.
	TierPowerup Tier = "powerup"
	// TierUnclassified — socle qu'aucun emplacement de la carte ne confirme.
	TierUnclassified Tier = "unclassified"
)

// Familles d'emplacement telles que la référence des cartes les écrit
// (`MapWeaponPadSpot.Family`, dérivée du `type_id` Forge).
const (
	familyRack    = "rack"
	familyPower   = "power"
	familyPowerup = "powerup"
)

// BaseShareMin est la part minimale des vies du match qu'une arme doit occuper au départ pour
// être tenue pour une arme de BASE.
//
// POURQUOI UN SEUIL, ET POURQUOI CELUI-LÀ. Mesure de l'étape 0 (4 406 vies en classé / partie
// rapide) : les trois armes de départ réelles pèsent 94,4 % des équipements de première
// émission (MA40 45,5 %, Sidekick 29,5 %, BR75 19,4 %), et une queue de 5,6 % porte des armes
// qui n'en sont pas — Skewer, Sniper, Épée — parce que la première émission publiée d'une vie
// arrive parfois après un ramassage. Sans seuil, un Skewer vu une fois au départ d'une vie
// ferait passer TOUTES ses prises de socle en « base » et viderait le niveau « puissance » de
// son sens. À 5 %, la queue tombe entièrement et les trois vraies armes de base passent avec
// quatre à neuf fois la marge.
//
// Ce seuil ne s'applique QU'À la promotion en « base » : il ne retire jamais une prise du
// tableau, elle retombe simplement sur le niveau de son emplacement.
const BaseShareMin = 0.05

// Pad est UN socle du match, reduit a ce que le niveau demande : la famille d'arme qui s'y
// trouve. Sa POSITION dans la tranche est son identite — c'est l'index que [Spot.Pad] cite et
// que [Match.TierOf] recoit.
type Pad struct {
	Weapon string
}

// Spot est UN emplacement de la carte CONFIRME par un socle du match : la nature que le fichier
// de carte lui donne, et l'index du socle qu'elle qualifie. Un socle qu'aucun Spot ne cite n'est
// pas confirme — son niveau sera « non classe ».
type Spot struct {
	Pad int
	// Family : `rack`, `power` ou `powerup`, tels que la reference des cartes les ecrit.
	Family string
}

// Spawn est UNE emission d'equipement de depart : la vie concernee et les armes en main.
//
// L'ORDRE DE LA TRANCHE COMPTE : seule la PREMIERE emission de chaque Slot est retenue (cf.
// [baseWeaponsOf]). L'appelant passe le canal dans l'ordre du film, il ne le trie pas.
type Spawn struct {
	Slot    uint32
	Weapons []string
}

// Match porte ce qu'un match apprend une fois pour toutes : la nature de chaque socle et les
// armes de départ. Le construire coûte une passe ; classer une prise coûte ensuite un accès.
//
// Le zéro est UTILISABLE : tous les socles y sont non classés et aucune arme n'est de base.
// C'est exactement ce qu'un match sans référence de carte et sans canal `loadouts` doit
// donner.
type Match struct {
	// familyByPad est indexé par l'index de socle de `weaponPads` ; "" = non confirmé.
	familyByPad []string
	// baseWeapons : familles d'arme (hexadécimal, même écriture que `WeaponPad.Weapon`)
	// retenues comme armes de départ du match.
	baseWeapons map[string]bool
	// randomStarts — le mode distribue des équipements de départ ALÉATOIRES (Fiesta et
	// consorts). Le niveau « base » n'y a aucun sens et n'est jamais attribué.
	randomStarts bool
	// lives est le nombre de vies dont un équipement de départ a été lu — le DÉNOMINATEUR de
	// BaseShareMin, et rien d'autre. Il n'est pas exposé : ce que l'écran affiche vient des
	// compteurs du bloc servi, pas d'un accesseur de ce paquet (0 code mort, revue 2026-09-14).
	lives int
}

// NewMatch assemble le classement d'un match.
//
// `randomStarts` est REÇU, jamais déduit : c'est la catégorie de mode qui le sait
// (Fiesta / Super Fiesta / Husky Raid), et ce paquet ne lit ni slug ni nom de mode.
func NewMatch(pads []Pad, spots []Spot, spawns []Spawn, randomStarts bool) Match {
	m := Match{familyByPad: make([]string, len(pads)), randomStarts: randomStarts}
	for _, spot := range spots {
		if spot.Pad >= 0 && spot.Pad < len(m.familyByPad) {
			m.familyByPad[spot.Pad] = spot.Family
		}
	}
	if !randomStarts {
		m.baseWeapons, m.lives = baseWeaponsOf(spawns)
	} else {
		_, m.lives = baseWeaponsOf(spawns)
	}
	return m
}

// FamilyOfPad rend la famille d'emplacement d'un socle du match ("" si non confirmé).
func (m Match) FamilyOfPad(padIndex int) string {
	if padIndex < 0 || padIndex >= len(m.familyByPad) {
		return ""
	}
	return m.familyByPad[padIndex]
}

// TierOf classe UNE prise : le socle où elle a eu lieu, et l'arme qui s'y trouvait.
//
// L'ORDRE DES TESTS EST LA RÈGLE DU PLAN : base > terrain > puissance > non classé. Une arme
// de départ ramassée sur un râtelier se lit « base » — c'est le niveau d'information voulu :
// reprendre son fusil d'assaut n'est pas contrôler la carte.
//
// Un socle de BONUS échappe à cet ordre : il n'est pas une arme, et il ne devient jamais
// « base » même si une famille de bonus portait par accident le nom d'une arme de départ.
func (m Match) TierOf(padIndex int, weapon string) Tier {
	family := m.FamilyOfPad(padIndex)
	if family == familyPowerup {
		return TierPowerup
	}
	if !m.randomStarts && m.baseWeapons[weapon] {
		return TierBase
	}
	switch family {
	case familyRack:
		return TierGround
	case familyPower:
		return TierPower
	default:
		return TierUnclassified
	}
}

// baseWeaponsOf lit les armes de départ du match dans le canal `loadouts`.
//
// UNE SEULE ÉMISSION COMPTE PAR SLOT, LA PREMIÈRE, et c'est mesuré (étape 0) : un `slot` est
// une VIE (100 slots distincts pour 113 pistes sur le match témoin `01e1f945`), et le canal
// RÉ-ÉMET en cours de vie après un changement d'arme. Prendre tout le canal dilue les trois
// armes de base de 94,4 % à 85,6 % en classé et fait monter le S7 Sniper de 0,73 % à 3,15 % —
// c'est-à-dire qu'il transforme une arme de puissance en arme de base.
//
// Rend les familles retenues et le nombre de vies lues.
func baseWeaponsOf(spawns []Spawn) (map[string]bool, int) {
	premiere := make(map[uint32]int, len(spawns))
	for i, l := range spawns {
		if _, vu := premiere[l.Slot]; !vu {
			premiere[l.Slot] = i
		}
	}
	vies := len(premiere)
	if vies == 0 {
		return nil, 0
	}
	compte := make(map[string]int)
	for _, i := range premiere {
		// Une même arme deux fois dans le même équipement de départ ne vaut qu'une vie :
		// le dénominateur est la VIE, pas l'emplacement d'inventaire.
		vues := make(map[string]bool, len(spawns[i].Weapons))
		for _, w := range spawns[i].Weapons {
			if w == "" || vues[w] {
				continue
			}
			vues[w] = true
			compte[w]++
		}
	}
	seuil := float64(vies) * BaseShareMin
	out := make(map[string]bool)
	for w, n := range compte {
		if float64(n) >= seuil {
			out[w] = true
		}
	}
	return out, vies
}
