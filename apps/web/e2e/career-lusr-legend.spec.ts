/**
 * E2E — Vérifie que le chart "Évolution LUSR / CSR" de la carrière :
 *   1. Montre UNE échelle à la fois (LUSR d'abord, bascule « Échelle de classement » vers
 *      le CSR quand les deux existent) et, pour l'échelle affichée, une série par chaîne
 *      présente dans le payload API — la chaîne est `playlist_group` (arena_slayer /
 *      arena_objectif / btb / chaos pour le LUSR, `ranked` pour le CSR), repli
 *      `arena_slayer` quand elle est absente. Les matchs de placement ne se tracent pas.
 *   2. Libelle chaque série par le libellé FR de sa chaîne suivi du type, sans UUID brut ni
 *      clé interne (« ranked (CSR) ») et sans doublon.
 *
 * Les noms de séries ne sont lisibles ni dans le DOM (légende dessinée dans le canvas) ni
 * via une API ECharts globale (modules importés, pas de `window.echarts`) : on journalise
 * les textes que le canvas dessine (`fillText`), donc ce que l'utilisateur voit réellement.
 */
import { test, expect, type Page } from '@playwright/test'
import { skipIfNoDemoData, E2E_PLAYER_SLUG } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
// Miroir des libellés FR de `career.lusr.chain.*` (manifests/career.toml).
const FR_CHAIN_LABELS: Record<string, string> = {
  arena_slayer: 'Social · Assassin',
  arena_objectif: 'Social · Objectif',
  btb: 'Grande Équipe',
  chaos: 'Chaos',
  ranked: 'Classé',
}
const DEFAULT_CHAIN = 'arena_slayer'
// Ordre des échelles (RATING_SCALES de lusrEvolution.logic.ts) ; libellé = bouton de bascule.
const SCALES = ['LUSR', 'CSR'] as const
// Miroir de PLACEMENT_TIER_SENTINEL (lib/skillTiers.ts).
const PLACEMENT = 'Placement'

type Checkpoint = {
  playlist_group: string | null
  playlist_name: string
  rating_type: string | null
  recorded_at: string | null
  tier_label: string | null
}

const scaleOf = (c: Checkpoint) => ((c.rating_type ?? '').toUpperCase() === 'CSR' ? 'CSR' : 'LUSR')

/** Séries attendues pour une échelle = (rating_type, chaîne) distincts parmi ses checkpoints traçables. */
function expectedFor(checkpoints: Checkpoint[], scale: string) {
  const own = checkpoints.filter((c) => scaleOf(c) === scale)
  const names = new Set(
    own.map((c) => {
      const chain = c.playlist_group ?? DEFAULT_CHAIN
      return `${FR_CHAIN_LABELS[chain] ?? chain} (${c.rating_type ?? 'LUSR'})`
    }),
  )
  const keys = new Set(own.map((c) => `${c.rating_type ?? 'LUSR'}:${c.playlist_group ?? DEFAULT_CHAIN}`))
  return { names, keys }
}

async function drawnSeriesNames(page: Page): Promise<string[]> {
  const texts = await page.evaluate(() => (window as unknown as { __canvasTexts: string[] }).__canvasTexts)
  return [...new Set(texts.filter((t) => /\((LUSR|CSR)\)$/.test(t)))]
}

function expectCanonicalLabels(seriesNames: string[]) {
  for (const n of seriesNames) {
    const labelOnly = n.replace(/\s*\((LUSR|CSR)\)\s*$/, '')
    expect(UUID_RE.test(labelOnly), `UUID brut en légende : "${n}"`).toBe(false)
    expect(Object.values(FR_CHAIN_LABELS), `Libellé non canonique en légende : "${n}"`).toContain(labelOnly)
  }
}

test("Carrière — chart LUSR : 1 entrée par chaîne, libellés FR localisés, pas d'UUID", async ({ page }) => {
  // Journalise tout texte dessiné dans un canvas, AVANT le script de page.
  await page.addInitScript(() => {
    const drawn: string[] = []
    ;(window as unknown as { __canvasTexts: string[] }).__canvasTexts = drawn
    const original = CanvasRenderingContext2D.prototype.fillText
    CanvasRenderingContext2D.prototype.fillText = function (text, ...rest) {
      drawn.push(String(text))
      return original.call(this, text, ...rest)
    }
  })

  const careerPromise = page.waitForResponse(
    (resp) =>
      resp.url().includes(`/api/v1/players/${E2E_PLAYER_SLUG}/pages/career`) &&
      !resp.url().includes('encounters') &&
      resp.status() === 200,
  )

  await page.goto(playerPath(E2E_PLAYER_SLUG, 'career'))
  const resp = await careerPromise
  const data = await resp.json()
  const chartable = (data.lusr.checkpoints as Checkpoint[]).filter(
    (c) => c.recorded_at && !(c.tier_label ?? '').trim().startsWith(PLACEMENT),
  )
  const scales = SCALES.filter((s) => chartable.some((c) => scaleOf(c) === s))
  console.log('[E2E] checkpoints traçables:', chartable.length, '— échelles:', scales)

  await page.locator('text=Évolution LUSR / CSR').waitFor({ state: 'visible', timeout: 15000 })
  const toggle = page.getByRole('group', { name: 'Échelle de classement' })
  // La bascule n'apparaît que si les deux échelles existent.
  await expect(toggle).toHaveCount(scales.length > 1 ? 1 : 0)

  for (const [i, scale] of scales.entries()) {
    if (i > 0) {
      await page.evaluate(() => {
        ;(window as unknown as { __canvasTexts: string[] }).__canvasTexts.length = 0
      })
      await toggle.getByRole('button', { name: scale, exact: true }).click()
    }
    await page.waitForTimeout(2500) // ECharts mount + dessin de la légende

    const { names, keys } = expectedFor(chartable, scale)
    // Deux chaînes distinctes ne doivent jamais se rendre sous le même libellé.
    expect(names.size).toBe(keys.size)
    const seriesNames = await drawnSeriesNames(page)
    console.log(`[E2E] ${scale} attendues:`, [...names], '— dessinées:', seriesNames)

    // 1. Aucun UUID ni clé interne en légende : chaque nom est un libellé FR connu.
    expectCanonicalLabels(seriesNames)
    // 2. Légende = exactement les séries de l'échelle affichée (une par chaîne, aucune en trop).
    expect([...seriesNames].sort()).toEqual([...names].sort())
  }
})
