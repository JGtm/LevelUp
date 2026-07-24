/**
 * E2E — Redirections legacy `/players/…` → `/t/{titleSlug}/players/…`.
 *
 * Exerce EN RÉEL, dans le navigateur, la matrice principale du splat de redirection
 * (routes/players/$.tsx, décision D-5 du PLAN_TITLE_SLUG_URL_2026-07) contre le dev
 * server : préfixe titre ajouté, préservation intégrale du suffixe + `?search`
 * (`?f=`) + `#hash`, remaps internes (objectifs→ascension, palmares→community), et
 * honneur de la session H5 déjà committée (trou n°1 de D-8 : un bookmark H5 ne
 * retombe PAS sur le titre par défaut). La logique PURE est déjà couverte en
 * table-driven (src/lib/title-routing/buildLegacyRedirect.test.ts) — CETTE spec
 * confirme la PROJECTION dans un vrai navigateur (course de redirection incluse).
 *
 * JOUEUR. Résolu à l'exécution via `GET /api/v1/players` (même pattern que
 * match-view-combat.spec.ts) — robuste au jeu de données du serveur (démo synthétique
 * en CI, joueurs réels en local). Un slug VALIDE est requis : un slug inconnu serait
 * réécrit par le filet `resolvePlayerFallback` du layout joueur (→ joueur courant +
 * `/home`), ce qui masquerait la préservation du suffixe.
 *
 * PAS de skipIfNoDemoData : les redirections sont de l'INFRA de routage (assertions
 * sur l'URL d'atterrissage, pas sur le contenu des pages) — elles doivent s'exécuter
 * même sans fixture démo, tant qu'au moins un joueur existe.
 *
 * Prérequis : `make dev` (API :8000 + Vite :5173). Titres attendus au registre :
 * halo_infinite (défaut) + halo_5 (active) — cf. bootstrap.
 */
import { test, expect, type APIRequestContext } from '@playwright/test'
import {
  legacyPlayerPath,
  playerPath,
  playerUrlPattern,
  titlePath,
  titleSegmentPattern,
} from './_helpers/routes'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8000'

let cachedPlayer: string | undefined

/**
 * Slug joueur VALIDE sur le serveur courant, mémoïsé (workers:1). Pattern repris de
 * match-view-combat.spec.ts (`items[0].player_slug ?? default_player_slug`).
 */
async function resolvePlayer(request: APIRequestContext): Promise<string> {
  if (cachedPlayer) return cachedPlayer
  const resp = await request.get(`${API_BASE}/api/v1/players`)
  expect(resp.status(), 'GET /api/v1/players doit répondre 200').toBe(200)
  const body = (await resp.json()) as {
    items?: { player_slug: string }[]
    default_player_slug?: string
  }
  const slug = body.items?.[0]?.player_slug ?? body.default_player_slug
  expect(slug, 'au moins un joueur doit exister pour exercer la redirection').toBeTruthy()
  cachedPlayer = slug as string
  return cachedPlayer
}

/** Vrai si `slug` est un titre ACTIF du registre (lu via bootstrap). */
async function titleActive(request: APIRequestContext, slug: string): Promise<boolean> {
  const resp = await request.get(`${API_BASE}/api/v1/bootstrap`)
  if (!resp.ok()) return false
  const data = (await resp.json()) as { available_titles?: { slug: string; status?: string }[] }
  return (data.available_titles ?? []).some((t) => t.slug === slug && t.status === 'active')
}

