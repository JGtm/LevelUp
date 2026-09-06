/**
 * Tests — equipmentUsageColumns : LA PARTITION DU REPLI « GAME CHANGERS » (plan 2026-09-05).
 *
 * CE QU'ILS PROTÈGENT, mutation par mutation (G1.3 du plan) :
 *   - la PARTITION n'est pas inversée : les élus du vote en avant, le reste replié — et
 *     l'ordre écrit des tables de référence SURVIT à l'intérieur de chaque partition ;
 *   - le PONT D5 est employé : les épisodes `camo`/`overshield` (vocabulaire d'ÉPISODE) sont
 *     en avant PARCE QUE `powerup_camo`/`powerup_overshield` (vocabulaire de SOCLE) sont élus
 *     — retirer le pont les fait tomber du bloc en avant ;
 *   - les GRENADES restent TOUJOURS visibles (décision D4), jamais dans le bloc replié ;
 *   - zéro colonne repliée = compte à zéro : le rendu n'affiche alors AUCUN bouton.
 *
 * La partition est une HIÉRARCHIE D'AFFICHAGE : elle ne touche ni aux mesures ni aux totaux
 * (`EquipmentUsage` ne passe pas par elle) — c'est éprouvé chez `equipmentUsageLogic.test.ts`.
 * Les fixtures passent par `testReplayDoc`, la seule porte du document de test.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import {
  partitionUsageGroups,
  uniqueUsageGroups,
  usageColumnGroups,
  type UsageColumnGroup,
} from './equipmentUsageColumns'
import { buildEquipmentUsage } from './equipmentUsageLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { testReplayDoc } from '../test/testDoc'

const t = REPLAY_TEXT.fr

const SB: MatchScoreboardRow[] = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Bravo', team_side: 't1' },
] as MatchScoreboardRow[]

/** Une vie : le slot, son propriétaire, et deux points pour que la fenêtre existe. */
function vie(slot: number, xuid: string) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 1, y: 1 },
    ],
  }
}

/** Une pose d'équipement (les champs que l'agrégation lit). */
function pose(family: string, origin: string, owner: number, id = '0xaaaa') {
  return { family, origin, owner, id, t0: 10, t1: 20, x: 0, y: 0 }
}

/** Les groupes de colonnes du document donné, par le VRAI pipeline (aucune colonne en dur). */
function groupesDe(over: Partial<ReplayDocument>): UsageColumnGroup[] {
  const doc = testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
      { filmIndex: 1, xuid: 'b1', name: 'Bravo' },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'b1')],
    ...over,
  } as Partial<ReplayDocument>)
  return usageColumnGroups(buildEquipmentUsage(doc, SB), doc, t, 'fr')
}

/**
 * LE TÉMOIN COMPLET : un geste par canal, avec dans chaque canal mixte (poses déployées,
 * objets lâchés) des familles ÉLUES et des familles REPLIÉES — c'est lui qui rend une
 * partition inversée, ou un tri parasite, immédiatement visibles.
 */
