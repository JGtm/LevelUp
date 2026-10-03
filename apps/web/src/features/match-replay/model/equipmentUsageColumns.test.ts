/**
 * Tests — equipmentUsageColumns : LES GROUPES DE COLONNES QUE LA DONNÉE JUSTIFIE.
 *
 * LE REPLI « GAME CHANGERS » ET LE GROUPE « ÉTATS ACTIFS » ONT ÉTÉ RETIRÉS LE 2026-09-19
 * (plan `.ai/V7.5/PLAN_AJUSTEMENTS_PRE_V75_2026-09-19.md`, lot 2, décision 6) : tout ce que la
 * donnée justifie s'affiche, et l'épisode de camouflage ou de surbouclier ne fait plus de
 * colonne à lui — il alimente le côté « utilisé » de la colonne d'équipement du power-up. Les
 * tests de la partition et du pont D5 sont partis avec le code qu'ils protégeaient.
 *
 * CE QU'ILS PROTÈGENT DÉSORMAIS :
 *   - l'ORDRE ÉCRIT des groupes (grappin, puis équipement) et l'ordre INTERNE des colonnes,
 *     celui des tables de référence (`PLACEMENT_RENDER` : sensor avant seeker) ;
 *   - AUCUN groupe « états actifs », quel que soit le document ;
 *   - `uniqueUsageGroups`, qui ne garde qu'une occurrence par famille de geste ;
 *   - la PILE d'issues d'une colonne d'équipement (utilisé / gardé / lâché).
 *
 * Les colonnes sont une HIÉRARCHIE D'AFFICHAGE : elles ne touchent ni aux mesures ni aux totaux
 * (`EquipmentUsage` ne passe pas par elles) — c'est éprouvé chez `equipmentUsageLogic.test.ts`.
 * Les fixtures passent par `testReplayDoc`, la seule porte du document de test.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import {
  uniqueUsageGroups,
  usageColumnGroups,
  type UsageColumnGroup,
} from './equipmentUsageColumns'
import { buildEquipmentUsage } from './equipmentUsageLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { testReplayDoc } from '../test/testDoc'
import { WALL_PANEL_IDS } from '../layers/placementWall'

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
  return usageColumnGroups(buildEquipmentUsage(doc, SB), t)
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

/** Les clés de colonnes d'un groupe, ou [] s'il n'y figure pas. */
function colonnes(groupes: UsageColumnGroup[], key: string): string[] {
  return groupes.find((g) => g.key === key)?.columns.map((c) => c.key) ?? []
}

describe('usageColumnGroups — tout ce que la donnée justifie, dans l’ordre écrit', () => {
  const groupes = groupesDe(TEMOIN)

  it('rend le grappin puis l’équipement, et RIEN d’autre', () => {
    // Plus de groupe « états actifs » depuis le 2026-09-19 (décision 6) : le témoin porte
    // pourtant des épisodes de camouflage et de surbouclier — ils ne font plus de groupe.
    expect(groupes.map((g) => g.key)).toEqual(['grapple', 'equipment'])
  })

  it('garde l’ordre INTERNE que la logique a posé, dans la colonne fusionnée', () => {
    // E2 (2026-09-09) : `deployed`/`dropped` ont fusionné en `equipment` — une colonne par
    // famille, empilée sur ses issues (P2/P3). L'ordre est celui de `usage.columns.equipment`
    // (equipmentUsageLogic) ; les deux power-ups ferment la marche, dans l'ordre de
    // `EPISODE_FAMILIES`. La mise en colonnes ne trie JAMAIS ce que la logique a rangé.
    expect(colonnes(groupes, 'equipment')).toEqual([
      'equipment.wall',
      'equipment.sensor',
      'equipment.translocator_beacon',
      'equipment.threat_seeker',
      'equipment.repair_field',
      'equipment.camo',
      'equipment.overshield',
    ])
  })

  // LES LANCERS DE GRENADE N'ONT PLUS DE GROUPE (2026-09-13, retrait demandé par l'utilisateur) :
  // un document qui n'apporte QUE des grenades et un grappin ne rend donc que le grappin.
  it('un document sans autre geste que des grenades ne rend que le grappin', () => {
    const seul = groupesDe({
      grappleLines: [{ slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 }],
      grenades: [{ slot: 1, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
      grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }],
    } as unknown as Partial<ReplayDocument>)
    expect(seul.map((g) => g.key)).toEqual(['grapple'])
  })

  it('un groupe sans colonne n’est pas rendu', () => {
    const pose_seule = groupesDe({
      equipmentPlacements: [pose('sensor', 'deployed', 1)],
    } as unknown as Partial<ReplayDocument>)
    expect(pose_seule.map((g) => g.key)).toEqual(['equipment'])
  })
})

