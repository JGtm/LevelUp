package grammar

// frame_vue_controle.go — LA VUE C (RANG 2) D UNE TRAME DELTA : LA VUE DE CONTROLE (lot 5.14).
//
// `FUN_1406cf548`, vtable `0x1436a8770`, rang 2 (`FUN_141f855b4`). Sa classe est nommee par son
// propre code : le prologue `FUN_142f2539c` porte le chemin source
// `shared\engine\source\blofeld\networking\replication\replication_control_view.cpp`
// (chaine `143c999f0`) — c est la VUE DE CONTROLE, et ses records ne sont pas des entites.
//
// L ordre des trois rangs et le partage du lecteur sont documentes dans `frame_vue_messages.go`.
//
// # CE QUE LE LOT M4b Y LIT : LE TIR CONTINU (campagne « retours rejeu », 2026-09-24)
//
// L entree `kind 0` d un participant porte, dans son bloc d action ([BlocDAction]), les GACHETTES
// TENUES dont le barillet n a pas emis de record de tir (type de prediction 1) et les BARILLETS de
// type 3 en tir : c est le canal que Theater rejoue pour le Ghost, les canons de la Banshee, le
// Chopper, la LAAG, les LMG du Falcon et du Wasp, et a pied le Rayon de Sentinelle (sonde P1-S3,
// `.ai/V7.5/retours_rejeu_2026-09-23/SONDE_P1S3_tir_continu_vue_controle.md`). L entree est
// desormais lue EN ENTIER : les trois branches que ce port refusait (le troisieme champ du couple
// analogique, le champ +0x10 et les drapeaux +0x14) sont resolues au site d appel, et le bloc
// d action est celui de `bloc_action.go`, dont deux sous-lecteurs ont ete corriges. Ce que la vue
// rend ne vaut que si le paquet se FERME ([vueCFermee]) : c est l oracle de cadrage de la sonde,
// et un paquet dont la vue C n est pas atteinte ou ne se ferme pas est un TROU, nomme et compte
// par l appelant ([LectureVueC]).

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

// ArretVueC nomme la cause pour laquelle la lecture d une vue C s est arretee avant son
// terminateur. Chaque cause est un TROU NOMME : le paquet ne rend aucune entree.
type ArretVueC int

// Les causes d arret de la vue C, et leur compte.
const (
	// ArretVueCAucun : la vue a lu jusqu a son terminateur.
	ArretVueCAucun ArretVueC = iota
	// ArretVueCDebordement : la lecture a atteint la fin du payload avant le terminateur.
	ArretVueCDebordement
	// ArretVueCKindNonPorte : un selecteur `kind` 1 (`FUN_142f29b38`) ou 2 (`FUN_142f29e54`), dont
	// la charge depend d un etat d execution (`FUN_142f2b574`) et n est pas portee.
	ArretVueCKindNonPorte
	// ArretVueCBlocBC : le second bloc de l entree (`FUN_141fdae44`, 0xbc octets), dont la premiere
	// largeur vient d un global et la suite de quatre sous-lecteurs non portes.
	ArretVueCBlocBC
	// ArretVueCPlafond : la boucle a atteint [plafondToursVueC] sans terminateur.
	ArretVueCPlafond
	// ArretVueCCount est le nombre de causes.
	ArretVueCCount = 5
)

// EntreeDeControle est l entree `kind 0` d un participant (`FUN_1406d0388` puis le bloc de 0x68
// octets `FUN_1406cd860`), dans ce que le tir continu en consomme.
type EntreeDeControle struct {
	// Index est l index de controle R(5) : l index de joueur du film, c est-a-dire la PLACE (le
	// meme espace que le tireur d un record de tir `action_weapon_fire`).
	Index int
	// Bloc dit que l entree porte le bloc de 0x68 octets (`a = R(1)` de `FUN_1406d0388`). Sans
	// lui, l entree ne dit RIEN des gachettes : ni tenues, ni lachees.
	Bloc bool
	// Action est le bloc d action de l entree ; vide quand l entree ne porte pas le bloc de 0x68.
	Action BlocDAction
	// Champs sont les autres valeurs de l entree, lues telles que l ecrivain les pose ; un champ
	// absent du flux vaut [ChampDeControleAbsent].
	Champs ChampsDeControle
}

