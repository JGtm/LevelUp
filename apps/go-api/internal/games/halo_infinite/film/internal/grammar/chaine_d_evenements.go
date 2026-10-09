package grammar

// chaine_d_evenements.go — LA GRAMMAIRE DE LA CHAINE D EVENEMENTS DU RATTRAPAGE DES KILLS
// ([rattrapageDesKills]) : un second lecteur des messages de la vue A, plus pauvre que la lecture
// unique ([lireLaVueA]) et qui ne la remplace nulle part. Il ne sert qu a VALIDER une position de
// message de kill que la recherche bit a bit propose, dans une trame dont la vue A ne se lit pas
// jusqu a son terminateur : une coincidence ne s enchaine pas.
//
// GRAMMAIRE D UN EVENEMENT (desassemblage FUN_14076a1c4 + FUN_14080a9d4, RE_LOG 7ter.25) :
//
//	[1 bit de continuation]   0 => fin de la liste
//	[R(7) code]               code >= 123 => le dispatcher renvoie une erreur
//	3 x { [1 bit de porte] ; si 1 -> champ de presence(cfgIdx[code][i]) }
//	[corps : vtable+0x68 du descripteur du code]
//
// LE PIEGE QUI A COUTE LE CHANTIER, et il est consigne parce qu il est CONTRE-INTUITIF : le bit
// de drapeau du champ de presence N EST PAS lu systematiquement. Dans FUN_1406d3140 le
// court-circuit `(param_3 == 1) && ReadBit(...)` n evalue le bit QUE si `param_3 == 1`, et
// `param_3` est le cfgIdx — une propriete du TYPE d evenement rendue par `vtable+0x58`, qui ne
// lit AUCUN bit — pas l indice de boucle. Le modele << drapeau systematique >> est le cas
// PARTICULIER cfg==1 : d ou 100 % sur le code 1 et 6 % sur le code 7. Grammaire validee a 98.3 %
// sur 3 963 paires consecutives, et **56/56 = 100 % sur le code 85** (RE_LOG 7ter.25 (2)).
//
// CE FICHIER NE DECIDE RIEN. Il rend des positions et des champs ; killsource confronte au
// kill-feed, et le kill-feed reste le seul juge.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// lignesDePresRange : le nombre de lignes de la table de quantification (9 lignes + terminateur).
const lignesDePresRange = 10

// presRange rend la colonne `range` de la ligne `c` de la table de quantification du moteur. Elle
// vaut ZERO dans l image PE et n est remplie qu au demarrage : seule une lecture memoire la donne
// (capture `tools/ce/config_qhit_1451f98c0.bin`, RE_LOG 7ter.2).
func presRange(c int) uint32 {
	return [lignesDePresRange]uint32{7679, 7679, 256, 256, 512, 256, 512, 8191, 8191, 0}[c]
}

// presSpecialRow : la ligne SPECIALE, prise par le court-circuit quand cfgIdx == 1 et que le bit
// de drapeau vaut 1.
const presSpecialRow = 4

// qbitsOf : largeur de quantification d un range (FUN_1406d310c). Zero quand le range est nul —
// et dans ce cas la lecture de la valeur est entierement SAUTEE, pas lue a zero bit.
func qbitsOf(r uint32) int {
	if r == 0 {
		return 0
	}
	m := 31
	for r>>uint(m) == 0 {
		m--
	}
	if r&((1<<uint(m))-1) != 0 {
		return m + 1
	}
	return m
}

// codesDEvenement : le nombre de codes du dispatcher ; un code au-dela renvoie une erreur.
const codesDEvenement = 123

