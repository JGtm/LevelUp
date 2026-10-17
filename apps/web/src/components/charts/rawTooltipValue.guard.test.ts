/**
 * Garde-rail (CLAUDE.md n°6) : une valeur numérique de graphe ne s'écrit JAMAIS brute dans une
 * infobulle ou une étiquette (« 1.3333333333333333 », milliers collés, point décimal sous une
 * interface française) — retours utilisateur du 2026-10-17. Une valeur ECharts passe par
 * `tipNumberAuto` / `tipNumber` (`./tipNumber`), jamais par `${p.value}` ni `String(p.value)`, ni par
 * un `toFixed` qui sort un texte.
 *
 * Garde 1 — interpolation : une interpolation ou un `String(...)` du champ `value` ou `data` d'un
 * paramètre ECharts (`p`, `param`, `params`, `item`, `point`, `d`, `it`, `r`). Fiable car le champ
 * `value` d'un paramètre ECharts EST la valeur numérique brute. Périmètre : TOUT `src/` hors tests
 * (relevé nul au 2026-10-17, plus aucune surface non migrée).
 *
 * Garde 1 bis — gabarit ECharts : un `formatter` en chaîne qui pose `{c}` (la valeur) ou `{d}` (la part) sort le
 * nombre sans la langue de l'application (« 33.33% »). Périmètre : tout `src/` hors tests, relevé nul au 2026-10-17.
 *
 * Garde 2 — `toFixed` : dans un fichier qui construit un graphe (il importe `getTooltipBase` ou type
 * son option `EChartsCoreOption`), un `.toFixed(` ne sort jamais un texte affiché. Périmètre :
 * `components/charts`, Sessions, Synthèse, Séries temporelles et le calendrier d'activité. RESTE
 * HORS PÉRIMÈTRE, nommé et non traité (autres pages) : `features/career` (axe des XP),
 * `features/explorer` (carte d'activité, cadence). Les arrondis qui rendent un NOMBRE (`parseFloat`,
 * `Number`, `+`) ne sont pas un affichage : ils sont listés ci-dessous.
 */
import { describe, expect, it } from 'vitest'

const RAW_VALUE =
  /\$\{\s*(?:p|param|params|item|point|d|it|r)\.(?:value|data)(?:\[\d+\]|\?\.\[\d+\])?\s*(?:\?\?[^}]*)?\}|String\(\s*(?:p|param|params|item)\.value\s*\)/

const RAW_TEMPLATE = /formatter:\s*['"`][^'"`]*\{(?:c|d)\}/

const TO_FIXED = /\.toFixed\(/

const FIXED_SCOPE = [
  /^\/src\/components\/charts\//,
  /^\/src\/features\/session-detail\//,
  /^\/src\/features\/synthesis\//,
  /^\/src\/features\/timeseries\//,
  /^\/src\/features\/ascension\/ActivityCalendarChart\.tsx$/,
]

/** Un fichier qui construit un graphe ECharts. */
const BUILDS_A_CHART = /getTooltipBase|EChartsCoreOption/

/** Arrondis qui rendent un nombre (jamais un texte affiché) : motif de ligne autorisé, justifié. */
const ROUNDING_TO_NUMBER = [
  /parseFloat\([^)]*\.toFixed\(/, // sessionBarsTrendChart.round1
  /Number\([^)]*\.toFixed\(/, // SessionFdaBars.fmt : valeur de donnée du bâton
  /\+\w[\w.]*\.toFixed\(\d\)/, // SynthesisTopWeeksChart : valeur de donnée de la courbe
  /left: `\$\{[^}]*\.toFixed\(\d\)\}%`/, // barStackedGroups : position CSS, pas un texte lu
]

function sources(): Array<[string, string]> {
  const files = import.meta.glob('/src/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<
    string,
    string
  >
  return Object.entries(files).filter(([path]) => !/\.test\.tsx?$/.test(path))
}

describe('valeur brute de graphe', () => {
  it('le motif d’interpolation reconnaît les écritures fautives', () => {
    for (const bad of ['`<b>${p.value}</b>`', '`${p.value[1]}`', '`${p.value?.[1] ?? 0}`', 'String(p.value)', '`${d.value[2]}`']) {
      expect(RAW_VALUE.test(bad), bad).toBe(true)
    }
    for (const good of ['tipNumberAuto(p.value)', '`${tipNumberAuto(p.value[1])}`', '`${p.name}`', 'String(p.name)']) {
      expect(RAW_VALUE.test(good), good).toBe(false)
    }
  })

  it('le motif de gabarit reconnaît `{c}` et `{d}`, pas `{b}`', () => {
    expect(RAW_TEMPLATE.test("formatter: '{b} : <b>{c}</b> ({d}%)'")).toBe(true)
    expect(RAW_TEMPLATE.test("formatter: '{d}%'")).toBe(true)
    expect(RAW_TEMPLATE.test("formatter: '{b}'")).toBe(false)
  })

  it('aucun graphe de l’application n’écrit une valeur brute', () => {
    const offenders: string[] = []
    for (const [path, src] of sources()) {
      src.split('\n').forEach((line, i) => {
        if (RAW_VALUE.test(line) || RAW_TEMPLATE.test(line)) offenders.push(`${path}:${i + 1} ${line.trim()}`)
      })
    }
    expect(offenders, `valeur brute dans un graphe (utiliser tipNumberAuto / tipNumber) :\n${offenders.join('\n')}`).toEqual([])
  })

  it('aucun fichier de graphe des pages Sessions, Synthèse, Séries temporelles et calendrier n’écrit un toFixed', () => {
    const offenders: string[] = []
    for (const [path, src] of sources()) {
      if (!FIXED_SCOPE.some((re) => re.test(path)) || !BUILDS_A_CHART.test(src)) continue
      src.split('\n').forEach((line, i) => {
        if (!TO_FIXED.test(line) || ROUNDING_TO_NUMBER.some((re) => re.test(line))) return
        if (/^\s*(\/\/|\*|\/\*)/.test(line)) return
        offenders.push(`${path}:${i + 1} ${line.trim()}`)
      })
    }
    expect(offenders, `toFixed dans un graphe (utiliser tipNumber, nombre à la langue) :\n${offenders.join('\n')}`).toEqual([])
  })
})
