// Package himap — sbsp_region.go : QUEL tag sbsp d'un module porte les bornes contre
// lesquelles le moteur quantifie les positions repliquees.
//
// POURQUOI CE FICHIER EXISTE. `ReadModuleBSPBounds` rend TOUS les tags sbsp d'un module,
// tries par TAILLE DE TAG decroissante, et `cmd/mapquant-build` retenait le premier. La
// taille du tag n'est pas un critere : elle mesure la quantite de geometrie compilee, pas le
// role du BSP. Sur les six canevas Forge de l'installation (`fo03_space`, `fo05_desert`,
// `fo06_deepsea`, `fo10_deadland`, `fo11_blank`, `fo13_frost`) le plus gros tag est le DECOR
// LOINTAIN — 3 867 x 3 662 x 2 664 unites — quand le jeu quantifie dans l'arene de
// 463 x 453 x 1 189. Mesure du 2026-08-16 : bornes -> [18 18 18], films -> [15 15 17], sur
// 12 films de trois canevas. Les deux canevas ou la chaine tombait juste (`fo08_wetland`,
// `fo09_academy`) y tombaient PAR HASARD : leur arene pese plus d'octets que leur decor.
//
// LE CRITERE, ET IL EST CELUI DU MOTEUR. Au chargement de carte, FUN_140be9a14 parcourt le
// bloc structure-BSP du tag de SCENARIO (`levl`) et precalcule W[r][L][axe] pour chaque
// REGION de compression r, dans l'ordre de ce bloc. Le composant i0 transporte ensuite un
// index de region de ceilLog2(nb de BSP) bits — quasi toujours 0 (mesure film : 3 records sur
// 291 288 portent 1 sur Cliffhanger). La region 0, donc le PREMIER sbsp du bloc structure-BSP
// du scenario, est le repere de dequantification. Ce n'est pas une heuristique sur les
// bornes : c'est l'ordre que le moteur lit lui-meme.
//
// COMMENT ON LE LIT SANS PLUGIN `levl`. Aucun plugin de scenario n'est embarque et on n'en
// ajoute pas un pour deux champs. Le bloc se reconnait a sa FORME : c'est un bloc de donnees
// du tag `levl` dont les enregistrements portent, chacun, le GlobalID d'un tag sbsp du module,
// et dont l'ensemble des GlobalID distincts couvre EXACTEMENT les tags sbsp presents. Les
// blocs qui satisfont ce signalement doivent tous donner le MEME ordre, sinon on refuse —
// jamais de repli silencieux sur la taille, qui est precisement le defaut corrige ici.
package himap

import (
	"fmt"

	"levelup/go-api/internal/himodule"
)

// ErrRegionBSPIndecidable signale que l'ordre des regions de compression n'a pas pu etre lu
// dans le tag de niveau alors que le module porte plusieurs tags sbsp. L'appelant doit
// s'ARRETER sur cette carte : produire des bornes prises au hasard parmi les candidats est
// exactement le defaut que ce fichier corrige.
var ErrRegionBSPIndecidable = fmt.Errorf("himap: ordre des regions de compression illisible")

// BSPQuantification rend le tag sbsp contre lequel le moteur quantifie les positions
// repliquees de ce module — la region 0 du bloc structure-BSP du scenario — ET la liste
// complete des candidats, triee par taille de tag decroissante.
//
// Les candidats sont rendus pour que l'appelant puisse DIRE ce qu'il a ecarte (un catalogue
// qui change de bornes sans le tracer est un catalogue qu'on ne peut pas relire). Le module
// n'est ouvert qu'UNE fois : certains pesent plusieurs centaines de Mo.
func BSPQuantification(modulePath string) (BSP, []BSP, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return BSP{}, nil, err
	}
	bsps, err := bspBoundsFrom(m, modulePath)
	if err != nil {
		return BSP{}, nil, err
	}
	if len(bsps) == 1 {
		return bsps[0], bsps, nil
	}
	ordre, err := ordreRegionsBSP(m, bsps)
	if err != nil {
		return BSP{}, bsps, fmt.Errorf("%s: %w", modulePath, err)
	}
	for _, b := range bsps {
		if b.GlobalID == ordre[0] {
			return b, bsps, nil
		}
	}
	return BSP{}, bsps, fmt.Errorf("%s: %w (region 0 = %08x, absent des tags sbsp)",
		modulePath, ErrRegionBSPIndecidable, ordre[0])
}

// ordreRegionsBSP rend les GlobalID des tags sbsp dans l'ordre des regions de compression
// declare par le tag de niveau.
func ordreRegionsBSP(m *himodule.Module, bsps []BSP) ([]uint32, error) {
	attendus := map[uint32]bool{}
	for _, b := range bsps {
		attendus[b.GlobalID] = true
	}
	levls := m.Files("levl")
	if len(levls) == 0 {
		return nil, fmt.Errorf("%w: aucun tag levl", ErrRegionBSPIndecidable)
	}
	var retenu []uint32
	for _, f := range levls {
		tag, err := m.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		for _, ordre := range ordresDansBlocs(ti, attendus) {
			if retenu == nil {
				retenu = ordre
				continue
			}
			if !memeOrdre(retenu, ordre) {
				return nil, fmt.Errorf("%w: deux blocs du levl donnent des ordres differents (%08x... / %08x...)",
					ErrRegionBSPIndecidable, retenu[0], ordre[0])
			}
		}
	}
	if retenu == nil {
		return nil, fmt.Errorf("%w: aucun bloc du levl ne reference les %d tags sbsp",
			ErrRegionBSPIndecidable, len(attendus))
	}
	return retenu, nil
}

// ordresDansBlocs rend, pour chaque bloc de donnees du tag qui reference EXACTEMENT l'ensemble
// des tags sbsp attendus, l'ordre de premiere apparition de leurs GlobalID.
func ordresDansBlocs(ti tagInfo, attendus map[uint32]bool) [][]uint32 {
	var out [][]uint32
	for b := 0; b < ti.dataBlocks; b++ {
		abs, size := ti.blockAbs(b)
		if abs < 0 || size <= 0 || abs+size > len(ti.tag) {
			continue
		}
		var ordre []uint32
		vus := map[uint32]bool{}
		for p := abs; p+4 <= abs+size; p += 4 {
			gid := uint32(u32(ti.tag, p)) //nolint:gosec // u32 rend deja un mot de 32 bits
			if !attendus[gid] || vus[gid] {
				continue
			}
			vus[gid] = true
			ordre = append(ordre, gid)
		}
		if len(ordre) == len(attendus) {
			out = append(out, ordre)
		}
	}
	return out
}

func memeOrdre(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
