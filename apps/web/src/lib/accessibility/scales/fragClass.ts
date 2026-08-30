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
  'vehicle',
  'turret',
  'equipment',
  'environmental',
  'unattributed',
] as const

export type FragClassKey = (typeof FRAG_CLASS_ORDER)[number]

/** Classe résidu (teinte neutre de la gamme, jamais une couleur de combat). */
export const FRAG_CLASS_UNATTRIBUTED: FragClassKey = 'unattributed'

/**
 * Mapping classe → TOKEN de la famille DÉDIÉE `frag-*` (2026-08-29).
 *
 * HISTORIQUE : les classes empruntaient des tokens d'autres gammes (perf-tier,
 * narrative, chart-series, extreme…), choisis pour leur distance perceptuelle sur
 * la palette DÉFAUT. Mais les palettes daltoniennes replient plusieurs de ces
 * tokens sur la MÊME teinte : sous Okabe-Ito, lourde ≡ grenade ≡ équipement
 * (Reddish Purple) et épaule ≡ environnement (Sky Blue) ; sous Cividis, lourde ≡
 * capacité spartan et épaule ≡ environnement — collisions EXACTES (ΔE = 0),
 * mesurées le 2026-08-29. Une famille dédiée permet d'accorder chaque palette
 * indépendamment, sans toucher aux tokens partagés par le reste de l'app.
 *
 * Les valeurs de la palette DÉFAUT reprennent À L'IDENTIQUE celles des anciens
 * tokens (zéro churn visuel), à UNE exception : frag-spartan-ability passe
 * d'indigo-400 à indigo-500 — la paire héritée avec unattributed (ΔE 6,89 < 8)
 * est RÉSOLUE par la refonte au lieu d'être exemptée au garde-rail.
 * Garde-rail par palette (défaut/Okabe ≥ 8, Cividis ≥ 5 documenté) :
 * fragClass.guard.test.ts.
 */
export const FRAG_CLASS_TOKENS: Record<FragClassKey, SemanticToken> = {
  shoulder: 'frag-shoulder', // cyan
  sidearm: 'frag-sidearm', // émeraude
  heavy: 'frag-heavy', // violet
  melee: 'frag-melee', // rose
  grenade: 'frag-grenade', // ambre
  spartan_ability: 'frag-spartan-ability', // indigo
  vehicle: 'frag-vehicle', // indigo profond
  turret: 'frag-turret', // orange brûlé
  equipment: 'frag-equipment', // fuchsia — hors famille bleue (décision 2026-08-29, lot A.5)
  environmental: 'frag-environmental', // bleu profond
  unattributed: 'frag-unattributed', // neutre (résidu)
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
/** Incrément NOMINAL d'éclaircissement par rôle suivant (tant qu'il tient sous le plafond). */
const ROLE_LIGHTNESS_STEP = 0.2
/**
 * PLAFOND d'éclaircissement du DERNIER rôle d'une classe. Sans lui, la rampe
 * `0.22 + index × 0.2` atteignait 1,02 dès l'index 4 : shiftLightness clampe l'amplitude
 * à 1, donc les 5ᵉ et 6ᵉ rôles d'une même classe (cas RÉEL : les 5 types de grenade, les
 * engins d'une classe véhicule, les objets d'une classe équipement) sortaient en BLANC
 * PUR — invisibles sur le fond de carte, et indiscernables entre eux. À 0,7 il reste 30 %
 * de la teinte de classe : l'arc garde sa hue, se distingue du blanc et de son voisin.
 */
const ROLE_LIGHTNESS_MAX = 0.7

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
 *
 * La rampe est NORMALISÉE sur `count` : le pas nominal tant qu'il tient sous
 * ROLE_LIGHTNESS_MAX, sinon un pas resserré qui place le DERNIER rôle exactement au
 * plafond. Les classes à 1-3 rôles (mêlée, capacités spartanes) gardent donc leur rendu
 * historique ; seules les classes à 4 rôles et plus sont recomprimées — c'est-à-dire
 * exactement celles qui déteignaient en blanc.
 */
export function fragRoleColor(className: string | null | undefined, index: number, count: number): string {
  const base = fragClassColor(className)
  if (index < 0) return base
  // `index + 1` : un index hors bornes (count sous-évalué par un appelant) reste borné
  // par le plafond au lieu de repartir vers le blanc.
  const n = Math.max(1, count, index + 1)
  const step = n <= 1 ? 0 : Math.min(ROLE_LIGHTNESS_STEP, (ROLE_LIGHTNESS_MAX - ROLE_LIGHTNESS_BASE) / (n - 1))
  return shiftLightness(base, ROLE_LIGHTNESS_BASE + index * step)
}

/**
 * Teinte de l'anneau externe d'une classe FEUILLE (poing/grenade/résidu, sans rôle) :
 * léger éclaircissement de la couleur de classe (pas de libellé, seulement la légende).
 */
export function fragLeafColor(className: string | null | undefined): string {
  return shiftLightness(fragClassColor(className), 0.12)
}
