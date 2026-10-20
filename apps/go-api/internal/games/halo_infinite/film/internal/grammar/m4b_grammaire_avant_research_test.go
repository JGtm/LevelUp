//go:build research

package grammar

// m4b_grammaire_avant_research_test.go — LA GRAMMAIRE DU BLOC D ACTION D AVANT LE LOT M4b, pour
// les SEULS instruments historiques qui la comparent a la lecture corrigee (sondes P1 S2 bis et
// P1-S3, `tcg_tir_continu_vuec_research_test.go`, `p1s3_vuec_tir_continu_research_test.go`).
//
// Le lot M4b (2026-09-24) a corrige deux sous-lecteurs de `FUN_1406d025c` (`bloc_action.go`) :
// la production ne porte plus ces deux formes. Les instruments qui mesuraient « la production
// contre la grammaire relue » en ont besoin pour rejouer leur colonne « production » telle
// qu elle etait : elles vivent ici, sous build tag `research`, et nulle part ailleurs.

// consumeQuatBlock1431a0cbc est l ANCIEN modele de `FUN_1431a0cbc` : R(1)[R(1)[R(1)]]. Faux :
// le jeu lit R(2) mode ; 1 -> R(19) ; 0 -> la position quantifiee de niveau 16 ; 2, 3 -> rien.
func consumeQuatBlock1431a0cbc(br *Lecteur) {
	if br.ReadBit() {
		return
	}
	if !br.ReadBit() {
		br.ReadBits(1)
	}
}

// ancienneQueue142f26740 est l ANCIENNE queue du bloc d action : `FUN_140c9e990` lu en
// categorie 0 dans ses deux genres. Faux : genre 1 = categorie 1 (sonde comprise), genre 2 =
// categorie 2.
func ancienneQueue142f26740(br *Lecteur) {
	if !br.ReadBit() {
		return
	}
	switch br.ReadBits(2) {
	case 1:
		readVarWidthInt(br, 0)
		if br.ReadBit() {
			br.ReadBits(6)
		}
	case 2:
		readVarWidthInt(br, 0)
	}
	if !br.ReadBit() {
		br.ReadBits(dequant140c9e4d8Width)
		br.ReadBits(dequant140c9e4d8Width)
		if !br.ReadBit() {
			return
		}
	}
	consume140c9e738(br, false)
}
