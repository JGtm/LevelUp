#!/usr/bin/env node
/**
 * Garde-fou — `apps/web/src/lib/api/generated.ts` est-il À JOUR vis-à-vis de
 * `apps/go-api/api/openapi.yaml` ?
 *
 * Chaînon manquant du verrouillage du contrat (contre-revue V72). Deux maillons
 * existaient déjà :
 *   - openapi.yaml ← code Go        : `make openapi-check` / TestOpenAPIYAMLIsUpToDate
 *   - generated.ts ← surface figée  : `contract-surface.guard.test.ts` (DISPARITIONS only)
 * Rien ne vérifiait que generated.ts DÉRIVE de l'openapi.yaml courant : une évolution
 * de contrat committée sans `make generate-types` laissait le front typé sur l'ANCIEN
 * contrat (schéma ajouté invisible, corps de réponse manquant, enum non resserré) —
 * `tsc` restant vert puisque les types sont simplement périmés, pas incohérents.
 *
 * Méthode : on rejoue EXACTEMENT la commande du script npm `generate-types`
 * (openapi-typescript, même version, mêmes options) vers un fichier temporaire, puis on
 * compare octet à octet. Rejouer le générateur — plutôt que ré-analyser le YAML — rend
 * la comparaison insensible au formatage et fidèle par construction.
 *
 * Usage      : node tools/check-generated-types-fresh.mjs
 * Exit code  : 0 = à jour · 1 = drift (ou générateur absent / en échec)
 * Réparation : cd apps/web && npm run generate-types
 *
 * Appelé par : `make openapi-check` (gate manuelle du contrat) ET
 * `apps/web/src/lib/api/generated-types-fresh.guard.test.ts` (CI, job Frontend).
 */

import { execFileSync } from 'node:child_process'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const WEB_DIR = join(REPO_ROOT, 'apps/web')
const OPENAPI_YAML = join(REPO_ROOT, 'apps/go-api/api/openapi.yaml')
const GENERATED_TS = join(WEB_DIR, 'src/lib/api/generated.ts')
// Entrée JS du CLI (et non node_modules/.bin/…) : invocable par `node` sur les trois
// plateformes, sans dépendre du shim shell/cmd.
const CLI = join(WEB_DIR, 'node_modules/openapi-typescript/bin/cli.js')

const FIX_HINT = 'Réparation : cd apps/web && npm run generate-types'

/**
 * Retourne null si generated.ts est à jour, sinon le message d'erreur expliquant
 * le drift (ou l'impossibilité de conclure). Aucun effet de bord process : le
 * caller décide (exit code côté CLI, assertion côté test).
 */
export function checkGeneratedTypesFresh() {
  for (const [label, path] of [
    ['contrat OpenAPI', OPENAPI_YAML],
    ['types générés', GENERATED_TS],
    ['CLI openapi-typescript (npm ci manquant ?)', CLI],
  ]) {
    if (!existsSync(path)) return `${label} introuvable : ${path}`
  }

  const tmp = mkdtempSync(join(tmpdir(), 'levelup-gen-types-'))
  const candidate = join(tmp, 'generated.ts')
  try {
    try {
      execFileSync(process.execPath, [CLI, OPENAPI_YAML, '-o', candidate], {
        cwd: WEB_DIR,
        stdio: 'pipe',
      })
    } catch (err) {
      const detail = err?.stderr?.toString().trim() || err?.message || String(err)
      return `échec de la génération de référence :\n${detail}\n  ${FIX_HINT}`
    }

    const expected = readFileSync(candidate, 'utf8')
    const actual = readFileSync(GENERATED_TS, 'utf8')
    if (expected === actual) return null

    // Première ligne divergente : suffit à identifier le schéma/chemin en cause.
    const exp = expected.split('\n')
    const act = actual.split('\n')
    let i = 0
    while (i < exp.length && i < act.length && exp[i] === act[i]) i++
    return (
      'DRIFT — generated.ts ne correspond pas à openapi.yaml.\n' +
      `  Première divergence ligne ${i + 1} :\n` +
      `    committé : ${JSON.stringify(act[i] ?? '<fin de fichier>')}\n` +
      `    attendu  : ${JSON.stringify(exp[i] ?? '<fin de fichier>')}\n` +
      `  ${FIX_HINT}`
    )
  } finally {
    rmSync(tmp, { recursive: true, force: true })
  }
}

// Exécution directe (make openapi-check) : import en tant que module = pas d'effet.
if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  const problem = checkGeneratedTypesFresh()
  if (problem) {
    console.error(`[generated-types] ${problem}`)
    process.exit(1)
  }
  console.log('[generated-types] OK — generated.ts dérive bien de openapi.yaml.')
}
