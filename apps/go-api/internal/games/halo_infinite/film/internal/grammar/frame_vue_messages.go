package grammar

// frame_vue_messages.go — LA VUE A (RANG 0) D UNE TRAME DELTA : UN FLUX DE MESSAGES (lot 5.14).
//
// # TROIS RANGS, TROIS CLASSES, TROIS GRAMMAIRES — ET L ORDRE EST PROUVE
//
// Le frame-processeur `FUN_142987460` lit UN bit de configuration
// (`DAT_144706104 = FUN_1406cf008(reader)`) puis parcourt SES TROIS VUES dans l ordre du
// tableau `param_1 + 0x228/0x230/0x238`, en appelant sur chacune `vtable[0x60]` (SANS le
// lecteur : zero bit) puis `vtable[0x40]` (la boucle de records). Les trois vtables ne portent
// PAS la meme fonction a `+0x40`, et le registraire dit QUEL RANG porte QUELLE CLASSE —
// `FUN_141f855b4` appelle `FUN_1409c9860(conteneur+8, rang, vue)` trois fois :
//
//	rang 0 -> conteneur + 0x3ce98 · vtable 0x1436a8700 · FUN_14076a1c4   VUE A
//	rang 1 -> conteneur + 0x21b70 · vtable 0x1436a87e0 · FUN_1406cd128   VUE B (entites)
//	rang 2 -> conteneur + 0x3d2d8 · vtable 0x1436a8770 · FUN_1406cf548   VUE C (controle)
//
// et `FUN_1409c9860` clot le rang dans `*(int *)(vue + 8)` — le champ que `FUN_142f2e174` met
// dans les deux bits de tete d un identifiant d image-cle. Le rang 1 mesure sur les images-cles
// des deux films temoins (lot 5.13.1) EST donc la vue B : la seule des trois grammaires que le
// depot portait.
//
// # CE QUE LE LOT 5.14 REFERME, ET C EST UN BIT QUE LE DEPOT NE SAVAIT PAS NOMMER
//
// [DefaultPacketPreambleBits] vaut 2 et sa documentation disait : « le desassemblage n en
// etablit qu UN ; le SECOND bit n est PAS localise, il est etabli par la MESURE ».
// **CE SECOND BIT EST LE TERMINATEUR `R(1) = 0` DE LA VUE A VIDE.** `FUN_14076a1c4` est une
// boucle `{ R(1) ; 0 -> fin ; corps }`, et sur `dad793c7` elle est vide sur les 5 345 paquets
// qui commencent par l amorce du frame-processeur — un bit chacun, jamais un corps. L amorce de
// 2 bits n est donc pas une amorce : c est `[bit de configuration][vue A vide]`.
//
// La consequence pratique : une vue A NON vide n est plus lue comme un bit d amorce aveugle mais
// DETECTEE, et le paquet est signale au lieu d etre decadre en silence.

// LargeurGenreVueA est la largeur du selecteur de genre d un corps de la vue A
// (`FUN_14080a9d4` : `R(7)` puis `if (genre < 0x7b)` — 123 genres).
const LargeurGenreVueA = 7

// GenresVueA est le nombre de genres que la table de la vue A porte (`FUN_14080a9d4` :
// `*(obj[0x18] + 0x210 + genre * 8)`, garde `genre < 0x7b`).
const GenresVueA = 0x7b

// FluxVueA est ce que la vue A (rang 0) a consomme sur un paquet.
//
// `FUN_14076a1c4` ne rend JAMAIS un record (`*param_6 = 0` sans condition) : la vue A est un
// flux de MESSAGES, pas d entites. `Genres` liste les selecteurs `R(7)` rencontres.
type FluxVueA struct {
	// Vide : la vue n a ecrit que son terminateur (UN bit). C est le cas de 100 % des paquets
	// de `dad793c7` (5 345 sur 5 345).
	Vide bool
	// Genres : les selecteurs `R(7)` des corps rencontres, dans l ordre.
	Genres []int
	// Porte : la vue a lu jusqu a son terminateur. `false` = un corps de message dont la
	// charge n est PAS portee (la table des 123 genres ne l est pas) — le curseur s arrete la,
	// et le paquet doit etre signale, jamais devine.
	Porte bool
}

// consumeVueA lit la vue A (rang 0) : `FUN_14076a1c4`.
//
//	si vue[0x11] != 0 : ZERO bit (la vue est desactivee)   <- etat runtime, hors flux
//	boucle {
//	   b = R(1) ; si 0 -> FIN
//	   si budget < 1 -> sortie (code 3), zero bit de corps
//	   corps = FUN_14080a9d4 : genre = R(7) ; si genre < 123 : charge du genre par
//	           handler->vtable[0x68] ; puis si HasExtraFields et R(1) : R(32)
//	}
//	*param_6 = 0   (aucun record)
//
// L ECRIVAIN BOUCLE ; CETTE MARCHE NE BOUCLE PAS, ET C EST UNE CONSEQUENCE, PAS UN CHOIX. La
// CHARGE d un genre n est pas portee — ce sont 123 grammaires de message, une par type
// (`initiate_mobility_action` en est un, lot 5.13.2) — donc des qu un corps s ouvre, le curseur
// ne peut plus avancer et un second tour serait une lecture inventee. `Porte` le dit, et
// `Genres` ne porte jamais plus d un genre tant que la table des 123 n est pas lue.
func consumeVueA(br *Lecteur, frameLen int) FluxVueA {
	out := FluxVueA{}
	if !placeDisponible(br, frameLen, 1) {
		return out
	}
	if !br.ReadBit() {
		out.Vide, out.Porte = true, true
		return out
	}
	if !placeDisponible(br, frameLen, LargeurGenreVueA) {
		return out
	}
	out.Genres = append(out.Genres, int(br.ReadBits(LargeurGenreVueA)))
	return out
}

// placeDisponible dit si `n` bits tiennent encore dans le payload. Elle existe pour que la
// marche s arrete SUR la frontiere au lieu de lire des zeros au-dela — le debordement que le
// lot 5.11.6 a mesure a 57 bits par paquet.
func placeDisponible(br *Lecteur, frameLen, n int) bool {
	return br.BitPos()+n <= frameLen
}
