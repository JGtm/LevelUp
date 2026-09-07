/**
 * squadIsolement.logic — les décisions PURES du nuage « isolement x couverture »
 * (item 7.7 du plan tactique : `SquadEchange.nuage_isolement`).
 *
 * Aucune dépendance React/ECharts : le quadrant d'un point, les médianes du nuage et la
 * règle d'atténuation d'un échantillon faible — les trois décisions qu'un test doit
 * pouvoir mettre en défaut sans monter de graphe. `SquadIsolementNuageCard` ne fait que
 * peindre ce que ce module décide.
 */
import type { SquadIsolementPoint } from '@/lib/api/types'

/**
 * Plancher de PUBLICATION d'une session : miroir de `domain.PlancherMortsSessionIsolement`
 * (5 morts examinées) côté Go. Le serveur applique déjà ce plancher — aucune session en
 * dessous n'est jamais servie — cette constante ne sert qu'au pied de carte, qui doit
 * NOMMER le seuil plutôt que le recopier en dur.
 */
export const PLANCHER_MORTS_SESSION = 5

/**
 * Opacité d'un point ATTÉNUÉ (échantillon sous 30 morts, `part_isolee.echantillon_faible`).
 * Même convention que `HistogramChart` (`ATTENUATION_OPACITE`) — reprise ici en dur parce
 * que cette constante-là est privée au wrapper et ce fichier ne l'importe pas.
 */
export const OPACITE_ATTENUEE = 0.35
/** Opacité d'un point à échantillon suffisant. */
export const OPACITE_PLEINE = 0.8

export type QuadrantIsolement = 'procheCouvert' | 'loinCouvert' | 'procheSeul' | 'loinSansSecours'

export interface MedianesNuage {
  /** Médiane de la part de morts isolées (axe X), en unité 0..1. */
  isolement: number
  /** Médiane du taux d'échange (axe Y), en unité 0..1. */
  couverture: number
}

function mediane(valeurs: number[]): number {
  const tri = [...valeurs].sort((a, b) => a - b)
  const milieu = Math.floor(tri.length / 2)
  return tri.length % 2 === 0 ? (tri[milieu - 1] + tri[milieu]) / 2 : tri[milieu]
}

/**
 * medianesNuage calcule les deux lignes de référence tracées sur le nuage : la médiane de
 * la part isolée et celle de la couverture, sur TOUS les points publiés (tous joueurs, toutes
 * sessions confondus — une seule paire de lignes pour tout le graphe).
 *
 * `null` sans point : il n'y a rien à tracer, et une valeur à 0 inventerait une référence.
 */
export function medianesNuage(points: SquadIsolementPoint[]): MedianesNuage | null {
  if (points.length === 0) return null
  return {
    isolement: mediane(points.map((p) => p.part_isolee.taux)),
    couverture: mediane(points.map((p) => p.couverture.taux)),
  }
}

/**
 * quadrantDuPoint range un point dans l'un des quatre quadrants nommés, par comparaison
 * aux médianes du nuage : « proche » exige une part isolée STRICTEMENT sous la médiane,
 * « couvert » accepte l'égalité. Un point EXACTEMENT sur les deux médianes n'entre donc
 * jamais dans le quadrant d'alerte (`loinSansSecours`) — arbitrage conservateur : un
 * point moyen ne désigne personne comme « seul et sans secours ».
 */
export function quadrantDuPoint(
  point: SquadIsolementPoint,
  medianes: MedianesNuage,
): QuadrantIsolement {
  const proche = point.part_isolee.taux < medianes.isolement
  const couvert = point.couverture.taux >= medianes.couverture
  if (proche && couvert) return 'procheCouvert'
  if (!proche && couvert) return 'loinCouvert'
  if (proche && !couvert) return 'procheSeul'
  return 'loinSansSecours'
}

/**
 * pointAttenue dit si un point doit se peindre en opacité réduite : son taux d'isolement
 * (le même dénominateur que la taille du point) reste sous le plancher de confiance (30
 * morts examinées). Ce n'est PAS le plancher de publication (5) — un point publié peut
 * très bien rester atténué.
 */
export function pointAttenue(point: SquadIsolementPoint): boolean {
  return point.part_isolee.echantillon_faible
}

/** Opacité d'un point : atténuée sous le plancher de confiance, pleine au-dessus. */
export function opaciteDuPoint(point: SquadIsolementPoint): number {
  return pointAttenue(point) ? OPACITE_ATTENUEE : OPACITE_PLEINE
}

const TAILLE_MIN = 6
const TAILLE_MAX = 30

/**
 * tailleDuPoint rend le rayon du marqueur (px, `symbolSize` ECharts) : croît avec le
 * nombre de morts examinées, plafonné pour qu'une session très chargée n'écrase pas les
 * autres points du nuage.
 */
export function tailleDuPoint(mortsExaminees: number): number {
  return Math.min(TAILLE_MAX, TAILLE_MIN + mortsExaminees)
}
