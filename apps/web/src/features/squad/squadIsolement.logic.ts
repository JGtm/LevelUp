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
 * pointAttenue dit si un point doit se peindre ATTÉNUÉ : son taux d'isolement (le même
 * dénominateur que la taille du point) reste sous le plancher de confiance (30 morts
 * examinées). Ce n'est PAS le plancher de publication (5) — un point publié peut très bien
 * rester atténué. Le rendu de l'atténuation (lot C3, D3 maquette) est un CERCLE POINTILLÉ,
 * pas une opacité réduite — décision de rendu portée par `SquadIsolementNuageCard`, cette
 * fonction ne fait que trancher le booléen.
 */
export function pointAttenue(point: SquadIsolementPoint): boolean {
  return point.part_isolee.echantillon_faible
}

/**
 * TAILLE_SESSION — diamètre (px) d'un point de SESSION, le même pour tous.
 *
 * FIXE, ET C'EST LA CORRECTION DU 2026-09-13. Auparavant il croissait avec les morts
 * examinées (6 + N, plafonné à 30) : sur des données réelles (jusqu'à 118 morts par
 * session), TOUS les points de session saturaient à 30 px et recouvraient les gros points
 * par joueur — l'utilisateur ne voyait plus qu'UNE tache. La taille est l'encodage du
 * GROS point (« combien de morts »), et un seul encodage de taille par graphe : les
 * sessions disent la DISPERSION, pas le volume.
 */
export const TAILLE_SESSION = 7

/**
 * Opacité d'un point de session (maquette 4c520da6 : « opacité ~0,45 ») : toutes les
 * sessions restent visibles sans masquer les repères par joueur.
 */
export const OPACITE_SESSION = 0.45

const TAILLE_MEDIANE_MIN_ECHELLE = 18
const TAILLE_MEDIANE_MAX_ECHELLE = 46

/**
 * tailleMedianeEchelle rend le diamètre (px) du gros point d'un joueur en projetant son
 * TOTAL de morts examinées sur la plage réelle du nuage (`min`..`max` des totaux par
 * joueur), et non sur une constante arbitraire.
 *
 * POURQUOI UNE ÉCHELLE RELATIVE. La version précédente (`16 + total / 2`, plafond 46)
 * saturait dès 60 morts : sur un roster réel (plusieurs centaines de morts par joueur),
 * les trois joueurs sortaient au MÊME diamètre et la taille ne disait plus rien. Projeté
 * sur les extrêmes observés, l'écart redevient lisible — et la légende de taille peut
 * nommer ces deux extrêmes (« N morts » / « M morts »), ce qu'une échelle absolue ne
 * permettait pas.
 *
 * `min === max` (un seul joueur, ou totaux égaux) → la taille médiane de la plage : il n'y
 * a pas d'écart à montrer.
 */
export function tailleMedianeEchelle(total: number, min: number, max: number): number {
  const bas = TAILLE_MEDIANE_MIN_ECHELLE
  const haut = TAILLE_MEDIANE_MAX_ECHELLE
  if (max <= min) return (bas + haut) / 2
  const t = Math.min(1, Math.max(0, (total - min) / (max - min)))
  return bas + t * (haut - bas)
}

export interface PointMedianJoueur {
  /** Médiane de la part de morts isolées (axe X) DU JOUEUR, unité 0..1. */
  isolement: number
  /** Médiane du taux d'échange (axe Y) DU JOUEUR, unité 0..1. */
  couverture: number
  /** SOMME (pas médiane) des morts examinées sur tous les points du joueur — la taille du
   *  gros point doit refléter le poids réel de son échantillon agrégé. */
  mortsExaminees: number
}

/**
 * pointMedianJoueur agrège tous les points d'UN joueur en un point unique : la MÉDIANE de
 * chaque axe (décision D4 — cohérent avec les deux lignes de repère du nuage, qui médianent
 * déjà tous les points), et le TOTAL des morts examinées pour la taille (une somme, pas une
 * médiane : sinon deux joueurs à 3 et 30 sessions produiraient un point de même poids).
 *
 * `null` sans point : rien à agréger, un point à l'origine inventerait une position.
 */
export function pointMedianJoueur(points: SquadIsolementPoint[]): PointMedianJoueur | null {
  if (points.length === 0) return null
  return {
    isolement: mediane(points.map((p) => p.part_isolee.taux)),
    couverture: mediane(points.map((p) => p.couverture.taux)),
    mortsExaminees: points.reduce((sum, p) => sum + p.morts_examinees, 0),
  }
}


/**
 * plafondAxe rend le HAUT d'un axe en pourcents : la plus grande valeur observée, arrondie
 * au multiple de 10 au-dessus, avec un minimum de 20 pour qu'un nuage très tassé garde une
 * échelle lisible.
 *
 * POURQUOI PAS 100 EN DUR. Les deux axes étaient bornés à 0..100 % : sur des données
 * réelles (isolement sous 40 %, couverture sous 50 %), plus de la moitié du canvas restait
 * vide et les trois repères par joueur se chevauchaient dans un mouchoir de poche —
 * l'utilisateur n'y voyait qu'une tache. Le BAS reste ancré à ZÉRO : c'est ce qui garde la
 * comparaison honnête entre joueurs, et ce qu'un axe tronqué ferait mentir.
 */
export function plafondAxe(valeurs: number[], minimum = 20): number {
  if (valeurs.length === 0) return 100
  const max = Math.max(...valeurs) * 100
  return Math.max(minimum, Math.min(100, Math.ceil(max / 10) * 10))
}
