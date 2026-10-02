package grammar

// frame_closure_temoins.go — LES TEMOINS DECALES DE LA CARTE DE FERMETURE DETAILLEE.
//
// La vue C se resynchronise d elle-meme : relue depuis un depart faux de quelques bits, elle ferme
// encore le paquet pour une part des paquets fermes. « Ferme au bit pres » n est donc qu un temoin
// de cadrage, et un gain de fermeture ne se lit que contre ce hasard. Pour chaque paquet ferme au
// bit pres, la carte relit la vue C depuis son depart DECALE de k bits, k de -8 a -1 et de 1 a 8,
// et note les k d ou elle referme le paquet sous la meme definition (reste nul, regles de la vue C
// de l ecrivain tenues, `ecrivain_invariants.go`). Les records de la vue B ne sont pas relus : leur
// verdict est celui du paquet.

// DecalageTemoinMax est l amplitude des temoins decales, dans chaque sens.
const DecalageTemoinMax = 8

// BitDeDecalage rend le bit de [PaquetDeCarte.TemoinsDecales] qui porte le decalage `k`
// (k de -8 a -1 : bits 0 a 7 ; k de 1 a 8 : bits 8 a 15).
func BitDeDecalage(k int) int {
	if k < 0 {
		return k + DecalageTemoinMax
	}
	return k + DecalageTemoinMax - 1
}

// temoinsDecales rend les decalages d ou la vue C, qui commence a `debutVueC`, referme le paquet.
func temoinsDecales(pay []byte, cfg FrameConfig, debutVueC int) uint16 {
	var out uint16
	for k := -DecalageTemoinMax; k <= DecalageTemoinMax; k++ {
		if k == 0 || debutVueC+k < 0 {
			continue
		}
		if vueCFermeDepuis(pay, cfg, debutVueC+k) {
			out |= uint16(1) << uint(BitDeDecalage(k)) //nolint:gosec // 0..15
		}
	}
	return out
}

// vueCFermeDepuis relit la vue C depuis `t` : ferme-t-elle le paquet sous la definition de la
// fermeture, regles de la vue C comprises ?
func vueCFermeDepuis(pay []byte, cfg FrameConfig, t int) bool {
	cfg.Obs = nil
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.SetBitPos(t)
	c := consumeVueC(br, len(pay)*8)
	if !c.Porte || !vueCFermee(pay, br.BitPos()) {
		return false
	}
	var j jugeEcrivain
	j.jugerLaVueC(c)
	return j.premiere == InvariantAucun
}
