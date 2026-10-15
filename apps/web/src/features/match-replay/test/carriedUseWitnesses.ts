/**
 * carriedUseWitnesses.ts — LES UTILISATIONS D'ÉQUIPEMENT PORTÉ, RELEVÉS RÉELS FIGÉS (schéma 96).
 *
 * POURQUOI CE FICHIER. Les documents de `fixtures/go/` portent l'équipement porté des huit builds,
 * mais aucun traqueur de menaces. Les deux documents ci-dessous sont ceux que la cuisson au schéma 96
 * publie, relevés le 2026-10-15 par une cuisson unitaire de chaque film :
 *
 *  - `5676a9ba` (Insolence) : détecteur, mur, écran occultant, champ de réparation ;
 *  - `1f8e8849` (Aquarius - Ranked) : traqueur de menaces.
 *
 * N'y figurent que les objets de ces familles qui ont au moins une utilisation, tels que le
 * document les publie (énergie arrondie au dix-millième). Chaque utilisation est suivie, une ou deux
 * images plus tard, d'une baisse d'énergie d'exactement un coût de la famille (30 sur 30) ; chaque
 * utilisation du mur précède de deux ou trois images une pose `deployed` du même porteur (les poses
 * `POSES_DU_MUR_565` en sont deux). `tracked` : les objets que l'équipement engendre (pile i28) et leur
 * suppression lue — pour le champ de réparation, sa fin.
 */
import { testReplayDoc } from './testDoc'

const LIBELLES_5676A9BA = {
  '1': { en: 'Threat Sensor', fr: 'detecteur de menaces', family: 'sensor' },
  '2': { en: 'Drop Wall', fr: 'mur de protection', family: 'wall' },
  '10': { en: 'Shroud Screen', fr: 'ecran occultant', family: 'shroud_screen' },
  '23': { en: 'Repair Field', fr: 'champ de reparation', family: 'repair_field' },
}

/** Les objets portés de `5676a9ba` qui ont au moins une utilisation. */
export const PORTES_5676A9BA = [
  { slot: 539, r: 23, t0: 657, t1: 1106, energy: [{ t: 657, e: 1, n: 2 }, { t: 941, e: 0.5, n: 1 }, { t: 1093, e: 0, n: 0 }], uses: [941, 1093], tracked: [{ t0: 943, t1: 1060 }, { t0: 1096, t1: 1113 }] },
  { slot: 557, r: 2, t0: 1231, t1: 1408, energy: [{ t: 1231, e: 0.9901 }], uses: [1406], tracked: [] },
  { slot: 565, r: 2, t0: 1486, t1: 1589, energy: [{ t: 1486, e: 0.6601 }, { t: 1520, e: 0.3301 }, { t: 1578, e: 0.0001 }], uses: [1518, 1576], tracked: [{ t0: 1521, t1: 1574 }, { t0: 1578, t1: 1602 }] },
  { slot: 570, r: 1, t0: 1518, t1: 1712, energy: [{ t: 1518, e: 1, n: 4 }, { t: 1602, e: 0.75, n: 3 }, { t: 1711, e: 0.5, n: 2 }], uses: [1601, 1710], tracked: [{ t0: 1602, t1: 1762 }, { t0: 1711, t1: 1864 }] },
  { slot: 571, r: 23, t0: 1621, t1: 1808, energy: [{ t: 1621, e: 1, n: 2 }, { t: 1709, e: 0.5, n: 1 }, { t: 1793, e: 0, n: 0 }], uses: [1709, 1793], tracked: [{ t0: 1711, t1: 1831 }, { t0: 1796, t1: 1815 }] },
  { slot: 565, r: 1, t0: 1768, t1: 1835, energy: [{ t: 1768, e: 0.5, n: 2 }, { t: 1778, e: 0.25, n: 1 }, { t: 1823, e: 0, n: 0 }], uses: [1776, 1822], tracked: [{ t0: 1778, t1: 1940 }, { t0: 1823, t1: 1984 }] },
  { slot: 580, r: 2, t0: 1839, t1: 2658, energy: [{ t: 1839, e: 0.9901 }, { t: 2626, e: 0.6601 }], uses: [2623], tracked: [{ t0: 2626, t1: 2693 }] },
  { slot: 575, r: 23, t0: 2450, t1: 3047, energy: [{ t: 2450, e: 1, n: 2 }, { t: 2456, e: 0.5, n: 1 }], uses: [2456], tracked: [{ t0: 2458, t1: 2580 }] },
  { slot: 608, r: 1, t0: 2680, t1: 2944, energy: [{ t: 2680, e: 1, n: 4 }, { t: 2701, e: 0.75, n: 3 }, { t: 2763, e: 0.5, n: 2 }, { t: 2871, e: 0.25, n: 1 }], uses: [2700, 2761, 2870], tracked: [{ t0: 2701, t1: 2721 }, { t0: 2763, t1: 2919 }, { t0: 2871, t1: 3029 }] },
  { slot: 622, r: 1, t0: 3072, t1: 3095, energy: [{ t: 3072, e: 0.25, n: 1 }, { t: 3095, e: 0, n: 0 }], uses: [3093], tracked: [{ t0: 3095, t1: 3103 }] },
  { slot: 634, r: 1, t0: 3302, t1: 3422, energy: [{ t: 3302, e: 1, n: 4 }, { t: 3404, e: 0.75, n: 3 }], uses: [3403], tracked: [{ t0: 3404, t1: 3617 }] },
  { slot: 634, r: 23, t0: 3422, t1: 3532, energy: [{ t: 3422, e: 1, n: 2 }, { t: 3445, e: 0.5, n: 1 }, { t: 3518, e: 0, n: 0 }], uses: [3445, 3518], tracked: [{ t0: 3447, t1: 3617 }, { t0: 3520, t1: 3641 }] },
  { slot: 663, r: 23, t0: 4266, t1: 4806, energy: [{ t: 4266, e: 1, n: 2 }, { t: 4433, e: 0.5, n: 1 }], uses: [4433], tracked: [{ t0: 4436, t1: 4554 }] },
  { slot: 686, r: 1, t0: 4477, t1: 4889, energy: [{ t: 4477, e: 1, n: 4 }, { t: 4658, e: 0.75, n: 3 }, { t: 4763, e: 0.5, n: 2 }], uses: [4657, 4762], tracked: [{ t0: 4658, t1: 4814 }, { t0: 4763, t1: 4918 }] },
  { slot: 725, r: 1, t0: 5459, t1: 5564, energy: [{ t: 5459, e: 1, n: 4 }, { t: 5484, e: 0.75, n: 3 }, { t: 5529, e: 0.5, n: 2 }], uses: [5483, 5528], tracked: [{ t0: 5484, t1: 5648 }, { t0: 5529, t1: 5689 }] },
  { slot: 735, r: 23, t0: 5697, t1: 5898, energy: [{ t: 5697, e: 1, n: 2 }, { t: 5753, e: 0.5, n: 1 }, { t: 5884, e: 0, n: 0 }], uses: [5753, 5884], tracked: [{ t0: 5755, t1: 5869 }, { t0: 5887, t1: 5928 }] },
  { slot: 753, r: 1, t0: 6167, t1: 6618, energy: [{ t: 6167, e: 1, n: 4 }, { t: 6299, e: 0.75, n: 3 }], uses: [6298], tracked: [{ t0: 6299, t1: 6453 }] },
  { slot: 757, r: 10, t0: 6287, t1: 6531, energy: [{ t: 6287, e: 1, n: 4 }, { t: 6308, e: 0.75, n: 3 }], uses: [6307], tracked: [{ t0: 6308, t1: 6315 }] },
  { slot: 713, r: 23, t0: 6303, t1: 6613, energy: [{ t: 6303, e: 1, n: 2 }, { t: 6325, e: 0.5, n: 1 }], uses: [6325], tracked: [{ t0: 6327, t1: 6446 }] },
]