// evCfgIdx rend le cfgIdx de chacun des 3 emplacements de presence du code d evenement `code`
// (zero hors de la table). Source : la table du registre FUN_140e453b4 (123 codes) plus quatre
// fournisseurs desassembles. La valeur -1 marque un fournisseur NON desassemble : l emplacement n est
// decodable que s il est ABSENT. C est une reserve honnete, pas un trou masque — la chaine s arrete
// au lieu d extrapoler.
func evCfgIdx(code int) [3]int {
	if code < 0 || code >= codesDEvenement {
		return [3]int{}
	}
	return [codesDEvenement][3]int{
		0: {1, 1, 7}, 1: {1, 8, 7}, 2: {1, 8, 7}, 3: {0, 8, 7},
		4: {0, 8, 7}, 5: {5, 8, 7}, 6: {5, 8, 7}, 7: {1, 8, 7},
		8: {2, 3, 7}, 9: {2, 8, 7}, 10: {8, 8, 7}, 11: {2, 8, 7},
		12: {8, 8, 7}, 13: {4, 8, 7}, 14: {-1, -1, -1}, 15: {8, 8, 7},
		16: {8, 8, 7}, 17: {-1, -1, -1}, 18: {6, 8, 7}, 19: {0, 8, 7},
		20: {1, 1, 7}, 21: {4, 8, 7}, 22: {1, 1, 7}, 23: {0, 8, 7},
		24: {0, 2, 7}, 25: {0, 2, 7}, 26: {0, 2, 7}, 27: {0, 0, 7},
		28: {0, 8, 7}, 29: {6, 8, 7}, 30: {4, 8, 7}, 31: {1, 8, 7},
		32: {1, 8, 7}, 33: {3, 4, 7}, 34: {8, 8, 7}, 35: {1, 8, 7},
		36: {1, 8, 7}, 37: {1, 8, 7}, 38: {2, 8, 7}, 39: {2, 8, 7},
		40: {2, 0, 7}, 41: {3, 8, 7}, 42: {2, 8, 7}, 43: {2, 0, 7},
		44: {2, 0, 7}, 45: {2, 8, 7}, 46: {4, 8, 7}, 47: {4, 8, 7},
		48: {4, 8, 7}, 49: {3, 2, 7}, 50: {-1, -1, -1}, 51: {2, 8, 7},
		52: {2, 0, 7}, 53: {1, 1, 7}, 54: {1, 1, 7}, 55: {6, 8, 7},
		56: {5, 1, 7}, 57: {2, 0, 7}, 58: {1, 8, 7}, 59: {0, 8, 7},
		60: {0, 8, 7}, 61: {0, 8, 7}, 62: {6, 8, 7}, 63: {2, 8, 7},
		64: {6, 8, 7}, 65: {3, 8, 7}, 66: {6, 8, 7}, 67: {6, 8, 7},
		68: {6, 8, 7}, 69: {0, 8, 7}, 70: {2, 8, 7}, 71: {2, 8, 7},
		72: {2, 8, 7}, 73: {2, 8, 7}, 74: {2, 8, 7}, 75: {1, 8, 7},
		76: {1, 8, 7}, 77: {6, 8, 7}, 78: {4, 8, 7}, 79: {-1, -1, -1},
		80: {8, 8, 7}, 81: {0, 8, 7}, 82: {0, 8, 7}, 83: {0, 8, 7},
		84: {0, 8, 7}, 85: {8, 8, 7}, 86: {1, 8, 7}, 87: {6, 8, 7},
		88: {6, 8, 7}, 89: {6, 8, 7}, 90: {6, 8, 7}, 91: {6, 8, 7},
		92: {6, 8, 7}, 93: {2, 0, 7}, 94: {6, 8, 7}, 95: {6, 8, 7},
		96: {6, 8, 7}, 97: {6, 8, 7}, 98: {1, 1, 7}, 99: {8, 8, 7},
		100: {1, 8, 7}, 101: {6, 8, 7}, 102: {2, 8, 7}, 103: {0, 0, 7},
		104: {0, -1, -1}, 105: {0, 0, 7}, 106: {0, -1, -1}, 107: {0, 8, 7},
		108: {6, 8, 7}, 109: {0, 0, 7}, 110: {1, 8, 7}, 111: {6, 8, 7},
		112: {6, 8, 7}, 113: {6, 8, 7}, 114: {6, 8, 7}, 115: {-1, -1, -1},
		116: {-1, -1, -1}, 117: {-1, -1, -1}, 118: {1, 8, 7}, 119: {2, 0, 7},
		120: {8, 8, 7}, 121: {8, 8, 7}, 122: {8, 8, 7},
	}[code]
}

