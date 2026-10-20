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
 * Nombre de CANVAS de la vitrine — différent du nombre de wrappers (11) : un chart peut
 * être rendu sur plusieurs couches zrender, et un wrapper peut ne rendre aucun canvas.
 *
 *   - `Heatmap2DChart` : 3 canvas superposés (une seule instance ECharts, trois couches)
 *     depuis `echarts` 6.1.0, là où la 5.6.0 n'en produisait qu'un — les captures de
 *     canvas qui suivent la heatmap sont décalées de +2 (cf. e2e/visual/README.md) ;
 *   - `FirstBloodLanes` : rendu en SVG (« un bitmap figé ne peut pas être net »,
 *     `ChartCard renderer="svg"`) — aucun canvas, donc -1 et les captures qui le suivent
 *     sont décalées de -1.
 *
 * Total : 8 wrappers à 1 canvas + heatmap (3) + `FirstBloodLanes` (0) + 1 canvas monté hors
 * `chart-card` (wrapper qui instancie ECharts sans ChartCard) = 12.
 *
 * Volontairement EN DUR : ajouter un wrapper à la page sans ajouter sa baseline
 * ferait passer le harnais à côté du nouveau composant. L'échec attendu ici est
 * un rappel, pas une gêne.
 */
const SHOWCASE_CANVAS_COUNT = 12

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
