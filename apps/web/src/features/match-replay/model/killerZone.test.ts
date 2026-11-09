/// <reference types="node" />
// @vitest-environment node
/**
 * Tests — killerZone (la zone nommée du tueur de chaque frag du fil).
 *
 * DEUX FAMILLES. La première lit le jeu de cas PARTAGÉ avec le Go (`NommerZone`,
 * `analysis/tactical/zones_test.go` lit le même fichier) : c'est elle qui prouve que le port
 * nomme comme son jumeau, règle et distance comprises — sans recopier un seul cas. La seconde
 * couvre ce qui n'existe que côté web : le document (tueur sans piste, embarqué, mort dans la
 * fenêtre après-mort, z absent, carte sans zone).
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchScoreboardRow, ReplayCalloutZone } from '@/lib/api/types'
import { calloutLabel, normalizeCalloutZones, type CalloutZoneReady } from '@/lib/replay/calloutsPaint'

import { racineDuDepot } from '../test/featureFiles'
import { testReplayDoc } from '../test/testDoc'
import { buildFeedEntries } from './killFeedLogic'
import { killerZones, nameZone } from './killerZone'

const JEU_DE_CAS = resolve(
  racineDuDepot(),
  'apps/go-api/internal/testfixtures/testdata/cartes/nommage_zone_cas.json',
)

interface CasPartage {
  nom: string
  zones: Array<Partial<ReplayCalloutZone> & { name: string; volume_index: number; z_bottom: number; z_top: number }>
  x: number
  y: number
  z: number | null
  attendu: string | null
  regle: string
  distance_m: number
}

/** Une zone du jeu de cas, passée par la MÊME normalisation que celle du document servi. */
function zonesOf(cas: CasPartage): CalloutZoneReady[] {
  return normalizeCalloutZones(
    cas.zones.map((z) => ({ x: 0, y: 0, z: 0, en: z.name, fr: z.name, ...z })),
  )
}

const jeu = JSON.parse(readFileSync(JEU_DE_CAS, 'utf8')) as { cas: CasPartage[] }

describe('nameZone — le jeu de cas partagé avec le Go', () => {
  it('le jeu n’est pas vide, et couvre les trois règles et l’absence de nom', () => {
    expect(jeu.cas.length).toBeGreaterThanOrEqual(24)
    const regles = new Set(jeu.cas.map((c) => c.regle))
    expect(regles).toEqual(new Set(['polygone', 'empilee', 'proche', '']))
  })

  for (const cas of jeu.cas) {
    it(cas.nom, () => {
      const got = nameZone(cas.x, cas.y, cas.z, zonesOf(cas))
      if (cas.attendu === null) {
        expect(got === null ? null : `${calloutLabel(got.zone, 'fr')} (${got.rule})`).toBeNull()
        return
      }
      expect(got).not.toBeNull()
      expect(calloutLabel(got!.zone, 'fr')).toBe(cas.attendu)
      expect(got!.rule).toBe(cas.regle)
      expect(got!.distanceM).toBeCloseTo(cas.distance_m, 9)
    })
  }
})

// --- ce qui n'existe que côté web : le document -----------------------------------------------

const carre = (x0: number, y0: number, x1: number, y1: number): number[][] => [
  [x0, y0],
  [x1, y0],
  [x1, y1],
  [x0, y1],
]

function zone(name: string, volume: number, polygon: number[][], zb: number, zt: number): CalloutZoneReady {
  return normalizeCalloutZones([
    { name, en: name, fr: name, x: 0, y: 0, z: 0, volume_index: volume, polygon, z_bottom: zb, z_top: zt },
  ])[0]
}

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000,
    xuid: 'A',
    ally: true,
    teamID: 0,
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: '',
    victimGamertag: '',
    ...over,
  }
}

/** Le fil d'un seul frag, tel que la page l'assemble (sans document : recalage brut, `replayMs` = `tMs`). */
const feedOf = (k: Partial<KillEvent>) => buildFeedEntries([kill(k)], [], 0)

const nameOf = (m: ReadonlyMap<string, CalloutZoneReady>, key: string) => {
  const z = m.get(key)
  return z ? calloutLabel(z, 'fr') : undefined
}

/** Deux étages superposés sur la même emprise : seul le z les distingue. */
const ETAGES = [zone('Bas', 1, carre(0, 0, 10, 10), 0, 3), zone('Haut', 2, carre(0, 0, 10, 10), 5, 8)]

/** Le tueur A immobile en (5, 5) à l'altitude `z` (ou sans altitude) pendant toute la vie. */
function docOf(z: number | undefined, over: Record<string, unknown> = {}) {
  const pt = (t: number) => ({ t, x: 5, y: 5, ...(z === undefined ? {} : { z }) })
  return testReplayDoc({
    tracks: [{ slot: 1, team: 0, xuid: 'A', points: [pt(0), pt(100)], startFrame: 0, endFrame: 100 }] as never,
    ...over,
  })
}

