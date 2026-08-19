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
 *    et la garde de mode doit en faire apparaître ZÉRO. Sans la mesure des 26, l'assertion
 *    « rien ne change » ne prouverait rien : on ne saurait pas si la garde a agi ou si le film
 *    n'avait simplement rien à montrer.
 *
 * LA GARDE DE MODE EST JOUÉE, PAS SIMULÉE : le test passe par `matchFiestaGuard` avec le
 * `mode_ui` que le serveur publie réellement pour chaque match (« Roi de la colline » et
 * « Super Fiesta »), pour que le câblage de la page soit couvert avec le calque.
 */
import { describe, expect, it } from 'vitest'

import type { MatchViewHeader, ReplayEquipmentPlacement } from '@/lib/api/types'

import { type PlacementTime } from './equipmentPlacementsLayer'
import { matchFiestaGuard } from './replayFiesta'
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

/** Un en-tête de Match View minimal : seuls `mode_ui` et `playlist_label` sont lus. */
function header(modeUi: string): MatchViewHeader {
  return {
    dominance_flag: false,
    had_bot_teammate: false,
    is_excluded: false,
    is_favorite: false,
    is_ranked: false,
    map_ui: 'Carte',
    match_id: 'temoin',
    mode_ui: modeUi,
    outcome_color: '',
    outcome_label: '',
    performance_display: '',
    playlist_label: 'Quick Play',
    replay_available: true,
    start_time_label: '',
  }
}

/**
 * Ce que la PAGE dessinerait : la bascule du tiroir (allumée par défaut) croisée avec la garde
 * de mode. C'est exactement l'expression câblée dans `ReplayCanvas`.
 */
function timeFor(modeUi: string): PlacementTime {
  const droppedAllowed = matchFiestaGuard(header(modeUi)) === 'clear'
  return { ...TIME, showDropped: droppedAllowed }
}

describe('témoin 01e1f945 — roi de la colline, Catalyst (hors Fiesta)', () => {
  const scene = poses(KOTH)

  it('le recensement fige bien 151 poses', () => {
    expect(scene).toHaveLength(151)
  })

  it('le mode n’est PAS une Fiesta : les lâchés ont le droit de se dessiner', () => {
    expect(matchFiestaGuard(header('Roi de la colline'))).toBe('clear')
  })

  it('EXACTEMENT une primitive de plus : le surbouclier, et rien des 130 autres lâchers', () => {
    const avant = painted(scene, { ...TIME, showDropped: false })
    const apres = painted(scene, timeFor('Roi de la colline'))
    expect(apres - avant).toBe(1)
  })
})

describe('témoin 000d5950 — Super Fiesta, Cliffhanger', () => {
  const scene = poses(FIESTA)

  it('le recensement fige bien 295 poses', () => {
    expect(scene).toHaveLength(295)
  })

  it('le mode EST une Fiesta, reconnu sur le `mode_ui` que le serveur publie', () => {
    expect(matchFiestaGuard(header('Super Fiesta'))).toBe('fiesta')
  })

  it('RIEN ne change : la garde annule la bascule, à la primitive près', () => {
    const avant = painted(scene, { ...TIME, showDropped: false })
    const apres = painted(scene, timeFor('Super Fiesta'))
    expect(apres).toBe(avant)
  })

  it('et la garde a bien AGI : sans elle, ce film gagnerait 26 marques', () => {
    // 15 capteurs lâchés + 11 murs lâchés. Sans ce chiffre, le test précédent ne prouverait
    // pas que la garde sert à quelque chose sur CE film.
    const avant = painted(scene, { ...TIME, showDropped: false })
    const sansGarde = painted(scene, { ...TIME, showDropped: true })
    expect(sansGarde - avant).toBe(26)
  })
})
