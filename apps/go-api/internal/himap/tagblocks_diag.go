// Package himap — tagblocks_diag.go : lecture generique des tag-blocks racine d'un tag deja
// decompresse (tout groupe), pour la reconnaissance de structure.
//
// POURQUOI. L'assemblage parent/enfant d'un vehicule (chassis + tourelle/canon, cf.
// objet_isole.go/RenduAssemblage) a demande de cartographier le render_model du chassis : ou
// sont les regions, les sections, les NOEUDS (squelette), les materiaux. Ces deux helpers ont
// servi cette reconnaissance (sonde `vehicle-sprite diag -roots/-hexroot`) et etabli le fait
// central : le render_model d'un chassis de vehicule NE PORTE PAS de bloc « marker groups »
// (offsets vides) ; le pivot de tourelle vit dans les NOEUDS (bloc a l'offset racine 64), et la
// pose de repos de l'objet-enfant est deja bakee dans le repere local du vehicule — d'ou un
// assemblage a translation nulle correct, sans marqueur a extraire.
package himap

import "fmt"

// RootBlocksOfTag rend l'inventaire des tag-blocks racine d'un tag deja decompresse (diagnostic
// generique, tout groupe) : field_offset, data-block cible, taille et compteur. Le premier
// candidat structurellement valide fait foi. Les noms de plugin restent ceux du sbsp (sans
// valeur hors sbsp) ; ce sont les OFFSETS et TAILLES qui servent.
func RootBlocksOfTag(tag []byte) ([]RootBlockInfo, error) {
	for _, ti := range tagCandidates(tag) {
		if _, _, err := ti.rootBlockRefs(); err != nil {
			continue
		}
		if info, err := ti.describeRootBlocks(); err == nil {
			return info, nil
		}
	}
	return nil, fmt.Errorf("himap: en-tête de tag non reconnu")
}

// RawRootBlock rend les octets bruts du data-block designe par un champ tag-block racine
// (field_offset). Diagnostic : mesurer le pas d'un enregistrement sur pieces.
func RawRootBlock(tag []byte, fieldOffset int) ([]byte, error) {
	for _, ti := range tagCandidates(tag) {
		offs, targets, err := ti.rootBlockRefs()
		if err != nil {
			continue
		}
		for i, o := range offs {
			if o != fieldOffset {
				continue
			}
			abs, size := ti.blockAbs(targets[i])
			if abs < 0 || size <= 0 || abs+size > len(ti.tag) {
				return nil, fmt.Errorf("himap: bloc hors borne (abs=%d size=%d)", abs, size)
			}
			b := make([]byte, size)
			copy(b, ti.tag[abs:abs+size])
			return b, nil
		}
	}
	return nil, fmt.Errorf("himap: champ racine %d introuvable", fieldOffset)
}
