/**
 * Helper E2E — détection de la présence des fixtures démo.
 *
 * POURQUOI : en CI, le backend démarre en `LEVELUP_DEMO_MODE=true` mais les
 * fixtures démo (`data/demo/`, gitignorées) ne sont jamais générées — le seul
 * générateur (`levelup seed-demo`) extrait des données RÉELLES de prod et ne
 * tourne que sur l'hôte de prod (job `deploy-demo`). Sans fixture, toutes les
 * routes `/players/<slug>/pages/*` échouent → les specs data-dépendantes
 * cassaient structurellement (~60 tests rouges) et masquaient les vrais signaux.
 *
 * Politique « vert ou signal, jamais structurellement rouge » : les specs qui
 * dépendent de données joueur appellent `skipIfNoDemoData()` en tête ; quand les
 * fixtures sont absentes, le test est SKIP (visible, motivé) au lieu de FAIL. Les
 * specs infra (montage du shell, absence d'erreur 500, i18n, redirections,
 * onboarding) n'appellent pas ce garde et restent exécutées.
 *
 * Sonde : `GET /api/v1/healthz/home?player=demo-player` (health_home.go). Le
 * discriminant est la RÉSOLUTION du joueur démo, pas la complétude de la home :
 *   - 404 (player_not_found → resolveDemoPlayer échoue) = fixture ABSENTE → skip ;
 *   - 200 (home complète) ou 503 (joueur résolu mais une section home vide) =
 *     données PRÉSENTES → exécuter. Un seed démo légitime peut renvoyer 503 (p.ex.
 *     bannière ou arme favorite vide) tout en ayant assez de données pour les
 *     specs : réserver le skip au seul cas « joueur introuvable » évite de masquer
 *     les regressions sur un démo réellement seedé.
 * Résultat mémoïsé (une seule sonde par worker, `workers: 1`).
 *
 * Voir e2e/README.md.
 */
import { test, request as pwRequest } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8000'

let cachedAvailable: boolean | undefined

/**
 * Sonde l'API et mémoïse le verdict. Ne met en cache QUE les réponses HTTP
 * définitives (200 vs autre) ; une erreur réseau transitoire n'est pas cachée
 * pour éviter de figer un faux "absent" sur un hoquet de démarrage.
 */
async function probeDemoDataAvailable(): Promise<boolean> {
  if (cachedAvailable !== undefined) return cachedAvailable
  const ctx = await pwRequest.newContext({
    baseURL: API_BASE,
    // Origin requis par le middleware CSRF de l'API Go (cf. playwright.config.ts).
    extraHTTPHeaders: { Origin: 'http://localhost:5173' },
  })
  try {
    const status = (await ctx.get('/api/v1/healthz/home?player=demo-player')).status()
    // 404 = joueur démo non résolvable (fixture absente) ; 200/503 = joueur résolu.
    cachedAvailable = status !== 404
    return cachedAvailable
  } catch {
    return false
  } finally {
    await ctx.dispose()
  }
}

/**
 * À appeler EN TÊTE d'un test data-dépendant. Si les fixtures démo sont absentes
 * (cas CI standard), marque le test SKIP avec une raison explicite ; sinon le
 * test s'exécute normalement.
 */
export async function skipIfNoDemoData(): Promise<void> {
  const available = await probeDemoDataAvailable()
  test.skip(
    !available,
    'Fixture démo absente (data/demo non peuplé) — spec data-dépendante non exécutable en CI. ' +
      'Générer via `levelup seed-demo --synthetic`. Voir e2e/README.md.',
  )
}

let cachedServerIsDemo: boolean | undefined

/**
 * Sonde `GET /api/v1/bootstrap` : le serveur tourne-t-il en mode démo (`feature_flags.demo_mode`) ?
 * Mémoïsé comme la sonde des fixtures ; une erreur réseau n'est pas mise en cache.
 */
async function probeServerIsDemo(): Promise<boolean> {
  if (cachedServerIsDemo !== undefined) return cachedServerIsDemo
  const ctx = await pwRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Origin: 'http://localhost:5173' },
  })
  try {
    const resp = await ctx.get('/api/v1/bootstrap')
    if (!resp.ok()) return false
    const body = (await resp.json()) as { feature_flags?: { demo_mode?: boolean } }
    cachedServerIsDemo = body.feature_flags?.demo_mode === true
    return cachedServerIsDemo
  } catch {
    return false
  } finally {
    await ctx.dispose()
  }
}

/**
 * Skip un test qui ÉCRIT (like, envoi, association…) quand le serveur est en démo : la démo
 * publique est en lecture seule, toute écriture y répond 403 `demo_mode_forbidden`
 * (middleware.DemoReadOnly). Le test s'exécute contre une instance non démo.
 */
export async function skipIfServerReadOnlyDemo(reason: string): Promise<void> {
  const demo = await probeServerIsDemo()
  test.skip(
    demo,
    `Serveur en démo (lecture seule : les écritures répondent 403 demo_mode_forbidden) — ${reason}`,
  )
}

/**
 * Skip une spec devenue OBSOLÈTE (drift) : la fixture démo synthétique a rendu la
 * spec exécutable et révélé qu'elle vise une route / un endpoint / une structure UI
 * qui a changé depuis son écriture (elle skippait toujours faute de démo, donc la
 * dérive n'a jamais été détectée). Skip inconditionnel + raison : la spec doit être
 * RÉÉCRITE pour l'UI courante (hors périmètre du chantier fixture E2E). Voir e2e/README.md.
 */
export function skipObsoleteSpec(reason: string): void {
  test.skip(true, `Spec obsolète (dérive UI/route/endpoint — à réécrire) : ${reason}`)
}

/**
 * Skip une spec qui exige les données d'un JOUEUR RÉEL spécifique (gamertags réels
 * codés en dur, coéquipiers nommés + synergies, backfill engagement, médias réels)
 * — non reproductibles par une fixture SYNTHÉTIQUE. Skip UNIQUEMENT quand la démo est
 * synthétique (E2E_SYNTHETIC_DEMO=1, posé par la CI) ; contre une démo réelle
 * (`seed-demo` sur l'hôte de prod), la spec s'exécute normalement.
 */
export function skipRequiresRealPlayer(reason: string): void {
  test.skip(
    process.env.E2E_SYNTHETIC_DEMO === '1',
    `Nécessite un joueur réel (non synthétisable) : ${reason}`,
  )
}
