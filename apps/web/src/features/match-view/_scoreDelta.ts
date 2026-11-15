/**
 * _scoreDelta.ts — logique pure de l'ÉCART DE SCORE (`MatchScoreDeltaChart`).
 *
 * CE QUE LA COURBE NE DISAIT PAS D'UN COUP D'OEIL. « Score dans le temps » trace les deux
 * camps l'un contre l'autre ; QUI MÈNE ET DE COMBIEN se lit en comparant deux escaliers. Cet
 * écart le pose en une seule grandeur signée : positif = l'équipe du joueur de la page mène,
 * négatif = l'équipe d'en face.
 *
 * RIEN N'EST RECALCULÉ. Les paliers viennent de `buildScoreCurve` (donc de `teams[].total`
 * du film, le score affiché en jeu) : même horloge du gameplay (`lib/replay/matchClock`), mêmes
 * bornes (coup d'envoi, fin du film), même règle « un camp sans série vaut zéro ». Le MENEUR
 * n'est pas redéfini : l'égalité à zéro et le signe disent ce que dit `leaderStates` (un test
 * de parité le tient) ; les retournements restent ceux de `leadChanges`, que la courbe pose déjà.
 *
 * LE SCORE DU REJEU, MÊME EN ASSASSIN. `header.score_timeline_kind = hidden` efface la COURBE
 * (« Frags cumulés » la remplace) ; il n'efface pas l'écart, qui dit autre chose — l'avance, pas
 * le total. Cette lecture ne consulte donc jamais ce champ (décision D-9).
 *
 * L'ÉCART EST UN ESCALIER, PAS UNE COURBE. Le film ne transmet que les changements : la valeur
 * ATTEND entre deux paliers (`scoreDeltaAt` lit le point tenu). Un mode à peu de points (drapeau,
 * colline, bombe) donne donc quelques marches, et rien n'est interpolé.
 *
 * LE CAMP DU JOUEUR EST UNE CONDITION, PAS UNE SUPPOSITION. Sans exactement une équipe alliée
 * connue, ou avec un camp d'en face d'identité inconnue, ou avec une série du calque sans camp
 * (`teamIdentity` non résolue — le score d'un camp muet serait inventé), il n'y a pas d'écart :
 * `null`, jamais une courbe devinée. Face à PLUS DE DEUX CAMPS, l'écart se prend contre la
 * MEILLEURE équipe adverse à chaque instant (`multiTeam` le dit, pour que l'écran l'écrive).
 *
 * Zéro dépendance React/ECharts : testable en pur (`_scoreDelta.test.ts`).
 */
import { buildScoreCurve, type ScoreCurveInput, type ScoreCurveSeries } from './_scoreCurve'

/** Un instant de l'écart : les deux scores comparés et leur différence signée. */
export interface ScoreDeltaPoint {
  /** Millisecondes depuis le COUP D'ENVOI. */
  ms: number
  /** Score de l'équipe du joueur. */
  ally: number
  /** Score de l'équipe d'en face (la meilleure s'il y en a plusieurs). */
  opponent: number
  /** `ally - opponent` : positif = l'équipe du joueur mène. */
  delta: number
}

export interface ScoreDelta {
  /** Du coup d'envoi à la fin du film ; chaque point tient jusqu'au suivant. */
  points: ScoreDeltaPoint[]
  endMs: number
  allyLabel: string
  /** Le nom du camp d'en face ; `null` quand il y en a plusieurs (cf. `multiTeam`). */
  opponentLabel: string | null
  /** Plus d'un camp en face : l'écart se lit contre la meilleure équipe adverse. */
  multiTeam: boolean
}

/**
 * buildScoreDelta projette la courbe de score en écart signé, ou rend `null` — donc RIEN À
 * L'ÉCRAN — dès que la courbe n'existe pas ou que le camp du joueur n'est pas établi (cf.
 * l'en-tête).
 */
export function buildScoreDelta(input: ScoreCurveInput): ScoreDelta | null {
  const curve = buildScoreCurve(input)
  if (!curve) return null
  // Une série publiée sans camp ne peut être ni comparée ni nommée : se taire, comme le bandeau.
  if (input.timeline?.teams.some((t) => t.teamId == null)) return null
  const allies = curve.series.filter((s) => s.ally === true)
  const others = curve.series.filter((s) => s.ally !== true)
  if (allies.length !== 1 || others.length === 0 || others.some((s) => s.ally === null)) return null
  const ally = allies[0]

  const points = compact(mergedSteps(ally, others))
  return {
    points,
    endMs: curve.endMs,
    allyLabel: ally.label,
    opponentLabel: others.length === 1 ? others[0].label : null,
    multiTeam: others.length > 1,
  }
}

/**
 * mergedSteps pose l'écart à CHAQUE instant où l'un des camps change de palier : l'union des
 * abscisses des séries, la valeur tenue de chacune à cet instant. Les séries sont croissantes en
 * `ms` (bornées au coup d'envoi et à la fin par `buildScoreCurve`) : un pointeur par série suffit.
 */
function mergedSteps(ally: ScoreCurveSeries, others: readonly ScoreCurveSeries[]): ScoreDeltaPoint[] {
  const all = [ally, ...others]
  const xs = [...new Set(all.flatMap((s) => s.points.map(([x]) => x)))].sort((a, b) => a - b)
  const cursor = all.map(() => 0)
  const held = all.map(() => 0)
  return xs.map((x) => {
    all.forEach((s, i) => {
      while (cursor[i] < s.points.length && s.points[cursor[i]][0] <= x) {
        held[i] = s.points[cursor[i]][1]
        cursor[i]++
      }
    })
    const opponent = Math.max(...held.slice(1))
    return { ms: x, ally: held[0], opponent, delta: held[0] - opponent }
  })
}

/**
 * compact retire les points qui ne changent ni le score de l'équipe du joueur ni celui de la
 * meilleure équipe adverse (un camp hors course qui marque) : le premier et le dernier restent,
 * ils bornent l'axe.
 */
function compact(points: ScoreDeltaPoint[]): ScoreDeltaPoint[] {
  return points.filter((p, i) => {
    if (i === 0 || i === points.length - 1) return true
    const prev = points[i - 1]
    return p.ally !== prev.ally || p.opponent !== prev.opponent
  })
}

/**
 * scoreDeltaAt rend le point TENU à `ms` : le dernier au plus tard à cet instant, le premier
 * avant le coup d'envoi, le dernier après la fin. Le score ne vieillit pas — il attend.
 * Dichotomie : l'infobulle suit le curseur au pixel.
 */
export function scoreDeltaAt(delta: ScoreDelta, ms: number): ScoreDeltaPoint {
  const pts = delta.points
  let lo = 0
  let hi = pts.length - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (pts[mid].ms <= ms) lo = mid
    else hi = mid - 1
  }
  return pts[lo]
}
