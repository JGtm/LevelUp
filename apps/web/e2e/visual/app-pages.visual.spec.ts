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
/**
 * Bandeau décoratif de l'accueil, MASQUÉ dans la capture de page.
 *
 * `HomeHeroBanner` tire son image AU HASARD (`Math.random`) parmi 7 visuels
 * statiques du titre, et en change toutes les 45 s. Le `Math.random` seedé de
 * `prepareVisualPage` ne suffit pas à le figer : le tirage dépend du NOMBRE
 * d'appels consommés avant le montage du composant, qui varie avec l'ordre
 * d'arrivée des réponses réseau. Deux passes sur des données strictement
 * identiques affichaient donc deux visuels différents (8 % des pixels de la page).
 *
 * Ce bandeau est purement décoratif — aucun graphe, aucune donnée joueur : le
 * masquer retire le seul bruit non déterministe de la capture sans réduire la
 * couverture de régression. Locator inoffensif sur les pages qui n'en ont pas
 * (il ne résout alors aucun élément).
 */
const HERO_BANNER_TESTID = '[data-testid="home-hero-banner-sticky"]'

/**
 * Mode STRICT : exige que les 7 pages rendent au moins un graphe (échec sinon,
 * au lieu du skip). Posé par `scripts/demo-visual-harness.sh`, qui vise une
 * fixture démo censée peupler toutes les surfaces. Hors démo (joueur réel du
 * poste, données partielles), le skip reste le comportement voulu.
 */
const REQUIRE_ALL_PAGES = process.env.E2E_VISUAL_REQUIRE_ALL === '1'

const PAGES: { id: string; suffix: string; unstableCanvases?: number[] }[] = [
  { id: 'home', suffix: 'home' },
  { id: 'timeseries-summary', suffix: 'stats/timeseries?tab=summary' },
  // canvas 06/07 = classements cartes et modes : à nombre de matchs ÉGAL, l'ordre
  // des libellés variait d'une réponse d'API à l'autre (tri sans clé de départage
  // côté backend). CAUSE CORRIGÉE le 2026-08-04 — sortMapEntries/sortModeEntries
  // (service/synthesis_service_legacy.go) départagent désormais par nom. Les
  // exclusions restent le temps d'une regénération de baselines qui confirme la
  // stabilité sur données réelles ; les retirer alors (ne pas les garder « au cas
  // où »). Ces deux charts étaient en 04/05 sur `echarts` 5.6.0 : la heatmap de la
  // page occupe 3 canvas depuis la 6.1.0, ce qui décale de +2 tout ce qui la suit.
  { id: 'synthesis', suffix: 'stats/synthesis', unstableCanvases: [6, 7] },
  { id: 'squad-synergies', suffix: 'squad/synergies' },
  { id: 'squad-dynamique', suffix: 'squad/dynamique' },
  // canvas 19 = classement des armes, même cause et même correction (départage par
  // label dans buildTopWeaponKills, service/synthesis_service_builders.go).
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
      if (REQUIRE_ALL_PAGES) {
        // Mode strict (fixture démo) : une page sans graphe est un ÉCHEC, pas un
        // skip. Sans ce durcissement, une démo qui sert les mauvaises données
        // rend un « zéro diff » parfaitement vert sur 6 pages SKIPPÉES — le faux
        // vert qui a laissé passer le bug de résolution des DB de la fixture.
        expect(
          canvasCount,
          `aucun graphe rendu pour ${VISUAL_PLAYER} sur /${suffix}. En mode démo strict ` +
            "c'est une régression : la fixture doit peupler les 7 pages. Vérifier que l'API " +
            'sert bien les DB de la fixture (log « demo_mode: utilisation fixture »).',
        ).toBeGreaterThan(0)
      } else {
        test.skip(
          canvasCount === 0,
          `aucun graphe rendu pour ${VISUAL_PLAYER} sur /${suffix} — données locales absentes. ` +
            'Viser un joueur peuplé via E2E_VISUAL_PLAYER.',
        )
      }

      await expect(page.locator('main')).toHaveScreenshot(
        [...APP_SNAPSHOT_SEGMENTS, `${id}-page.png`],
        { mask: [page.locator(HERO_BANNER_TESTID)] },
      )
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
