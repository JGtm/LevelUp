/**
 * fragClass.ts — Échelle couleur des CLASSES d'arme du sunburst « Répartition des
 * frags » v2 (classe→rôle). SOURCE UNIQUE réutilisée par toutes les surfaces
 * (Synthesis, Match view, Timeseries, Sessions, Escouade) — cf.
 * .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P1.
 *
 * Couleurs INDÉPENDANTES DE LA PALETTE ACTIVE : les classes de frags ont leur PROPRE
 * jeu de hex fixes CVD-safe (fragClassColors.ts, teintes Okabe-Ito), sur le modèle
 * de l'exception des couleurs de rareté. Motif : sous la palette DÉFAUT, les tokens
 * chart-series-1..5 forment une rampe indigo (all-pairs ΔE ~5.4, SOUS le plancher
 * CVD) → 5 classes indistinguables. Sortir de la palette active garantit la
 * distinction quelle que soit la palette (all-pairs ΔE 11.0 en deutan, > cible 8 du
 * validateur ; normal-vision 15.6). « Non attribué » = neutre rendu HACHURÉ côté chart.
 *
 * Double encodage (P1.2) : la couleur ne porte JAMAIS seule l'information — les
 * charts consommateurs ajoutent label + position (anneau/segment). Les rôles
 * (niveau 2) reçoivent des TEINTES de luminosité de la couleur de leur classe
 * (fragRoleColor) : même hue, luminosité ordonnée + label + position.
 *
 * Garde-rail anti-collision et anti-recopie : fragClass.guard.test.ts.
 */
import { fragClassFixedHex } from './fragClassColors'

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

/** Classe résidu (rendue hachurée par les charts, jamais une couleur pleine). */
export const FRAG_CLASS_UNATTRIBUTED: FragClassKey = 'unattributed'

/**
 * Couleur hex FIXE de la classe (hors palette active). Client ET SSR (aucun accès
 * DOM). Les hex vivent uniquement dans fragClassColors.ts (précédent rarity.ts).
 */
export function fragClassColor(className: string | null | undefined): string {
  return fragClassFixedHex(className)
}

// ── Teintes de rôle (P1.2) ──────────────────────────────────────────────────────

const HEX_RE = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i

/** Amplitude max de variation de luminosité entre le rôle le plus clair et le plus foncé. */
const ROLE_LIGHTNESS_SPREAD = 0.32

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
 * luminosité STRUCTURELLE (comme hexToRgba pour l'alpha) : le hue reste celui de
 * la classe, seule la clarté change. Renvoie l'entrée telle quelle si non parsable.
 */
export function shiftLightness(hex: string, t: number): string {
  const rgb = parseHex(hex)
  if (!rgb) return hex
  const target = t >= 0 ? 255 : 0
  const amt = Math.abs(t)
  return toHex(rgb.map((c) => c + (target - c) * amt) as [number, number, number])
}

/**
 * Couleur hex d'un rôle (niveau 2) : teinte de luminosité de la couleur de sa
 * classe, ordonnée par `index` sur `count` rôles (premier = plus clair, dernier =
 * plus foncé). Un rôle unique (count<=1) garde la couleur de classe. Double
 * encodage : la position + le label du rôle désambiguïsent au-delà de la teinte.
 */
export function fragRoleColor(className: string | null | undefined, index: number, count: number): string {
  const base = fragClassColor(className)
  if (count <= 1 || index < 0) return base
  // frac ∈ [-0.5, 0.5] centré → t ∈ [-spread/2, +spread/2].
  const frac = index / (count - 1) - 0.5
  return shiftLightness(base, -frac * ROLE_LIGHTNESS_SPREAD)
}
