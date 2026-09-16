/**
 * Tests — usagePadTiersModel (les prises de socle par niveau d'arme).
 *
 * CE QU'ILS PROTÈGENT :
 *   - l'ORDRE des niveaux est ÉCRIT, jamais le volume — un classement qui change d'ordre d'une
 *     session à l'autre ne se compare pas ;
 *   - un niveau que le serveur ne publie pas n'a PAS de ligne à zéro : « ce niveau n'existe pas
 *     sur ce scope » n'est pas « aucune prise » ;
 *   - le détail par arme part AU SURVOL, et une arme hors catalogue garde sa CLÉ ;
 *   - les trois notes de mesure ne s'écrivent que quand il y a quelque chose à signaler ;
 *   - les deux formes (barres et jauges) rangent dans le MÊME ordre et nomment les armes de la
 *     MÊME façon.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsagePadTiersBlock } from '@/lib/api/types'

import { USAGE_TEXT } from './usageI18n'
import {
  USAGE_PAD_TIER_ORDER,
  buildPadTierGaugeRows,
  buildPadTierRows,
  padTierLabel,
  padTiersNotes,
} from './usagePadTiersModel'

const t = USAGE_TEXT.fr

/** Le témoin : trois niveaux, dont la puissance pèse le plus lourd. */
function temoin(over: Partial<SessionUsagePadTiersBlock> = {}): SessionUsagePadTiersBlock {
  return {
    matches_measured: 5,
    matches_with_pads: 5,
    matches_tiers_established: 5,
    matches_random_starts: 0,
    tiers: [
      {
        tier: 'puissance',
        player_total: 9,
        lobby_total: 20,
        weapons: [
          { family_key: '9d6aaed2', family_label: 'S7 Sniper', player_pickups: 6, lobby_pickups: 12 },
          { family_key: 'deadbeef', player_pickups: 3, lobby_pickups: 8 },
        ],
      },
      {
        tier: 'terrain',
        player_total: 4,
        lobby_total: 15,
        weapons: [
          { family_key: 'b619d84a', family_label: 'Hydra', player_pickups: 4, lobby_pickups: 15 },
        ],
      },
      { tier: 'base', player_total: 2, lobby_total: 6, weapons: [] },
    ],
    ...over,
  } as SessionUsagePadTiersBlock
}

const paritesNulles = {
  teamParityPct: null,
  lobbyParityPct: null,
  teamOfLobbyParityPct: null,
  t,
  locale: 'fr' as const,
}

describe('buildPadTierRows', () => {
  it('range dans l’ordre ÉCRIT, pas par volume', () => {
    const rows = buildPadTierRows(temoin(), t)
    expect(rows.map((r) => r.key)).toEqual(['base', 'terrain', 'puissance'])
    // La puissance pèse 9 contre 2 pour la base : un tri par volume l'aurait mise en tête.
    expect(USAGE_PAD_TIER_ORDER.indexOf('base')).toBeLessThan(USAGE_PAD_TIER_ORDER.indexOf('puissance'))
  })

  it('ne fabrique AUCUNE ligne pour un niveau que le serveur ne publie pas', () => {
    const rows = buildPadTierRows(temoin(), t)
    expect(rows.find((r) => r.key === 'non_classe')).toBeUndefined()
    expect(rows.find((r) => r.key === 'bonus')).toBeUndefined()
  })

  it('rend une liste vide sans bloc, et sans niveau', () => {
    expect(buildPadTierRows(null, t)).toEqual([])
    expect(buildPadTierRows(undefined, t)).toEqual([])
    expect(buildPadTierRows(temoin({ tiers: [] }), t)).toEqual([])
  })

  it('compose le détail par arme au survol, et garde la CLÉ d’une arme hors catalogue', () => {
    const rows = buildPadTierRows(temoin(), t)
    const puissance = rows.find((r) => r.key === 'puissance')
    expect(puissance?.hint).toBe('Armes de puissance — S7 Sniper 6, deadbeef 3')
  })

  it('n’écrit aucun survol quand le joueur n’a rien pris à ce niveau', () => {
    const rows = buildPadTierRows(temoin(), t)
    expect(rows.find((r) => r.key === 'base')?.hint).toBeUndefined()
  })
})

describe('padTiersNotes', () => {
  it('se tait quand il n’y a rien à signaler', () => {
    expect(padTiersNotes(temoin(), t)).toEqual([])
  })

  it('dit les matchs sans aucun socle, les cartes hors référence et les départs aléatoires', () => {
    const notes = padTiersNotes(
      temoin({ matches_with_pads: 3, matches_tiers_established: 1, matches_random_starts: 2 }),
      t,
    )
    expect(notes).toHaveLength(3)
    expect(notes[0]).toContain('2 matchs')
    expect(notes[1]).toContain('2 matchs')
    expect(notes[2]).toContain('2 matchs')
  })

  it('rend une liste vide sans bloc', () => {
    expect(padTiersNotes(null, t)).toEqual([])
  })
})

describe('buildPadTierGaugeRows', () => {
  it('range dans le MÊME ordre et nomme les armes de la MÊME façon que les barres', () => {
    const barres = buildPadTierRows(temoin(), t)
    const jauges = buildPadTierGaugeRows(temoin(), paritesNulles)
    expect(jauges.map((r) => r.key)).toEqual(barres.map((r) => `tier-${r.key}`))
    expect(jauges.map((r) => r.hint)).toEqual(barres.map((r) => r.hint))
    expect(jauges.map((r) => r.label)).toEqual(barres.map((r) => r.label))
  })

  it('rend une liste vide sans bloc', () => {
    expect(buildPadTierGaugeRows(null, paritesNulles)).toEqual([])
  })
})

describe('padTierLabel', () => {
  it('nomme les cinq niveaux du contrat', () => {
    for (const tier of USAGE_PAD_TIER_ORDER) {
      expect(padTierLabel(tier, t)).not.toBe(tier)
    }
  })

  it('garde la clé d’une valeur inconnue, jamais un nom voisin', () => {
    expect(padTierLabel('quelque_chose', t)).toBe('quelque_chose')
    // Et surtout pas une entrée de la chaîne de prototypes.
    expect(padTierLabel('constructor', t)).toBe('constructor')
  })
})
