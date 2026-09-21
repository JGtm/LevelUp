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
  isCollapsedTierRowKey,
  padTierLabel,
  padTierUnclassifiedCount,
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
  it('range dans l’ordre ÉCRIT (puissance d’abord), pas par volume', () => {
    const rows = buildPadTierRows(temoin(), t)
    expect(rows.map((r) => r.key)).toEqual(['puissance', 'terrain', 'base'])
    // L'ordre est celui de la LECTURE depuis le 2026-09-21 (D2) : le plus lourd en tête,
    // la base en dernier — et repliée derrière un dépliable chez l'appelant.
    expect(USAGE_PAD_TIER_ORDER.indexOf('puissance')).toBeLessThan(
      USAGE_PAD_TIER_ORDER.indexOf('base'),
    )
    expect(isCollapsedTierRowKey('base')).toBe(true)
    expect(isCollapsedTierRowKey('tier-base')).toBe(true)
    expect(isCollapsedTierRowKey('puissance')).toBe(false)
  })

  it('ne fabrique AUCUNE ligne pour un niveau que le serveur ne publie pas', () => {
    const rows = buildPadTierRows(temoin(), t)
    expect(rows.find((r) => r.key === 'non_classe')).toBeUndefined()
    expect(rows.find((r) => r.key === 'bonus')).toBeUndefined()
  })

  /**
   * D2 amendée (décision utilisateur du 2026-09-21) : `bonus` et `non_classe` n'ont plus de
   * ligne MÊME SERVIS. Les socles de bonus sont des équipements (déjà comptés par
   * `equipment_powerup_*`) ; les prises sans emplacement identifié sont une réserve de
   * mesure, dont le compte part dans l'infobulle du titre.
   */
  it('n’affiche NI les socles de bonus NI les prises non classées, même servis', () => {
    const bloc = temoin({
      tiers: [
        ...(temoin().tiers ?? []),
        { tier: 'bonus', player_total: 7, lobby_total: 12, weapons: [] },
        { tier: 'non_classe', player_total: 3, lobby_total: 5, weapons: [] },
      ],
    } as Partial<SessionUsagePadTiersBlock>)
    expect(buildPadTierRows(bloc, t).map((r) => r.key)).toEqual(['puissance', 'terrain', 'base'])
    expect(buildPadTierGaugeRows(bloc, paritesNulles).map((r) => r.key)).toEqual([
      'tier-puissance',
      'tier-terrain',
      'tier-base',
    ])
    // Le compte non classé n'est pas perdu : il rejoint les notes de mesure (infobulle).
    expect(padTierUnclassifiedCount(bloc)).toBe(3)
    expect(padTiersNotes(bloc, t)).toContain(t.padTierUnclassifiedFmt(3))
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
  it('nomme les trois niveaux rendus', () => {
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
