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

// 3. Fonctions de couleur CSS : oklch(), rgb()/rgba(), hsl()/hsla(), color()...
//    AJOUTE LE 2026-09-06 (lot v2, correction C5 de la revue R1). Deux gardes du rejeu
//    supprimes en D.12 portaient ce controle (oklch sur les encres d'effet de tir, oklch et
//    rgba sur les fichiers du lacher d'equipement) : sans lui, le lint canonique ne couvrait
//    PAS ce qu'il remplacait — une couleur litterale ecrite dans layers/fxInk.ts passait le
//    gate global sans un seul rouge (mutation M10 de la revue). Le theme reste la seule
//    source : ces valeurs vivent dans styles/globals.css, jamais dans une feature.
//    `color(` est VOLONTAIREMENT ABSENT de la liste : c'est aussi le nom d'une variable de
//    rendu dans le depot (`color(rowIndex, colIndex)`, valueGridModel.ts), et un motif qui
//    crie faux se desactive.
const CSS_COLOR_FN_PATTERN = /\b(?:oklch|oklab|lch|rgba?|hsla?|hwb)\s*\(/g

// Whitelist de fichiers / patterns explicitement tolérés.
const ALLOWED_FILES = [
  // Palettes brutes — c'est leur seule raison d'être.
  'lib/accessibility/palettes/',
  'lib/accessibility/_palettes.ts',
  // Rareté Halo (BattlePass) — couleurs de rareté hors charte sémantique.
  // Cf. CLAUDE.md §20 : couleurs de rareté Halo (Battlepass, `rarity.ts`) tolérées.
  'features/home/rarity.ts',
  'features/battlepass/rarity.ts',
  'features/palmares/rarity.ts',
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
  // Lignes purement commentaires (`//` ou `*`) — tolérées : une ligne de commentaire ne
  // peint rien.
  //
  // L'OUVERTURE DE BLOC (`/*`) N'EST PAS TOLÉRÉE, et c'est une décision (2026-09-06, ronde 2
  // de la revue, constat N1) : ce test ne regarde que le premier caractère non blanc, donc
  // `/* rien */ const c = '#ff00aa'` serait passé — le gate HISTORIQUE sur les hex et les
  // classes Tailwind, lui, l'attrapait. Élargir le lint aux fonctions de couleur ne doit pas
  // le rétrécir ailleurs. Les deux lignes de PROSE que cette tolérance protégeait (les
  // en-têtes qui décrivent la rampe de la carte de chaleur) portent un `color-allow` nommé,
  // comme les dix-sept autres exceptions du lot.
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
      // Fonctions de couleur CSS (oklch / rgb / hsl / color...)
      const fnMatches = line.match(CSS_COLOR_FN_PATTERN)
      if (fnMatches) {
        for (const m of fnMatches) {
          violations.push({ file: relPath, line: i + 1, kind: 'fonction-couleur', match: m })
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
