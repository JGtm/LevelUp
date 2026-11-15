#!/usr/bin/env node
/**
 * Lint custom — anti-hardcode des couleurs côté React.
 *
 * P8.1 (revue 2026-04-29 axe 5) : interdit les littéraux hex `#RRGGBB[AA]`
 * et les classes Tailwind couleur (`text-red-*`, `bg-green-*`, etc.) dans
 * `apps/web/src/{features,components}/`.
 *
 * Toute couleur sémantique doit passer par :
 *   - `tokenCssVar(token)` (JSX style)
 *   - `resolveToken(token)` (Plotly/SVG)
 *   - `getSeriesColors(n, tokens[])` (séries)
 *   - les palettes brutes centralisées dans `apps/web/src/lib/accessibility/palettes/`.
 *
 * Exceptions tolérées (cf. CLAUDE.md §20) :
 *   - couleurs de rareté Halo (Battlepass, `rarity.ts`)
 *   - couleurs structurelles de layout SVG (fond de piste, bordure)
 *   - couleurs UI génériques sans signification métier (liked/rose, warning/amber)
 *
 * Les motifs, les tolérances et l'analyse d'une ligne vivent dans `lint-no-hardcoded-colors.lib.mjs` (testé).
 * Exception ponctuelle : `color-allow(<genre>): <raison>` sur la ligne, qui n'excuse QUE le genre nommé
 * (hex, tailwind, fonction-couleur).
 *
 * Usage :
 *   node tools/lint-no-hardcoded-colors.mjs
 *
 * Exit code :
 *   0 = clean
 *   1 = violations détectées
 */

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { scanLine, isAllowedFile } from './lint-no-hardcoded-colors.lib.mjs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const WEB_SRC = join(REPO_ROOT, 'apps/web/src')

function* walk(dir) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) {
      // Skip node_modules, build outputs.
      if (entry === 'node_modules' || entry === 'dist' || entry === '.tanstack') continue
      yield* walk(full)
    } else if (
      entry.endsWith('.ts') ||
      entry.endsWith('.tsx') ||
      entry.endsWith('.js') ||
      entry.endsWith('.jsx')
    ) {
      yield full
    }
  }
}

const violations = []

const featuresDir = join(WEB_SRC, 'features')
const componentsDir = join(WEB_SRC, 'components')

// lib/replay/ EST DU PRODUIT, pas de l'outillage (ajoute le 2026-09-06, correction C5) :
// le lot v2 D.13 y a descendu le document du rejeu, sa normalisation, sa logique de lecture,
// le roster et le chargement — sept modules que la Match View et la page de rejeu lisent a
// egalite. Les laisser hors du balayage aurait ouvert un angle mort de la taille d'un
// deplacement de fichier. Aucune allowlist n'y est necessaire : mesure du 2026-09-06, aucun
// de ces modules ne DEFINIT de couleur (ils lisent des tokens, comme le reste du produit).
const libReplayDir = join(WEB_SRC, 'lib/replay')

for (const rootDir of [featuresDir, componentsDir, libReplayDir]) {
  let root
  try {
    root = statSync(rootDir)
  } catch {
    continue
  }
  if (!root.isDirectory()) continue

  for (const file of walk(rootDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    if (isAllowedFile(relPath)) continue

    const lines = readFileSync(file, 'utf8').split('\n')
    for (let i = 0; i < lines.length; i++) {
      for (const v of scanLine(relPath, lines[i])) {
        violations.push({ file: relPath, line: i + 1, kind: v.kind, match: v.match })
      }
    }
  }
}

if (violations.length === 0) {
  console.log('lint-no-hardcoded-colors: clean (0 violation)')
  process.exit(0)
}

// Group by file for diagnostic.
const byFile = new Map()
for (const v of violations) {
  if (!byFile.has(v.file)) byFile.set(v.file, [])
  byFile.get(v.file).push(v)
}

console.log(`lint-no-hardcoded-colors: ${violations.length} violation(s) détectée(s).\n`)
for (const [file, items] of byFile.entries()) {
  console.log(file + ' :')
  for (const v of items.slice(0, 5)) {
    console.log(`  ${file}:${v.line} — ${v.kind} "${v.match}"`)
  }
  if (items.length > 5) {
    console.log(`  ... et ${items.length - 5} autre(s)`)
  }
}

console.log(
  '\nutilisez tokenCssVar(token) / resolveToken(token) / getSeriesColors() ' +
    'au lieu de littéraux hardcodés.\n' +
    'Marqueur inline pour exception ponctuelle : `// color-allow(<genre>): <raison>` (genres : hex, tailwind, fonction-couleur).',
)

// Ratchet : abaissé à 0 le 2026-05-02 après nettoyage complet du backlog
// dans le PR settings.show_progression. Toute nouvelle violation doit
// soit être remplacée par un token (`tokenCssVar` / classe thème
// `bg-warning`/`text-success`/etc.), soit annotée `// color-allow:` avec
// une justification (couleur structurelle SVG, fallback API, thématique
// Spartan UI, like rose, etc. — cf. CLAUDE.md §20).
const RATCHET_THRESHOLD = 0

if (violations.length > RATCHET_THRESHOLD) {
  console.log(
    `\nERREUR : ${violations.length} > plafond P8.1 (${RATCHET_THRESHOLD}). ` +
      "Cleanup requis avant merge.",
  )
  process.exit(1)
}

console.log(
  `\nInfo : ${violations.length} <= plafond P8.1 (${RATCHET_THRESHOLD}). ` +
    'Pas d\'échec — ratchet en place pour bloquer toute régression.',
)
process.exit(0)
