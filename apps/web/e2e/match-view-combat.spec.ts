/**
 * E2E — Validation visuelle onglet Combat (badges + 4 charts du mock).
 *
 * Vérifie que les blocs ajoutés par fix/media-player-ux pour aligner sur
 * .ai/charts_specs/_generated/match_view/mock-echarts.html sont rendus :
 *   - Bandeau "Faits marquants" (MatchImpactBadgesBar)
 *   - match_view.09 — Frags cumulés par équipe
 *   - match_view.10 — Dominance par tranche de temps
 *   - match_view.11 — Cadence des frags
 *   - match_view.12 — Némésis et Souffre-douleur (cartes sombres)
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173) actif avec au moins un
 * joueur synchronisé.
 */
import { test, expect } from '@playwright/test'
import { skipObsoleteSpec } from './_helpers/demoData'

test.describe('Match view — onglet Combat (refonte 2026-05-06)', () => {
  // Le match view a été redessiné : plus d'onglet "Combat" (onglets actuels :
  // "Général" / "Détails"). Les charts kd_timeline/tug_of_war existent toujours
  // mais sous une autre structure — spec à réécrire pour les onglets courants.
  test.beforeEach(() => {
    skipObsoleteSpec("onglet 'Combat' du match view remplacé par 'Général'/'Détails'")
  })
  test('rend la barre de badges et les 4 charts en tête de l\'onglet', async ({
    page,
    request,
  }) => {
    await page.setViewportSize({ width: 1440, height: 1100 })
    // 1. Récupérer un joueur disponible
    const playersResp = await request.get('http://localhost:8000/api/v1/players')
    expect(playersResp.status()).toBe(200)
    const players = await playersResp.json()
    test.skip(
      !players?.items?.length,
      'Aucun joueur configuré — impossible de tester la match view',
    )
    const playerSlug = (players.items[0].player_slug ?? players.default_player_slug) as string

    // 2. Récupérer le match_id d'un match valide via l'explorer.
    //    Réponse : { summary, table: { items: [{match_id, ...}] } }.
    //    On parcourt l'historique pour prendre le 1er match dont le combat_tab
    //    expose les charts qu'on vient d'ajouter (kd_timeline, tug_of_war,
    //    cadence) — sinon le test devient fragile selon le 1er match.
    const histResp = await request.post(
      `http://localhost:8000/api/v1/players/${playerSlug}/pages/explorer/matches-query`,
      { data: { filters: {} } },
    )
    test.skip(
      histResp.status() !== 200,
      `Historique inaccessible (HTTP ${histResp.status()})`,
    )
    const histData = await histResp.json()
    const histItems = histData?.table?.items ?? []
    test.skip(histItems.length === 0, "Aucun match dans l'historique du joueur")

    let matchId: string | null = null
    for (const candidate of histItems.slice(0, 10)) {
      const mvResp = await request.get(
        `http://localhost:8000/api/v1/players/${playerSlug}/matches/${candidate.match_id}`,
      )
      if (mvResp.status() !== 200) continue
      const mv = await mvResp.json()
      const ct = mv?.combat_tab ?? {}
      if (
        Array.isArray(ct.kd_timeline) &&
        ct.kd_timeline.length > 0 &&
        Array.isArray(ct.tug_of_war) &&
        ct.tug_of_war.length > 0
      ) {
        matchId = candidate.match_id
        break
      }
    }
    test.skip(matchId == null, 'Aucun match trouvé avec kd_timeline + tug_of_war peuplés')

    // 3. Naviguer vers la page détail du match
    await page.goto(`/players/${playerSlug}/matches/${matchId}`)

    // 4. Cliquer sur l'onglet Combat
    const combatTab = page.getByRole('button', { name: 'Combat' })
    await expect(combatTab).toBeVisible({ timeout: 15_000 })
    await combatTab.click()

    // 5. Vérifier la présence des blocs nouvellement câblés.
    //    Utilise des selecteurs textuels stables (libellés FR i18n).
    await expect(page.getByText('Faits marquants', { exact: false })).toBeVisible({
      timeout: 10_000,
    })
    await expect(
      page.getByText('Frags cumulés par équipe', { exact: false }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(
      page.getByText('Dominance par tranche de temps', { exact: false }),
    ).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Cadence des frags', { exact: false })).toBeVisible({
      timeout: 10_000,
    })

    // Némésis / Souffre-douleur peuvent être absents si aucun duel direct,
    // mais doivent au moins ne pas casser la page. On regarde non-strict.
    const nemesisVisible = await page
      .getByText('Némésis', { exact: false })
      .first()
      .isVisible()
      .catch(() => false)
    const bullyVisible = await page
      .getByText('Souffre-douleur', { exact: false })
      .first()
      .isVisible()
      .catch(() => false)
    console.log(
      `[combat] nemesis_visible=${nemesisVisible} bully_visible=${bullyVisible}`,
    )

    // 6. Pas d'erreur React fatale ni de "Aucune donnée" partout
    const errorTexts = await page.getByText(/erreur|Match introuvable/i).all()
    expect(errorTexts.length).toBe(0)

    // 7. Captures d'écran en attestation : page entière + zoom sur la zone
    //    haute (badges + KD cumul) + zoom sur Tug-of-war/Cadence + zoom Némésis.
    //    Le scroll force le rendu lazy des charts ECharts avant capture.
    const cadence = page.getByText('Cadence des frags', { exact: false }).first()
    await cadence.scrollIntoViewIfNeeded()
    await page.waitForTimeout(500)
    const nemesisHeading = page.getByText('Némésis', { exact: false }).first()
    await nemesisHeading.scrollIntoViewIfNeeded()
    await page.waitForTimeout(500)
    await page.evaluate(() => window.scrollTo(0, 0))
    await page.waitForTimeout(500)
    await page.screenshot({
      path: '../../tests/e2e-results/match-view-combat-tab.png',
      fullPage: true,
    })
  })
})
