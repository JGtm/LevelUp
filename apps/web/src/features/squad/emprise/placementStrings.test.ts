/**
 * placementStrings.test.ts — les aides ⓘ du bloc « Groupés ou isolés » : trois phrases au plus
 * (V9 / §2.1 dernier point), les comptes de couverture dits en toutes lettres, dans les deux
 * langues. La parité FR / EN des clés est garantie par le typage `Record<Locale, …>`.
 */
import { describe, expect, it } from 'vitest'

import { PLACEMENT_TEXT } from './placementStrings'

const sentences = (s: string) => s.split(/(?<=[.!?])\s+/).filter(Boolean)

describe.each(['fr', 'en'] as const)('aides du bloc « Groupés ou isolés » — %s', (locale) => {
  const t = PLACEMENT_TEXT[locale]

  it('l’aide du nuage tient en trois phrases au plus et dit vies mesurées, vies au total et matchs sans portée', () => {
    const info = t.life.info(14, 15, 3)
    expect(sentences(info).length).toBeLessThanOrEqual(3)
    expect(info).toContain('14')
    expect(info).toContain('15')
    expect(info).toMatch(/: 3\.$/)
  })

  it('l’aide de la barre tient en trois phrases au plus et dit les deux seuils', () => {
    const info = t.quarts.info('1,25', 1.25, 3)
    expect(sentences(info).length).toBeLessThanOrEqual(3)
    expect(info).toContain('1,25')
    expect(info).toMatch(/3 (frags|kills)/)
  })

  it('les quatre quarts ont un titre en capitales et un nom de segment', () => {
    for (const q of Object.values(t.life.quadrants)) {
      expect(q.title).toBe(q.title.toUpperCase())
      expect(q.sub.length).toBeGreaterThan(0)
    }
    expect(Object.keys(t.quarts.names)).toHaveLength(4)
  })
})
