/**
 * Garde-rail (CLAUDE.md n°6) : une valeur numérique de graphe ne s'écrit JAMAIS brute dans une
 * infobulle ou une étiquette (« 1.3333333333333333 », milliers collés) — retours utilisateur du
 * 2026-10-17. Dans les graphes de `components/charts`, de l'Escouade et de la Vue match, une valeur
 * ECharts passe par `tipNumberAuto` / `tipNumber` (`./tipNumber`), jamais par `${p.value}` ni
 * `String(p.value)`.
 *
 * Motif : une interpolation ou un `String(...)` du champ `value` ou `data` d'un paramètre ECharts
 * (`p`, `param`, `params`, `item`, `point`, `d`, `it`, `r`). Fiable car le champ `value` d'un paramètre
 * ECharts EST la valeur numérique brute ; faux positif impossible dans ces trois dossiers (relevé nul
 * au 2026-10-17).
 *
 * HORS PÉRIMÈTRE, à migrer (découvertes du même jour, non traitées) : `features/session-detail`
 * (Engagement, Perf, Participation, Placement, Mode), `features/synthesis` (Résultats par groupe, Top
 * semaines), `features/ascension/ActivityCalendarChart`. Les ajouter ici quand ils seront migrés.
 */
import { describe, expect, it } from 'vitest'

const RAW_VALUE =
  /\$\{\s*(?:p|param|params|item|point|d|it|r)\.(?:value|data)(?:\[\d+\]|\?\.\[\d+\])?\s*(?:\?\?[^}]*)?\}|String\(\s*(?:p|param|params|item)\.value\s*\)/

const SCOPE = [/^\/src\/components\/charts\//, /^\/src\/features\/squad\//, /^\/src\/features\/match-view\//]

describe('valeur brute de graphe', () => {
  it('le motif reconnaît les écritures fautives', () => {
    for (const bad of ['`<b>${p.value}</b>`', '`${p.value[1]}`', '`${p.value?.[1] ?? 0}`', 'String(p.value)', '`${d.value[2]}`']) {
      expect(RAW_VALUE.test(bad), bad).toBe(true)
    }
    for (const good of ['tipNumberAuto(p.value)', '`${tipNumberAuto(p.value[1])}`', '`${p.name}`', 'String(p.name)']) {
      expect(RAW_VALUE.test(good), good).toBe(false)
    }
  })

  it('aucun graphe de components/charts, de l’Escouade ou de la Vue match n’écrit une valeur brute', () => {
    const files = import.meta.glob('/src/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
      string,
      string
    >
    const offenders: string[] = []
    for (const [path, src] of Object.entries(files)) {
      if (!SCOPE.some((re) => re.test(path)) || /\.test\.tsx?$/.test(path)) continue
      src.split('\n').forEach((line, i) => {
        if (RAW_VALUE.test(line)) offenders.push(`${path}:${i + 1} ${line.trim()}`)
      })
    }
    expect(offenders, `valeur brute dans un graphe (utiliser tipNumberAuto / tipNumber) :\n${offenders.join('\n')}`).toEqual([])
  })
})