// ChampDeControleAbsent : le champ facultatif n est pas dans le flux (son bit de presence vaut 0).
const ChampDeControleAbsent = -1

// ChampsDeControle sont les valeurs de l entree hors bloc d action : le champ de sept bits de
// `FUN_1406cdc04`, puis ceux du bloc de 0x68 (`FUN_1406cd860`) dans leur ordre d ecriture.
type ChampsDeControle struct {
	Cdc04, Court, Troisieme, Champ10, Drapeaux int
	// Analogique est le couple de `FUN_1406d6ef4`.
	Analogique [2]int
}

// champsAbsents rend des champs tous absents.
func champsAbsents() ChampsDeControle {
	a := ChampDeControleAbsent
	return ChampsDeControle{Cdc04: a, Court: a, Troisieme: a, Champ10: a, Drapeaux: a, Analogique: [2]int{a, a}}
}

// FluxVueC est ce que la vue C (rang 2) a consomme sur un paquet.
type FluxVueC struct {
	// Vide : la vue n a ecrit que son terminateur (UN bit).
	Vide bool
	// Kinds : les selecteurs `R(2)` rencontres, dans l ordre.
	Kinds []int
	// Porte : la vue a lu jusqu a son terminateur.
	Porte bool
	// Arret : la cause, quand la vue ne s est pas lue jusqu a son terminateur.
	Arret ArretVueC
	// Entrees : les entrees `kind 0` lues, dans l ordre du flux.
	Entrees []EntreeDeControle
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
// Les trois autres portent une charge ; [lireEntreeDeControle] porte celle du `kind` 0.
func consumeVueC(br *Lecteur, frameLen int) FluxVueC {
	out := FluxVueC{Arret: ArretVueCPlafond}
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1) {
			out.Arret = ArretVueCDebordement
			return out
		}
		if !br.ReadBit() {
			out.Vide = tour == 0
			out.Porte, out.Arret = true, ArretVueCAucun
			return out
		}
		if !placeDisponible(br, frameLen, LargeurKindVueC) {
			out.Arret = ArretVueCDebordement
			return out
		}
		k := int(br.ReadBits(LargeurKindVueC))
		out.Kinds = append(out.Kinds, k)
		switch k {
		case kindVueCNeant:
			continue // zero bit : l ecrivain reboucle
		case kindVueCControle:
			e, arret := lireEntreeDeControle(br, frameLen)
			if arret != ArretVueCAucun {
				out.Arret = arret
				return out
			}
			out.Entrees = append(out.Entrees, e)
		default:
			out.Arret = ArretVueCKindNonPorte // kinds 1 et 2 : charge non portee
			return out
		}
	}
	return out
}

// plafondToursVueC borne la boucle de la vue C. Ce n est PAS une largeur de grammaire : le
// budget de l ecrivain est `param_4` (le nombre de records restants), que la marche hors ligne
// n a pas. Un flux sain sort sur son terminateur bien avant — le maximum mesure est huit tours
// sur `bfecd02b` (huit participants).
const plafondToursVueC = 64

