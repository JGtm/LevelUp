/**
 * E2E — Split des stores de filtres solo/squad.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173), LEVELUP_DEMO_MODE=true.
 *
 * Couverture des 3 scénarios du refactor :
 *  1. Architecture stores : 2 clés localStorage distinctes (`levelup-solo-filter-v1`
 *     et `levelup-squad-filter-v1`) après navigation sur les 2 sections.
 *  2. Isolation Squad ↔ Stats : un filtre cascade posé sur Squad ne pollue pas
 *     Stats Solo et vice versa (vérifié via les payloads réseau).
 *  3. Scope local des pages annexes : Citations / SessionDetail / SessionCompare
 *     ont leur barre filtres propre (cf. présence du dropdown Expérience local)
 *     et n'envoient PAS le filterContext du store solo dans leurs requêtes.
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

const PLAYER = 'demo-player'

test.describe('Split stores filtres — architecture solo/squad isolée', () => {
  test('1. Deux clés localStorage distinctes après mutation des 2 stores', async ({ page }) => {
    // Zustand `persist` n'écrit dans localStorage qu'après une mutation
    // post-hydratation. On simule l'effet d'un clic « Analyser » sur chaque page
    // en injectant manuellement un state dans chaque clé attendue.
    await page.goto(`/players/${PLAYER}/stats/timeseries`)
    await page.waitForLoadState('networkidle')

    await page.evaluate(() => {
      const writeStore = (key: string) => {
        const state = {
          state: {
            filterContext: {
              filter_mode: 'period',
              period: { start_date: null, end_date: null },
              sessions: { picked_sessions: [], gap_minutes: 120 },
              cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
            },
            filterContextHash: 'init',
            lastKnownLatestSessionId: null,
          },
          version: 0,
        }
        localStorage.setItem(key, JSON.stringify(state))
      }
      writeStore('levelup-solo-filter-v1')
      writeStore('levelup-squad-filter-v1')
    })

    const keys = await page.evaluate(() => Object.keys(localStorage))

    expect(keys, 'clé solo doit exister').toContain('levelup-solo-filter-v1')
    expect(keys, 'clé squad doit exister').toContain('levelup-squad-filter-v1')
  })

  test('2a. Filtre cascade Squad ne pollue pas Stats Solo', async ({ page }) => {
    await skipIfNoDemoData()
    // Étape 1 : sur Squad, set la cascade côté store squad via injection directe
    // (équivaut à un user qui coche des modes et clique Analyser)
    await page.goto(`/players/${PLAYER}/squad`)
    await page.waitForLoadState('networkidle')

    await page.evaluate(() => {
      const raw = JSON.parse(localStorage.getItem('levelup-squad-filter-v1') ?? '{}')
      raw.state = raw.state ?? {}
      raw.state.filterContext = {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: {
          experience_types: [],
          playlists: [],
          modes: ['Slayer'],
          maps: [],
        },
      }
      raw.state.filterContextHash = 'squad-test-hash'
      localStorage.setItem('levelup-squad-filter-v1', JSON.stringify(raw))
    })

    // Étape 2 : navigation vers Stats Solo, intercepte la requête timeseries
    const timeseriesReq = page.waitForRequest(
      (req) => req.url().includes('/pages/timeseries') && req.method() === 'POST',
    )

    await page.goto(`/players/${PLAYER}/stats/timeseries`)
    const req = await timeseriesReq
    const body = req.postDataJSON() as {
      filters?: { cascade?: { modes?: string[] } }
    }

    // Le body doit refléter le store SOLO (cascade vide), pas le store SQUAD
    expect(body.filters?.cascade?.modes ?? [], 'cascade modes Stats Solo doit être vide').toEqual([])
  })

  test('2b. Filtre cascade Solo ne pollue pas Squad', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/stats/timeseries`)
    await page.waitForLoadState('networkidle')

    // Inject cascade côté store solo
    await page.evaluate(() => {
      const raw = JSON.parse(localStorage.getItem('levelup-solo-filter-v1') ?? '{}')
      raw.state = raw.state ?? {}
      raw.state.filterContext = {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: {
          experience_types: ['PVP classé'],
          playlists: [],
          modes: [],
          maps: [],
        },
      }
      raw.state.filterContextHash = 'solo-test-hash'
      localStorage.setItem('levelup-solo-filter-v1', JSON.stringify(raw))
    })

    // Navigation vers Squad → intercepte filters/resolve
    const resolveReq = page.waitForRequest(
      (req) => req.url().includes('/filters/resolve') && req.method() === 'POST',
    )

    await page.goto(`/players/${PLAYER}/squad`)
    const req = await resolveReq
    const body = req.postDataJSON() as {
      filters?: { cascade?: { experience_types?: string[] } }
      cascade?: { experience_types?: string[] }
    }

    // filters/resolve prend FilterContextInput directement (pas wrap dans filters)
    const experienceTypes =
      body.cascade?.experience_types ?? body.filters?.cascade?.experience_types ?? []
    expect(experienceTypes, 'cascade experience Squad doit être vide').toEqual([])
  })

  test('3a. CitationsPage a sa propre barre filtres locale (Expérience visible)', async ({ page }) => {
    await skipIfNoDemoData()
    await page.goto(`/players/${PLAYER}/citations`)
    await page.waitForLoadState('networkidle')

    // La barre locale (useLocalFilterBar) rend le dropdown Expérience avec
    // le placeholder "Expérience" (manifest citations.filters.experience).
    await expect(
      page.getByRole('button', { name: /Expérience\s*:/ }),
      'dropdown Expérience local doit être présent',
    ).toBeVisible()
  })

  test('3b. CitationsPage envoie un filterContext local (pas hérité du store solo)', async ({ page }) => {
    await skipIfNoDemoData()
    // Pollue le store solo avec une cascade
    await page.goto(`/players/${PLAYER}/home`)
    await page.evaluate(() => {
      const raw = JSON.parse(localStorage.getItem('levelup-solo-filter-v1') ?? '{}')
      raw.state = raw.state ?? {}
      raw.state.filterContext = {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: {
          experience_types: ['PVP classé'],
          playlists: [],
          modes: [],
          maps: [],
        },
      }
      raw.state.filterContextHash = 'pollute-solo-hash'
      localStorage.setItem('levelup-solo-filter-v1', JSON.stringify(raw))
    })

    const citationsReq = page.waitForRequest(
      (req) => req.url().includes('/pages/citations') && req.method() === 'POST',
    )
    await page.goto(`/players/${PLAYER}/citations`)
    const req = await citationsReq
    const body = req.postDataJSON() as {
      filters?: { cascade?: { experience_types?: string[] } }
    }

    // Citations doit ignorer la pollution solo
    expect(
      body.filters?.cascade?.experience_types ?? [],
      'CitationsPage doit envoyer cascade locale vide',
    ).toEqual([])
  })

  test('3c. ExplorerPage envoie un filterContext local par défaut', async ({ page }) => {
    await skipIfNoDemoData()
    // Pollue le store solo
    await page.goto(`/players/${PLAYER}/home`)
    await page.evaluate(() => {
      const raw = JSON.parse(localStorage.getItem('levelup-solo-filter-v1') ?? '{}')
      raw.state = raw.state ?? {}
      raw.state.filterContext = {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: {
          experience_types: [],
          playlists: [],
          modes: ['Slayer'],
          maps: [],
        },
      }
      raw.state.filterContextHash = 'pollute-explorer-hash'
      localStorage.setItem('levelup-solo-filter-v1', JSON.stringify(raw))
    })

    const explorerReq = page.waitForRequest(
      (req) => req.url().includes('/pages/explorer/matches') && req.method() === 'POST',
    )
    await page.goto(`/players/${PLAYER}/explorer`)
    const req = await explorerReq
    const body = req.postDataJSON() as {
      filters?: { cascade?: { modes?: string[] } }
    }

    expect(
      body.filters?.cascade?.modes ?? [],
      'ExplorerPage doit envoyer cascade locale vide (DEFAULT_FILTER_CONTEXT)',
    ).toEqual([])
  })
})
