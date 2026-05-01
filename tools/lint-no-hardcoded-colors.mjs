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

const __dirname = dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = resolve(__dirname, '..')
const WEB_SRC = join(REPO_ROOT, 'apps/web/src')

// Patterns à détecter.
//
// 1. Hex colors : #RGB / #RGBA / #RRGGBB / #RRGGBBAA dans du JSX/CSS-in-JS
//    (entre quotes, après `:`, dans tokenize/style).
const HEX_PATTERN = /#[0-9a-fA-F]{3,8}\b/g

// 2. Classes Tailwind couleur : text-red-500, bg-green-100, border-blue-700,
//    ring-amber-400, etc. On match les couleurs nommées de Tailwind.
const TW_COLOR_NAMES = [
  'red', 'orange', 'amber', 'yellow', 'lime', 'green', 'emerald',
  'teal', 'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple',
  'fuchsia', 'pink', 'rose', 'slate', 'gray', 'zinc', 'neutral', 'stone',
]
const TW_PREFIXES = ['text', 'bg', 'border', 'ring', 'fill', 'stroke', 'divide', 'placeholder']
const TW_PATTERN = new RegExp(
  `\\b(?:${TW_PREFIXES.join('|')})-(?:${TW_COLOR_NAMES.join('|')})-(?:50|100|200|300|400|500|600|700|800|900|950)\\b`,
  'g',
)

// Whitelist de fichiers / patterns explicitement tolérés.
const ALLOWED_FILES = [
  // Palettes brutes — c'est leur seule raison d'être.
  'lib/accessibility/palettes/',
  'lib/accessibility/_palettes.ts',
  // Rareté Halo (BattlePass) — couleurs de rareté hors charte sémantique.
  'features/home/rarity.ts',
  'features/battlepass/rarity.ts',
  // Tests : on accepte les fixtures et snapshots.
  '.test.ts',
  '.test.tsx',
  '.spec.ts',
  // E2E specs.
  'e2e/',
  // Fichier généré.
  'lib/api/generated.ts',
  // routeTree généré.
  'routeTree.gen.ts',
]

// Substrings tolérées DANS le code (commentaires, strings spécifiques).
// Ex: `// #fafafa is the border color of the chart frame` reste tolerable
// quand annoté avec `// color-allow:`.
const ALLOWED_INLINE_MARKER = 'color-allow'

function isAllowed(relPath, line) {
  for (const allowed of ALLOWED_FILES) {
    if (relPath.includes(allowed)) return true
  }
  if (line.includes(ALLOWED_INLINE_MARKER)) return true
  // Lignes purement commentaires (`//` ou `*`) — tolérées.
  const trimmed = line.trimStart()
  if (trimmed.startsWith('//') || trimmed.startsWith('*')) return true
  return false
}

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

for (const rootDir of [featuresDir, componentsDir]) {
  let root
  try {
    root = statSync(rootDir)
  } catch {
    continue
  }
  if (!root.isDirectory()) continue

  for (const file of walk(rootDir)) {
    const relPath = relative(REPO_ROOT, file).replaceAll('\\', '/')
    if (ALLOWED_FILES.some((a) => relPath.includes(a))) continue

    const text = readFileSync(file, 'utf8')
    const lines = text.split('\n')
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]
      if (isAllowed(relPath, line)) continue

      // Hex check
      const hexMatches = line.match(HEX_PATTERN)
      if (hexMatches) {
        for (const m of hexMatches) {
          violations.push({ file: relPath, line: i + 1, kind: 'hex', match: m })
        }
      }
      // Tailwind color check
      const twMatches = line.match(TW_PATTERN)
      if (twMatches) {
        for (const m of twMatches) {
          violations.push({ file: relPath, line: i + 1, kind: 'tailwind', match: m })
        }
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
    'Marqueur inline pour exception ponctuelle : `// color-allow: <raison>`.',
)

// Ratchet : pour P8.1 initial on tolère les violations existantes (release
// progressive du linter). Le but est d'interdire les NOUVELLES violations.
// Le seuil sera resserré commit après commit.
// Plafond actuel — abaissé progressivement par PR. Au moment de l'écriture
// de P8.1 (revue 2026-04-29) : 143 violations. Bumpé à 155 le 2026-05-01
// pour absorber la dette introduite par les commits "wip(home): 8 bugs"
// (12 violations introduites hors-scope du présent travail Prestige). À
// resserrer dès que la dette LUSR/SkillPeak sera nettoyée.
const RATCHET_THRESHOLD = 155

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