// lireEntreeDeControle lit la charge du `kind` 0 de la vue C : `FUN_1406d0388`, L ENTREE DE
// CONTROLE D UN PARTICIPANT.
//
//	FUN_1406cdc04(reader)            R(1) ; si 1 -> R(7)          un index, sentinelle 0xff
//	R(5)                             l index de controle (0..31), garde `> 0x1f -> return 3`
//	                                 (inatteignable : cinq bits ne valent jamais plus de 31)
//	branche FUN_14048ee34() == 0 :
//	   a = R(1) ; si a -> FUN_1406cd860(reader, bloc de 0x68)   [consumeEntreeControle]
//	   b = R(1) ; si b -> FUN_141fdae44(reader, bloc de 0xbc)   NON PORTE : [ArretVueCBlocBC]
//
// La branche `FUN_14048ee34() != 0` n est pas celle du film : la fermeture des paquets l arbitre
// (lot 5.14), comme elle arbitre les largeurs courtes (`DAT_145121140 != 1`).
func lireEntreeDeControle(br *Lecteur, frameLen int) (EntreeDeControle, ArretVueC) {
	e := EntreeDeControle{Champs: champsAbsents()}
	if !placeDisponible(br, frameLen, 1) {
		return e, ArretVueCDebordement
	}
	if br.ReadBit() { // FUN_1406cdc04 : R(1) puis R(7), sentinelle 0xff quand le bit est 0
		if !placeDisponible(br, frameLen, largeurIndexCdc04) {
			return e, ArretVueCDebordement
		}
		e.Champs.Cdc04 = int(br.ReadBits(largeurIndexCdc04)) //nolint:gosec // R(7)
	}
	if !placeDisponible(br, frameLen, largeurIndexControle+1) {
		return e, ArretVueCDebordement
	}
	e.Index = int(br.ReadBits(largeurIndexControle)) //nolint:gosec // R(5) : 0..31
	if e.Bloc = br.ReadBit(); e.Bloc {               // a : le bloc de 0x68 (FUN_1406cd860)
		action, ok := consumeEntreeControle(br, frameLen, &e.Champs)
		if !ok {
			return e, ArretVueCDebordement
		}
		e.Action = action
	}
	if !placeDisponible(br, frameLen, 1) {
		return e, ArretVueCDebordement
	}
	if br.ReadBit() { // b : FUN_141fdae44 (bloc de 0xbc), non porte
		return e, ArretVueCBlocBC
	}
	return e, ArretVueCAucun
}

// largeurIndexCdc04 est la largeur du champ que `FUN_1406cdc04` ouvre derriere son bit de
// presence (`if (0x40 - iVar1 < 7)` : sept bits).
const largeurIndexCdc04 = 7

// largeurIndexControle est la largeur de l index de controle de `FUN_1406d0388`
// (`if (iVar20 < 5)` : cinq bits, et la garde `0x1f < uVar21` qui suit est inatteignable).
const largeurIndexControle = 5

// Les trois largeurs que `FUN_1406cd860` passe par la PILE a `FUN_1406d84b4`, relues au site
// d appel (lot M4b ; le port les refusait, faute de les avoir lues) :
const (
	// largeurTroisiemeChamp : le champ optionnel de `FUN_1406d6ef4` (`MOV [RSP+0x20],5`
	// @1422f6bdb).
	largeurTroisiemeChamp = 5
	// largeurChamp10 : le champ +0x10 (`MOV [RSP+0x20],6` @142265fcc, stocke par
	// `MOVSS [RSI+0x10]`).
	largeurChamp10 = 6
	// largeurDrapeaux14 : les drapeaux +0x14 (`(DAT_145121140 == 1) * 2 + 5`, forme courte ;
	// `MOV word [RSI+0x14],R10W` @142265fe3, puis `JMP 0x1406cd991` : LE BLOC D ACTION SUIT).
	largeurDrapeaux14 = 5
)

// consumeEntreeControle lit `FUN_1406cd860` : le bloc de 0x68 octets de l entree de controle, et
// rend son bloc d action.
//
//	R(1) g ; si g : R(2)                       +0x00, +0x01 (forme courte, DAT_145121140 != 1)
//	FUN_1406d6ef4 : R(6) R(6) ; R(1) ; si 1 : R(5)
//	R(1) ; si 1 : FUN_1406d84b4 R(6)           +0x10
//	R(1) ; si 1 : R(5)                         +0x14
//	FUN_1406d025c                              le bloc d action, DANS LES DEUX BRANCHES
//
// LA BORNE EST POSEE A LA SORTIE, PAS A L ENTREE : la largeur du bloc d action depend de ses
// gardes. Il est lu, puis le curseur est compare a la trame ; un debordement rend `false`.
func consumeEntreeControle(br *Lecteur, frameLen int, ch *ChampsDeControle) (BlocDAction, bool) {
	if !placeDisponible(br, frameLen, 1) {
		return BlocDAction{}, false
	}
	if br.ReadBit() { // le second champ du bloc
		if !placeDisponible(br, frameLen, largeurCourteControle) {
			return BlocDAction{}, false
		}
		ch.Court = int(br.ReadBits(largeurCourteControle)) //nolint:gosec // R(2)
	}
	if !placeDisponible(br, frameLen, 2*LargeurScalaireAnalogique+1) {
		return BlocDAction{}, false
	}
	for k := range ch.Analogique { // FUN_1406d6ef4 : le couple analogique
		ch.Analogique[k] = int(br.ReadBits(LargeurScalaireAnalogique)) //nolint:gosec // R(6)
	}
	facultatifs := []*int{&ch.Troisieme, &ch.Champ10, &ch.Drapeaux}
	for k, largeur := range []uint{largeurTroisiemeChamp, largeurChamp10, largeurDrapeaux14} {
		if !placeDisponible(br, frameLen, 1) {
			return BlocDAction{}, false
		}
		if br.ReadBit() {
			if !placeDisponible(br, frameLen, int(largeur)) {
				return BlocDAction{}, false
			}
			*facultatifs[k] = int(br.ReadBits(largeur)) //nolint:gosec // R(5) / R(6)
		}
	}
	if !placeDisponible(br, frameLen, 1) {
		return BlocDAction{}, false
	}
	action := lireBlocDAction(br) // FUN_1406d025c
	return action, br.BitPos() <= frameLen
}