test.describe('Redirections legacy /players/ → /t/{slug}/players/', () => {
  test('bookmark home : /players/{p}/home → /t/halo_infinite/players/{p}/home (session par défaut)', async ({
    page,
    request,
  }) => {
    const player = await resolvePlayer(request)
    await page.goto(legacyPlayerPath(player, 'home'))
    // Le splat réécrit (replace) vers la forme title-préfixée du titre de session (défaut).
    await page.waitForURL(playerUrlPattern(player, 'home'), { timeout: 15_000 })
    expect(page.url()).toContain(playerPath(player, 'home'))
  })

  test('deep-link : suffixe + ?f= + #hash préservés à la redirection', async ({
    page,
    request,
  }) => {
    const player = await resolvePlayer(request)
    const F_VALUE = 'e2e-legacy-deeplink'

    // FORME D'ASSERTION CHOISIE (documentée) : on asserte la PREMIÈRE URL post-redirect.
    // Le CONTRAT du splat (D-5) est de préserver suffixe + search + hash À L'IDENTIQUE ;
    // `waitForURL(predicate)` suit les changements History API (SPA) et résout à L'INSTANT
    // du `history.replace`, AVANT que la page timeseries ne ré-encode/purge un `?f=` opaque
    // (comportement de PAGE, hors périmètre de la redirection). C'est la forme la plus
    // robuste ET la plus fidèle au rôle du splat ; la préservation byte-identique est aussi
    // couverte par la matrice unitaire buildLegacyRedirect.test.ts. Asserter l'URL FINALE
    // serait fragile (la page peut légitimement réécrire un `?f=` invalide).
    await page.goto(legacyPlayerPath(player, `stats/timeseries?f=${F_VALUE}#h`))
    // I10 : le premier hop ÉMET désormais le segment de langue de session
    // (`/{lang}/t/…`) — on asserte donc que le pathname SE TERMINE par le chemin
    // title-scoped (suffixe byte-exact) plutôt qu'une égalité stricte (le préfixe
    // `/fr`|`/en` variable est légitime). Le contrat de préservation ?f=+#hash est
    // inchangé et reste asserté byte-exact.
    await page.waitForURL(
      (url) =>
        url.pathname.endsWith(playerPath(player, 'stats/timeseries')) &&
        url.searchParams.get('f') === F_VALUE &&
        url.hash === '#h',
      { timeout: 15_000 },
    )
  })

  test('remap interne objectifs : /players/{p}/objectifs → …/ascension/objectifs', async ({
    page,
    request,
  }) => {
    const player = await resolvePlayer(request)
    await page.goto(legacyPlayerPath(player, 'objectifs'))
    await page.waitForURL(playerUrlPattern(player, 'ascension/objectifs'), { timeout: 15_000 })
    expect(page.url()).toContain(playerPath(player, 'ascension/objectifs'))
  })

  test('remap interne palmares : /players/{p}/palmares → …/community', async ({
    page,
    request,
  }) => {
    const player = await resolvePlayer(request)
    await page.goto(legacyPlayerPath(player, 'palmares'))
    await page.waitForURL(playerUrlPattern(player, 'community'), { timeout: 15_000 })
    expect(page.url()).toContain(playerPath(player, 'community'))
  })

  test('session H5 committée : bookmark legacy honore halo_5, pas le titre par défaut', async ({
    page,
    request,
  }) => {
    // Deux navigations + convergence de session : budget de temps élargi.
    test.setTimeout(60_000)
    // halo_5 doit être un titre ACTIF du registre pour exercer la matrice H5. Absent
    // (ex. démo synthétique CI = halo_infinite seul) → skip explicite, pas d'échec.
    test.skip(
      !(await titleActive(request, 'halo_5')),
      'halo_5 absent/inactif du registre (ex. démo synthétique CI) — matrice H5 non exécutable',
    )
    const player = await resolvePlayer(request)

    // 1. Committer la session sur halo_5 en visitant une route H5 explicite : le layout
    //    titre détecte le segment ≠ titre du store et applique applyActiveTitle
    //    (POST /session/context → session serveur = halo_5) puis re-bootstrappe.
    await page.goto(titlePath('halo_5', player, 'home'))
    await page.waitForURL(titleSegmentPattern('halo_5'), { timeout: 20_000 })
    // Attendre la CONVERGENCE de la session serveur sur halo_5 (le fresh-load suivant la
    // lira). `page.request` partage le cookie de session mais N'envoie PAS le header
    // X-LevelUp-Title → /bootstrap résout le titre par la SESSION. poll = robuste au
    // timing d'applyActiveTitle et à un éventuel 429 transitoire.
    await expect
      .poll(
        async () => {
          const r = await page.request.get(`${API_BASE}/api/v1/bootstrap`)
          return r.ok() ? ((await r.json()) as { current_title_slug?: string }).current_title_slug : null
        },
        { timeout: 20_000, message: 'la visite de /t/halo_5/… doit committer la session serveur sur halo_5' },
      )
      .toBe('halo_5')

    // 2. Bookmark LEGACY sans segment de titre : au fresh-load, le titre actif vient de la
    //    SESSION (halo_5), pas du défaut. La redirection doit donc viser halo_5 (trou n°1).
    await page.goto(legacyPlayerPath(player, 'home'))
    await page.waitForURL(titleSegmentPattern('halo_5'), { timeout: 20_000 })
    const url = page.url()
    expect(url, 'la redirection legacy doit viser la session H5').toMatch(titleSegmentPattern('halo_5'))
    expect(url, 'elle ne doit PAS retomber sur le titre par défaut').not.toMatch(
      titleSegmentPattern('halo_infinite'),
    )
  })
})
