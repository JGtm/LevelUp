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
		default:
			return out // kinds 0, 1 et 2 : charge non portee par ce commit
		}
	}
	return out
}

// plafondToursVueC borne la boucle de la vue C. Ce n est PAS une largeur de grammaire : le
// budget de l ecrivain est `param_4` (le nombre de records restants), que la marche hors ligne
// n a pas. Un flux sain sort sur son terminateur bien avant — le maximum mesure est huit tours
// sur `bfecd02b` (huit participants).
const plafondToursVueC = 64