// largeurCourteControle est la largeur du second champ du bloc de controle. `FUN_1406cd860` la
// tire du global `DAT_145121140` : `(DAT_145121140 == 1) * 2 + 2`, donc 2 ou 4 — et la meme
// expression `* 2 + 5` (5 ou 7) gouverne les drapeaux +0x14. Le port retient la forme COURTE
// (global a 0), et c est la FERMETURE DES PAQUETS qui l arbitre, comme elle arbitre
// `HasExtraFields` : les deux formes viennent de l ecrivain, et seul le film dit laquelle il porte.
const largeurCourteControle = 2

// LargeurScalaireAnalogique est la largeur d un scalaire quantifie du couple analogique de
// `FUN_1406d6ef4` : six bits, codes 0 et 0x3e aux bornes, code 0x1f a ZERO EXACT.
const LargeurScalaireAnalogique = 6

// bourrageMaxBits est le reste maximal d un paquet qui se FERME : la trame s arrete sur une
// frontiere d octet, donc au plus sept bits de bourrage, tous nuls.
const bourrageMaxBits = 7

// vueCFermee applique l ORACLE DE CADRAGE de la vue C (sonde P1-S3) : la vue C est le DERNIER rang
// d une trame delta, donc une lecture juste finit sur son terminateur avec un reste de 0 a 7 bits,
// TOUS NULS. Une vue qui « se lit » jusqu a un terminateur sans fermer le paquet a ete lue a une
// position fausse — le plus souvent une fin de vue B fausse.
func vueCFermee(pay []byte, curseur int) bool {
	reste := len(pay)*8 - curseur
	if reste < 0 || reste > bourrageMaxBits {
		return false
	}
	for i := curseur; i < len(pay)*8; i++ {
		if pay[i/8]&(1<<uint(7-i%8)) != 0 { //nolint:gosec // i%8 in [0,7]
			return false
		}
	}
	return true
}

// LectureVueC est le VERDICT d un paquet sur sa vue C, tel que la marche du frame-processeur le
// publie ([Observation.VueControleHook]) : UN par paquet delta marche.
type LectureVueC struct {
	// Atteinte : la vue B a clos sa liste, la vue C a donc ete lue.
	Atteinte bool
	// Fermee : la vue C s est lue jusqu a son terminateur ET le paquet se ferme ([vueCFermee]).
	// Seule une vue fermee rend des entrees.
	Fermee bool
	// Arret : la cause, quand la vue C atteinte ne s est pas lue jusqu a son terminateur.
	Arret ArretVueC
	// Entrees : les entrees de controle d une vue FERMEE ; vide sinon.
	Entrees []EntreeDeControle
}

// publierVueC rend au hook, s il y en a un, le verdict d un paquet sur sa vue C.
func (b *Lecteur) publierVueC(l LectureVueC) {
	if b.obs == nil || b.obs.VueControleHook == nil {
		return
	}
	b.obs.VueControleHook(l)
}
