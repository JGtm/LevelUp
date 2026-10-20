/**
 * vehicles.fixtures.ts — la soirée témoin du 22/09 (`emprise.fixtures.ts`) augmentée de la ressource
 * « véhicules » au format du contrat (lot L7.3 du plan PLAN_EMPRISE_VEHICULES_2026-09-28). Chiffres
 * choisis pour exercer chaque règle, pas relevés : un Warthog pris 3 fois par nous et 1 fois par
 * eux, un Banshee pris 2 fois par eux, une tourelle fixe (libellé du titre), un châssis inconnu ;
 * temps à bord 210 s / 100 s ; 23 frags de classe véhicule dont 11 appariés à un passage daté de
 * leur tueur ; Shogun (m6) sans passe de véhicules, Detachment (m5) sans film mais mesuré.
 */
import type { SquadEmpriseBlock, SquadEmpriseMatch, SquadEmpriseObject } from '@/lib/api/types'

import { EMPRISE_2209, XUID } from './emprise.fixtures'

/** Une part : qui chez nous a pris (et passé du temps à bord), dans l'ordre des fiches puis le reste du camp. */
function squad(taken: [number, number, number, number], aboard: [number, number, number, number]) {
  const xuids = [XUID.jgtm, XUID.choco, XUID.madina, undefined]
  return xuids.map((xuid, i) => ({ ...(xuid ? { xuid } : {}), taken: taken[i], aboard_ms: aboard[i] }))
}

function obj(key: string, us: number, them: number, ms: [number, number], sq: SquadEmpriseObject['squad'], label?: string): SquadEmpriseObject {
  return { resource: 'vehicle', key, ...(label ? { label } : {}), taken: { us, them }, aboard_ms: { us: ms[0], them: ms[1] }, squad: sq }
}

/** Les objets de la soirée (l'ordre du serveur : prises de notre camp décroissantes). */
const EVENING_OBJECTS: SquadEmpriseObject[] = [
  obj('warthog', 3, 1, [180_000, 60_000], squad([2, 1, 0, 0], [120_000, 60_000, 0, 0])),
  obj('tourelle_fixe', 1, 0, [10_000, 0], squad([0, 0, 0, 1], [0, 0, 0, 10_000]), 'Tourelle fixe'),
  obj('unknown', 1, 0, [20_000, 0], squad([1, 0, 0, 0], [20_000, 0, 0, 0])),
  obj('banshee', 0, 2, [0, 40_000], squad([0, 0, 0, 0], [0, 0, 0, 0])),
]

/** Une ressource de match (Starboard : le Warthog seul ; Detachment : la tourelle et le châssis inconnu). */
function matchResource(objects: SquadEmpriseObject[]) {
  const us = objects.reduce((a, o) => a + o.taken.us, 0)
  const them = objects.reduce((a, o) => a + o.taken.them, 0)
  return { resource: 'vehicle', taken: { us, them }, objects }
}

function withVehicles(m: SquadEmpriseMatch): SquadEmpriseMatch {
  switch (m.match_id) {
    case 'm1':
      return { ...m, vehicles: 'measured', resources: [...(m.resources ?? []), matchResource([EVENING_OBJECTS[0]])] }
    case 'm5':
      return { ...m, vehicles: 'measured', resources: [...(m.resources ?? []), matchResource([EVENING_OBJECTS[1], EVENING_OBJECTS[2], EVENING_OBJECTS[3]])] }
    case 'm6':
      return { ...m, vehicles: 'not_measured' }
    default:
      return { ...m, vehicles: 'measured' }
  }
}

const VEHICLE_RESOURCE: NonNullable<SquadEmpriseBlock['resources']>[number] = { resource: 'vehicle', taken: { us: 5, them: 3 }, matches_measured: 6 }

/** Insère la ressource à sa place : après les armes spéciales, avant le reste. */
function inOrder<T extends { resource: string }>(list: T[] | null | undefined, vehicle: T): T[] {
  const base = list ?? []
  const at = base.findIndex((r) => r.resource === 'rack')
  return at < 0 ? [...base, vehicle] : [...base.slice(0, at), vehicle, ...base.slice(at)]
}

const share = (us: number, them: number) => ({ resource: 'vehicle', taken: { us, them }, share: us / (us + them) })

/** Le bloc du 22/09 avec les véhicules. */
export const VEHICLES_2209: SquadEmpriseBlock = {
  ...EMPRISE_2209,
  resources: inOrder(EMPRISE_2209.resources, VEHICLE_RESOURCE),
  objects: [...(EMPRISE_2209.objects ?? []).filter((o) => o.resource !== 'rack'), ...EVENING_OBJECTS, ...(EMPRISE_2209.objects ?? []).filter((o) => o.resource === 'rack')],
  matches: (EMPRISE_2209.matches ?? []).map(withVehicles),
  production: [
    ...(EMPRISE_2209.production ?? []),
    {
      resource: 'vehicle',
      kills: { us: 14, them: 9 },
      // La barre épaisse garde tous les frags (D5) ; le rendement ne compte que les appariés (D9).
      exposure: { kind: 'aboard_ms', value: { us: 210_000, them: 100_000 }, kills: { us: 14, them: 9 }, paired_kills: { us: 8, them: 3 } },
      yield_us: 8 / 3.5,
      yield_them: 3 / (100 / 60),
      relative_gap: 8 / 3.5 / (3 / (100 / 60)) - 1,
    },
  ],
  habit: {
    ...EMPRISE_2209.habit!,
    current: { ...EMPRISE_2209.habit!.current, shares: [...(EMPRISE_2209.habit!.current.shares ?? []), share(5, 3)] },
    previous: (EMPRISE_2209.habit!.previous ?? []).map((e, i) => ({ ...e, shares: [...(e.shares ?? []), share(2 + i, 3)] })),
  },
}
