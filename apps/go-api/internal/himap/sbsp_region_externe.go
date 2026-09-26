// Package himap — sbsp_region_externe.go : les regions de compression d'une carte dont le
// module ne porte AUCUN tag sbsp.
//
// POURQUOI CE FICHIER EXISTE (lot C catalogues, 2026-08-27). Live Fire est prouvee
// level_id -> `sgh_interlock` (TestPreuveLevelIDCartes), mais ce module est SANS tag sbsp
// (`ErrAucunTagSbsp`, mesure du 2026-08-13) — la carte etait donc hors du catalogue de
// bornes et ses films indecodables. Le piege inter-modules documente par moduleindex.go
// (les references d'une carte vivent aussi dans globals/*) vaut pour le BSP : le tag de
// niveau (`levl`) de sgh_interlock reference par GlobalID quatre tags sbsp portes par
// `ds/globals/common`, et DEUX blocs du levl les enumerent dans le MEME ordre — la forme
// exacte du bloc structure-BSP que `ordreRegionsBSP` reconnait.
//
// LE CRITERE RESTE CELUI DU MOTEUR (sbsp_region.go) : l'ordre des regions est celui du bloc
// structure-BSP du levl. Seule la RESOLUTION change : les tags sbsp sont cherches dans des
// modules PORTEURS fournis par l'appelant (les globals de la meme variante), et l'ensemble
// attendu est celui des GlobalID sbsp que le levl reference — quand les sbsp sont dans le
// module, cet ensemble est le meme qu'avant, par construction.
//
// LE CHOIX DE LA REGION N'EST PAS FAIT ICI. Sur les cartes a sbsp locaux, la region jouee
// est la 0 (mesure film : quasi tous les records portent l'index 0). Sur Live Fire, la
// region jouee est la 1 — etabli par TROIS faits independants (test de preuve
// `TestPreuveRegionsLiveFire`) : l'emprise des ancres d'objectifs du catalogue, le plus
// petit AABB (le controle croise historique du catalogue), et l'index 01 porte par
// 59 376/59 377 records i0 de ses 2 films. L'appelant (cmd/mapquant-build) DECLARE la
// region avec sa preuve, comme il declare le module.
package himap

import (
	"errors"
	"levelup/go-api/internal/himodule"

	"fmt"
	"sort"
)

// RegionExterne est une region de compression resolue hors du module de la carte.
type RegionExterne struct {
	BSP BSP
	// Module est le chemin du module PORTEUR du tag sbsp (tracabilite du catalogue).
	Module string
}

// RegionsBSPExternes rend les regions de compression d'une carte dont le module ne porte
// aucun tag sbsp, dans l'ordre du bloc structure-BSP de son tag de niveau. Les tags sbsp
// sont resolus dans les modules porteurs donnes ; toute illisibilite est une ERREUR, jamais
// un ordre partiel silencieux.
func RegionsBSPExternes(modulePath string, porteurs []string) ([]RegionExterne, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = m.Close() }()
	if len(m.Files("sbsp")) > 0 {
		return nil, fmt.Errorf("himap: %s porte ses propres tags sbsp — utiliser BSPQuantification", modulePath)
	}
	parGID, err := sbspDesPorteurs(porteurs)
	if err != nil {
		return nil, err
	}
	attendus := sbspReferences(m, parGID)
	if len(attendus) == 0 {
		return nil, fmt.Errorf("himap: le levl de %s ne reference aucun tag sbsp des %d module(s) porteur(s)",
			modulePath, len(porteurs))
	}
	ordre, err := ordreRegionsBSP(m, attendus)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", modulePath, err)
	}
	out := make([]RegionExterne, 0, len(ordre))
	for _, gid := range ordre {
		for _, b := range attendus {
			if b.GlobalID == gid {
				out = append(out, RegionExterne{BSP: b, Module: parGID[gid].module})
				break
			}
		}
	}
	if len(out) != len(ordre) {
		return nil, fmt.Errorf("himap: %s : %d region(s) ordonnee(s) mais %d resolue(s)",
			modulePath, len(ordre), len(out))
	}
	return out, nil
}

type sbspPorte struct {
	bsp    BSP
	module string
}

// sbspDesPorteurs lit les bornes de TOUS les tags sbsp des modules porteurs. Un porteur
// illisible est une erreur (un index partiel rendrait un ordre faux sans le dire) — SAUF le
// porteur qui ne porte simplement aucun sbsp (`ds/globals/forge`) : il n'apporte rien,
// c'est son etat normal, pas une panne.
func sbspDesPorteurs(porteurs []string) (map[uint32]sbspPorte, error) {
	out := map[uint32]sbspPorte{}
	for _, p := range porteurs {
		bsps, err := ReadModuleBSPBounds(p)
		if errors.Is(err, ErrAucunTagSbsp) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("himap: porteur %s: %w", p, err)
		}
		for _, b := range bsps {
			if _, deja := out[b.GlobalID]; !deja {
				out[b.GlobalID] = sbspPorte{bsp: b, module: p}
			}
		}
	}
	return out, nil
}

// sbspReferences rend, tries par GlobalID, les tags sbsp des porteurs que les blocs de
// donnees des tags levl du module referencent.
func sbspReferences(m *himodule.Module, parGID map[uint32]sbspPorte) []BSP {
	vus := map[uint32]bool{}
	for _, f := range m.Files("levl") {
		tag, err := m.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		for b := 0; b < ti.dataBlocks; b++ {
			abs, size := ti.blockAbs(b)
			if abs < 0 || size <= 0 || abs+size > len(ti.tag) {
				continue
			}
			for p := abs; p+4 <= abs+size; p += 4 {
				gid := uint32(u32(ti.tag, p)) //nolint:gosec // u32 rend deja un mot de 32 bits
				if _, ok := parGID[gid]; ok {
					vus[gid] = true
				}
			}
		}
	}
	out := make([]BSP, 0, len(vus))
	for gid := range vus {
		out = append(out, parGID[gid].bsp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GlobalID < out[j].GlobalID })
	return out
}
