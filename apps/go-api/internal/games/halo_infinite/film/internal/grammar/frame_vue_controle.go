package grammar

// frame_vue_controle.go — LA VUE C (RANG 2) D UNE TRAME DELTA : LA VUE DE CONTROLE (lot 5.14).
//
// `FUN_1406cf548`, vtable `0x1436a8770`, rang 2 (`FUN_141f855b4`). Sa classe est nommee par son
// propre code : le prologue `FUN_142f2539c` porte le chemin source
// `shared\engine\source\blofeld\networking\replication\replication_control_view.cpp`
// (chaine `143c999f0`) — c est la VUE DE CONTROLE, et ses records ne sont pas des entites.
//
// L ordre des trois rangs et le partage du lecteur sont documentes dans `frame_vue_messages.go`.

// LargeurKindVueC est la largeur du selecteur de handler de la vue C
// (`FUN_1406cf548` : `kind = R(2)`).
const LargeurKindVueC = 2

// kindVueC* nomment les quatre valeurs du selecteur de la vue C (`FUN_1406cf548`).
const (
	kindVueCControle = 0 // FUN_1406d0388 — l entree de controle d un participant
	kindVueCSecond   = 1 // FUN_142f29b38
	kindVueCTiers    = 2 // FUN_142f29e54
	kindVueCNeant    = 3 // aucun handler : ZERO bit, la boucle continue
)

// FluxVueC est ce que la vue C (rang 2) a consomme sur un paquet.
type FluxVueC struct {
	// Vide : la vue n a ecrit que son terminateur (UN bit).
	Vide bool
	// Kinds : les selecteurs `R(2)` rencontres, dans l ordre.
	Kinds []int
	// Porte : la vue a lu jusqu a son terminateur.
	Porte bool
}

// consumeVueC lit la vue C (rang 2) : `FUN_1406cf548`, la vue de CONTROLE
// (`replication_control_view.cpp`).
//
//	si FUN_1409c94b8() : prologue FUN_142f2539c — ZERO BIT. Il ne lit pas le flux : il RECOPIE
//	   le tampon (`memcpy(dst, reader+8, reader+0x18)`), une photo du payload pour la suite.
//	   La garde etant hors flux et le prologue gratuit, la marche hors ligne n a rien a decider.
//	boucle {
//	   b = R(1) ; si 0 -> FIN
//	   si budget <= rendus -> sortie (code 3)
//	   kind = R(2)
//	   0 -> FUN_1406d0388 · 1 -> FUN_142f29b38 · 2 -> FUN_142f29e54 · 3 -> ZERO BIT
//	}
//
// `kind == 3` ne coute rien et la boucle continue : c est le seul cas ou la vue C enchaine.
// Les trois autres portent une charge ; `consumeControleVueC` porte celle du `kind` 0.
func consumeVueC(br *Lecteur, frameLen int) FluxVueC {
	out := FluxVueC{}
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1) {
			return out
		}
		if !br.ReadBit() {
			out.Vide = tour == 0
			out.Porte = true
			return out
		}
		if !placeDisponible(br, frameLen, LargeurKindVueC) {
			return out
		}
		k := int(br.ReadBits(LargeurKindVueC))
		out.Kinds = append(out.Kinds, k)
		switch k {
		case kindVueCNeant:
			continue // zero bit : l ecrivain reboucle
		case kindVueCControle:
			if !consumeControleVueC(br, frameLen) {
				return out
			}
		default:
			return out // kinds 1 et 2 : charge non portee
		}
	}
	return out
}

// plafondToursVueC borne la boucle de la vue C. Ce n est PAS une largeur de grammaire : le
// budget de l ecrivain est `param_4` (le nombre de records restants), que la marche hors ligne
// n a pas. Un flux sain sort sur son terminateur bien avant — le maximum mesure est huit tours
// sur `bfecd02b` (huit participants).
const plafondToursVueC = 64

