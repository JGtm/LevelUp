/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail ratchet (D-10, CLAUDE.md n°6) — le module `lib/title-routing/` est la
 * source UNIQUE de l'interprétation/construction des segments de route title-scoped
 * (`/t/{slug}` ET `/players/{playerSlug}`).
 *
 * Deux règles :
 *  1. `/t/` (armée Phase 1) : aucun littéral de PARSING du namespace titre hors module.
 *  2. `/players/` (armée Phase 2e, 2026-07-23) : aucun littéral de CHEMIN DE ROUTE
 *     front `/players/` hors allowlist. Les routes ont été physiquement déplacées sous
 *     `/{-$lang}/t/$titleSlug/players/$playerSlug/…` : un lien doit passer par le `to`
 *     typé (`FileRouteTypes['to']`) + `params`, JAMAIS par un littéral `/players/`
 *     recopié (sinon la nav re-émet des URLs mortes — cause corrigée au lot 2-B).
 *
 * BORNAGE anti-faux-positif (crucial) : les URL d'API backend
 * `` `/players/${playerSlug}/…` `` (backtick + interpolation `${`) sont des chemins
 * d'API `/api/v1`, PAS des routes front. La règle `/players/` matche donc :
 *   - un littéral quote SIMPLE/DOUBLE immédiatement suivi de `/players/`
 *     (`'/players/$playerSlug/…'`, forme legacy des routes front) ;
 *   - un échappement regex `\/players\/` (matchers de pathname) ;
 *   - [2026-07-23, lot 2-C] un backtick `` `/players/${ `` UNIQUEMENT en CONTEXTE DE
 *     CIBLE DE NAVIGATION, i.e. précédé (même expression) d'un `to:` / `to=` / `href:` /
 *     `href=` (Link/navigate/redirect/`<a>`). C'était la DERNIÈRE famille de littéraux de
 *     route legacy (cibles `` to: `/players/${…}` ``) échappant au typecheck ET à cette
 *     règle : émettait des URLs ancien format (mortes puis 1 hop de redirect).
 * Elle EXCLUT le backtick NU (appels API interpolés `api.get(`/players/${…}`)`, JAMAIS
 * précédés de `to`/`href` : le `\s*` de la règle ne franchit pas `api.get(` non blanc) et
 * les templates canoniques `'/{-$lang}/…/players/…'` (le `/players/` n'y est jamais collé
 * au quote). RÉSIDUEL ASSUMÉ : une route bâtie via une variable intermédiaire non-`to`/
 * `href` (`const u = `/players/…``) échapperait — même angle mort que la règle quote ; en
 * pratique une cible passe toujours par `to`/`href`. Les fichiers `*.test.ts(x)` sont hors
 * scan (walk) — fixtures autorisées à contenir des URLs concrètes.
 *
 * Allowlist datée :
 *   - [2026-07-22] `/t/`      : `lib/title-routing/` UNIQUEMENT.
 *   - [2026-07-23] `/players/`: `lib/title-routing/` (helpers playerScopedPath +
 *     buildLegacyRedirect) + `routes/players/` (splat de redirection legacy créé en
 *     Phase 3 — inscrit PAR AVANCE, regex legacy volontaire du splat) +
 *     `features/feedback-drawer/` (regex legacy volontaire : `matchArea` /
 *     `classifyFeedback` classent le PATHNAME courant en aire de feedback — matchers
 *     read-only, PAS des cibles de route ; corrects sur les nouvelles URLs via match
 *     de sous-chaîne ; hors périmètre lot 2-B → migration vers `playerRelativePath`
 *     reportée en Découverte).
 * Fichiers GÉNÉRÉS (jamais édités) exclus du walk : `routeTree.gen.ts` (routeur),
 * `generated.ts` (types OpenAPI — chemins d'API `/players/{player_slug}/…`, pas des
 * routes front).
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Littéral de parsing du namespace `/t/` : `'/t/`, `"/t/`, `` `/t/ ``, `/t/${`, ou
// l'échappement regex `\/t\/`. Un `/t/` NU dans un commentaire ou une prose
// non-quotée (ex. « handleChange/t/frozen ») n'est PAS matché (pas de marqueur).
const TITLE_NS_LITERAL = /(['"`]\/t\/)|(\/t\/\$\{)|(\\\/t\\\/)/

// Littéral de CHEMIN DE ROUTE front `/players/`, trois formes (cf. en-tête BORNAGE) :
//  1. quote SIMPLE/DOUBLE collée (route legacy `'/players/$playerSlug…'`) ;
//  2. échappement regex `\/players\/` (matchers de pathname) ;
//  3. backtick `` `/players/${ `` en CONTEXTE de cible de navigation — `to`/`href` suivi de
//     `:`/`=` (puis `{` JSX optionnel) immédiatement avant le backtick (lot 2-C). Le
//     backtick NU (appels API `api.get(`/players/${…}`)`) reste EXCLU.
const PLAYERS_ROUTE_LITERAL =
  /(['"]\/players\/)|(\\\/players\\\/)|(\b(?:to|href)\s*[:=]\s*\{?\s*`\/players\/\$\{)/

const srcRoot = resolve(process.cwd(), 'src')
const titleRoutingDir = join(srcRoot, 'lib', 'title-routing')
// Allowlist `/players/` : module title-routing + splat legacy (Phase 3) + classifieur
// feedback-drawer (regex legacy volontaire — cf. en-tête).
const PLAYERS_ALLOWED_DIRS = [
  titleRoutingDir,
  join(srcRoot, 'routes', 'players'),
  join(srcRoot, 'features', 'feedback-drawer'),
]

// Fichiers générés (jamais édités) : exclus du scan (chemins d'API / routeur généré).
const GENERATED_FILES = new Set(['routeTree.gen.ts', 'generated.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      if (GENERATED_FILES.has(entry.name)) continue
      out.push(full)
    }
  }
  return out
}

const allFiles = walk(srcRoot)

describe('garde-rail title-routing (source unique du parsing /t/)', () => {
  it('aucun littéral /t/ de parsing hors lib/title-routing/', () => {
    const offenders = allFiles
      .filter((f) => !f.startsWith(titleRoutingDir))
      .filter((f) => TITLE_NS_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(srcRoot, 'src'))
    expect(
      offenders,
      `Littéral /t/ à router via @/lib/title-routing (parseRouteSegments/buildLegacyRedirect) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})

describe('garde-rail routes (aucun littéral de chemin /players/ hors allowlist)', () => {
  it('aucun littéral de route /players/ hors title-routing + splat legacy', () => {
    const offenders = allFiles
      .filter((f) => !PLAYERS_ALLOWED_DIRS.some((dir) => f.startsWith(dir)))
      .filter((f) => PLAYERS_ROUTE_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(srcRoot, 'src'))
    expect(
      offenders,
      `Littéral de route /players/ interdit (utiliser le \`to\` typé + params) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