/** Les deux poses `deployed` du mur que porte 565 dans `5676a9ba`, deux images après ses deux utilisations. */
export const POSES_DU_MUR_565 = [
  { t0: 1520, t1: 1524, x: -17.889069, y: -8.537842, z: 67.82031, family: 'wall', id: '0x686b40c9', owner: 565, h: 71.674805, origin: 'deployed', until: 1557, untilMax: 1757, end: 'seen' },
  { t0: 1578, t1: 1583, x: -18.030258, y: -9.451126, z: 67.321594, family: 'wall', id: '0x686b40c9', owner: 565, h: 72.81738, origin: 'deployed', until: 1578, untilMax: 1757, end: 'seen' },
]

/** Les objets portés de `1f8e8849` qui ont au moins une utilisation. */
export const PORTES_1F8E8849 = [
  { slot: 549, r: 12, t0: 1534, t1: 1635, energy: [{ t: 1534, e: 0.2501, n: 1 }, { t: 1623, e: 0.0001, n: 0 }], uses: [1622], tracked: [{ t0: 1623, t1: 1636 }] },
  { slot: 556, r: 12, t0: 2270, t1: 2460, energy: [{ t: 2270, e: 0.2501, n: 1 }, { t: 2448, e: 0.0001, n: 0 }], uses: [2447], tracked: [{ t0: 2448, t1: 2457 }] },
]

/** Le document témoin de `5676a9ba`, réduit à son équipement porté (et, sur demande, aux poses du mur de 565). */
export function doc5676a9ba(avecPosesDuMur = false) {
  return testReplayDoc({
    frameCount: 6618,
    frameIntervalMs: 100,
    abilityLabels: LIBELLES_5676A9BA,
    carriedEquipment: PORTES_5676A9BA,
    equipmentPlacements: avecPosesDuMur ? POSES_DU_MUR_565 : [],
  })
}

/** Le document témoin de `1f8e8849`, réduit à son équipement porté. */
export function doc1f8e8849() {
  return testReplayDoc({
    frameCount: 4039,
    frameIntervalMs: 100,
    abilityLabels: { '12': { en: 'Threat Seeker', fr: 'traqueur de menaces', family: 'threat_seeker' } },
    carriedEquipment: PORTES_1F8E8849,
  })
}

/** Les documents témoins, pour les garde-rails qui exigent un déclencheur réel. */
export function documentsTemoinsDeLEquipementPorte() {
  return [doc5676a9ba(), doc1f8e8849()]
}