// curseurEv : LE CURSEUR MEFIANT DE LA CHAINE D EVENEMENTS.
//
// Il porte le lecteur de bits canonique de la couche source ([source.Bits]) et lui ajoute la seule
// chose que le moteur n a pas — LE REFUS DE LIRE AU-DELA DU PAQUET. Le drapeau est indispensable :
// le lecteur du moteur bourre a zero, et sans ce refus une chaine desynchronisee << lit >> des
// evenements parfaitement valides apres la fin du paquet.
//
// LE DRAPEAU EST ICI, ET PAS DANS LE LECTEUR (arbitrage V15 (3)). La mefiance appartient au
// MARCHEUR de chaine, pas au moteur : le lecteur canonique garde la semantique du jeu
// (bourrage a zero), et le refus se teste par [source.Bits.Remaining] AVANT chaque lecture.
// Aucune valeur lue ne change — `bp+n > len(pl)*8` et `Remaining() < n` sont la meme condition,
// et l equivalence bit a bit est prouvee par `chaine_d_evenements_equivalence_test.go`.
type curseurEv struct {
	b    *source.Bits
	over bool
}

// nouveauCurseurEv ouvre un curseur sur `pl`, positionne au bit `bp`.
func nouveauCurseurEv(pl []byte, bp int) *curseurEv {
	b := source.NewBits(pl)
	b.SetBitPos(bp)
	return &curseurEv{b: b}
}

// pos : la position de lecture, en bits.
func (r *curseurEv) pos() int { return r.b.BitPos() }

// aller : repositionne le curseur a une position ABSOLUE deja calculee (la fin des champs d un
// message de kill, par exemple). Ne leve pas le drapeau : la position vient d une lecture qui a
// deja fait ses bornes.
func (r *curseurEv) aller(p int) { r.b.SetBitPos(p) }

// octets : les octets du paquet parcouru.
func (r *curseurEv) octets() []byte { return r.b.Octets() }

// epuise : plus aucun bit a lire.
func (r *curseurEv) epuise() bool { return r.b.Remaining() <= 0 }

func (r *curseurEv) rd(n int) uint64 {
	if n <= 0 {
		return 0
	}
	if r.b.Remaining() < n {
		r.over = true
		return 0
	}
	return r.b.ReadBits(uint(n))
}

func (r *curseurEv) g1() int { return int(r.rd(1)) } //nolint:gosec // R(1)

// skip : avance de `n` bits sans construire de valeur. Necessaire au-dela de 64 bits, ou `rd`
// n aurait plus de sens.
func (r *curseurEv) skip(n int) {
	if n <= 0 {
		return
	}
	if r.b.Remaining() < n {
		r.over = true
		return
	}
	r.b.Skip(n)
}

// evPresence : la boucle de presence, 3 emplacements FIXES. Rend faux quand un emplacement
// PRESENT porte un cfgIdx non resolu : la longueur est alors inconnue, et une longueur inconnue
// arrete la chaine.
func evPresence(r *curseurEv, code int) bool {
	cfg := evCfgIdx(code)
	for i := range 3 {
		if r.g1() == 0 {
			continue
		}
		c := cfg[i]
		if c < 0 || c >= lignesDePresRange {
			return false
		}
		if c == 1 && r.g1() == 1 {
			c = presSpecialRow
		}
		if w := qbitsOf(presRange(c)); w > 0 {
			r.rd(w)
		}
		r.rd(2) // tag, toujours lu
	}
	return !r.over
}

// lireRefDEntite5 : ECS_ReadEntityRefIndex5. Porte a 1 => ABSENT ([lecture.RefAbsente]) ; porte a
// 0 => 5 bits d indice local.
func lireRefDEntite5(r *curseurEv) int8 {
	if r.g1() != 0 {
		return lecture.RefAbsente
	}
	return int8(r.rd(5)) //nolint:gosec // R(5)
}

