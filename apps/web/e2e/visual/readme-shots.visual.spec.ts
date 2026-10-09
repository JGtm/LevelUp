/**
 * readme-shots.visual.spec.ts — CAPTURES CANDIDATES POUR LE README.
 *
 * OUTIL À LA DEMANDE, pas une régression visuelle : sans `README_SHOTS=1` ce fichier
 * ne prend AUCUNE capture. Il vit sous `e2e/visual/` pour réutiliser le projet
 * Playwright `visual` (thème figé, aléatoire neutralisé, attente de stabilité des
 * canvas ECharts) — pas pour tourner avec lui.
 *
 * POURQUOI PAS `app-pages.visual.spec.ts` : celui-là compare des CANVAS pour détecter
 * une dérive de rendu. Ici on veut l'inverse — une image de PAGE, cadrée large,
 * présentable.
 *
 * PRÉ-REQUIS D'AUTHENTIFICATION. Un contexte Playwright neuf n'a pas de cookie de
 * session : avec `LEVELUP_AUTH_MODE=xbox` (le défaut d'un poste de développement),
 * `__root.tsx` éjecte vers `/login` et les 16 captures montrent l'écran de connexion.
 * Lancer l'API avec `LEVELUP_AUTH_MODE=none` le temps de la passe.
 *
 * LANCEMENT
 *   README_SHOTS=1 E2E_BASE_URL=http://localhost:5173 E2E_VISUAL_PLAYER=JGtm \
 *     E2E_VISUAL_SQUAD_TEAMMATES=Chocoboflor,Madina97294 \
 *     E2E_README_MATCH_ID=<uuid> \
 *     npx playwright test --project=visual readme-shots
 *
 * Sortie : `tests/readme-shots/` (hors dépôt versionné).
 */
import { test } from '@playwright/test'
import path from 'node:path'

import { gotoSettled, prepareVisualPage, VISUAL_PLAYER } from '../_helpers/visual'
import { playerPath } from '../_helpers/routes'

/**
 * Préfixe de langue de l'URL (`{-$lang}`) : le README racine est en ANGLAIS, les
 * captures doivent l'être aussi. `playerPath` rend un chemin SANS segment de langue
 * (l'app retombe alors sur la langue des réglages, `fr` ici).
 */
const LANG = process.env.E2E_README_LANG ?? 'en'
const route = (suffix: string) => `/${LANG}${playerPath(VISUAL_PLAYER, suffix)}`

const OUT = path.resolve(process.cwd(), '../../tests/readme-shots')

/**
 * Cadre « README ». HAUTEUR VOLONTAIREMENT GRANDE : les pages scrollent dans
 * `<main overflow-y-auto>` et non dans le body, donc `fullPage` ne capture RIEN de
 * plus que le viewport (même raison que le projet `visual` de playwright.config.ts).
 */
const VIEWPORT = { width: 1600, height: 2200 }

/** Match porteur d'un artefact de rejeu au schéma courant (sinon vue de match seule). */
const MATCH_ID = process.env.E2E_README_MATCH_ID ?? ''

const SHOTS: { id: string; suffix: string }[] = [
  { id: '01-home', suffix: 'home' },
  { id: '02-synthesis', suffix: 'stats/synthesis' },
  { id: '03-timeseries', suffix: 'stats/timeseries?tab=progression' },
  { id: '04-sessions', suffix: 'stats/sessions' },
  { id: '05-squad-synergies', suffix: 'squad/synergies' },
  { id: '06-squad-dynamique', suffix: 'squad/dynamique' },
  { id: '07-explorer', suffix: 'explorer' },
  { id: '08-career', suffix: 'career' },
  { id: '09-medals', suffix: 'career/medals' },
  { id: '10-citations', suffix: 'career/citations' },
  { id: '11-season-pass', suffix: 'career/season-pass' },
  { id: '12-ascension-profil', suffix: 'ascension' },
  { id: '13-ascension-objectifs', suffix: 'ascension/objectifs' },
  { id: '14-tactique', suffix: 'ascension/tactique' },
  { id: '15-community-relations', suffix: 'community/relations' },
  { id: '16-community-leaderboard', suffix: 'community' },
  { id: '17-media', suffix: 'media' },
  { id: '20-prestige', suffix: 'community/prestige' },
  { id: '21-coaching', suffix: 'ascension/coaching' },
  { id: '22-realisations', suffix: 'ascension/realisations' },
  { id: '23-squad-contributions', suffix: 'squad/contributions' },
]

test.describe('Captures candidates README', () => {
  test.skip(process.env.README_SHOTS !== '1', 'README_SHOTS=1 requis')
  test.use({ viewport: VIEWPORT })
  // Les pages Escouade portent des dizaines de graphes : la stabilisation des
  // canvas dépasse le défaut de 30 s (deux échecs mesurés le 2026-09-14).
  test.setTimeout(180_000)

  test.beforeEach(async ({ page }) => {
    await prepareVisualPage(page)
  })

  for (const { id, suffix } of SHOTS) {
    test(`${id}`, async ({ page }) => {
      const errors: string[] = []
      page.on('pageerror', (err) => errors.push(err.message))
      await gotoSettled(page, route(suffix))
      // Une capture est prise même si la page est pauvre : c'est un CANDIDAT, pas
      // une assertion. Les erreurs de page sont journalisées, pas levées.
      if (errors.length > 0) console.log(`[${id}] pageerror: ${errors.join(' | ')}`)
      await page.screenshot({ path: path.join(OUT, `${id}.png`) })
    })
  }

  test('18-match-view', async ({ page }) => {
    test.skip(MATCH_ID === '', 'E2E_README_MATCH_ID requis')
    await gotoSettled(page, route(`matches/${MATCH_ID}`))
    await page.waitForTimeout(2500)
    await page.screenshot({ path: path.join(OUT, '18-match-view.png') })
  })

  test('19-replay', async ({ page }) => {
    test.skip(MATCH_ID === '', 'E2E_README_MATCH_ID requis')
    // La page de rejeu tient dans ~1100 px de haut : le cadre commun (2200) la
    // noierait dans du vide.
    await page.setViewportSize({ width: 1600, height: 1150 })
    await page.goto(route(`matches/${MATCH_ID}/replay`), {
      waitUntil: 'domcontentloaded',
    })
    // Le rejeu ouvre EN PAUSE au coup d'envoi (auto-play désactivé par défaut) :
    // une capture immédiate montrerait un terrain vide. On lance la lecture au
    // raccourci documenté (Espace) et on laisse le match se dérouler un peu.
    await page.waitForTimeout(8000)
    // Le coup d'envoi montre un peloton groupé au spawn : sans intérêt. On saute
    // en avant (bouton « +10 s ») jusqu'au coeur du match, puis on laisse jouer
    // quelques secondes pour que trainées, fil des éliminations et objectifs aient
    // de quoi se peupler.
    const forward = page.getByRole('button', { name: /Forward 10|Avancer de 10/ })
    if (await forward.count()) {
      for (let i = 0; i < 30; i++) await forward.first().click({ timeout: 2000 }).catch(() => {})
    }
    await page.keyboard.press('Space')
    await page.waitForTimeout(15000)
    await page.screenshot({ path: path.join(OUT, '19-replay.png') })
  })
})
