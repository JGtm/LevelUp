/**
 * Tests — antagonistBandLogic : la bande des titres de la frise.
 *
 * Ils protègent :
 *  1. UNE ICÔNE PAR PRISE DE TITRE, AUCUNE POUR UNE ÉGALITÉ : le titre ne change pas de main sur
 *     une égalité, la frise ne le dit donc pas ;
 *  2. LES TEXTES, EXACTS, en français et en anglais : titre (« Némésis : X »), puis phrase
 *     (première prise, ou dépassement avec l'ancien tenant et son compte) ;
 *  3. LA GÉOMÉTRIE : le ratio est celui des marques de la frise (`ratioOfMs`), l'image à viser est
 *     celle des portes de présence (`Math.round(t / frameIntervalMs)`) ; une prise hors de la
 *     fenêtre n'a pas de place sur l'axe et n'est pas posée.
 */
import { describe, expect, it } from 'vitest'

import type { AntagonistChange } from '@/lib/antagonists/antagonists'

import { ANTAGONISTS_TEXT } from '../i18n/antagonists.i18n'
import { buildAntagonistBand } from './antagonistBandLogic'

const NAMES: Record<string, string> = { 'foe-1': 'Rival', 'foe-2': 'Fantome' }
const nameOf = (key: string) => NAMES[key] ?? '?'

function change(over: Partial<AntagonistChange>): AntagonistChange {
  return { t: 10_000, role: 'nemesis', kind: 'prise', holder: 'foe-1', previousHolder: null, count: 1, previousCount: 0, ...over }
}

const CHANGES: AntagonistChange[] = [
  change({ t: 10_000 }),
  change({ t: 20_000, role: 'victim', holder: 'foe-2' }),
  change({ t: 30_000, kind: 'egalite', holder: 'foe-2', previousHolder: 'foe-1', count: 1, previousCount: 1 }),
  change({ t: 40_000, holder: 'foe-2', previousHolder: 'foe-1', count: 2, previousCount: 1 }),
  change({ t: 50_000, role: 'victim', holder: 'foe-1', previousHolder: 'foe-2', count: 2, previousCount: 1 }),
]

function band(locale: 'fr' | 'en', changes: readonly AntagonistChange[] = CHANGES, scale = { from: 0, span: 1000 }) {
  return buildAntagonistBand(changes, {
    nameOf, viewpointName: 'JGtm', frameIntervalMs: 100, scale, text: ANTAGONISTS_TEXT[locale],
  })
}

describe('buildAntagonistBand — une icône par prise, aucune pour une égalité', () => {
  it('rend les quatre prises dans l ordre, et ne rend rien pour l égalité', () => {
    const items = band('fr')
    expect(items.map((i) => [i.role, i.holder])).toEqual([
      ['nemesis', 'foe-1'], ['victim', 'foe-2'], ['nemesis', 'foe-2'], ['victim', 'foe-1'],
    ])
  })

  it('une liste sans changement rend une bande vide', () => {
    expect(band('fr', [])).toEqual([])
  })

  it('une liste d égalités seules rend une bande vide', () => {
    expect(band('fr', [CHANGES[2]])).toEqual([])
  })

  it('chaque marque porte une clé distincte', () => {
    const keys = band('fr').map((i) => i.key)
    expect(new Set(keys).size).toBe(keys.length)
  })
})

describe('buildAntagonistBand — la place sur l axe', () => {
  it('le ratio est celui de la frise, l image celle des portes de présence', () => {
    const [premier] = band('fr')
    expect(premier.ratio).toBeCloseTo(0.1)
    expect(premier.frame).toBe(100)
  })

  it('une prise hors de la fenêtre n est pas posée', () => {
    const hors = band('fr', [change({ t: 500_000 }), change({ t: -1_000 })])
    expect(hors).toEqual([])
  })

  it('une échelle sans étendue ne pose rien', () => {
    expect(band('fr', CHANGES, { from: 0, span: 0 })).toEqual([])
  })
})

describe('buildAntagonistBand — textes en français', () => {
  it('première prise de la némésis', () => {
    expect(band('fr')[0]).toMatchObject({ title: 'Némésis : Rival', phrase: 'Premier adversaire à éliminer JGtm.' })
  })

  it('première prise du souffre-douleur', () => {
    expect(band('fr')[1]).toMatchObject({ title: 'Souffre-douleur : Fantome', phrase: 'Premier adversaire éliminé par JGtm.' })
  })

  it('dépassement pour la némésis : le compte et l ancien tenant avec son compte', () => {
    expect(band('fr')[2]).toMatchObject({
      title: 'Némésis : Fantome',
      phrase: 'A éliminé JGtm 2 fois et passe devant Rival (1).',
    })
  })

  it('dépassement pour le souffre-douleur', () => {
    expect(band('fr')[3]).toMatchObject({
      title: 'Souffre-douleur : Rival',
      phrase: 'Éliminé 2 fois par JGtm, passe devant Fantome (1).',
    })
  })
})

describe('buildAntagonistBand — textes en anglais', () => {
  it('première prise de la némésis', () => {
    expect(band('en')[0]).toMatchObject({ title: 'Nemesis: Rival', phrase: 'First opponent to eliminate JGtm.' })
  })

  it('première prise du souffre-douleur', () => {
    expect(band('en')[1]).toMatchObject({ title: 'Bully target: Fantome', phrase: 'First opponent eliminated by JGtm.' })
  })

  it('dépassement pour la némésis', () => {
    expect(band('en')[2]).toMatchObject({
      title: 'Nemesis: Fantome',
      phrase: 'Eliminated JGtm 2 times and moves ahead of Rival (1).',
    })
  })

  it('dépassement pour le souffre-douleur', () => {
    expect(band('en')[3]).toMatchObject({
      title: 'Bully target: Rival',
      phrase: 'Eliminated 2 times by JGtm, moves ahead of Fantome (1).',
    })
  })
})
