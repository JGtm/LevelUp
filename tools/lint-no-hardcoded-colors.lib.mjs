/**
 * Coeur du lint anti-hardcode des couleurs (`tools/lint-no-hardcoded-colors.mjs`) : motifs, tolérances et
 * analyse d'UNE ligne. Séparé du balayage de l'arbre pour être testé
 * (`apps/web/src/lib/accessibility/lintNoHardcodedColors.guard.test.ts`).
 *
 * Trois familles de motifs (le « genre » d'une violation) :
 *   - `hex`              : #RGB / #RGBA / #RRGGBB / #RRGGBBAA ;
 *   - `tailwind`         : classe Tailwind couleur nommée (`text-red-500`, `ring-offset-amber-300`...) et noir /
 *                          blanc absolus (`bg-black/60`, `outline-white`) qui contournent les jetons
 *                          (`scrim` / `on-scrim`) ;
 *   - `fonction-couleur` : oklch() / oklab() / lch() / rgb() / rgba() / hsl() / hsla() / hwb().
 *
 * Exception ponctuelle : un commentaire `color-allow(<genre>[, <genre>]): <raison>` sur la MÊME ligne
 * n'excuse QUE les genres qu'il nomme. Une ligne qui porte un hex excusé et gagne ensuite une classe
 * Tailwind couleur rougit : l'exception ne couvre pas « toute couleur de la ligne ». Un `color-allow` sans
 * genre n'excuse rien.
 */

/** Genres de violation, tels qu'un `color-allow(...)` les nomme. */
export const KINDS = ['hex', 'tailwind', 'fonction-couleur']

// 1. Hex colors.
const HEX_PATTERN = /#[0-9a-fA-F]{3,8}\b/g

// 2. Classes Tailwind couleur : text-red-500, bg-green-100, border-blue-700, ring-offset-amber-400, etc.
//    On match les couleurs nommées de Tailwind. Ajoutés le 2026-11-14 (plan compléments 8.2.0, 5.2) :
//    ring-offset, outline, decoration, accent, caret — ils échappaient au balayage.
const TW_COLOR_NAMES = [
  'red', 'orange', 'amber', 'yellow', 'lime', 'green', 'emerald',
  'teal', 'cyan', 'sky', 'blue', 'indigo', 'violet', 'purple',
  'fuchsia', 'pink', 'rose', 'slate', 'gray', 'zinc', 'neutral', 'stone',
]
const TW_PREFIXES = [
  'text', 'bg', 'border', 'ring', 'ring-offset', 'fill', 'stroke', 'divide', 'placeholder',
  'outline', 'decoration', 'accent', 'caret',
]
const TW_PATTERN = new RegExp(
  `\\b(?:${TW_PREFIXES.join('|')})-(?:${TW_COLOR_NAMES.join('|')})-(?:50|100|200|300|400|500|600|700|800|900|950)\\b`,
  'g',
)

// 2 bis. Noir et blanc absolus (plan 8.1.2, 1.4) : `bg-black/60`, `text-white`, etc. Ils
//    contournent les jetons (`scrim` / `on-scrim`) : un noir ou un blanc écrit en dur ne suit
//    pas le thème. Compté dans la famille `tailwind`.
const ABSOLUTE_PREFIXES = [
  'bg', 'text', 'border', 'ring', 'ring-offset', 'fill', 'stroke', 'divide', 'from', 'via', 'to', 'shadow',
  'outline', 'decoration', 'accent', 'caret',
]
const ABSOLUTE_COLOR_PATTERN = new RegExp(
  `\\b(?:${ABSOLUTE_PREFIXES.join('|')})-(?:black|white)(?:\\/\\d+)?\\b`,
  'g',
)

// 3. Fonctions de couleur CSS : oklch(), rgb()/rgba(), hsl()/hsla()...
//    AJOUTE LE 2026-09-06 (lot v2, correction C5 de la revue R1) : sans lui, le lint canonique ne couvrait
//    pas ce que deux gardes du rejeu supprimés portaient. Le thème reste la seule source : ces valeurs
//    vivent dans styles/globals.css, jamais dans une feature.
//    `color(` est VOLONTAIREMENT ABSENT de la liste : c'est aussi le nom d'une variable de rendu dans le
//    dépôt (`color(rowIndex, colIndex)`), et un motif qui crie faux se désactive.
const CSS_COLOR_FN_PATTERN = /\b(?:oklch|oklab|lch|rgba?|hsla?|hwb)\s*\(/g

// Fichiers explicitement tolérés (chemins relatifs à la racine du dépôt, comparés par sous-chaîne).
export const ALLOWED_FILES = [
  // Palettes brutes — c'est leur seule raison d'être.
  'lib/accessibility/palettes/',
  'lib/accessibility/_palettes.ts',
  // Rareté Halo (BattlePass) — couleurs de rareté hors charte sémantique.
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

const ALLOW_MARKER = /color-allow(?:\(([^)]*)\))?/

/**
 * Genres excusés par le `color-allow(...)` de la ligne : `null` sans marqueur, ensemble vide pour un
 * marqueur sans genre ou dont aucun genre n'est connu.
 */
export function allowedKinds(line) {
  const m = ALLOW_MARKER.exec(line)
  if (!m) return null
  const named = (m[1] ?? '').split(',').map((k) => k.trim())
  return new Set(named.filter((k) => KINDS.includes(k)))
}

/**
 * Une ligne de commentaire (`//` ou `*`) ne peint rien : tolérée. L'OUVERTURE DE BLOC (`/*`) N'EST PAS
 * TOLÉRÉE (2026-09-06, ronde 2 de la revue, constat N1) : ce test ne regarde que le premier caractère non
 * blanc, donc `/* rien *\/ const c = '#ff00aa'` passerait. Les lignes de prose qui ouvrent un bloc portent
 * un `color-allow(<genre>)` nommé.
 */
function isCommentLine(line) {
  const trimmed = line.trimStart()
  return trimmed.startsWith('//') || trimmed.startsWith('*')
}

/** Le fichier est-il toléré en entier ? `relPath` : chemin relatif à la racine, séparateurs `/`. */
export function isAllowedFile(relPath) {
  return ALLOWED_FILES.some((allowed) => relPath.includes(allowed))
}

function collect(line, pattern, kind, out) {
  const matches = line.match(pattern)
  if (matches) for (const match of matches) out.push({ kind, match })
}

/**
 * Violations d'UNE ligne : `[{ kind, match }]`, hors tolérances (fichier, ligne de commentaire, genres
 * nommés par un `color-allow(...)`).
 */
export function scanLine(relPath, line) {
  if (isAllowedFile(relPath) || isCommentLine(line)) return []
  const found = []
  collect(line, HEX_PATTERN, 'hex', found)
  collect(line, TW_PATTERN, 'tailwind', found)
  collect(line, ABSOLUTE_COLOR_PATTERN, 'tailwind', found)
  collect(line, CSS_COLOR_FN_PATTERN, 'fonction-couleur', found)
  const allowed = allowedKinds(line)
  return allowed ? found.filter((v) => !allowed.has(v.kind)) : found
}
