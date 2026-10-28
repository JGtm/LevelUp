/**
 * spawnedObjectWitnesses.ts — UN EXTRAIT RÉEL PAR FAMILLE DESSINÉE, schéma 105 (patron de
 * `carriedUseWitnesses.ts`).
 *
 * POURQUOI. Une famille de `PLACEMENT_RENDER` qui a une règle de dessin doit se dessiner sur un
 * document REEL, pas sur une fixture écrite à la main : c'est ce qui a manqué quand le capteur, le
 * traqueur, l'écran et le champ ont cessé de s'afficher (la règle d'origine du 15/09/2026 les avait
 * tous classés « lâchers », et leurs tests, bâtis sur des poses `deployed` de fixture, restaient
 * verts). Chaque extrait ci-dessous est tiré, sans modification, d'un document cuit le 2026-10-28
 * par le code Go du schéma 105 ; n'y figurent que les objets utiles au témoin.
 *
 *  - capteur : `2ba491cf` (Recharge), porteur slot 543, rang 22, deux premiers objets situés ;
 *  - champ de réparation : `5676a9ba` (Insolence), slot 539, rang 23, deux objets situés ;
 *  - écran occultant : `5676a9ba`, slot 757, rang 10, un objet situé ;
 *  - traqueur : `bc0bdda3` (Forbidden), slot 513, rang 12, deux objets situés ;
 *  - mur : `5676a9ba`, ses trois poses `deployed` (panneaux) ;
 *  - faille : `a0c36016` (Forest), la première pose de faille (slot 520, rang 11) et la première
 *    téléportation du même porteur.
 */
import { testReplayDoc } from './testDoc'

/** Un extrait : le document réduit à ce que la famille publie. */
export const SPAWNED_WITNESSES = {
  sensor: () =>
    testReplayDoc({
      frameCount: 4501,
      frameIntervalMs: 100,
      abilityLabels: { '22': { en: 'Threat Sensor', fr: 'detecteur de menaces', family: 'sensor' } },
      carriedEquipment: [
        {
          slot: 543,
          r: 22,
          t0: 2103,
          t1: 2439,
          tracked: [
            { t0: 2169, t1: 2252, land: 2176, x: 15.2, y: -12.26, z: 3.24 },
            { t0: 2252, t1: 2313, land: 2258, x: 17.75, y: -9.69, z: 3.85 },
          ],
        },
      ],
    }),
  repair_field: () =>
    testReplayDoc({
      frameCount: 6618,
      frameIntervalMs: 100,
      abilityLabels: { '23': { en: 'Repair Field', fr: 'champ de reparation', family: 'repair_field' } },
      carriedEquipment: [
        {
          slot: 539,
          r: 23,
          t0: 657,
          t1: 1106,
          tracked: [
            { t0: 943, t1: 1060, land: 948, x: -14.57, y: 4.61, z: 64.92 },
            { t0: 1096, t1: 1113, land: 1098, x: -13.22, y: 9.01, z: 64.26 },
          ],
        },
      ],
    }),
  shroud_screen: () =>
    testReplayDoc({
      frameCount: 6618,
      frameIntervalMs: 100,
      abilityLabels: { '10': { en: 'Shroud Screen', fr: 'ecran occultant', family: 'shroud_screen' } },
      carriedEquipment: [
        {
          slot: 757,
          r: 10,
          t0: 6287,
          t1: 6531,
          tracked: [{ t0: 6308, t1: 6315, land: 6313, x: -1.34, y: 5.88, z: 65.6 }],
        },
      ],
    }),
  threat_seeker: () =>
    testReplayDoc({
      frameCount: 6464,
      frameIntervalMs: 100,
      abilityLabels: { '12': { en: 'Threat Seeker', fr: 'traqueur de menaces', family: 'threat_seeker' } },
      carriedEquipment: [
        {
          slot: 513,
          r: 12,
          t0: 380,
          t1: 557,
          tracked: [
            { t0: 452, t1: 469, land: 468, x: -7.26, y: 5.85, z: 2.79 },
            { t0: 545, t1: 565, land: 561, x: 7.52, y: 1.96, z: 3.32 },
          ],
        },
      ],
    }),
  wall: () =>
    testReplayDoc({
      frameCount: 6618,
      frameIntervalMs: 100,
      equipmentPlacements: [
        { t0: 1520, t1: 1524, x: -17.889069, y: -8.537842, z: 67.82031, family: 'wall', id: '0x686b40c9', owner: 565, h: 71.674805, origin: 'deployed', until: 1557, untilMax: 1757, end: 'seen' },
        { t0: 1578, t1: 1583, x: -18.030258, y: -9.451126, z: 67.321594, family: 'wall', id: '0x686b40c9', owner: 565, h: 72.81738, origin: 'deployed', until: 1578, untilMax: 1757, end: 'seen' },
        { t0: 2626, t1: 2629, x: 21.191055, y: -24.520294, z: 67.947266, family: 'wall', id: '0x686b40c9', owner: 580, h: 249.56543, origin: 'deployed', until: 2626, untilMax: 2758, end: 'seen' },
      ],
    }),
  translocator_beacon: () =>
    testReplayDoc({
      frameCount: 7684,
      frameIntervalMs: 100,
      abilityLabels: {
        '11': { en: 'Quantum Translocator', fr: 'translocateur quantique', family: 'translocator_beacon' },
      },
      riftPoses: [{ t: 555, a: 539, slot: 520, x: -18.814835, y: -4.859276, z: -0.6392288, src: 'objet', t1: 858 }],
      translocations: [
        { t: 690, slot: 520, fx: -24.54, fy: -11.67, fz: 1.84, tx: -18.8, ty: -4.88, tz: -0.18 },
      ],
    }),
} as const

/** Les familles qui ont un témoin réel. */
export type SpawnedWitnessFamily = keyof typeof SPAWNED_WITNESSES
