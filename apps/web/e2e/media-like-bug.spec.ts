/**
 * E2E — Reproduction du bug "le clic like change la vidéo dans le cover-flow".
 *
 * Scénario :
 *   1. Aller sur la page médias
 *   2. Ouvrir un clip (le cover-flow s'ouvre sur cet item)
 *   3. Cliquer sur le bouton like
 *   4. Vérifier que la vidéo affichée n'a PAS changé
 *
 * Capture screenshot + DOM avant/après pour debug visuel.
 */
import { test, expect } from '@playwright/test'
import { skipIfNoDemoData } from './_helpers/demoData'

// Fixtures démo absentes en CI (data/demo gitignoré) → spec entière data-dépendante.
test.beforeEach(async () => {
  await skipIfNoDemoData()
})

test.describe('Bug : like change la vidéo', () => {
  test('clic like → la même vidéo reste affichée', async ({ page }) => {
    // Capture les console.log du browser
    const consoleLogs: string[] = []
    page.on('console', (msg) => {
      const text = msg.text()
      if (text.includes('BUG-DEBUG') || msg.type() === 'error' || msg.type() === 'warning') {
        consoleLogs.push(`[${msg.type()}] ${text}`)
      }
    })
    page.on('pageerror', (err) => {
      consoleLogs.push(`[pageerror] ${err.message}`)
    })

    // Capture toutes les requêtes réseau pour debug
    const networkLog: string[] = []
    page.on('request', (req) => {
      if (req.url().includes('/api/v1') || req.url().includes('/media/')) {
        networkLog.push(`→ ${req.method()} ${req.url()}`)
      }
    })
    page.on('response', (res) => {
      if (res.url().includes('/api/v1') || res.url().includes('/media/')) {
        networkLog.push(`← ${res.status()} ${res.url()}`)
      }
    })

    // 1. Aller sur la page médias d'un joueur qui a des clips.
    // On goto d'abord pour créer une session, puis on POST /session/context
    // pour set current_player_slug=JGtm (sinon le backend ne peut pas auto-injecter
    // le liker dans la mutation /likes).
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // Set le player courant dans la session (= JGtm pour avoir des médias)
    await page.request.post('/api/v1/session/context', {
      data: { player_slug: 'JGtm' },
    })

    const candidates = ['JGtm', 'Madina97294', 'XxDaemonGamerxX', 'Chocoboflor']
    let found = false
    for (const slug of candidates) {
      // Re-set current_player_slug avant chaque tentative pour que section_filter=mine match le slug visité
      await page.request.post('/api/v1/session/context', { data: { player_slug: slug } })
      await page.goto(`/players/${slug}/media`)
      try {
        await page.waitForLoadState('networkidle', { timeout: 15000 })
      } catch { /* peut timeout sur poll feed-version */ }
      await page.waitForTimeout(2000)
      const c = await page.locator('article[role="button"]').count()
      console.log(`[debug] ${slug}: ${c} cards`)
      if (c > 0) {
        found = true
        break
      }
    }
    if (!found) {
      console.log('[debug] Aucun joueur de la nav n\'a de médias visibles')
      test.skip()
      return
    }

    // Screenshot après chargement
    await page.screenshot({ path: 'test-results/01-media-page-loaded.png', fullPage: true })

    // Debug : afficher le DOM en mode survol
    const articleCount = await page.locator('article').count()
    const videoCount = await page.locator('video').count()
    const imgCount = await page.locator('img').count()
    console.log(`[debug] articles=${articleCount}, videos=${videoCount}, imgs=${imgCount}`)

    // 2. Trouver toutes les cards de média (essayer plusieurs sélecteurs)
    const cards = page.locator('article[role="button"]')
    let cardCount = await cards.count()
    console.log(`[debug] article[role="button"]=${cardCount}`)

    if (cardCount === 0) {
      console.log('[debug] Pas de cards, content principal:')
      const main = await page.locator('body').textContent()
      console.log(main?.slice(0, 500))
      console.log('[network log]\n' + networkLog.join('\n'))
      test.skip()
      return
    }

    // Cliquer sur la 1ère card pour ouvrir le cover-flow
    await cards.nth(0).click()

    // Attendre que le cover-flow soit visible (overlay z-50 fixed inset-0)
    const coverFlow = page.locator('div.fixed.inset-0.z-50')
    await expect(coverFlow).toBeVisible({ timeout: 5000 })

    // Screenshot du cover-flow ouvert
    await page.screenshot({ path: 'test-results/02-coverflow-opened.png', fullPage: false })

    // 3. Capturer le heading actuel (compteur X/Y + map + date)
    const headingBefore = await coverFlow.locator('span.truncate.text-sm').first().textContent()
    console.log(`[debug] Heading AVANT like : "${headingBefore}"`)

    // 4. Capturer le file_path de la vidéo affichée (slot center, opacity 1)
    const videoSrcBefore = await page.evaluate(() => {
      const videos = document.querySelectorAll('video')
      // Chercher la vidéo "centrale" (parent avec opacity ~1)
      for (const v of Array.from(videos)) {
        const parent = v.closest('[style*="opacity"]') as HTMLElement | null
        if (parent && parent.style.opacity === '1') {
          return v.getAttribute('src')
        }
      }
      return null
    })
    console.log(`[debug] Video src AVANT like : ${videoSrcBefore}`)

    // Capture les items dans le query cache AVANT le clic
    const itemsBefore = await page.evaluate(() => {
      // @ts-ignore — debug TanStack Query cache
      const qc = (window as any).__queryClient
      if (!qc) return null
      const queries = qc.getQueriesData({ queryKey: ['media'] })
      const result: { key: string; itemsCount: number; firstThree: string[] }[] = []
      for (const [key, data] of queries) {
        if (data && (data as any).items?.items) {
          result.push({
            key: JSON.stringify(key),
            itemsCount: (data as any).items.items.length,
            firstThree: (data as any).items.items.slice(0, 3).map((i: any) => `${i.file_path}|liked=${i.liked}`),
          })
        }
      }
      return result
    })
    console.log(`[debug] Cache AVANT like:`, JSON.stringify(itemsBefore, null, 2))

    networkLog.length = 0 // Reset le log juste avant le clic like

    // Debug : combien de boutons "Liker" dans le DOM ?
    const allLikeButtons = await page.locator('button[aria-label="Liker"], button[aria-label="Retirer le like"]').count()
    console.log(`[debug] Total boutons like dans le DOM : ${allLikeButtons}`)

    // 5. Cliquer sur le bouton like SPÉCIFIQUEMENT dans le cover-flow header
    // Le cover-flow header a la classe bg-black/60 et contient le bouton non-compact
    const likeButton = coverFlow.locator('button[aria-label="Liker"], button[aria-label="Retirer le like"]').first()
    await expect(likeButton).toBeVisible()
    const likeAriaBefore = await likeButton.getAttribute('aria-label')
    const likeBox = await likeButton.boundingBox()
    console.log(`[debug] Like button avant clic : aria=${likeAriaBefore}, position=${JSON.stringify(likeBox)}`)

    await likeButton.click({ force: true })

    // 6. Attendre un peu pour laisser le pipeline mutation + invalidate + refetch
    await page.waitForTimeout(5000)

    // Screenshot après le clic
    await page.screenshot({ path: 'test-results/03-coverflow-after-like.png', fullPage: false })

    // 7. Re-capturer le heading
    const headingAfter = await coverFlow.locator('span.truncate.text-sm').first().textContent()
    console.log(`[debug] Heading APRÈS like : "${headingAfter}"`)

    const videoSrcAfter = await page.evaluate(() => {
      const videos = document.querySelectorAll('video')
      for (const v of Array.from(videos)) {
        const parent = v.closest('[style*="opacity"]') as HTMLElement | null
        if (parent && parent.style.opacity === '1') {
          return v.getAttribute('src')
        }
      }
      return null
    })
    console.log(`[debug] Video src APRÈS like : ${videoSrcAfter}`)

    console.log('[CONSOLE BROWSER]\n' + consoleLogs.join('\n'))
    console.log('[network log post-clic]\n' + networkLog.join('\n'))

    // Capture les items APRÈS le clic
    const itemsAfter = await page.evaluate(() => {
      const qc = (window as any).__queryClient
      if (!qc) return null
      const queries = qc.getQueriesData({ queryKey: ['media'] })
      const result: { key: string; itemsCount: number; firstThree: string[] }[] = []
      for (const [key, data] of queries) {
        if (data && (data as any).items?.items) {
          result.push({
            key: JSON.stringify(key),
            itemsCount: (data as any).items.items.length,
            firstThree: (data as any).items.items.slice(0, 3).map((i: any) => `${i.file_path}|liked=${i.liked}`),
          })
        }
      }
      return result
    })
    console.log(`[debug] Cache APRÈS like:`, JSON.stringify(itemsAfter, null, 2))

    // 8. ASSERTION CRITIQUE : la vidéo affichée doit être la MÊME
    expect(videoSrcAfter, 'La vidéo a changé après le clic like ! Le bug est reproduit.').toBe(videoSrcBefore)

    // Le map_name dans le heading doit aussi être le même (extrait après "/N · ")
    const mapBefore = headingBefore?.split(' · ')[1] ?? ''
    const mapAfter = headingAfter?.split(' · ')[1] ?? ''
    expect(mapAfter, 'Le map_name dans le heading a changé').toBe(mapBefore)

    // Et le like doit être enregistré (aria-label inversé)
    const likeAriaAfter = await likeButton.getAttribute('aria-label')
    console.log(`[debug] Like button après clic : ${likeAriaAfter}`)
    expect(likeAriaAfter, 'Le bouton like n\'a pas changé d\'état').not.toBe(likeAriaBefore)

    // Vérifier que le like a été partagé (media_likes shared rempli) :
    // re-fetch les médias et chercher le badge total_likers > 0 sur le current item.
    const sharedLikersInfo = await page.evaluate(async (filePath) => {
      const r = await fetch('/api/v1/players/JGtm/pages/media', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sort: 'date_desc', pagination: { page: 1, page_size: 100 } }),
      })
      const data = await r.json()
      const items = data?.items?.items ?? []
      const found = items.find((i: { file_path: string }) => i.file_path === filePath)
      return found ? { liked: found.liked, total_likers: found.total_likers, likers: found.likers } : null
    }, videoSrcBefore)
    console.log('[debug] Shared likers info après PATCH:', JSON.stringify(sharedLikersInfo))
    expect(sharedLikersInfo, 'L\'item liké doit être trouvé dans la galerie').toBeTruthy()
    expect(sharedLikersInfo!.total_likers, 'Le like doit être enregistré dans media_likes shared').toBeGreaterThan(0)
  })
})