// lireLeKillDeLaChaine decode les champs d un message de kill depuis le premier bit de son CORPS,
// sans sa queue (`FUN_14104bd08`, cf. `vue_a_charges_execution.go`) : victime, tueur, part du tueur,
// drapeau, assistant, part de l assistant — L ORDRE EST VICTIME PUIS TUEUR, mesure contre le
// kill-feed (RE_LOG 7ter.75 (2)). Rend aussi la fin de ces champs, -1 en cas de debordement.
func lireLeKillDeLaChaine(pl []byte, corps int) (lecture.MessageDeKill, int) {
	r := nouveauCurseurEv(pl, corps)
	var k lecture.MessageDeKill
	k.Victime = lireRefDEntite5(r)
	k.Tueur = lireRefDEntite5(r)
	k.PartDuTueur = uint32(r.rd(32)) //nolint:gosec // R(32)
	k.Drapeau = uint8(r.g1())        //nolint:gosec // R(1)
	k.Assistant = lireRefDEntite5(r)
	k.PartDeLAssistant = uint32(r.rd(32)) //nolint:gosec // R(32)
	if r.over {
		return k, -1
	}
	return k, r.pos()
}

// killPlausible : la grammaire MINIMALE d un message de kill — tueur et victime presents et
// distincts, champs lus jusqu au bout. Aucune heuristique de contenu : c est la chaine qui valide,
// pas le contenu.
func killPlausible(k lecture.MessageDeKill, fin int) bool {
	return fin > 0 && k.Tueur >= 0 && k.Victime >= 0 && k.Tueur != k.Victime
}

// evStep : decode UN evenement complet a partir de son bit de continuation.
// `fin` vaut vrai quand le bit de continuation est a 0 — fin NORMALE de la liste, pas une erreur.
func evStep(r *curseurEv, gate15 bool) (fin, ok bool) {
	fin, ok, _ = evStepAvecVerdict(r, gate15)
	return fin, ok
}

// evStepAvecVerdict est [evStep], plus `nonModelise` : l arret vient d un code dont la longueur
// n est pas modelisee (presence a cfgIdx non resolu, ou corps inconnu) et non d un debordement —
// le verdict de `repli_chaine_evenement_code_non_modelise`.
func evStepAvecVerdict(r *curseurEv, gate15 bool) (fin, ok, nonModelise bool) {
	if r.epuise() {
		return false, false, false
	}
	if r.g1() == 0 {
		return true, true, false
	}
	code := int(r.rd(7)) //nolint:gosec // R(7)
	if r.over || code >= codesDEvenement {
		return false, false, false
	}
	if !evPresence(r, code) {
		return false, false, !r.over
	}
	if !evBody(r, code, gate15) {
		return false, false, !r.over
	}
	return false, !r.over, false
}

// evChainLen : nombre d evenements enchaines depuis `p`, borne a `maxEv`. C EST LE CRITERE DE
// LOCALISATION : une coincidence ne s enchaine pas. Seuil retenu : 3 evenements — 132/133 vrais
// gardes (rappel 99.2 %), 11/1739 faux gardes, dont 10 sont des kill-events REELS du meme paquet
// (multi-kill atteste) et 1 porte un indice de victime hors roster (RE_LOG 7ter.25 (3)).
func evChainLen(pl []byte, p int, gate15 bool, maxEv int) int {
	n, _ := evChainLenAvecVerdict(pl, p, gate15, maxEv)
	return n
}

// evChainLenAvecVerdict est [evChainLen], plus `arretee` : la chaine s est arretee sur un code non
// modelise ([evStepAvecVerdict]) avant sa fin normale et avant `maxEv`.
func evChainLenAvecVerdict(pl []byte, p int, gate15 bool, maxEv int) (n int, arretee bool) {
	r := nouveauCurseurEv(pl, p)
	for n < maxEv {
		fin, ok, nonModelise := evStepAvecVerdict(r, gate15)
		if fin || !ok {
			return n, nonModelise
		}
		n++
	}
	return n, false
}
