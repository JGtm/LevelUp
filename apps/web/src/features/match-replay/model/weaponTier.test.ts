/**
 * Tests — weaponTier (base / terrain / puissance / non classé).
 *
 * CE QU'ILS PROTÈGENT, dans l'ordre des pièges du domaine :
 *   - le niveau vient de la CARTE, jamais du nom ni du rôle de l'arme ;
 *   - un socle qu'aucun emplacement ne confirme reste NON CLASSÉ, jamais rangé ailleurs ;
 *   - « base » se lit sur la PREMIÈRE émission de chaque slot — le canal ré-émet en cours de
 *     vie, et tout prendre ferait d'une arme de puissance une arme de base ;
 *   - la queue d'équipements de départ (5,6 % mesurés) ne promeut rien : seuil `BASE_SHARE_MIN` ;
 *   - un mode à départs ALÉATOIRES ne publie aucun niveau « base », les deux autres restent.
 *
 * Ce fichier est le jumeau de `internal/analysis/weapontier/weapontier_test.go` : mêmes cas,
 * mêmes valeurs. Une divergence entre les deux est un bug, pas une variante.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { buildPadTierMatch, hasRandomStarts, padTierOf } from './weaponTier'
import { testReplayDoc } from '../test/testDoc'

const AR = '0x48C19D2D'
const PISTOLET = '0xF408190F'
const SNIPER = '0x9D6AAED2'
const HYDRA = '0xB619D84A'
const BONUS = 'powerup_overshield'

function socle(weapon: string) {
  return { weapon, x: 0, y: 0, spawns: [], presence: [] }
}

/** `n` vies partant avec `depart`, chacune sur son slot, plus une ré-émission si `apres`. */
function vies(n: number, depart: string[], apres?: string[]) {
  const out: { t: number; slot: number; w: string[] }[] = []
  for (let i = 0; i < n; i += 1) {
    out.push({ t: 74, slot: 512 + i, w: depart })
    if (apres) out.push({ t: 600, slot: 512 + i, w: apres })
  }
  return out
}

/**
 * LE TÉMOIN. Quatre socles : un râtelier, un socle de puissance, un socle de bonus, et un
 * quatrième qu'aucun emplacement ne confirme.
 */
function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 1000,
    frameIntervalMs: 100,
    weaponPads: [socle(HYDRA), socle(SNIPER), socle(BONUS), socle(AR)],
    mapWeaponPads: {
      catalogN: 9,
      pads: [
        { x: 0, y: 0, pad: 0, family: 'rack' },
        { x: 1, y: 0, pad: 1, family: 'power' },
        { x: 2, y: 0, pad: 2, family: 'powerup' },
      ],
    },
    loadouts: vies(20, [AR, PISTOLET], [SNIPER, PISTOLET]),
    ...over,
  } as Partial<ReplayDocument>)
}

describe('padTierOf — les quatre niveaux', () => {
  it('lit la nature de l’emplacement sur la CARTE, et l’arme de base sur le film', () => {
    const m = buildPadTierMatch(temoin(), 'Assassin')
    expect(padTierOf(m, 0, HYDRA)).toBe('ground')
    expect(padTierOf(m, 1, SNIPER)).toBe('power')
    expect(padTierOf(m, 2, BONUS)).toBe('powerup')
    expect(padTierOf(m, 3, AR)).toBe('base')
    expect(m.lives).toBe(20)
    expect(m.tiersMeasured).toBe(true)
  })

  it('ne juge JAMAIS au rôle de l’arme : l’Hydra sur râtelier reste de terrain', () => {
    // L'Hydra a un rôle `power` au registre canonique, et 17 socles du parc la posent sur un
    // râtelier. Si le niveau se lisait sur l'arme, cette ligne rendrait 'power'.
    const m = buildPadTierMatch(temoin(), 'Assassin')
    expect(padTierOf(m, 0, HYDRA)).toBe('ground')
  })

  it('laisse NON CLASSÉ ce qu’aucun emplacement ne confirme, et le dit', () => {
    const m = buildPadTierMatch(temoin({ mapWeaponPads: undefined } as Partial<ReplayDocument>), 'Assassin')
    expect(m.tiersMeasured).toBe(false)
    expect(padTierOf(m, 1, SNIPER)).toBe('unclassified')
    // Un index hors bornes ne pioche pas le voisin.
    expect(padTierOf(m, 99, SNIPER)).toBe('unclassified')
  })

  it('fait primer « base » sur l’emplacement (base > terrain > puissance)', () => {
    const m = buildPadTierMatch(
      temoin({
        weaponPads: [socle(AR)],
        mapWeaponPads: { catalogN: 1, pads: [{ x: 0, y: 0, pad: 0, family: 'rack' }] },
        loadouts: vies(20, [AR, PISTOLET]),
      } as Partial<ReplayDocument>),
      'Assassin',
    )
    expect(padTierOf(m, 0, AR)).toBe('base')
  })
})

describe('les armes de départ', () => {
  it('ne lit que la PREMIÈRE émission de chaque slot', () => {
    // Les vingt vies démarrent AR + Sidekick puis ramassent toutes le sniper. Si tout le canal
    // comptait, le sniper deviendrait « base » et le niveau « puissance » disparaîtrait.
    const m = buildPadTierMatch(temoin(), 'Assassin')
    expect(padTierOf(m, 1, SNIPER)).toBe('power')
  })

  it('ne promeut pas la queue : une vie sur 41 reste sous le seuil', () => {
    const doc = temoin({
      loadouts: [...vies(40, [AR, PISTOLET]), { t: 74, slot: 999, w: [SNIPER, PISTOLET] }],
    } as Partial<ReplayDocument>)
    expect(padTierOf(buildPadTierMatch(doc, 'Assassin'), 1, SNIPER)).toBe('power')
  })

  it('retient en revanche une arme de départ franche (12 vies sur 52)', () => {
    const enPlus = Array.from({ length: 12 }, (_, i) => ({
      t: 74,
      slot: 900 + i,
      w: [SNIPER, PISTOLET],
    }))
    const doc = temoin({
      loadouts: [...vies(40, [AR, PISTOLET]), ...enPlus],
    } as Partial<ReplayDocument>)
    expect(padTierOf(buildPadTierMatch(doc, 'Assassin'), 1, SNIPER)).toBe('base')
  })
})

describe('les modes à départs aléatoires', () => {
  it('reconnaît les catégories concernées, et elles seules', () => {
    expect(hasRandomStarts('Fiesta')).toBe(true)
    expect(hasRandomStarts('Super Fiesta')).toBe(true)
    expect(hasRandomStarts('Husky Raid')).toBe(true)
    expect(hasRandomStarts('Assassin')).toBe(false)
    expect(hasRandomStarts('BTB')).toBe(false)
    expect(hasRandomStarts(undefined)).toBe(false)
    expect(hasRandomStarts(null)).toBe(false)
  })

  it('ne publie AUCUN niveau « base », et garde les deux autres', () => {
    const m = buildPadTierMatch(temoin(), 'Super Fiesta')
    expect(m.randomStarts).toBe(true)
    // L'AR est bien distribué au départ en Fiesta, mais le niveau n'existe pas.
    expect(padTierOf(m, 3, AR)).toBe('unclassified')
    expect(padTierOf(m, 0, HYDRA)).toBe('ground')
    expect(padTierOf(m, 1, SNIPER)).toBe('power')
    // Les vies restent comptées : l'écran doit pouvoir dire sur quoi il s'appuie.
    expect(m.lives).toBe(20)
  })
})