describe('killerZones — le document', () => {
  it('le z du tueur départage les étages : bas à 1 m, haut à 6 m', () => {
    const entries = feedOf({ tMs: 1_000 })
    expect(nameOf(killerZones(entries, docOf(1), ETAGES), entries[0].key)).toBe('Bas')
    expect(nameOf(killerZones(entries, docOf(6), ETAGES), entries[0].key)).toBe('Haut')
  })

  it('un document sans altitude : une seule forme nomme, des formes empilées ne nomment rien', () => {
    const entries = feedOf({ tMs: 1_000 })
    expect(killerZones(entries, docOf(undefined), ETAGES).size).toBe(0)
    expect(nameOf(killerZones(entries, docOf(undefined), [ETAGES[0]]), entries[0].key)).toBe('Bas')
  })

  it('un tueur SANS PISTE n’a pas d’entrée : jamais un nom deviné', () => {
    const entries = feedOf({ xuid: 'inconnu' })
    expect(killerZones(entries, docOf(1), ETAGES).size).toBe(0)
  })

  it('un tueur mort DANS LA FENÊTRE de 1,5 s garde sa dernière position, pas au-delà', () => {
    // Vie close à la frame 50 ; cadence de repli 60 i/s : 1 s = frame 60 (10 images après, dans
    // la fenêtre de 90), 3 s = frame 180 (130 images après, hors fenêtre).
    const doc = testReplayDoc({
      tracks: [
        { slot: 1, team: 0, xuid: 'A', points: [{ t: 0, x: 5, y: 5, z: 1 }, { t: 50, x: 5, y: 5, z: 1 }], startFrame: 0, endFrame: 50 },
      ] as never,
    })
    const dans = feedOf({ tMs: 1_000 })
    expect(nameOf(killerZones(dans, doc, ETAGES), dans[0].key)).toBe('Bas')
    const hors = feedOf({ tMs: 3_000 })
    expect(killerZones(hors, doc, ETAGES).size).toBe(0)
  })

  it('un tueur EMBARQUÉ est nommé d’après son véhicule, pas d’après la ligne droite de son bipède', () => {
    // Le bipède file en X (frame 50 -> x = 50) ; le véhicule monte en Y (frame 50 -> y = 50).
    const doc = testReplayDoc({
      tracks: [
        { slot: 1, team: 0, xuid: 'A', points: [{ t: 0, x: 0, y: 0, z: 0 }, { t: 100, x: 100, y: 0, z: 0 }], startFrame: 0, endFrame: 100 },
      ] as never,
      vehicles: [
        {
          slot: 700,
          gen: 1,
          t0: 0,
          t1: 100,
          t1max: 100,
          end: 'unknown',
          family: 'warthog',
          samples: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 0, y: 100 }],
          rides: [{ t0: 10, t1: 80, slot: 1, seat: 0, src: 'event', xuid: 'A' }],
        },
      ] as never,
    })
    const surLeVehicule = zone('Vehicule', 1, carre(-5, 40, 5, 60), -1, 1)
    const surLeBipede = zone('Bipede', 2, carre(45, -5, 55, 5), -1, 1)
    // Frame 50 = 833 ms à la cadence de repli de 60 images par seconde.
    const entries = feedOf({ tMs: 833 })
    expect(nameOf(killerZones(entries, doc, [surLeVehicule, surLeBipede]), entries[0].key)).toBe('Vehicule')
  })

  it('une carte SANS zone : aucune entrée', () => {
    const entries = feedOf({ tMs: 1_000 })
    expect(killerZones(entries, docOf(1), []).size).toBe(0)
  })

  it('une ligne qui n’est pas un frag n’a pas d’entrée', () => {
    const [entry] = feedOf({ tMs: 1_000 })
    const medaille = { ...entry, key: 'm-A-0', kill: null }
    expect(killerZones([medaille], docOf(1), ETAGES).size).toBe(0)
  })

  it('l’instant du frag (`replayMs` brut du fil) décide de la position relue', () => {
    // Le tueur est à (5, 5) jusqu'à la frame 50 puis en (500, 500) : seul l'instant décide.
    const doc = testReplayDoc({
      tracks: [
        { slot: 1, team: 0, xuid: 'A', points: [{ t: 0, x: 5, y: 5, z: 1 }, { t: 50, x: 5, y: 5, z: 1 }, { t: 51, x: 500, y: 500, z: 1 }, { t: 100, x: 500, y: 500, z: 1 }], startFrame: 0, endFrame: 100 },
      ] as never,
    })
    const avant = feedOf({ tMs: 500 })
    const apres = feedOf({ tMs: 1_500 })
    expect(nameOf(killerZones(avant, doc, ETAGES), avant[0].key)).toBe('Bas')
    expect(killerZones(apres, doc, ETAGES).size).toBe(0)
  })
})

