/**
 * Helper E2E — régression VISUELLE des graphes (harnais `*.visual.spec.ts`).
 *
 * POURQUOI CE HARNAIS EXISTE. Tous les graphes de l'app passent par ECharts, et
 * `vitest` mocke `echarts` partout : aucun test n'exerçait le rendu réel. C'est
 * précisément ce trou qui a fait fermer sans merger la PR Dependabot #49
 * (`echarts` 5.6.0 -> 6.1.0, correctif XSS CVE-2026-45249) : impossible de mesurer
 * l'impact d'un changement du moteur de rendu de TOUTE l'application. Ce helper
 * fournit le socle DÉTERMINISTE qui manquait.
 *
 * CE QUE CE HELPER GARANTIT (sans quoi une capture canvas n'est pas comparable) :
 *   1. Thème figé (dark) — le thème est lu depuis localStorage et pilote la palette
 *      ECharts via `data-theme` (cf. lib/echarts/themeColors.ts).
 *   2. Aléatoire neutralisé — `Math.random` est remplacé par une suite déterministe
 *      AVANT tout script de page : la vitrine `/lab/charts` s'en sert pour ses
 *      données de démo (heatmap, barres K/D), sinon chaque run produit un chart
 *      différent.
 *   3. Rendu canvas TERMINÉ — `animations: 'disabled'` de Playwright ne couvre que
 *      CSS/Web Animations, PAS l'animation d'entrée d'ECharts (active par défaut
 *      sur presque tous les wrappers). On attend donc que la signature de tous les
 *      canvas soit stable sur deux échantillons consécutifs.
 *
 * PORTÉE DES BASELINES. Les captures de `/lab/charts` (données statiques, aucun
 * backend) sont VERSIONNÉES : reproductibles par n'importe qui. Les captures des
 * pages joueur dépendent des données locales et peuvent contenir des gamertags
 * réels : elles sont écrites sous `__screenshots__/app/` que le `.gitignore` local
 * exclut. Voir e2e/visual/README.md.
 */
import { expect, type Page } from '@playwright/test'

/** Joueur visé par les specs de pages applicatives (`E2E_VISUAL_PLAYER`). */
export const VISUAL_PLAYER = process.env.E2E_VISUAL_PLAYER ?? 'demo-player'

/**
 * Coéquipiers pré-confirmés pour les pages Escouade.
 *
 * POURQUOI. `SquadLayout` n'affiche AUCUN graphe tant qu'aucun coéquipier n'est
 * confirmé (`EmptyStateNotice` « aucune sélection ») : la sélection est un état
 * UI persisté dans `localStorage['squad-teammates-<playerSlug>']`, pas une donnée
 * serveur. Sans amorçage, `squad/synergies` et `squad/dynamique` rendent 0 canvas
 * et le harnais les SKIPPE — quelle que soit la richesse de la fixture.
 *
 * Le défaut ne vaut que pour `demo-player`, dont le roster synthétique est fixe
 * (DemoPlayer2/DemoPlayer3, cf. `DemoRoster` dans internal/config/player_resolver.go) ;
 * viser un joueur réel suppose de nommer ses coéquipiers via
 * `E2E_VISUAL_SQUAD_TEAMMATES` (CSV), sinon la sélection reste vide et les deux
 * pages skippent comme avant.
 */
export const VISUAL_SQUAD_TEAMMATES = (
  process.env.E2E_VISUAL_SQUAD_TEAMMATES ??
  (VISUAL_PLAYER === 'demo-player' ? 'DemoPlayer2,DemoPlayer3' : '')
)
  .split(',')
  .map((gt) => gt.trim())
  .filter(Boolean)

/**
 * Segments de chemin des captures NON versionnées (pages joueur, données locales).
 * Un TABLEAU, pas une chaîne : `toHaveScreenshot('a/b.png')` aplatit le `/` en `-`
 * alors que les segments créent de vrais sous-dossiers — ce dont dépend la règle
 * `app/` du `.gitignore` de `__screenshots__`.
 */
export const APP_SNAPSHOT_SEGMENTS = ['app', VISUAL_PLAYER]

/** Pas et plafond du polling de stabilité canvas. */
const SETTLE_STEP_MS = 400
const SETTLE_TIMEOUT_MS = 25_000

/**
 * Prépare le contexte déterministe. À appeler AVANT toute navigation : les scripts
 * d'init s'exécutent avant le bundle de l'app, donc avant l'évaluation des données
 * de démo du showcase (constantes de module) et avant le montage du thème.
 */
