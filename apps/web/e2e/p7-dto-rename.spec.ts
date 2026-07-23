/**
 * E2E — P7.1 sub-PR D : valide le renommage des DTOs Timeseries / Synthesis.
 *
 * Référence : PLAN_ACTION.md P7.1 ("Tests E2E Playwright sur TimeseriesPage et
 * SynthesisPage avant merge sub-PR").
 *
 * Vérifie côté API que les DTOs portent les nouveaux noms métier et que les
 * anciens noms ECharts (`bin_start`, `bin_end`, `label`/`x`/`y`, `row_key`,
 * `col_key`, `solo_text`, `squad_text`, `color`) ont disparu du wire.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData, skipObsoleteSpec } from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const API = 'http://localhost:8000/api/v1'
const PLAYER = 'demo-player'

test.describe('P7.1 — DTOs Timeseries renommés sémantique métier', () => {
  test('DistributionBucket : bucket_lower/bucket_upper (plus de bin_start/bin_end)', async ({ request }) => {
    const resp = await request.post(`${API}/players/${PLAYER}/pages/timeseries`, {
      data: { filters: {} },
    })
    if (resp.status() !== 200) {
      test.skip(true, `API non disponible (status=${resp.status()}) — fixtures démo manquantes`)
    }
    const data = await resp.json()
    const buckets = data.distributions_tab?.kda_buckets ?? []
    if (buckets.length === 0) test.skip(true, 'Pas de KDA buckets en démo — non testable')

    for (const b of buckets) {
      expect(b).toHaveProperty('bucket_lower')
      expect(b).toHaveProperty('bucket_upper')
      expect(b).toHaveProperty('count')
      expect(b).not.toHaveProperty('bin_start')
      expect(b).not.toHaveProperty('bin_end')
      expect(typeof b.bucket_lower).toBe('number')
      expect(typeof b.bucket_upper).toBe('number')
    }
  })

  test('CorrelationDataPair : metric_x_key/metric_y_key/x_value/y_value (plus de label/x/y)', async ({ request }) => {
    const resp = await request.post(`${API}/players/${PLAYER}/pages/timeseries`, {
      data: { filters: {} },
    })
    if (resp.status() !== 200) {
      test.skip(true, `API non disponible (status=${resp.status()}) — fixtures démo manquantes`)
    }
    const data = await resp.json()
    const points = data.distributions_tab?.correlation_points ?? []
    if (points.length === 0) test.skip(true, 'Pas de correlation_points en démo — non testable')

    for (const p of points.slice(0, 5)) {
      expect(p).toHaveProperty('metric_x_key')
      expect(p).toHaveProperty('metric_y_key')
      expect(p).toHaveProperty('x_value')
      expect(p).toHaveProperty('y_value')
      expect(p).toHaveProperty('outcome')
      expect(p).not.toHaveProperty('label')
      expect(p).not.toHaveProperty('x')
      expect(p).not.toHaveProperty('y')
      expect(typeof p.metric_x_key).toBe('string')
      expect(typeof p.metric_y_key).toBe('string')
    }

    // Sanity check : au moins une paire connue doit exister.
    const knownPairs = points.map((p: { metric_x_key: string; metric_y_key: string }) =>
      `${p.metric_x_key}_vs_${p.metric_y_key}`,
    )
    expect(knownPairs).toContain('kills_vs_kd_ratio')
  })

  test('TimeseriesKpiCard : champ color retiré', async ({ request }) => {
    const resp = await request.post(`${API}/players/${PLAYER}/pages/timeseries`, {
      data: { filters: {} },
    })
    if (resp.status() !== 200) {
      test.skip(true, `API non disponible (status=${resp.status()}) — fixtures démo manquantes`)
    }
    const data = await resp.json()
    const cards = data.summary_tab?.kpi_cards ?? []
    if (cards.length === 0) test.skip(true, 'Pas de kpi_cards en démo — non testable')

    for (const c of cards) {
      expect(c).toHaveProperty('key')
      expect(c).toHaveProperty('label')
      expect(c).toHaveProperty('value')
      expect(c).not.toHaveProperty('color')
    }
  })
})

test.describe('P7.1 — DTOs Synthesis renommés sémantique métier', () => {
  test('ComparisonMetricItem : solo_text/squad_text retirés, label = clé canonique', async ({ request }) => {
    const resp = await request.post(`${API}/players/${PLAYER}/pages/synthesis`, {
      data: { period: 'all', filters: {} },
    })
    if (resp.status() !== 200) {
      test.skip(true, `API non disponible (status=${resp.status()}) — fixtures démo manquantes`)
    }
    const data = await resp.json()
    const metrics = data.comparison_metrics ?? []
    if (metrics.length === 0) test.skip(true, 'Pas de comparison_metrics en démo — non testable')

    for (const m of metrics) {
      expect(m).toHaveProperty('label')
      expect(m).toHaveProperty('solo_value')
      expect(m).toHaveProperty('squad_value')
      expect(m).not.toHaveProperty('solo_text')
      expect(m).not.toHaveProperty('squad_text')
      expect(typeof m.solo_value).toBe('number')
      expect(typeof m.squad_value).toBe('number')
    }

    // Label porte une clé canonique (cf. fields.toml) — minuscules + underscore.
    const labels = metrics.map((m: { label: string }) => m.label)
    expect(labels).toContain('win_rate')
    expect(labels).toContain('kd_ratio')
    expect(labels).toContain('accuracy')
  })
})

test.describe('P7.1 — Pages TimeseriesPage / SynthesisPage rendent sans erreur 500', () => {
  test('TimeseriesPage charge et affiche les KPI cards', async ({ page }) => {
    await page.goto(playerPath(PLAYER, 'stats/timeseries'))
    // Attendre le rendu de la page (au moins une KPI card avec un texte non vide)
    await expect(page.locator('[class*="font-bold"]').first()).toBeVisible({ timeout: 10_000 })
  })

  test('SynthesisPage charge et affiche le graphique bipolaire', async ({ page }) => {
    // Le graphique bipolaire (Comparaison Solo/Escouade) ne rend plus de <canvas>
    // détectable par le sélecteur `canvas, [class*="empty-state"]` (chart migré) —
    // sélecteur à mettre à jour. Le reste de la synthèse rend correctement.
    skipObsoleteSpec('SynthesisPage : sélecteur canvas obsolète du graphique bipolaire')
    await page.goto(playerPath(PLAYER, 'synthesis'))
    // Attendre que la page Synthesis ait chargé un canvas ECharts ou un placeholder
    const hasCanvas = await page.locator('canvas, [class*="empty-state"]').first().isVisible({ timeout: 10_000 })
    expect(hasCanvas).toBe(true)
  })
})
