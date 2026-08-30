/**
 * firstBloodLanes — modèle PUR du chart « Premier frag / première mort » (lanes).
 *
 * Aucune dépendance React ni ECharts : agrégation (médianes, écart), tri des
 * lignes, formatage des durées et géométrie de la bande. Le rendu vit dans
 * `FirstBloodLanes.tsx` ; ce module est testé seul (firstBloodLanes.test.ts).
 *
 * Vocabulaire :
 *   - « lane » = une bande horizontale = un joueur.
 *   - « écart / fenêtre d'avance » = médiane(première mort) − médiane(premier frag).
 *     Positif = le joueur frappe avant de tomber (avance) ; négatif = l'inverse.
 *
 * Règles de calcul (spec produit) :
 *   - Un événement absent (`null`) est EXCLU des points et des médianes, mais le
 *     match reste compté dans le total affiché (« 11/12 matchs »).
 *   - Médiane = quantile 0.5 par interpolation linéaire sur les valeurs non nulles
 *     triées, arrondie à la seconde.
 *   - Lignes triées par médiane(premier frag) croissante — le plus rapide en haut ;
 *     les joueurs sans premier frag exploitable ferment la marche.
 */

// ── Géométrie (px) — partagée par le builder d'option et le calcul de hauteur ──

/** Hauteur d'une bande joueur. */
export const LANE_HEIGHT = 54
/** Décalage vertical des nuages de points au-dessus / en dessous de la ligne. */
export const POINT_OFFSET = 14
/** Hauteur de la barre « fenêtre d'avance ». */
export const GAP_BAR_HEIGHT = 8
/** Largeur de la colonne de libellés à gauche du tracé. */
export const LABEL_WIDTH = 130
/** Marges du `grid` ECharts. */
export const GRID_TOP = 8
export const GRID_BOTTOM = 28

/** Fenêtre temporelle par défaut de l'axe X (secondes). */
export const DEFAULT_MAX_SEC = 300

// ── Entrées ───────────────────────────────────────────────────────────────────

export interface FirstBloodMatch {
  matchId: string
  /** Secondes depuis le début du match, `null` si le joueur n'a pas frag. */
  firstKillSec: number | null
  /** Secondes depuis le début du match, `null` si le joueur n'est pas mort. */
  firstDeathSec: number | null
  /** Libellé de carte résolu — absent si le titre/le match n'en expose pas
   *  (dégradation tooltip, DEC-4 : jamais l'uuid). */
  mapUI?: string
  /** Libellé de mode résolu — absent si le titre/le match n'en expose pas. */
  modeUI?: string
  /** Date de début du match, ISO 8601 (déjà en UTC canonique côté API — ne pas
   *  recalculer, seulement formater à l'affichage). */
  startTime?: string
}

export interface FirstBloodPlayerSeries {
  player: string
  matches: FirstBloodMatch[]
}

// ── Sorties ───────────────────────────────────────────────────────────────────

export interface FirstBloodEventPoint {
  matchId: string
  sec: number
  /** Reprises de FirstBloodMatch — alimentent le tooltip du nuage (DEC-4). */
  mapUI?: string
  modeUI?: string
  startTime?: string
}

export interface FirstBloodLane {
  player: string
  /** Nombre total de matchs du joueur (dénominateur des « 11/12 matchs »). */
  totalMatches: number
  /** Premiers frags exploitables (non nuls, finis). */
  kills: FirstBloodEventPoint[]
  /** Premières morts exploitables (non nulles, finies). */
  deaths: FirstBloodEventPoint[]
  /** Médiane des premiers frags, arrondie à la seconde. */
  medianKillSec: number | null
  /** Médiane des premières morts, arrondie à la seconde. */
  medianDeathSec: number | null
  /** medianDeathSec − medianKillSec ; `null` si l'une des deux manque. */
  gapSec: number | null
}

// ── Calcul ────────────────────────────────────────────────────────────────────

/**
 * Quantile par interpolation linéaire sur un tableau DÉJÀ trié croissant.
 * Renvoie `null` sur tableau vide. Convention identique à numpy/percentile
 * (méthode « linear ») : position = (n − 1) × q.
 */
export function quantileSorted(sorted: number[], q: number): number | null {
  if (sorted.length === 0) return null
  if (sorted.length === 1) return sorted[0]
  const pos = (sorted.length - 1) * q
  const lo = Math.floor(pos)
  const hi = Math.ceil(pos)
  if (lo === hi) return sorted[lo]
  return sorted[lo] + (sorted[hi] - sorted[lo]) * (pos - lo)
}