const TEMOIN: Partial<ReplayDocument> = {
  grappleLines: [{ slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 }],
  equipmentEpisodes: [
    { slot: 1, fam: 'camo', t0: 10, t1: 60 },
    { slot: 2, fam: 'overshield', t0: 0, t1: 30 },
  ],
  equipmentPlacements: [
    // Déployées — deux élues (sensor, threat_seeker), deux repliées (wall, translocator).
    pose('sensor', 'deployed', 1),
    pose('threat_seeker', 'deployed', 2),
    pose('wall', 'deployed', 1, '0x528fce46'),
    pose('translocator_beacon', 'deployed', 2),
    // Lâchées — une élue (powerup_overshield), deux repliées (wall, repair_field).
    pose('powerup_overshield', 'dropped', 1),
    pose('wall', 'dropped', 2),
    pose('repair_field', 'dropped', 1),
  ],
  grenades: [{ slot: 1, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
  grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
} as unknown as Partial<ReplayDocument>

/** Les clés de colonnes d'un groupe d'une partition, ou [] s'il n'y figure pas. */
function colonnes(groupes: UsageColumnGroup[], key: string): string[] {
  return groupes.find((g) => g.key === key)?.columns.map((c) => c.key) ?? []
}

describe('partitionUsageGroups — les élus en avant, le reste replié', () => {
  const partition = partitionUsageGroups(groupesDe(TEMOIN))

  it('met EN AVANT les familles élues, dans l’ordre écrit des groupes', () => {
    expect(partition.forward.map((g) => g.key)).toEqual([
      'episodes',
      'deployed',
      'dropped',
      'grenades',
    ])
    // L'ordre INTERNE de PLACEMENT_RENDER survit dans la partition (sensor avant seeker).
    expect(colonnes(partition.forward, 'deployed')).toEqual([
      'deployed.sensor',
      'deployed.threat_seeker',
    ])
    expect(colonnes(partition.forward, 'dropped')).toEqual(['dropped.powerup_overshield'])
  })

  it('REPLIE le grappin et les familles votées non, dans le même ordre écrit', () => {
    expect(partition.collapsed.map((g) => g.key)).toEqual(['grapple', 'deployed', 'dropped'])
    // L'ordre INTERNE survit aussi côté replié (wall avant translocator, wall avant field).
    expect(colonnes(partition.collapsed, 'deployed')).toEqual([
      'deployed.wall',
      'deployed.translocator_beacon',
    ])
    expect(colonnes(partition.collapsed, 'dropped')).toEqual([
      'dropped.wall',
      'dropped.repair_field',
    ])
  })

  it('compte les colonnes masquées — le N du bouton « Voir plus (N) »', () => {
    // 1 grappin + 2 déployées + 2 lâchées.
    expect(partition.collapsedColumnCount).toBe(5)
  })

  it('ne perd AUCUNE colonne : partition = repartition exacte des groupes d’entrée', () => {
    const total = (gs: UsageColumnGroup[]) => gs.reduce((n, g) => n + g.columns.length, 0)
    expect(total(partition.forward) + total(partition.collapsed)).toBe(total(groupesDe(TEMOIN)))
  })
})

describe('partitionUsageGroups — le pont D5 (socle -> épisode)', () => {
  it('met les épisodes camo/surbouclier EN AVANT — par le pont, pas par leur propre nom', () => {
    // `camo`/`overshield` ne figurent PAS dans GAME_CHANGER_EQUIPMENT_FAMILIES (vocabulaire
    // de socle) : seuls `powerup_camo`/`powerup_overshield` y sont. Retirer le pont
    // EPISODE_FAMILY_OF_POWERUP fait donc tomber ces colonnes du bloc en avant (mutation G1.3).
    const partition = partitionUsageGroups(groupesDe(TEMOIN))
    expect(colonnes(partition.forward, 'episodes')).toEqual([
      'camo.count',
      'camo.ms',
      'camo.kills',
      'overshield.count',
      'overshield.ms',
      'overshield.kills',
    ])
    expect(partition.collapsed.map((g) => g.key)).not.toContain('episodes')
  })
})

describe('partitionUsageGroups — les grenades (D4) et le cas sans repli', () => {
  it('garde les grenades TOUJOURS visibles, même quand tout le reste est replié', () => {
    const partition = partitionUsageGroups(
      groupesDe({
        grappleLines: [{ slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 }],
        grenades: [{ slot: 1, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
        grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
      } as unknown as Partial<ReplayDocument>),
    )
    expect(partition.forward.map((g) => g.key)).toEqual(['grenades'])
    expect(partition.collapsed.map((g) => g.key)).toEqual(['grapple'])
  })

  it('zéro colonne repliée = compte à zéro (le rendu n’affiche alors aucun bouton)', () => {
    const partition = partitionUsageGroups(
      groupesDe({
        equipmentPlacements: [pose('sensor', 'deployed', 1)],
      } as unknown as Partial<ReplayDocument>),
    )
    expect(partition.collapsed).toEqual([])
    expect(partition.collapsedColumnCount).toBe(0)
    expect(partition.forward.map((g) => g.key)).toEqual(['deployed'])
  })
})

describe('uniqueUsageGroups — une famille de geste, une occurrence', () => {
  it('fusionne les deux morceaux d’un groupe mixte pour la légende et la vue des parts', () => {
    const partition = partitionUsageGroups(groupesDe(TEMOIN))
    const deplie = uniqueUsageGroups([...partition.forward, ...partition.collapsed])
    expect(deplie.map((g) => g.key)).toEqual([
      'episodes',
      'deployed',
      'dropped',
      'grenades',
      'grapple',
    ])
  })
})