export async function prepareVisualPage(page: Page): Promise<void> {
  await page.addInitScript(
    ({ player, teammates }: { player: string; teammates: string[] }) => {
      // Thème figé : même clé/forme que le store `levelup-ui-prefs` (persist zustand).
      window.localStorage.setItem(
        'levelup-ui-prefs',
        JSON.stringify({ state: { localUiPrefs: { theme: 'dark' } }, version: 0 }),
      )
      // Sélection Escouade — même clé que SquadLayout (`squad-teammates-<slug>`).
      // Posée AVANT le montage : le state s'initialise depuis localStorage une
      // seule fois (lazy initializer), un set après coup n'aurait aucun effet.
      if (teammates.length > 0) {
        window.localStorage.setItem(`squad-teammates-${player}`, JSON.stringify(teammates))
      }
      // Générateur congruentiel linéaire seedé — remplace Math.random pour la durée
      // du test. Suite fixe => données de démo du showcase reproductibles.
      // `Math.imul` est OBLIGATOIRE : avec un `*` flottant, le produit dépasse 2^53,
      // les bits de poids faible sont perdus et la suite dégénère vers 0 (la heatmap
      // du showcase devient un aplat de zéros).
      let seed = 0x2f6e2b1
      Math.random = () => {
        seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
        return seed / 0x100000000
      }
    },
    { player: VISUAL_PLAYER, teammates: VISUAL_SQUAD_TEAMMATES },
  )
}

/**
 * Signature de l'état pixel de tous les canvas de la page. Sert uniquement à
 * détecter qu'un rendu a cessé de bouger (pas à comparer deux runs).
 */
async function canvasSignature(page: Page): Promise<string> {
  return page.evaluate(() => {
    const parts: string[] = []
    document.querySelectorAll('canvas').forEach((c) => {
      try {
        const data = (c as HTMLCanvasElement).toDataURL('image/png')
        let hash = 0
        for (let i = 0; i < data.length; i += 97) {
          hash = (hash * 31 + data.charCodeAt(i)) | 0
        }
        parts.push(`${c.width}x${c.height}:${data.length}:${hash}`)
      } catch {
        parts.push('unreadable')
      }
    })
    return parts.join('|')
  })
}

/**
 * Attend que le rendu canvas soit figé : deux signatures consécutives identiques
 * ET non vides. Retourne le nombre de canvas observés (0 = page sans graphe).
 */
export async function waitForChartsSettled(page: Page): Promise<number> {
  await page
    .locator('[data-testid="chart-card-loading"]')
    .first()
    .waitFor({ state: 'detached', timeout: 15_000 })
    .catch(() => {
      /* aucun loader monté : rien à attendre */
    })

  let previous = ''
  const deadline = Date.now() + SETTLE_TIMEOUT_MS
  while (Date.now() < deadline) {
    const current = await canvasSignature(page)
    if (current !== '' && current === previous) break
    previous = current
    await page.waitForTimeout(SETTLE_STEP_MS)
  }
  return page.locator('canvas').count()
}

/**
 * Navigue puis attend un état visuellement stable. Retourne le nombre de canvas
 * rendus — une spec de page applicative s'en sert pour se SKIPPER proprement quand
 * les données locales sont absentes (page vide = rien à protéger).
 */
export async function gotoSettled(page: Page, path: string): Promise<number> {
  await page.goto(path, { waitUntil: 'domcontentloaded' })
  await page.waitForLoadState('networkidle').catch(() => {
    /* flux long (SSE/polling) : on retombe sur l'attente de stabilité canvas */
  })
  return waitForChartsSettled(page)
}

/**
 * Capture UN PNG par canvas de la page. `locator.screenshot()` fait défiler
 * l'élément dans la vue : les graphes situés sous la ligne de flottaison sont donc
 * couverts eux aussi. Granularité voulue — quand une diff apparaît après un bump,
 * elle désigne le graphe fautif au lieu d'une page entière.
 */
export async function expectCanvasesMatch(
  page: Page,
  segments: string[],
  baseName: string,
  expectedCount: number,
  skipIndexes: readonly number[] = [],
): Promise<void> {
  const canvases = page.locator('canvas')
  await expect(canvases).toHaveCount(expectedCount)
  for (let i = 0; i < expectedCount; i += 1) {
    if (skipIndexes.includes(i)) continue
    const index = String(i).padStart(2, '0')
    await expect(canvases.nth(i)).toHaveScreenshot([
      ...segments,
      `${baseName}-canvas-${index}.png`,
    ])
  }
}
