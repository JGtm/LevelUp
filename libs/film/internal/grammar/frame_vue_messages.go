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
//
// # LA LECTURE DE LA VUE A VIT DANS `vue_a_lecture.go` (lot VA)
//
// Elle n y lit plus seulement la tete : la table des 123 genres et les lecteurs de leurs charges
// sont portes (`vue_a_genres.go`, `vue_a_charges*.go`), et [lireLaVueA] lit la vue A message par
// message jusqu a son terminateur quand le film la rend lisible ([FluxVueA]).

// LargeurGenreVueA est la largeur du selecteur de genre d un corps de la vue A
// (`FUN_14080a9d4` : `R(7)` puis `if (genre < 0x7b)` — 123 genres).
const LargeurGenreVueA = 7

// GenresVueA est le nombre de genres que la table de la vue A porte (`FUN_14080a9d4` :
// `*(obj[0x18] + 0x210 + genre * 8)`, garde `genre < 0x7b`).
const GenresVueA = 0x7b

// placeDisponible dit si `n` bits tiennent encore dans le payload. Elle existe pour que la
// marche s arrete SUR la frontiere au lieu de lire des zeros au-dela — le debordement que le
// lot 5.11.6 a mesure a 57 bits par paquet.
func placeDisponible(br *Lecteur, frameLen, n int) bool {
	return br.BitPos()+n <= frameLen
}