describe('killerZones — le z suit la position', () => {
  /** Le bipède file en X à z = 0 ; le véhicule monte en Y (frame 50 -> y = 50) et de z = 0 à z = 6. */
  function docVehicule(vehicle: Record<string, unknown>) {
    return testReplayDoc({
      tracks: [
        { slot: 1, team: 0, xuid: 'A', points: [{ t: 0, x: 0, y: 0, z: 0 }, { t: 100, x: 100, y: 0, z: 0 }], startFrame: 0, endFrame: 100 },
      ] as never,
      vehicles: [
        {
          slot: 700,
          gen: 1,
          t0: 0,
          t1: 100,
          t1max: 100,
          end: 'unknown',
          family: 'warthog',
          rides: [{ t0: 10, t1: 80, slot: 1, seat: 0, src: 'event', xuid: 'A' }],
          ...vehicle,
        },
      ] as never,
    })
  }
  /** Deux étages empilés sur la trajectoire du véhicule à mi-parcours : (0, 50). */
  const EMPILEES = [
    zone('Bas', 1, carre(-5, 40, 5, 60), 0, 1),
    zone('Haut', 2, carre(-5, 40, 5, 60), 2, 8),
  ]
  // Frame 50 = 833 ms à la cadence de repli de 60 images par seconde.
  const entries = feedOf({ tMs: 833 })

  it('embarqué : le z est celui du véhicule (3 m à mi-trajet) -> « Haut », pas celui du bipède (0 -> « Bas »)', () => {
    const doc = docVehicule({ samples: [{ t: 0, x: 0, y: 0, z: 0 }, { t: 100, x: 0, y: 100, z: 6 }] })
    expect(nameOf(killerZones(entries, doc, EMPILEES), entries[0].key)).toBe('Haut')
  })

  it('embarqué sur un véhicule SANS altitude : z inconnu, des étages empilés ne nomment rien (jamais le z du bipède)', () => {
    const doc = docVehicule({ samples: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 0, y: 100 }] })
    expect(killerZones(entries, doc, EMPILEES).size).toBe(0)
  })

  it('embarqué sur un véhicule dont la position vient de sa NAISSANCE : z inconnu, jamais celui du bipède', () => {
    const doc = docVehicule({ samples: [], spawn: { x: 0, y: 50, z: 0 } })
    expect(killerZones(entries, doc, EMPILEES).size).toBe(0)
    expect(nameOf(killerZones(entries, doc, [EMPILEES[0]]), entries[0].key)).toBe('Bas')
  })
})

describe('killerZones — zones sans nom, bots', () => {
  const muette = normalizeCalloutZones([
    { name: 'muette', en: '', fr: '', x: 0, y: 0, z: 0, volume_index: 9, polygon: carre(0, 0, 10, 10), z_bottom: 0, z_top: 1 },
  ])[0]

  it('une zone sans aucun nom est écartée avant le nommage : elle ne prend pas la place d’une zone nommée', () => {
    const entries = feedOf({ tMs: 1_000 })
    const doc = docOf(0.5)
    // Plus étroite que « Bas » : sans l'écart, elle l'emporterait et la ligne resterait sans nom.
    expect(nameOf(killerZones(entries, doc, [muette, ETAGES[0]]), entries[0].key)).toBe('Bas')
    expect(killerZones(entries, doc, [muette]).size).toBe(0)
  })

  const BOT_FILM = { slot: 4, team: 1, bot: 'Spark [bot]', points: [{ t: 0, x: 5, y: 5, z: 1 }, { t: 100, x: 5, y: 5, z: 1 }], startFrame: 0, endFrame: 100 }
  const ROW_BOT = { xuid: 'bid(1.0)', gamertag: 'Spark', is_bot: true } as unknown as MatchScoreboardRow

  it('tueur BOT : le xuid de base du fil est relié à sa piste par la feuille (gamertag nu), sans table nouvelle', () => {
    const doc = testReplayDoc({ tracks: [BOT_FILM] as never })
    const entries = feedOf({ tMs: 1_000, xuid: 'bid(1.0)' })
    expect(nameOf(killerZones(entries, doc, ETAGES, [ROW_BOT]), entries[0].key)).toBe('Bas')
  })

  it('tueur bot SANS ligne de feuille pour le joindre : pas d’entrée', () => {
    const doc = testReplayDoc({ tracks: [BOT_FILM] as never })
    const entries = feedOf({ tMs: 1_000, xuid: 'bid(1.0)' })
    expect(killerZones(entries, doc, ETAGES).size).toBe(0)
    expect(killerZones(entries, doc, ETAGES, [{ ...ROW_BOT, gamertag: 'Autre' } as MatchScoreboardRow]).size).toBe(0)
  })
})
