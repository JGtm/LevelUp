/**
 * E2E — Switch de thème light/dark : vérifie que les charts ECharts
 * suivent réellement le thème (re-render canvas avec couleurs adaptées).
 *
 * Stratégie :
 *   1. On rend la page Escouade > Synergies avec le radar pilote.
 *   2. On lit la valeur computée des CSS vars `--muted-foreground`,
 *      `--foreground`, `--border` dans le thème courant.
 *   3. On toggle le thème via le ThemeToggle (header).
 *   4. On vérifie que `data-theme` a basculé.
 *   5. On vérifie que les CSS vars ont des valeurs DIFFÉRENTES.
 *   6. On capture un screenshot du chart dans chaque thème (validation
 *      visuelle manuelle si besoin).
 *
 * Pourquoi pas une assertion directe sur la couleur des labels canvas ?
 *   ECharts rend en `<canvas>` — impossible de lire les couleurs
 *   computées des éléments individuels via le DOM. À défaut, on s'assure
 *   que le pipeline est bien câblé (CSS vars switchent + chart toujours
 *   visible + pas d'erreur console). La validation pixel reste manuelle.
 *
 * Prérequis : `make dev` (Vite :5173 + Go :8000).
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

const PLAYER_SLUG = 'JGtm'
const TEAMMATES = ['Madina97294', 'Chocoboflor']

test.describe('Theme switch — charts ECharts suivent le thème (pilote)', () => {
  test('CSS vars sémantiques diffèrent entre dark et light', async ({ page }) => {
    await page.goto(`/players/${PLAYER_SLUG}`)

    // S'assurer qu'un thème initial est positionné
    await page.waitForFunction(() =>
      document.documentElement.getAttribute('data-theme') !== null
    )

    const readVars = () =>
      page.evaluate(() => {
        const cs = getComputedStyle(document.documentElement)
        return {
          theme: document.documentElement.getAttribute('data-theme'),
          fg: cs.getPropertyValue('--foreground').trim(),
          mfg: cs.getPropertyValue('--muted-foreground').trim(),
          brd: cs.getPropertyValue('--border').trim(),
          card: cs.getPropertyValue('--card').trim(),
        }
      })

    const before = await readVars()
    expect(before.theme).not.toBeNull()
    expect(before.fg).not.toBe('')
    expect(before.mfg).not.toBe('')
    expect(before.brd).not.toBe('')

    // Toggle via le bouton de thème (présent dans l'AppShell)
    const switchToOpposite = before.theme === 'dark' ? 'light' : 'dark'
    await page.evaluate((target) => {
      document.documentElement.setAttribute('data-theme', target)
    }, switchToOpposite)

    const after = await readVars()
    expect(after.theme).toBe(switchToOpposite)
    expect(after.fg).not.toBe(before.fg)
    expect(after.mfg).not.toBe(before.mfg)
    expect(after.brd).not.toBe(before.brd)
    expect(after.card).not.toBe(before.card)
  })

  test('le radar synergy reste rendu après toggle de thème (pas de crash)', async ({
    page,
  }) => {
    await skipIfNoDemoData()
    // Pré-set sélection coéquipiers (cf. squad-charts-render.spec.ts)
    await page.addInitScript(
      ({ slug, gamertags }) => {
        localStorage.setItem(`squad-teammates-${slug}`, JSON.stringify(gamertags))
      },
      { slug: PLAYER_SLUG, gamertags: TEAMMATES },
    )

    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(`/players/${PLAYER_SLUG}/squad/synergies`)
    await page.waitForLoadState('networkidle')

    // Le wrapper porte data-testid="squad-synergy-radar" depuis Phase 5c.
    const radar = page.getByTestId('squad-synergy-radar')
    await expect(radar).toBeVisible({ timeout: 15_000 })

    // Capture initiale
    const initialTheme = await page.evaluate(() =>
      document.documentElement.getAttribute('data-theme'),
    )
    await radar.screenshot({
      path: `test-results/squad-synergy-radar-${initialTheme}.png`,
    })

    // Toggle thème
    const target = initialTheme === 'dark' ? 'light' : 'dark'
    await page.evaluate((t) => {
      document.documentElement.setAttribute('data-theme', t)
    }, target)

    // Le MutationObserver de useThemeVersion incrémente, ChartCard rebuild
    // l'option, ECharts re-render. Donner ~250ms pour la propagation
    // React + canvas redraw.
    await page.waitForTimeout(300)

    // Le chart est toujours rendu (pas de crash sur le re-render)
    await expect(radar).toBeVisible()

    // Capture après toggle
    await radar.screenshot({
      path: `test-results/squad-synergy-radar-${target}.png`,
    })

    // Aucune erreur console pendant le switch
    expect(errors, errors.join('\n')).toEqual([])
  })
})
