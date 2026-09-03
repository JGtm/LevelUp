/**
 * _scoreEvents.ts — logique pure des POINTS MARQUÉS DANS LE TEMPS (`MatchScoreEventsChart`).
 *
 * POURQUOI CE CALCUL EXISTE À CÔTÉ DE LA COURBE, ET NE LA REMPLACE PAS. Les deux lisent le
 * MÊME calque de score du film (`lib/replay/scoreTimeline`, schéma 12) et n'ajoutent aucune
 * requête. Ce qu'ils en font diffère parce que les modes diffèrent : en Oddball ou en
 * Bastion, le score monte en continu et la courbe en escalier est la bonne lecture ; en
 * capture de drapeau, en roi de la colline et en assaut, un match entier tient en TROIS à
 * CINQ paliers — une courbe y est un escalier vide, et ce sont les INSTANTS de marque qui
 * portent toute l'information. La règle qui dit lequel des deux s'affiche vit en DONNÉE
 * (`regulation.toml [score_timeline]`), jamais ici.
 *
 * UN PALIER EST UNE MARQUE, ET SON DELTA EST CE QUI A ÉTÉ MARQUÉ. Le film ne transmet que
 * les CHANGEMENTS : chaque `{t, v}` de `teams[].total` est donc l'instant où le compteur a
 * bougé, et la valeur marquée est l'écart avec le palier précédent (le premier se lit depuis
 * zéro — le match commence à 0-0, c'est une mesure et non un remplissage).
 *
 * UN DELTA NUL OU NÉGATIF N'EST PAS UNE MARQUE. Le calque peut republier la même valeur
 * (recalage) ; une barre de hauteur zéro se lirait comme un point marqué qui n'existe pas.
 * Ces paliers sont écartés, et leur écart reste porté par le palier suivant puisque le
 * CUMUL, lui, est toujours celui du film.
 *
 * UN CAMP SANS SÉRIE N'A AUCUNE MARQUE, ET C'EST UNE MESURE (témoin CTF 3-0 : le film ne
 * publie qu'une série pour deux camps). Il garde sa place dans la liste — avec sa couleur et
 * son nom — pour que la légende dise « ce camp n'a pas marqué » plutôt que de le faire
 * disparaître du graphe.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_scoreEvents.test.ts`).
 */
// MÊME LECTURE DU CALQUE QUE LA COURBE, jamais une deuxième : `teamSeriesFor` (un camp sans
// série vaut zéro) est écrit et éprouvé dans `lib/replay/scoreTimeline`, partagé par le
// rejeu 2D et la vue match (ratchet P8.5 : la logique commune vit dans `lib/`).
import { teamSeriesFor, type ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

/** Une MARQUE : l'instant où un camp a pris des points, et le compteur juste après. */
export interface ScoreEvent {
  /** Instant de la marque, en millisecondes depuis le début du rejeu. */
  ms: number
  /** Points pris à cet instant (l'écart avec le palier précédent). */
  points: number
  /** Score CUMULÉ du camp juste après cette marque — la valeur publiée par le film. */
  total: number
}

/** Un camp, ses marques, et de quoi le nommer et le colorer. */
export interface ScoreEventsTeam {
  teamId: number
  /** Camp du point de vue du joueur de la page ; `null` = inconnu (encre neutre). */
  ally: boolean | null
  label: string
  events: ScoreEvent[]
  /** Le film publie-t-il une série pour ce camp ? `false` = camp MESURÉ sans aucune marque. */
  published: boolean
}

export interface ScoreEvents {
  teams: ScoreEventsTeam[]
  /** Durée couverte par le graphe, en ms — l'axe des temps s'y arrête. */
  durationMs: number
}

export interface ScoreEventsInput {
  timeline: ReplayScoreTimelineReady | undefined
  /** Durée d'une image du document. Absente = pas d'échelle temporelle, pas de graphe. */
  frameIntervalMs: number | undefined
  frameCount: number
  /** Les camps à tracer, dans l'ordre d'affichage (scoreboard d'abord, film ensuite). */
  teamIds: number[]
  allyOf: (teamId: number) => boolean | null
  labelOf: (teamId: number) => string
}

/**
 * buildScoreEvents projette le calque de score en marques datées.
 *
 * Rend `null` — donc RIEN À L'ÉCRAN — dès qu'une condition du tracé manque : pas de calque,
 * pas d'échelle temporelle, moins de deux camps, aucun camp publié, ou AUCUNE marque à
 * poser. Mêmes portes que la courbe (`buildScoreCurve`), plus la dernière, qui lui est
 * propre : une courbe sans palier reste une paire de lignes plates lisibles, un graphe de
 * barres sans barre est un cadre vide — et un cadre vide est une promesse non tenue.
 */
export function buildScoreEvents(input: ScoreEventsInput): ScoreEvents | null {
  const { timeline, frameIntervalMs, frameCount, teamIds, allyOf, labelOf } = input
  if (!timeline || !frameIntervalMs || frameCount <= 1 || teamIds.length < 2) return null
  const durationMs = (frameCount - 1) * frameIntervalMs
  const teams = teamIds.map<ScoreEventsTeam>((teamId) => {
    const team = teamSeriesFor(timeline, teamId)
    return {
      teamId,
      ally: allyOf(teamId),
      label: labelOf(teamId),
      events: markedEvents(team?.total ?? [], frameIntervalMs),
      published: team !== null,
    }
  })
  if (!teams.some((t) => t.published)) return null
  if (!teams.some((t) => t.events.length > 0)) return null
  return { teams, durationMs }
}

/**
 * markedEvents transforme les paliers d'un camp en marques datées.
 *
 * LE PREMIER PALIER SE LIT DEPUIS ZÉRO : le match commence à 0-0, la première capture vaut
 * donc sa valeur publiée entière. Les paliers qui ne font pas monter le compteur sont
 * écartés (cf. l'en-tête) — le cumul reporté reste celui du film, jamais un recalcul.
 */
function markedEvents(
  paliers: ReadonlyArray<{ t: number; v: number }>,
  frameIntervalMs: number,
): ScoreEvent[] {
  const out: ScoreEvent[] = []
  let previous = 0
  for (const p of paliers) {
    const points = p.v - previous
    previous = p.v
    if (points <= 0) continue
    out.push({ ms: p.t * frameIntervalMs, points, total: p.v })
  }
  return out
}
