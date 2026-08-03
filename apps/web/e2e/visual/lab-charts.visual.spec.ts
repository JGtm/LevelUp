/**
 * Régression visuelle — vitrine `/lab/charts` (baselines VERSIONNÉES).
 *
 * Cette page monte 11 wrappers ECharts sur des données STATIQUES : aucun backend,
 * aucune fixture, aucun joueur. C'est donc la seule surface dont les captures sont
 * reproductibles à l'identique par n'importe qui, et le garde-rail durable contre
 * un changement de rendu du moteur (bump majeur d'`echarts`, changement de thème,
 * régression d'un wrapper).
 *
 * Wrappers couverts, dans l'ordre de la page : TimeseriesLineChart, BarStackedChart,
 * BarGroupedChart, DonutChart, Heatmap2DChart, HistogramChart, ScatterChart,
 * RadarChart, OutcomeSequenceTape, FirstBloodLanes, TimeseriesKdaBars.
 *
 * Prérequis : Vite sur :5173 (`make dev`). Aucune donnée backend nécessaire.
 */
import { test, expect } from '@playwright/test'

import { expectCanvasesMatch, gotoSettled, prepareVisualPage } from '../_helpers/visual'

/**
 * Nombre de wrappers de la vitrine. Volontairement EN DUR : ajouter un wrapper à
 * la page sans ajouter sa baseline ferait passer le harnais à côté du nouveau
 * composant. L'échec attendu ici est un rappel, pas une gêne.
 */
const SHOWCASE_CANVAS_COUNT = 11

test.describe('Régression visuelle — vitrine des wrappers ECharts', () => {
  test.beforeEach(async ({ page }) => {
    await prepareVisualPage(page)
  })

  test('la page vitrine rend ses 11 wrappers sans dérive visuelle', async ({ page }) => {
    const canvasCount = await gotoSettled(page, '/lab/charts')
    expect(canvasCount).toBe(SHOWCASE_CANVAS_COUNT)

    // Contexte de mise en page (grille, cartes, titres) — détecte une régression de
    // layout que les captures de canvas isolés ne verraient pas.
    await expect(page.locator('main')).toHaveScreenshot(['lab', 'showcase-page.png'])

    // Puis chaque wrapper isolément : une diff désigne le composant fautif.
    await expectCanvasesMatch(page, ['lab'], 'showcase', SHOWCASE_CANVAS_COUNT)
  })
})
