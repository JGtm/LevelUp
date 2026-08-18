/**
 * _scoreCurve.ts — logique pure de la COURBE DE SCORE du match (`MatchScoreCurveChart`).
 *
 * D'OÙ VIENT LA DONNÉE, ET POURQUOI ELLE NE RESSEMBLE À AUCUNE AUTRE DE CETTE PAGE. Toutes
 * les autres courbes de la vue match se recomposent depuis les événements servis par l'API ;
 * celle-ci vient de l'ARTEFACT DE REJEU, décodé du film Theater (schéma 12, décision D2 du
 * plan d'exploitation du registre : aucune table DuckDB nouvelle). Elle n'existe donc que
 * pour les matchs dont un artefact a été construit — comme le lien « rejeu », et par le
 * même chemin.
 *
 * LA COURBE EST EN ESCALIER, ET CE N'EST PAS UN CHOIX ESTHÉTIQUE. Le film ne transmet que
 * les CHANGEMENTS : entre deux paliers, le score n'évolue pas, il ATTEND. Une interpolation
 * linéaire dessinerait une montée continue là où il y a un palier puis un saut — elle
 * inventerait des valeurs intermédiaires que le match n'a jamais affichées.
 *
 * UN CAMP SANS SÉRIE SE TRACE À PLAT, À ZÉRO. Le film ne publie rien pour une équipe qui
 * n'a jamais marqué (témoin CTF 3-0 : une seule série pour deux camps). Son zéro est une
 * mesure, pas une lacune — et sans sa ligne, un 3-0 se lirait comme un match à un seul
 * participant.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_scoreCurve.test.ts`).
 */
// LES LECTURES DU CALQUE VIENNENT DU REJEU, ET NE SONT PAS RECOPIÉES ICI. `teamSeriesFor`
// (un camp sans série vaut zéro) et `leadChanges` (le meneur unique, l'égalité qui suspend)
// sont déjà écrits et éprouvés dans `match-replay/scoreTimelineLogic.ts` : les réécrire
// serait la deuxième vérité que la règle « ≤ 2 copies » interdit. La dépendance va dans le
// sens de la donnée — l'artefact appartient au rejeu, cette page ne fait que le lire.
import type { ReplayScoreTimelineReady } from '@/features/match-replay/replayNormalize'
import { leadChanges, teamSeriesFor } from '@/features/match-replay/scoreTimelineLogic'
// L'instant en M:SS a UN foyer dans le dépôt (`lib/formatters`) : la carte et la frise du
// rejeu l'appellent toutes deux, aucune ne le réécrit.
export { formatClockMMSS as formatClock } from '@/lib/formatters'

/** Une équipe et sa courbe, prête à poser dans ECharts. */
export interface ScoreCurveSeries {
  teamId: number
  /** Camp du point de vue du joueur de la page ; `null` = inconnu (encre neutre). */
  ally: boolean | null
  label: string
  /** Paliers `[ms depuis le début du rejeu, valeur]`, bornés au début et à la fin du match. */
  points: Array<[number, number]>
  /** Le film publie-t-il une série pour ce camp ? `false` = ligne plate MESURÉE à zéro. */
  published: boolean
  /** Valeur finale — celle que le jeu affiche à l'écran de fin. */
  final: number
}

/** Un retournement, daté en millisecondes de rejeu. */
export interface ScoreCurveLead {
  ms: number
  teamId: number
}

export interface ScoreCurve {
  series: ScoreCurveSeries[]
  leads: ScoreCurveLead[]
  /** Durée couverte par la courbe, en ms — l'axe des temps s'y arrête. */
  durationMs: number
}

export interface ScoreCurveInput {
  timeline: ReplayScoreTimelineReady | undefined
  /** Durée d'une image du document. Absente = pas d'échelle temporelle, pas de courbe. */
  frameIntervalMs: number | undefined
  frameCount: number
  /** Les camps à tracer, dans l'ordre d'affichage (scoreboard d'abord, film ensuite). */
  teamIds: number[]
  allyOf: (teamId: number) => boolean | null
  labelOf: (teamId: number) => string
}

/**
 * buildScoreCurve projette le calque de score en courbes ECharts.
 *
 * Rend `null` — donc RIEN À L'ÉCRAN — dès qu'une des conditions du tracé manque : pas de
 * calque, pas d'échelle temporelle, moins de deux camps, ou aucun camp publié. Le plan
 * l'exige explicitement : « affiche seulement si l'artefact existe, sinon rien, pas de
 * placeholder ». Un cadre vide est une promesse non tenue ; l'absence n'en fait aucune.
 */
export function buildScoreCurve(input: ScoreCurveInput): ScoreCurve | null {
  const { timeline, frameIntervalMs, frameCount, teamIds, allyOf, labelOf } = input
  if (!timeline || !frameIntervalMs || frameCount <= 1 || teamIds.length < 2) return null
  const durationMs = (frameCount - 1) * frameIntervalMs
  const series = teamIds.map<ScoreCurveSeries>((teamId) => {
    const team = teamSeriesFor(timeline, teamId)
    const paliers = team?.total ?? []
    return {
      teamId,
      ally: allyOf(teamId),
      label: labelOf(teamId),
      points: stepPoints(paliers, frameIntervalMs, durationMs),
      published: team !== null,
      final: paliers.length > 0 ? paliers[paliers.length - 1].v : 0,
    }
  })
  if (!series.some((s) => s.published)) return null
  return {
    series,
    leads: leadChanges(timeline).map((c) => ({ ms: c.frame * frameIntervalMs, teamId: c.teamId })),
    durationMs,
  }
}

/**
 * stepPoints borne la série aux deux extrémités du match.
 *
 * LES DEUX BORNES SONT DES MESURES, PAS DU REMPLISSAGE. Au coup d'envoi le score EST nul :
 * sans le point de départ, la courbe commencerait au premier but et laisserait croire que
 * le match a démarré là. À l'arrivée, le dernier palier tient jusqu'à la fin : sans le point
 * final, l'escalier s'arrêterait au dernier changement et la courbe paraîtrait plus courte
 * que le match — d'autant plus qu'un dernier but tombe rarement à la dernière seconde.
 */
function stepPoints(
  paliers: ReadonlyArray<{ t: number; v: number }>,
  frameIntervalMs: number,
  durationMs: number,
): Array<[number, number]> {
  const out: Array<[number, number]> = []
  if (paliers.length === 0 || paliers[0].t > 0) out.push([0, 0])
  for (const p of paliers) out.push([p.t * frameIntervalMs, p.v])
  const last = out[out.length - 1]
  if (last[0] < durationMs) out.push([durationMs, last[1]])
  return out
}

/**
 * teamIdsOf rend les camps à tracer, dans un ordre stable.
 *
 * LE SCOREBOARD EST LA SOURCE, LE FILM LE COMPLÉMENT. Les camps du match sont ceux du
 * scoreboard — c'est lui qui sait combien d'équipes ont joué, y compris celles qui n'ont
 * jamais marqué et dont le film ne dit donc rien. Un camp que seul le film connaît est
 * ajouté quand même : mieux vaut une courbe de plus qu'un but qui n'appartient à personne.
 */
export function teamIdsOf(
  scoreboardTeamIds: readonly (number | null)[],
  timeline: ReplayScoreTimelineReady | undefined,
): number[] {
  const ids = new Set<number>()
  for (const id of scoreboardTeamIds) if (id != null) ids.add(id)
  for (const team of timeline?.teams ?? []) if (team.teamId != null) ids.add(team.teamId)
  return [...ids].sort((a, b) => a - b)
}