// consumeControleVueC lit la charge du `kind` 0 de la vue C : `FUN_1406d0388`, L ENTREE DE
// CONTROLE D UN PARTICIPANT.
//
// # CE QUE CE CANAL PORTE, ET C EST NEUF
//
//	FUN_1406cdc04(reader)            R(1) ; si 1 -> R(7)          un index, sentinelle 0xff
//	R(5)                             l index de controle (0..31), garde `> 0x1f -> return 3`
//	                                 (inatteignable : cinq bits ne valent jamais plus de 31)
//	branche FUN_14048ee34() == 0 :
//	   a = R(1) ; si a -> FUN_1406cd860(reader, bloc de 0x68)
//	   b = R(1) ; si b -> FUN_141fdae44(reader, bloc de 0xbc)
//	branche FUN_14048ee34() != 0 :
//	   FUN_1404f1ca4() ? FUN_142f2a17c (= R(1)) : FUN_142f29954 (= R(1) [+ R(5) + charge])
//
// et `FUN_1406cd860` — le bloc de 0x68 octets — est L ENTREE ELLE-MEME :
//
//	R(1)                             presence d un second champ
//	  si 1 : R(2)                    (R(4) quand `DAT_145121140 == 1`)
//	FUN_1406d6ef4(reader, &f[0], &f[1])   DEUX SCALAIRES QUANTIFIES SUR 6 BITS chacun, code 0
//	                                 et code 0x3e = les deux bornes, code 0x1f = ZERO EXACT,
//	                                 sinon `(code - 1) * pas - origine` : un couple analogique
//	                                 symetrique. Puis R(1) ; si 1 : un troisieme champ de
//	                                 largeur passee par la pile (non resolue au site d appel).
//	R(1)                             si 1 : un champ de largeur passee par la pile
//	R(1)                             si 0 : FUN_1406d025c — LES BITS D ACTION (ci-dessous),
//	                                 puis FIN. Si 1 : R(5) (R(7) quand `DAT_145121140 == 1`).
//
// `FUN_1406d025c` est le bloc d ACTIONS, et sa premiere garde est un bit : `g = R(1)` ; si `g`
// vaut 0 la charge est VIDE. Sinon viennent un groupe de 6 bits (`FUN_1431ab1ec`, indices
// (0,0) (0,1) (0,2) (1,0) (1,1) (1,2)), un groupe de 4 bits (`FUN_1431ab1cc`, (0,0) (0,1)
// (1,0) (1,1)), un bit de visee qui ouvre deux bits de plus et un point
// (`FUN_1431a0bbc` / `FUN_1431a0abc` / `FUN_1431a0cbc`), puis `FUN_1406d0f20`, deux
// `FUN_1406d00ec` gardes par les bits d action deja lus, et `FUN_142f26740`.
//
// # CE QUE CETTE FONCTION PORTE, ET CE QU ELLE REFUSE DE DEVINER
//
// Elle porte le chemin dont TOUTES les largeurs sont resolues chez l ecrivain : la branche
// `FUN_14048ee34() == 0`, largeurs courtes (`DAT_145121140 != 1`), chemin quantifie de
// `FUN_1406d6ef4` (`DAT_145173840 == 0`), et les gardes de presence LUES DANS LE FLUX. Des
// qu une garde ouvre un champ dont la largeur vient de la pile ou d un sous-arbre non porte,
// elle rend `false` : le curseur s arrete sur le bit de garde, et le paquet est SIGNALE. Aucune
// largeur n est inventee ici.
func consumeControleVueC(br *Lecteur, frameLen int) bool {
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	if br.ReadBit() { // FUN_1406cdc04 : R(1) puis R(7), sentinelle 0xff quand le bit est 0
		if !placeDisponible(br, frameLen, largeurIndexCdc04) {
			return false
		}
		br.Skip(largeurIndexCdc04)
	}
	if !placeDisponible(br, frameLen, largeurIndexControle) {
		return false
	}
	br.Skip(largeurIndexControle) // R(5) : l index de controle, 0..31
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	if br.ReadBit() && !consumeEntreeControle(br, frameLen) { // FUN_1406cd860
		return false
	}
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	// `b` ouvre `FUN_141fdae44` (bloc de 0xbc octets), dont la premiere largeur vient deja d un
	// global (`DAT_145121140`) et dont la suite appelle quatre sous-lecteurs non portes.
	return !br.ReadBit()
}

// largeurIndexCdc04 est la largeur du champ que `FUN_1406cdc04` ouvre derriere son bit de
// presence (`if (0x40 - iVar1 < 7)` : sept bits).
const largeurIndexCdc04 = 7

// largeurIndexControle est la largeur de l index de controle de `FUN_1406d0388`
// (`if (iVar20 < 5)` : cinq bits, et la garde `0x1f < uVar21` qui suit est inatteignable).
const largeurIndexControle = 5

// consumeEntreeControle lit `FUN_1406cd860` : le bloc de 0x68 octets de l entree de controle.
func consumeEntreeControle(br *Lecteur, frameLen int) bool {
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	if br.ReadBit() { // le second champ du bloc
		if !placeDisponible(br, frameLen, largeurCourteControle) {
			return false
		}
		br.Skip(largeurCourteControle)
	}
	if !consumeCoupleAnalogique(br, frameLen) { // FUN_1406d6ef4
		return false
	}
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	if br.ReadBit() {
		return false // champ de largeur passee par la pile : non resolue
	}
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	if br.ReadBit() {
		return false // la branche `R(5)` + FUN_142265fe3 : non portee
	}
	return consumeActionsControle(br, frameLen) // FUN_1406d025c
}

// largeurCourteControle est la largeur du second champ du bloc de controle. `FUN_1406cd860` la
// tire du global `DAT_145121140` : `(DAT_145121140 == 1) * 2 + 2`, donc 2 ou 4 — et la meme
// expression `* 2 + 5` (5 ou 7) gouverne l autre champ, celui de la branche que ce port ne
// prend pas. Le port retient la forme COURTE (global a 0), et c est la FERMETURE DES PAQUETS
// qui l arbitre, comme elle arbitre `HasExtraFields` : les deux formes viennent de l ecrivain,
// et seul le film dit laquelle il porte.
const largeurCourteControle = 2

// LargeurScalaireAnalogique est la largeur d un scalaire quantifie du couple analogique de
// `FUN_1406d6ef4` : six bits, codes 0 et 0x3e aux bornes, code 0x1f a ZERO EXACT.
const LargeurScalaireAnalogique = 6

// consumeCoupleAnalogique lit `FUN_1406d6ef4` : deux scalaires quantifies sur six bits, puis un
// bit de presence pour un troisieme champ dont la largeur vient de la pile.
func consumeCoupleAnalogique(br *Lecteur, frameLen int) bool {
	if !placeDisponible(br, frameLen, 2*LargeurScalaireAnalogique+1) {
		return false
	}
	br.Skip(2 * LargeurScalaireAnalogique)
	return !br.ReadBit()
}

// consumeActionsControle lit `FUN_1406d025c` : les bits d ACTION de l entree de controle.
// Sa premiere garde est un bit du flux ; a 0 la charge est vide, et c est le seul chemin dont
// toutes les largeurs sont resolues.
func consumeActionsControle(br *Lecteur, frameLen int) bool {
	if !placeDisponible(br, frameLen, 1) {
		return false
	}
	return !br.ReadBit()
}
