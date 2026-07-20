/**
 * fragClass.ts — Échelle couleur des CLASSES d'arme du sunburst « Répartition des
 * frags » v2 (classe→rôle). SOURCE UNIQUE réutilisée par toutes les surfaces
 * (Synthesis, Match view, Timeseries, Sessions, Escouade) — cf.
 * .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P1.
 *
 * GAMME « ANTAGONISTES » (réactive à la palette) : chaque classe est mappée sur un
 * TOKEN sémantique de la gamme Antagonistes (la même que MatchAntagonistChart :
 * teintes choisies pour une distance perceptuelle maximale sur la roue des hues).
 * La couleur est RÉSOLUE au runtime via resolveToken → elle suit la palette active
 * (défaut/Okabe-Ito/Cividis…) exactement comme les autres charts. Ceci REMPLACE
 * l'ancien jeu de hex Okabe FIXES (fragClassColors.ts, supprimé) : plus aucun hex de
 * classe en dur — la source de vérité est le mapping classe→token ci-dessous.
 *
 * Double encodage (P1.2) : la couleur ne porte JAMAIS seule l'information — les
 * charts consommateurs ajoutent label + position (anneau/segment, ligne de rappel,
 * légende). Les rôles (niveau 2) reçoivent des TEINTES ÉCLAIRCIES de la couleur de
 * leur classe (fragRoleColor) : même hue, luminosité ordonnée + label + position.
 *
 * Garde-rail anti-collision et source unique : fragClass.guard.test.ts +
 * fragClass.colorSource.guard.test.ts.
 */
import { resolveToken } from '../resolveToken'
import type { SemanticToken } from '../semantic-tokens'

/**
 * Ordre canonique FIXE des classes (miroir de canonicalFragClassOrder côté Go,
 * §4 du plan). La couleur suit l'ENTITÉ, jamais son rang : ce mapping ne dépend
 * pas des données présentes dans un scope donné.
 */
export const FRAG_CLASS_ORDER = [
  'shoulder',
  'sidearm',
  'heavy',
  'melee',
  'grenade',
  'spartan_ability',
  'unattributed',
] as const

export type FragClassKey = (typeof FRAG_CLASS_ORDER)[number]

/** Classe résidu (teinte neutre de la gamme, jamais une couleur de combat). */
export const FRAG_CLASS_UNATTRIBUTED: FragClassKey = 'unattributed'

/**
 * Mapping classe → TOKEN de la gamme « Antagonistes » (réactif à la palette).
 * 1 token DISTINCT par classe (anti-collision) — même famille de teintes que
 * MatchAntagonistChart. Le résidu prend un neutre divergent.
 *
 * Rappel des teintes sous la palette DÉFAUT (pour lecture) : shoulder=cyan,
 * sidearm=émeraude, heavy=violet, melee=rose, grenade=ambre, spartan=indigo.
 */
export const FRAG_CLASS_TOKENS: Record<FragClassKey, SemanticToken> = {
  shoulder: 'perf-tier-2', // cyan
  sidearm: 'chart-series-6', // émeraude
  heavy: 'narrative-humiliation', // violet
  melee: 'chart-series-8', // rose
  grenade: 'chart-series-7', // ambre
  spartan_ability: 'compare-a', // indigo
  unattributed: 'divergent-neutral', // neutre (résidu)
}

/** Neutre de repli pour une clé inconnue du front (jamais une couleur de combat). */
const FRAG_CLASS_NEUTRAL_TOKEN: SemanticToken = 'divergent-neutral'

/** Token de la gamme Antagonistes pour une classe (fallback neutre si inconnue). */
export function fragClassToken(className: string | null | undefined): SemanticToken {
  if (className != null && className in FRAG_CLASS_TOKENS) {
    return FRAG_CLASS_TOKENS[className as FragClassKey]
  }
  return FRAG_CLASS_NEUTRAL_TOKEN
}

/**
 * Couleur hex RÉSOLUE de la classe dans la palette active (via resolveToken —
 * contexte non-CSS : ECharts canvas et SVG inline, comme MatchAntagonistChart).
 * Réactive au changement de palette au prochain rebuild (useColorPaletteVersion).
 */
export function fragClassColor(className: string | null | undefined): string {
  return resolveToken(fragClassToken(className))
}

// ── Teintes de rôle (P1.2) ──────────────────────────────────────────────────────

const HEX_RE = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i

/** Éclaircissement du 1er rôle (teinte de base déjà un cran plus claire que la classe). */
const ROLE_LIGHTNESS_BASE = 0.22
/** Incrément d'éclaircissement par rôle suivant. */
const ROLE_LIGHTNESS_STEP = 0.2

function parseHex(hex: string): [number, number, number] | null {
  const m = HEX_RE.exec(hex.trim())
  if (!m) return null
  return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)]
}

function toHex(rgb: [number, number, number]): string {
  return (
    '#' +
    rgb
      .map((c) => Math.max(0, Math.min(255, Math.round(c))).toString(16).padStart(2, '0'))
      .join('')
  )
}

/**
 * Mélange un hex vers blanc (t>0) ou noir (t<0) de |t| dans [0,1] — variation de
 * luminosité STRUCTURELLE : le hue reste celui de la classe, seule la clarté change.
 * Renvoie l'entrée telle quelle si non parsable (ex. chaîne vide en SSR).
 */
export function shiftLightness(hex: string, t: number): string {
  const rgb = parseHex(hex)
  if (!rgb) return hex
  const target = t >= 0 ? 255 : 0
  const amt = Math.min(1, Math.abs(t))
  return toHex(rgb.map((c) => c + (target - c) * amt) as [number, number, number])
}

/**
 * Couleur hex d'un rôle (niveau 2) : teinte ÉCLAIRCIE de la couleur de sa classe,
 * de plus en plus claire selon `index` (premier = le plus proche de la classe,
 * dernier = le plus clair). Double encodage : la position (anneau externe) + le
 * label du rôle (ligne de rappel) désambiguïsent au-delà de la teinte.
 */
export function fragRoleColor(className: string | null | undefined, index: number, count: number): string {
  const base = fragClassColor(className)
  if (index < 0) return base
  const t = count <= 1 ? ROLE_LIGHTNESS_BASE : ROLE_LIGHTNESS_BASE + index * ROLE_LIGHTNESS_STEP
  return shiftLightness(base, t)
}

/**
 * Teinte de l'anneau externe d'une classe FEUILLE (poing/grenade/résidu, sans rôle) :
 * léger éclaircissement de la couleur de classe (pas de libellé, seulement la légende).
 */
export function fragLeafColor(className: string | null | undefined): string {
  return shiftLightness(fragClassColor(className), 0.12)
}
