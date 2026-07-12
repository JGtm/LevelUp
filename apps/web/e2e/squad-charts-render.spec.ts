/**
 * E2E — Squad : vérifie que les graphes s'affichent vraiment.
 *
 * CE TEST AURAIT DÛ EXISTER. Le bug "Aucun graphe ne s'affiche" pour
 * les coéquipiers existants venait d'une erreur SQL Binder côté backend
 * (Q30SquadMatches référençait p1.perfect_kills, colonne inexistante
 * dans shared.match_participants — `perfect_kills` doit être agrégée
 * depuis shared.medals_earned). Le service retournait silencieusement
 * teammates=[] et le frontend affichait l'empty state "Aucune donnée
 * commune". Les tests Vitest mockent useSquadContext et n'ont jamais
 * exercé le pipeline complet front → API → DB → response → render.
 *
 * Prérequis : `make dev` (Vite :5173 + Go :8000) avec données réelles
 * pour le joueur JGtm + Madina97294 + Chocoboflor.
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const PLAYER_SLUG = 'JGtm'
const TEAMMATES = ['Madina97294', 'Chocoboflor']

test.describe('Escouade — graphes rendus pour des coéquipiers existants', () => {
  test('POST /pages/teammates retourne au moins une row pour des amis avec matchs communs', async ({
    request,
  }) => {
    const resp = await request.post(
      `http://localhost:8000/api/v1/players/${PLAYER_SLUG}/pages/teammates`,
      {
        headers: {
          'Content-Type': 'application/json',
          Origin: 'http://localhost:5173',
        },
        data: { selected_gamertags: TEAMMATES },
      },
    )
    expect(resp.status()).toBe(200)
    const body = await resp.json()
    expect(body.teammates).toBeDefined()
    // Les deux teammates ont des matchs communs avec JGtm — au moins une
    // row doit revenir, sinon Q30 ou la résolution XUID est cassée.
    expect(body.teammates.length).toBeGreaterThanOrEqual(1)
    for (const row of body.teammates) {
      expect(TEAMMATES.map((g) => g.toLowerCase())).toContain(
        row.gamertag.toLowerCase(),
      )
      expect(row.with_kpis.match_count).toBeGreaterThan(0)
    }
  })

  test('la page Escouade affiche le KPI block + un PlotlyChart pour les coéquipiers sélectionnés', async ({
    page,
  }) => {
    // Pré-set localStorage avec la sélection — évite la friction de
    // saisie combobox (déjà testée par d'autres tests).
    await page.addInitScript(
      ({ slug, gamertags }) => {
        localStorage.setItem(
          `squad-teammates-${slug}`,
          JSON.stringify(gamertags),
        )
      },
      { slug: PLAYER_SLUG, gamertags: TEAMMATES },
    )

    // Capture les erreurs JS pour faire échouer le test en cas de stack
    // trace pendant le rendu (TypeError dans un chart builder, etc.).
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(`/players/${PLAYER_SLUG}/squad/synergies`)
    await page.waitForLoadState('networkidle')

    // 1. Le KPI block "Synergies avec les coéquipiers sélectionnés"
    //    apparaît seulement si selectedRows.length > 0.
    await expect(
      page.getByText('Synergies avec les coéquipiers sélectionnés'),
    ).toBeVisible({ timeout: 10_000 })

    // 2. Au moins un teammate sélectionné apparaît avec son label "Avec X"
    //    dans le KPI block (le même libellé apparaît aussi dans la legend
    //    Plotly — qu'on adresse séparément en assertion #3).
    await expect(
      page.getByRole('heading', { name: `Avec ${TEAMMATES[0]}` }),
    ).toBeVisible()

    // 3. Le graphe Plotly est rendu — on cible le conteneur interne
    //    `.js-plotly-plot` que react-plotly.js monte une fois le chart
    //    construit. Si selectedRows = 0 ou chart = null, il n'existe pas.
    await expect(page.locator('.js-plotly-plot').first()).toBeVisible({
      timeout: 10_000,
    })

    // 4. L'empty state "Aucune donnée commune" (invalid_selection) NE
    //    doit PAS être visible — c'est le symptôme du bug Q30 perfect_kills
    //    qui faisait silencieusement passer dans cette branche.
    await expect(page.getByText('Aucune donnée commune')).not.toBeVisible()

    // 5. Pas d'erreur JS pendant le rendu.
    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("la table 'Tous les coéquipiers' est visible avec des lignes", async ({
    page,
  }) => {
    await page.addInitScript(
      ({ slug, gamertags }) => {
        localStorage.setItem(
          `squad-teammates-${slug}`,
          JSON.stringify(gamertags),
        )
      },
      { slug: PLAYER_SLUG, gamertags: TEAMMATES },
    )
    await page.goto(`/players/${PLAYER_SLUG}/squad/synergies`)
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Tous les coéquipiers')).toBeVisible()
    // Au moins une row de teammate dans la table — visible via le gamertag.
    await expect(
      page.locator('table').getByText(TEAMMATES[0], { exact: true }),
    ).toBeVisible()
  })
})
