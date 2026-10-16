/**
 * E2E — Vérifie que le chart "Évolution LUSR / CSR" de la carrière :
 *   1. Produit une série par (rating_type, chaîne) présent dans le payload API — la chaîne
 *      est `playlist_group` (arena_slayer / arena_objectif / btb / chaos pour le LUSR,
 *      `ranked` pour le CSR), repli `arena_slayer` quand elle est absente.
 *   2. Libelle chaque série par le libellé FR de sa chaîne suivi du type, sans UUID brut ni
 *      clé interne (« ranked (CSR) ») et sans doublon.
 *
 * Les noms de séries ne sont lisibles ni dans le DOM (légende dessinée dans le canvas) ni
 * via une API ECharts globale (modules importés, pas de `window.echarts`) : on journalise
 * les textes que le canvas dessine (`fillText`), donc ce que l'utilisateur voit réellement.
 */
import { test, expect } from '@playwright/test'
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
      !resp.url().includes('top-matches') &&
      !resp.url().includes('encounters') &&
      resp.status() === 200,
  )

  await page.goto(playerPath(E2E_PLAYER_SLUG, 'career'))
  const resp = await careerPromise
  const data = await resp.json()
  const checkpoints = data.lusr.checkpoints as Array<{
    playlist_group: string | null
    playlist_name: string
    rating_type: string
    recorded_at: string | null
  }>

  // Séries attendues = (rating_type, chaîne) distincts parmi les checkpoints datés.
  const expectedNames = new Set(
    checkpoints
      .filter((c) => c.recorded_at)
      .map((c) => {
        const chain = c.playlist_group ?? DEFAULT_CHAIN
        return `${FR_CHAIN_LABELS[chain] ?? chain} (${c.rating_type})`
      }),
  )
  const expectedKeys = new Set(
    checkpoints
      .filter((c) => c.recorded_at)
      .map((c) => `${c.rating_type}:${c.playlist_group ?? DEFAULT_CHAIN}`),
  )
  // Deux chaînes distinctes ne doivent jamais se rendre sous le même libellé.
  expect(expectedNames.size).toBe(expectedKeys.size)

  await page.locator('text=Évolution LUSR / CSR').waitFor({ state: 'visible', timeout: 15000 })
  await page.waitForTimeout(2500) // ECharts mount + dessin de la légende

  const seriesNames = [
    ...new Set(
      (await page.evaluate(
        () => (window as unknown as { __canvasTexts: string[] }).__canvasTexts,
      )).filter((t) => /\((LUSR|CSR)\)$/.test(t)),
    ),
  ]

  console.log('[E2E] checkpoints (total):', checkpoints.length)
  console.log('[E2E] expected series:', expectedNames.size, '→', [...expectedNames])
  console.log('[E2E] series rendered:', seriesNames.length, '→', seriesNames)

  // 1. Aucun UUID ni clé interne en légende : chaque nom est un libellé FR connu.
  for (const n of seriesNames) {
    const labelOnly = n.replace(/\s*\((LUSR|CSR)\)\s*$/, '')
    expect(UUID_RE.test(labelOnly), `UUID brut en légende : "${n}"`).toBe(false)
    expect(
      Object.values(FR_CHAIN_LABELS),
      `Libellé non canonique en légende : "${n}"`,
    ).toContain(labelOnly)
  }

  // 2. Légende = exactement les séries attendues (une par chaîne, aucune en trop).
  expect([...seriesNames].sort()).toEqual([...expectedNames].sort())
})
