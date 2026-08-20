/**
 * Tests — LES DEUX TÉMOINS DU LOT « objets lâchés », rejoués sur leur recensement RÉEL.
 *
 * POURQUOI UN RECENSEMENT ET PAS L'ARTEFACT. Les artefacts de rejeu vivent dans
 * `data/cache/replays/halo_infinite/` et ne sont pas versionnés : un test qui les lirait ne
 * tournerait ni en CI ni dans un arbre de travail frais. La méthode du dépôt est donc celle
 * qu'emploient déjà `killFeedLogic.test.ts` et `mapBackground.test.ts` — MESURER hors ligne,
 * puis figer le chiffre mesuré dans le test. Les deux tableaux ci-dessous sont le décompte
 * `famille/origine` exact des deux films, relevé le 2026-08-19.
 *
 * CE QUE CES DEUX TÉMOINS PROUVENT, ET QU'AUCUN TEST UNITAIRE NE PROUVE :
 *  - `01e1f945` (roi de la colline, Catalyst) porte UNE pose de puissance lâchée dans 151
 *    poses — un surbouclier — noyée dans 108 grenades à fragmentation et 11 répulseurs. Le
 *    lot doit en faire apparaître EXACTEMENT une marque de plus, pas 130 ;
 *  - `000d5950` (Super Fiesta, Cliffhanger) en porte 26 (15 capteurs, 11 murs) sur 295 poses,
 *    et ces 26 marques DOIVENT apparaître quand la bascule est allumée.
 *
 * LA GARDE DE MODE A DISPARU LE 2026-08-20 (demande utilisateur : « je veux les voir »). Elle
 * masquait ces 26 lâchers réels au nom d'une décision produit du 18/08 ; le réglage
 * « Objets lâchés au sol » gouverne désormais seul, dans TOUS les modes. Ce témoin mesure
 * donc exactement l'inverse de ce qu'il mesurait : les 26 marques sont ce que la Fiesta gagne.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { type PlacementTime } from './equipmentPlacementsLayer'
import { DEVICE_ID, OVERSHIELD_ID, painted, PANEL_ID, SENSOR_ID, TIME } from './test/placementFixtures'

/** Un recensement : `famille/origine` -> nombre de poses, tel que relevé sur l'artefact. */
type Census = Record<string, number>

/**
 * `01e1f945` — ROI DE LA COLLINE sur Catalyst, schéma 16, 5 343 images, 151 poses.
 * Le surbouclier lâché y est à l'image 4 104, à 0,24 / 6,80 en monde, poseur au slot 589.
 */
const KOTH: Census = {
  'grapple/dropped': 9,
  'grenade_frag/deployed': 5,
  'grenade_frag/dropped': 108,
  'grenade_plasma/dropped': 6,
  'grenade_spike/dropped': 5,
  'other/dropped': 4,
  'powerup_overshield/dropped': 1,
  'repulsor/deployed': 2,
  'repulsor/dropped': 11,
}

/**
 * `000d5950` — SUPER FIESTA sur Cliffhanger, schéma 17, 4 985 images, 295 poses.
 *
 * Les 17 murs déployés se partagent en 15 PANNEAUX (`0x528fce46`, l'arc) et 2 appareils ; les
 * 11 murs lâchés portent tous l'identifiant de l'APPAREIL. C'est ce détail qui interdisait de
 * réutiliser la forme de la famille pour un lâcher.
 */
const FIESTA: Census = {
  'grapple/deployed': 1,
  'grapple/dropped': 29,
  'grenade_dynamo/deployed': 2,
  'grenade_dynamo/dropped': 27,
  'grenade_frag/deployed': 3,
  'grenade_frag/dropped': 43,
  'grenade_frag/unknown': 2,
  'grenade_plasma/deployed': 4,
  'grenade_plasma/dropped': 45,
  'grenade_spike/deployed': 4,
  'grenade_spike/dropped': 53,
  'grenade_spike/unknown': 2,
  'sensor/deployed': 4,
  'sensor/dropped': 15,
  'thruster/deployed': 4,
  'thruster/dropped': 27,
  'thruster/unknown': 2,
  'wall/deployed': 17,
  'wall/dropped': 11,
}

/** L'identifiant qu'une famille publie — panneaux pour un mur déployé, appareil sinon. */
function idOf(family: string, origin: string): string {
  if (family === 'wall') return origin === 'deployed' ? PANEL_ID : DEVICE_ID
  if (family === 'sensor') return SENSOR_ID
  if (family === 'powerup_overshield') return OVERSHIELD_ID
  return '0x00000000'
}

/** Le recensement déplié en poses. Positions étalées : elles ne changent aucun compte. */
function poses(census: Census): ReplayEquipmentPlacement[] {
  const out: ReplayEquipmentPlacement[] = []
  for (const [key, n] of Object.entries(census)) {
    const [family, origin] = key.split('/')
    for (let i = 0; i < n; i++) {
      out.push({
        t0: 10,
        t1: 100,
        x: 1 + ((out.length * 7) % 8),
        y: 1 + ((out.length * 3) % 8),
        family,
        id: idOf(family, origin),
        owner: 3,
        origin,
      })
    }
  }
  return out
}

/**
 * Ce que la PAGE dessinerait : la bascule du tiroir, ALLUMÉE par défaut, et rien d'autre.
 * C'est exactement l'expression câblée dans `ReplayCanvas` depuis le retrait de la garde de
 * mode — le mode du match n'entre plus dans le calcul.
 */
const AVEC_LACHERS: PlacementTime = { ...TIME, showDropped: true }
const SANS_LACHERS: PlacementTime = { ...TIME, showDropped: false }

describe('témoin 01e1f945 — roi de la colline, Catalyst', () => {
  const scene = poses(KOTH)

  it('le recensement fige bien 151 poses', () => {
    expect(scene).toHaveLength(151)
  })

  it('EXACTEMENT une primitive de plus : le surbouclier, et rien des 130 autres lâchers', () => {
    const avant = painted(scene, SANS_LACHERS)
    const apres = painted(scene, AVEC_LACHERS)
    expect(apres - avant).toBe(1)
  })
})

describe('témoin 000d5950 — Super Fiesta, Cliffhanger', () => {
  const scene = poses(FIESTA)

  it('le recensement fige bien 295 poses', () => {
    expect(scene).toHaveLength(295)
  })

  /**
   * LES 26 LÂCHERS DE LA FIESTA SONT VISIBLES — c'est le changement du 2026-08-20. La garde de
   * mode les masquait tous ; le réglage les rend, et ce chiffre est celui du film réel :
   * 15 capteurs lâchés + 11 murs lâchés.
   */
  it('la Fiesta gagne bien ses 26 marques : le mode ne masque plus rien', () => {
    const avant = painted(scene, SANS_LACHERS)
    const apres = painted(scene, AVEC_LACHERS)
    expect(apres - avant).toBe(26)
  })
})
