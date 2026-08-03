/**
 * Régression visuelle — pages applicatives denses en graphes (baselines LOCALES).
 *
 * Complète `lab-charts.visual.spec.ts` : la vitrine couvre les wrappers un par un
 * avec des données statiques, ces specs-ci couvrent les graphes TELS QU'ILS SONT
 * COMPOSÉS dans l'app (séries réelles, thème, densité, empilement de cartes) —
 * c'est là que se voient les régressions de rendu qu'un wrapper isolé ne montre pas.
 *
 * BASELINES NON VERSIONNÉES. Ces captures dépendent des données locales et peuvent
 * contenir des gamertags réels : elles sont écrites sous `__screenshots__/app/`,
 * exclu par le `.gitignore` du dossier. Elles servent de référence AVANT/APRÈS pour
 * une opération donnée (bump de dépendance, refonte d'un chart) :
 *
 *   1. avant l'opération : `npx playwright test --project=visual --update-snapshots`
 *   2. après l'opération  : `npx playwright test --project=visual`
 *
 * Joueur visé : `E2E_VISUAL_PLAYER` (défaut `demo-player`). Une page sans graphe
 * (données absentes) SKIPPE au lieu d'échouer — le harnais reste exécutable partout.
 */
import { test, expect } from '@playwright/test'

import { playerPath } from '../_helpers/routes'
import {
  APP_SNAPSHOT_SEGMENTS,
  VISUAL_PLAYER,
  expectCanvasesMatch,
  gotoSettled,
  prepareVisualPage,
} from '../_helpers/visual'

/**
 * Surfaces couvertes. Sélection : les pages les plus denses en graphes, incluant
 * celles dont les charts ont été retouchés en v7.3 lot 2 (bande d'issues en
 * encoches, pistes Rendement/Résistance, courbe d'intensité d'équipe).
 */
const PAGES: { id: string; suffix: string; unstableCanvases?: number[] }[] = [
  { id: 'home', suffix: 'home' },
  { id: 'timeseries-summary', suffix: 'stats/timeseries?tab=summary' },
  // canvas 06/07 = classements cartes et modes : à nombre de matchs ÉGAL, l'ordre
  // des libellés varie d'une réponse d'API à l'autre (tri sans clé de départage
  // côté backend). Les barres sont identiques, seuls les libellés permutent —
  // captures exclues faute d'être comparables, pas parce qu'elles « gênent ».
  // Instabilité MESURÉE (deux générations comparées), pas supposée. Ces deux
  // charts étaient en 04/05 sur `echarts` 5.6.0 : la heatmap de la page occupe
  // 3 canvas depuis la 6.1.0, ce qui décale de +2 tout ce qui la suit. Voir README.
  { id: 'synthesis', suffix: 'stats/synthesis', unstableCanvases: [6, 7] },
  { id: 'squad-synergies', suffix: 'squad/synergies' },
  { id: 'squad-dynamique', suffix: 'squad/dynamique' },
  // canvas 19 = classement des armes, même cause que ci-dessus (ex aequo à 2 et 3).
  { id: 'session-detail', suffix: 'stats/sessions', unstableCanvases: [19] },
  { id: 'community-relations', suffix: 'community/relations' },
]

test.describe(`Régression visuelle — pages applicatives (${VISUAL_PLAYER})`, () => {
  test.beforeEach(async ({ page }) => {
    await prepareVisualPage(page)
  })

  for (const { id, suffix, unstableCanvases } of PAGES) {
    test(`${id} — graphes rendus sans dérive visuelle`, async ({ page }) => {
      const errors: string[] = []
      page.on('pageerror', (err) => errors.push(err.message))

      const canvasCount = await gotoSettled(page, playerPath(VISUAL_PLAYER, suffix))
      test.skip(
        canvasCount === 0,
        `aucun graphe rendu pour ${VISUAL_PLAYER} sur /${suffix} — données locales absentes. ` +
          'Viser un joueur peuplé via E2E_VISUAL_PLAYER.',
      )

      await expect(page.locator('main')).toHaveScreenshot([
        ...APP_SNAPSHOT_SEGMENTS,
        `${id}-page.png`,
      ])
      await expectCanvasesMatch(
        page,
        APP_SNAPSHOT_SEGMENTS,
        id,
        canvasCount,
        unstableCanvases ?? [],
      )

      // Un chart qui explose au rendu peut laisser un canvas vide sans casser la
      // page : l'absence d'erreur JS fait partie du contrat visuel.
      expect(errors.filter((e) => !e.includes('ResizeObserver'))).toHaveLength(0)
    })
  }
})