describe('uniqueUsageGroups — une famille de geste, une occurrence', () => {
  it('ne garde que la première occurrence d’une même clé de famille', () => {
    const groupes = groupesDe(TEMOIN)
    const double = uniqueUsageGroups([...groupes, ...groupes])
    expect(double.map((g) => g.key)).toEqual(['grapple', 'equipment'])
  })
})

describe('equipmentGroup — la pile empilée (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md)', () => {
  /** La colonne fusionnée d'une seule famille, et la ligne d'Alpha, par le VRAI pipeline. */
  function colonneEquipement(over: Partial<ReplayDocument>, family: string) {
    const doc = testReplayDoc({
      frameCount: 200,
      frameIntervalMs: 100,
      roster: [{ filmIndex: 0, xuid: 'a1', name: 'Alpha' }],
      tracks: [vie(1, 'a1')],
      ...over,
    } as Partial<ReplayDocument>)
    const usage = buildEquipmentUsage(doc, SB)
    const groups = usageColumnGroups(usage, t)
    const group = groups.find((g) => g.key === 'equipment')
    const column = group?.columns.find((c) => c.key === `equipment.${family}`)
    const alpha = usage.byPlayer.find((r) => r.name === 'Alpha')
    return { column, alpha }
  }

  it('une cellule à UN SEUL segment non nul reste lisible : les trois segments existent, deux à zéro', () => {
    // Famille MUR (et son panneau, `WALL_PANEL_IDS`) : c'est la SEULE dont une pose `deployed`
    // seule vaut « utilisé » depuis le lot 5.7 (`isFamilyWithSpawnedPiece`) — une pose de
    // capteur seule, elle, ne vaudrait plus rien (cf. `equipmentUsageLogic.kept.test.ts`).
    const { column, alpha } = colonneEquipement(
      { equipmentPlacements: [pose('wall', 'deployed', 1, WALL_PANEL_IDS[0])] } as Partial<ReplayDocument>,
      'wall',
    )
    expect(column?.value(alpha!)).toBe(1)
    const segments = column?.segments?.(alpha!)
    expect(segments?.map((s) => [s.key, s.value])).toEqual([
      ['used', 1],
      ['kept', 0],
      ['dropped', 0],
    ])
    // Chaque segment garde son ENCRE et son NOM — la lisibilité d'UN segment ne dépend pas
    // des deux autres, à zéro ou pas.
    expect(segments?.every((s) => typeof s.color === 'string' && s.color.length > 0)).toBe(true)
    expect(segments?.every((s) => typeof s.label === 'string' && s.label.length > 0)).toBe(true)
  })

  it('une cellule à TROIS segments non nuls empile utilisé, gardé et lâché (P1)', () => {
    // Trois prises (`taken`) du même mur : une déployée, une lâchée, une GARDÉE (dérivée :
    // 3 pris - 1 utilisé - 1 lâché = 1 gardé, décision utilisateur du 2026-09-09).
    const { column, alpha } = colonneEquipement(
      {
        abilityLabels: { '5': { fr: 'Mur de protection', en: 'Drop wall', family: 'wall' } },
        equipmentChanges: [
          { t: 1, slot: 1, kind: 'taken', r: 5, from: -1 },
          { t: 2, slot: 1, kind: 'taken', r: 5, from: -1 },
          { t: 3, slot: 1, kind: 'taken', r: 5, from: -1 },
        ],
        equipmentPlacements: [
          pose('wall', 'deployed', 1, '0x528fce46'),
          pose('wall', 'dropped', 1, '0xdead'),
        ],
      } as unknown as Partial<ReplayDocument>,
      'wall',
    )
    expect(column?.value(alpha!)).toBe(3)
    expect(column?.segments?.(alpha!)?.map((s) => [s.key, s.value])).toEqual([
      ['used', 1],
      ['kept', 1],
      ['dropped', 1],
    ])
  })
})