/** Médiane (quantile 0.5) arrondie à la seconde. Valeurs non finies ignorées. */
export function medianSeconds(values: number[]): number | null {
  const clean = values.filter((v) => Number.isFinite(v)).sort((a, b) => a - b)
  const q = quantileSorted(clean, 0.5)
  return q == null ? null : Math.round(q)
}

/** Un événement est exploitable s'il est renseigné ET fini (garde-fou API). */
function isUsable(sec: number | null | undefined): sec is number {
  return sec != null && Number.isFinite(sec)
}

/**
 * Projette les séries joueur en lanes prêtes à dessiner, triées par médiane du
 * premier frag croissante (le plus rapide en haut). Les lanes sans médiane de
 * premier frag sont reléguées en fin de liste (ordre d'entrée conservé entre
 * elles : `Array.prototype.sort` est stable).
 */
export function buildFirstBloodLanes(data: FirstBloodPlayerSeries[]): FirstBloodLane[] {
  const lanes: FirstBloodLane[] = data.map((s) => {
    const matches = s.matches ?? []
    const kills: FirstBloodEventPoint[] = []
    const deaths: FirstBloodEventPoint[] = []
    for (const m of matches) {
      // Métadonnées d'affichage communes aux deux points (frag/mort) d'un même
      // match — DEC-4 : alimentent le tooltip (carte · mode · date), jamais l'uuid.
      const meta = { mapUI: m.mapUI, modeUI: m.modeUI, startTime: m.startTime }
      if (isUsable(m.firstKillSec)) kills.push({ matchId: m.matchId, sec: m.firstKillSec, ...meta })
      if (isUsable(m.firstDeathSec)) deaths.push({ matchId: m.matchId, sec: m.firstDeathSec, ...meta })
    }
    const medianKillSec = medianSeconds(kills.map((p) => p.sec))
    const medianDeathSec = medianSeconds(deaths.map((p) => p.sec))
    return {
      player: s.player,
      totalMatches: matches.length,
      kills,
      deaths,
      medianKillSec,
      medianDeathSec,
      gapSec:
        medianKillSec != null && medianDeathSec != null ? medianDeathSec - medianKillSec : null,
    }
  })

  return lanes.sort((a, b) => {
    if (a.medianKillSec == null && b.medianKillSec == null) return 0
    if (a.medianKillSec == null) return 1
    if (b.medianKillSec == null) return -1
    return a.medianKillSec - b.medianKillSec
  })
}

/** Hauteur totale du chart pour N lanes (bandes + marges du grid). */
export function firstBloodLanesHeight(laneCount: number): number {
  return Math.max(1, laneCount) * LANE_HEIGHT + GRID_TOP + GRID_BOTTOM
}

// ── Formatage ─────────────────────────────────────────────────────────────────

/** Signe typographique du moins (U+2212) — jamais le tiret d'union ASCII. */
const MINUS = '−'

/**
 * Durée absolue : « 45s » sous la minute, « 1m05 » au-delà (secondes sur 2
 * chiffres). Format numérique volontairement identique FR/EN (m/s sont des
 * symboles d'unité, pas des mots à traduire).
 */
export function formatLaneSeconds(sec: number): string {
  const total = Math.max(0, Math.round(sec))
  if (total < 60) return `${total}s`
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}m${String(s).padStart(2, '0')}`
}

/** Écart signé : « +14s », « −1m05 ». Zéro compte comme une avance (« +0s »). */
export function formatGapSeconds(gapSec: number): string {
  const sign = gapSec < 0 ? MINUS : '+'
  return `${sign}${formatLaneSeconds(Math.abs(gapSec))}`
}

/** Étiquette de tick de l'axe X : « 0 », « 1m », « 2m »… */
export function formatAxisTick(sec: number): string {
  if (sec === 0) return '0'
  return `${Math.round(sec / 60)}m`
}

/**
 * Neutralise les métacaractères du texte enrichi ECharts (`{`, `}`, `|`) et les
 * retours à la ligne dans une valeur non constante (gamertag) : un `|` dans un
 * pseudo casserait le parsing `{style|contenu}` de l'axe.
 */
export function sanitizeRichText(s: string): string {
  return s.replace(/[{}|\r\n]/g, ' ')
}
