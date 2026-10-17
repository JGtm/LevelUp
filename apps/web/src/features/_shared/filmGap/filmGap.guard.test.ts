/**
 * Garde-rail (CLAUDE.md n°6) : la décision « analyse du film en cours » / « film non analysé » se lit
 * en UN endroit, `filmGap.logic.ts`. Trois pages (Escouade, Séries temporelles, Sessions) lisaient ou
 * allaient lire `matches_film_pending` / `matches_film_missing` : sans garde, la règle re-diverge d'une
 * page à l'autre (un « en cours » ici, un « non analysé » là pour le même bloc). Le littéral de l'état
 * vide (`AnalysisPendingSlot` + « Film non analysé ») vit lui aussi en un seul composant.
 */
import { describe, expect, it } from 'vitest'

const COUNTERS = /matches_film_(?:pending|missing)/
const ALLOWED = [/\/_shared\/filmGap\/filmGap\.logic\.ts$/, /\/lib\/api\/(?:generated|types)\.ts$/]

/** Le code sans ses commentaires : une phrase qui NOMME l'état dans un commentaire n'est pas un second littéral. */
const withoutComments = (src: string) => src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '')

function sources(): Array<[string, string]> {
  const files = import.meta.glob('/src/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
    string,
    string
  >
  return Object.entries(files).filter(([path]) => !/\.test\.tsx?$/.test(path) && !/\/fixtures?\b|\.fixtures\./.test(path))
}

describe('garde « film non lu »', () => {
  it('seul filmGap.logic.ts lit les compteurs du bloc d’Emprise', () => {
    const offenders: string[] = []
    for (const [path, src] of sources()) {
      if (ALLOWED.some((re) => re.test(path))) continue
      src.split('\n').forEach((line, i) => {
        if (COUNTERS.test(line)) offenders.push(`${path}:${i + 1} ${line.trim()}`)
      })
    }
    expect(offenders, `compteurs lus hors de filmGapOfBlock :\n${offenders.join('\n')}`).toEqual([])
  })

  it('« Film non analysé » ne s’écrit qu’une fois (filmGapStrings.ts) dans l’Emprise, la Synthèse, Sessions et Séries temporelles', () => {
    const offenders: string[] = []
    for (const [path, src] of sources()) {
      if (/\/_shared\/filmGap\/filmGapStrings\.ts$/.test(path)) continue
      if (!/\/features\/(squad|synthesis|session-detail|timeseries)\//.test(path)) continue
      if (/Film non analysé|Film not analysed/.test(withoutComments(src))) offenders.push(path)
    }
    expect(offenders).toEqual([])
  })
})
