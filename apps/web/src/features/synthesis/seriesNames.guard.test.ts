/**
 * Garde-rail : aucun nom de série, d'axe ou de ligne d'infobulle écrit en dur dans les graphes de la
 * Synthèse. « Top 1 », « Autres », « Taux Top (%) » et « Win % » (affichés tels quels en français comme en
 * anglais) venaient d'un `name: '…'` littéral dans le constructeur d'option ; les noms viennent des manifestes
 * i18n (`synthesis.charts.top_weeks_*`, `synthesis.scope.*`).
 */
import { describe, expect, it } from 'vitest'

const LITERAL_NAME = /\bname:\s*['"`][^'"`]*[A-Za-zÀ-ÿ][^'"`]*['"`]/
const withoutComments = (src: string) => src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '')

describe('graphes de la Synthèse', () => {
  it('aucun nom de série ou d’axe littéral dans les sources (hors tests)', () => {
    const files = import.meta.glob('/src/features/synthesis/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
      string,
      string
    >
    const offenders: string[] = []
    for (const [path, src] of Object.entries(files)) {
      if (/\.test\.tsx?$/.test(path)) continue
      withoutComments(src)
        .split('\n')
        .forEach((line, i) => {
          if (LITERAL_NAME.test(line)) offenders.push(`${path}:${i + 1} ${line.trim()}`)
        })
    }
    expect(offenders, `nommer par un manifeste i18n :\n${offenders.join('\n')}`).toEqual([])
  })
})
