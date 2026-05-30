#!/usr/bin/env node
/**
 * Ratchet knip — fige le code mort frontend (unused files / exports / types) à
 * un plafond et bloque toute régression. Même esprit que :
 *   - tools/lint-cross-feature-imports.mjs (plafond 10)
 *   - tools/lint-no-hardcoded-colors.mjs   (ratchet)
 *
 * Lancé en pre-push (cf. lefthook.yml). En local :
 *   node tools/knip-ratchet.mjs
 *
 * RÈGLE : les plafonds ne se réduisent jamais seuls. Quand tu nettoies du code
 * mort, ré-exécute et ABAISSE THRESHOLDS au nouveau compte. Ne JAMAIS relever un
 * plafond sans justification (= laisser entrer du code mort).
 *
 * Note : ce ratchet ne fait QUE garder la ligne. Le détail des findings reste
 * accessible via `cd apps/web && npx knip`.
 *
 * Exit : 0 = OK (compte <= plafond) · 1 = régression (compte > plafond).
 */
import { execSync } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const WEB_DIR = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'apps', 'web')

// Plafonds figés le 2026-05-30 (post-sprint explorer enrichi + session charts).
// files=31 : SessionKDATimeline + SessionOcdrScatter sont des composants session
//   créés lors de sprints précédents et jamais intégrés dans le layout final —
//   dette UI à nettoyer (hors scope immédiat).
// types=84 temporaire : explorerScope.ts WIP (refactor/arch-port-abstractions) compte 1 type non câblé.
// Remettre à 83 dès que le type est consommé ou supprimé.
const THRESHOLDS = { files: 31, exports: 87, types: 84 }

function knipJson() {
  try {
    return execSync('npx knip --reporter json', {
      cwd: WEB_DIR,
      encoding: 'utf8',
      maxBuffer: 64 * 1024 * 1024,
      stdio: ['ignore', 'pipe', 'ignore'],
    })
  } catch (err) {
    // knip sort en code 1 dès qu'il y a des findings — cas normal ici.
    if (err.stdout) return err.stdout
    console.error('knip-ratchet : impossible d\'exécuter knip —', err.message)
    process.exit(1)
  }
}

let data
try {
  data = JSON.parse(knipJson())
} catch (err) {
  console.error('knip-ratchet : sortie knip non-JSON —', err.message)
  process.exit(1)
}

const actual = { files: 0, exports: 0, types: 0 }
for (const issue of data.issues ?? []) {
  actual.files += (issue.files ?? []).length
  actual.exports += (issue.exports ?? []).length
  actual.types += (issue.types ?? []).length
}

let regressed = false
let underBudget = false
const rows = []
for (const key of ['files', 'exports', 'types']) {
  const a = actual[key]
  const max = THRESHOLDS[key]
  let tag
  if (a > max) {
    tag = 'RÉGRESSION'
    regressed = true
  } else if (a < max) {
    tag = 'à abaisser'
    underBudget = true
  } else {
    tag = 'OK'
  }
  rows.push(`  ${key.padEnd(8)} ${String(a).padStart(3)} / plafond ${max}  [${tag}]`)
}

console.log('knip-ratchet (code mort frontend) :')
console.log(rows.join('\n'))

if (regressed) {
  console.error(
    '\nERREUR : régression de code mort (compte > plafond).\n' +
      'Supprime le code mort ajouté, ou — si l\'ajout est justifié — relève le\n' +
      'plafond correspondant dans tools/knip-ratchet.mjs (avec justification).',
  )
  process.exit(1)
}
if (underBudget) {
  console.log(
    '\nInfo : compte sous le plafond — pense à abaisser THRESHOLDS dans\n' +
      'tools/knip-ratchet.mjs pour verrouiller le gain.',
  )
}
process.exit(0)
