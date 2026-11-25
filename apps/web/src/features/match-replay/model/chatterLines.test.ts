import { describe, expect, it } from 'vitest'

import {
  buildChatterLines,
  CHATTER_FILTER_ALL,
  chatterFamilyOf,
  countByFamily,
  filterChatterLines,
  passesChatterFilter,
  type ChatterFilter,
  type ChatterLine,
} from './chatterLines'
import { parseChatterTable } from './chatterTable'

const TABLE = parseChatterTable({
  version: 1,
  sons: {
    '0000000a': { texte: { fr: 'Ennemi aperçu !', en: 'Enemy there.' }, audio: ['fr'] },
    '0000000b': { texte: { fr: 'Boucliers à zéro !', en: 'Shields down.' }, audio: ['fr', 'en'] },
    '0000000c': { texte: { en: 'Only English.' }, audio: [] },
  },
})

describe('chatterFamilyOf', () => {
  it('range un nom court dans sa famille, premier critère qui correspond', () => {
    expect(chatterFamilyOf('seeingenemynearby')).toBe('enemy')
    expect(chatterFamilyOf('seeingenemyactiveovershield')).toBe('enemy')
    expect(chatterFamilyOf('enemyshieldspopped')).toBe('shields')
    expect(chatterFamilyOf('shieldsdepleted')).toBe('shields')
    expect(chatterFamilyOf('lamentsquaddeathheadshot')).toBe('squadDown')
    expect(chatterFamilyOf('praisesquadkill')).toBe('praise')
    expect(chatterFamilyOf('calloutovershield')).toBe('other')
    expect(chatterFamilyOf('landhard')).toBe('other')
  })
})

describe('buildChatterLines', () => {
  const src = [
    { t: 10, player: '1', sound: '0000000a', name: 'seeingenemynearby' },
    { t: 20, player: '2', sound: '0000000b', name: 'shieldsdepleted' },
    { t: 30, player: '1', sound: '0000000a', name: 'seeingenemybelow' },
    { t: 40, sound: '0000000a', name: 'seeingenemynearby' }, // sans joueur : un bot, ou une unité sans piste
    { t: 50, player: '1', sound: '0000000a' }, // sans nom : le décodeur ne connaît pas ce build
    { t: 60, player: '1', sound: '00000099', name: 'landhard' }, // un cri : absent de la table
    { t: 70, player: '3', sound: '0000000c', name: 'praisesquadkill' },
  ]

  it('ne garde que les répliques nommées, parlées et portées par un joueur', () => {
    const lines = buildChatterLines(src, TABLE, 'fr')
    expect(lines.map((l) => [l.frame, l.xuid, l.family])).toEqual([
      [10, '1', 'enemy'],
      [20, '2', 'shields'],
      [30, '1', 'enemy'],
      [70, '3', 'praise'],
    ])
  })

  it('clé le texte par le son et compte chaque phrase dans le match', () => {
    const lines = buildChatterLines(src, TABLE, 'fr')
    expect(lines[0]).toMatchObject({ text: 'Ennemi aperçu !', sound: '0000000a', count: 2 })
    expect(lines[1]).toMatchObject({ text: 'Boucliers à zéro !', count: 1 })
  })

  it("lit le texte et la voix dans la langue de l'interface, l'anglais ailleurs", () => {
    const fr = buildChatterLines(src, TABLE, 'fr')
    expect(fr[1].voice).toBe('repliques/fr/0000000b')
    const en = buildChatterLines(src, TABLE, 'en')
    expect(en[1]).toMatchObject({ text: 'Shields down.', voice: 'repliques/en/0000000b' })
  })

  it("n'annonce pas de voix que la langue du texte n'a pas", () => {
    const en = buildChatterLines(src, TABLE, 'en')
    expect(en[0]).toMatchObject({ text: 'Enemy there.', voice: null }) // la voix versée est française, le texte est anglais
    const fr = buildChatterLines(src, TABLE, 'fr')
    const english = fr.find((l) => l.sound === '0000000c')
    expect(english).toMatchObject({ text: 'Only English.', voice: null })
  })
})

function line(over: Partial<ChatterLine>): ChatterLine {
  return { key: 'k', frame: 0, xuid: '1', sound: 's', family: 'enemy', text: 't', voice: null, count: 1, ...over }
}

describe('passesChatterFilter', () => {
  it('suit les familles', () => {
    const noShields: ChatterFilter = { ...CHATTER_FILTER_ALL, families: { ...CHATTER_FILTER_ALL.families, shields: false } }
    expect(passesChatterFilter(line({ family: 'shields' }), noShields)).toBe(false)
    expect(passesChatterFilter(line({ family: 'enemy' }), noShields)).toBe(true)
  })

  it('écarte les phrases fréquentes (quatre fois ou plus) sur « sans les plus fréquentes »', () => {
    const f: ChatterFilter = { ...CHATTER_FILTER_ALL, frequency: 'noFrequent' }
    expect(passesChatterFilter(line({ count: 3 }), f)).toBe(true)
    expect(passesChatterFilter(line({ count: 4 }), f)).toBe(false)
  })

  it('ne garde que les phrases dites deux fois ou moins sur « rares seulement »', () => {
    const f: ChatterFilter = { ...CHATTER_FILTER_ALL, frequency: 'rare' }
    expect(passesChatterFilter(line({ count: 1 }), f)).toBe(true)
    expect(passesChatterFilter(line({ count: 2 }), f)).toBe(true)
    expect(passesChatterFilter(line({ count: 3 }), f)).toBe(false)
  })

  it('filtre une liste sans la modifier, et compte par famille', () => {
    const all = [line({ family: 'enemy' }), line({ family: 'shields' }), line({ family: 'enemy' })]
    const f: ChatterFilter = { ...CHATTER_FILTER_ALL, families: { ...CHATTER_FILTER_ALL.families, enemy: false } }
    expect(filterChatterLines(all, f)).toHaveLength(1)
    expect(all).toHaveLength(3)
    expect(countByFamily(all)).toEqual({ enemy: 2, shields: 1, squadDown: 0, praise: 0, other: 0 })
  })
})
