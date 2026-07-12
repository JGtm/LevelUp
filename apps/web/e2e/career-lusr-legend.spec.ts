/**
 * E2E — Vérifie que le chart "Évolution LUSR / CSR" :
 *   1. Produit au plus 4 séries (1 par playlist_group canonique).
 *   2. Tous les libellés sont des libellés localisés FR (Arène / Grand combat
 *      / Social / Classé) — aucun UUID brut, aucun doublon.
 *   3. Le compte rendu correspond aux groupes canoniques effectivement
 *      présents dans le payload API (après normalisation social→arena +
 *      filtrage UUIDs).
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const FR_GROUP_LABELS = new Set(['Arène', 'Grand combat', 'Social', 'Classé'])

function normalizeGroup(g: string | null | undefined): 'arena' | 'btb' | 'fun' | 'ranked' {
  const v = (g ?? '').trim().toLowerCase()
  if (v === 'btb') return 'btb'
  if (v === 'fun') return 'fun'
  if (v === 'ranked') return 'ranked'
  return 'arena'
}

test('Carrière — chart LUSR : 1 entrée par playlist_group, libellés FR localisés, pas d\'UUID', async ({ page }) => {
  const careerPromise = page.waitForResponse(
    (resp) =>
      resp.url().includes('/api/v1/players/JGtm/pages/career') &&
      !resp.url().includes('top-matches') &&
      !resp.url().includes('encounters') &&
      resp.status() === 200,
  )

  await page.goto('/players/JGtm/career')
  const resp = await careerPromise
  const data = await resp.json()
  const checkpoints = data.lusr.checkpoints as Array<{
    playlist_group: string | null
    playlist_name: string
    rating_type: string
  }>

  // Compte attendu = (rating_type, group canonique) distincts après normalisation + filtre UUID
  const expectedKeys = new Set(
    checkpoints
      .filter((c) => c.playlist_name && !UUID_RE.test(c.playlist_name))
      .map((c) => `${c.rating_type}:${normalizeGroup(c.playlist_group)}`),
  )

  await page.locator('text=Évolution LUSR / CSR').waitFor({ state: 'visible', timeout: 15000 })
  await page.waitForTimeout(2500) // ECharts mount

  const seriesNames: string[] = await page.evaluate(() => {
    const containers = Array.from(document.querySelectorAll('[_echarts_instance_]')) as HTMLElement[]
    for (const el of containers) {
      const instId = el.getAttribute('_echarts_instance_')
      if (!instId) continue
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const w = window as any
      const inst = w.echarts?.getInstanceByDom?.(el) ?? w.__echartsInstanceMap__?.[instId]
      if (!inst || typeof inst.getOption !== 'function') continue
      const opt = inst.getOption()
      const series = (opt.series ?? []) as Array<{ name?: string }>
      const names = series.map((s) => s.name ?? '').filter((n) => /\((LUSR|CSR)\)/.test(n))
      if (names.length > 0) return names
    }
    return []
  })

  console.log('[E2E] checkpoints (total):', checkpoints.length)
  console.log('[E2E] expected canonical keys:', expectedKeys.size, '→', [...expectedKeys])
  console.log('[E2E] series rendered:', seriesNames.length, '→', seriesNames)

  // 1. Au plus 4 séries par rating_type (4 groupes canoniques max)
  expect.soft(seriesNames.length).toBeLessThanOrEqual(8)

  // 2. Aucun UUID, aucun doublon, libellé FR canonique
  const seen = new Map<string, number>()
  for (const n of seriesNames) {
    seen.set(n, (seen.get(n) ?? 0) + 1)
    const labelOnly = n.replace(/\s*\((LUSR|CSR)\)\s*$/, '')
    expect(UUID_RE.test(labelOnly), `UUID brut en légende : "${n}"`).toBe(false)
    expect(FR_GROUP_LABELS.has(labelOnly), `Libellé non canonique en légende : "${n}"`).toBe(true)
  }
  for (const [label, count] of seen) {
    expect(count, `Label "${label}" apparaît ${count} fois`).toBe(1)
  }

  // 3. Compte = nb de (rating_type, group) distincts
  expect(seriesNames.length).toBe(expectedKeys.size)
})
