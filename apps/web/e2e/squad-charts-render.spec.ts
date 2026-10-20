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
 * Prérequis : `make dev` avec la fixture démo synthétique (joueur demo-player +
 * coéquipiers DemoPlayer2/DemoPlayer3, 36 matchs communs) ou, via E2E_PLAYER_SLUG et
 * E2E_TEAMMATES, un jeu de données réel.
 */
import { test, expect } from '@playwright/test'
import {
  skipIfNoDemoData,
  E2E_API_BASE,
  E2E_PLAYER_SLUG,
  E2E_TEAMMATES,
} from './_helpers/demoData'
import { playerPath } from './_helpers/routes'

test.beforeEach(async () => {
  await skipIfNoDemoData()
})

const PLAYER_SLUG = E2E_PLAYER_SLUG
const TEAMMATES = E2E_TEAMMATES

/** Pré-set la composition dans localStorage : évite la saisie dans le combobox. */
async function presetTeammates(page: import('@playwright/test').Page) {
  await page.addInitScript(
    ({ slug, gamertags }) => {
      localStorage.setItem(`squad-teammates-${slug}`, JSON.stringify(gamertags))
    },
    { slug: PLAYER_SLUG, gamertags: TEAMMATES },
  )
}

test.describe('Escouade — graphes rendus pour des coéquipiers existants', () => {
  test('POST /pages/teammates retourne au moins une row pour des amis avec matchs communs', async ({
    request,
  }) => {
    const resp = await request.post(
      `${E2E_API_BASE}/api/v1/players/${PLAYER_SLUG}/pages/teammates`,
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
    // Les coéquipiers ont des matchs communs avec le joueur — au moins une
    // row doit revenir, sinon Q30 ou la résolution XUID est cassée.
    expect(body.teammates.length).toBeGreaterThanOrEqual(1)
    for (const row of body.teammates) {
      expect(TEAMMATES.map((g) => g.toLowerCase())).toContain(
        row.gamertag.toLowerCase(),
      )
      expect(row.with_kpis.match_count).toBeGreaterThan(0)
    }
  })

  test('la page Synergies affiche le briefing de session + des graphes pour les coéquipiers sélectionnés', async ({
    page,
  }) => {
    await presetTeammates(page)

    // Capture les erreurs JS pour faire échouer le test en cas de stack
    // trace pendant le rendu (TypeError dans un chart builder, etc.).
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto(playerPath(PLAYER_SLUG, 'squad/synergies'))
    await page.waitForLoadState('networkidle')

    // 1. Le briefing de session nomme chaque coéquipier sélectionné.
    for (const teammate of TEAMMATES) {
      await expect(
        page.getByText(teammate, { exact: true }).filter({ visible: true }).first(),
      ).toBeVisible({
        timeout: 10_000,
      })
    }

    // 2. Au moins un graphe ECharts (canvas) est rendu. Si la sélection est
    //    vide ou si le chart builder échoue, aucun canvas n'est monté.
    await expect(page.locator('canvas').first()).toBeVisible({ timeout: 10_000 })

    // 3. L'empty state "Aucune donnée commune" (invalid_selection) NE
    //    doit PAS être visible — c'est le symptôme du bug Q30 perfect_kills
    //    qui faisait silencieusement passer dans cette branche.
    await expect(page.getByText('Aucune donnée commune')).not.toBeVisible()

    // 4. Pas d'erreur JS pendant le rendu.
    expect(
      errors.filter((e) => !e.includes('ResizeObserver')),
    ).toHaveLength(0)
  })

  test("le tableau « Impact des coéquipiers » est visible avec une ligne par coéquipier", async ({
    page,
  }) => {
    await presetTeammates(page)
    await page.goto(playerPath(PLAYER_SLUG, 'squad/contributions'))
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Impact des coéquipiers')).toBeVisible({
      timeout: 10_000,
    })
    // Une row par coéquipier de la composition — visible via le gamertag.
    for (const teammate of TEAMMATES) {
      await expect(
        page.locator('table').getByText(teammate, { exact: true }),
      ).toBeVisible()
    }
  })
})
