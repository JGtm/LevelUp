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
 * L'AXE DES TEMPS EST CELUI DU GAMEPLAY, ET C'EST UNE CORRECTION (registre 2026-09-05,
 * P0-7). Jusqu'ici cette courbe posait ses paliers en `frame × frameIntervalMs`, c'est-à-dire
 * en millisecondes depuis le PREMIER PAQUET DE POSITION du film — 3,6 à 50,8 s avant le coup
 * d'envoi selon le match — et l'étiquette « 0m00s » se lisait juste sous « Frags cumulés »,
 * dont le zéro est le coup d'envoi. Deux instants nommés pareil, distants de −24 à +4,5 s
 * selon le témoin. La conversion vit désormais dans `lib/replay/matchClock` et RIEN ne se
 * date ici sans passer par elle : sans horloge établie (artefact sans origine), il n'y a pas
 * de courbe — un axe faux se lit comme juste.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_scoreCurve.test.ts`).
 */
// LES LECTURES DU CALQUE NE SONT PAS RECOPIÉES ICI. `teamSeriesFor` (un camp sans série
// vaut zéro) et `leadChanges` (le meneur unique, l'égalité qui suspend) sont déjà écrits et
// éprouvés dans `lib/replay/scoreTimeline` : les réécrire serait la deuxième vérité que la
// règle « ≤ 2 copies » interdit. Le rejeu 2D lit le MÊME module — la logique partagée par
// deux features vit dans `lib/`, jamais dans l'une que l'autre irait importer (ratchet P8.5).
import type { MatchClock } from '@/lib/replay/matchClock'
import {
  leadChanges,
  teamSeriesFor,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
// L'instant en M:SS a UN foyer dans le dépôt (`lib/formatters`) : la carte et la frise du
// rejeu l'appellent toutes deux, aucune ne le réécrit.
export { formatClockMMSS as formatClock } from '@/lib/formatters'

/** Une équipe et sa courbe, prête à poser dans ECharts. */
export interface ScoreCurveSeries {
  teamId: number
  /** Camp du point de vue du joueur de la page ; `null` = inconnu (encre neutre). */
  ally: boolean | null
  label: string
  /**
   * Paliers `[ms depuis le COUP D'ENVOI, valeur]`, bornés au coup d'envoi et à la fin du
   * film. Même axe que `event_time_ms`, donc que « Frags cumulés » juste au-dessus.
   */
  points: Array<[number, number]>
  /** Le film publie-t-il une série pour ce camp ? `false` = ligne plate MESURÉE à zéro. */
  published: boolean
  /** Valeur finale — celle que le jeu affiche à l'écran de fin. */
  final: number
}

/** Un retournement, daté en millisecondes depuis le coup d'envoi. */
export interface ScoreCurveLead {
  ms: number
  teamId: number
}

export interface ScoreCurve {
  series: ScoreCurveSeries[]
  leads: ScoreCurveLead[]
  /**
   * Fin de l'axe des temps, en ms depuis le coup d'envoi : la dernière image du film, lue
   * sur l'horloge du gameplay. L'axe part de 0 (le coup d'envoi) et s'y arrête.
   */
  endMs: number
}

export interface ScoreCurveInput {
  timeline: ReplayScoreTimelineReady | undefined
  /**
   * L'horloge du match (`lib/replay/matchClock`). `null` = origine du film non établie :
   * aucun palier ne peut être daté sur l'horloge du gameplay, donc pas de courbe.
   */
  clock: MatchClock | null | undefined
  /** Les camps à tracer, dans l'ordre d'affichage (scoreboard d'abord, film ensuite). */
  teamIds: number[]
  allyOf: (teamId: number) => boolean | null
  labelOf: (teamId: number) => string
}

/**
 * buildScoreCurve projette le calque de score en courbes ECharts.
 *
 * Rend `null` — donc RIEN À L'ÉCRAN — dès qu'une des conditions du tracé manque : pas de
 * calque, pas d'horloge établie, moins de deux camps, aucun camp publié, ou un film qui
 * s'arrête avant le coup d'envoi. Le plan l'exige explicitement : « affiche seulement si
 * l'artefact existe, sinon rien, pas de placeholder ». Un cadre vide est une promesse non
 * tenue ; l'absence n'en fait aucune.
 */
export function buildScoreCurve(input: ScoreCurveInput): ScoreCurve | null {
  const { timeline, clock, teamIds, allyOf, labelOf } = input
  if (!timeline || !clock || teamIds.length < 2) return null
  const endMs = clock.gameplayMsOfFrame(clock.frameCount - 1)
  if (endMs <= 0) return null
  const series = teamIds.map<ScoreCurveSeries>((teamId) => {
    const team = teamSeriesFor(timeline, teamId)
    const paliers = team?.total ?? []
    return {
      teamId,
      ally: allyOf(teamId),
      label: labelOf(teamId),
      points: stepPoints(paliers, clock, endMs),
      published: team !== null,
      final: paliers.length > 0 ? paliers[paliers.length - 1].v : 0,
    }
  })
  if (!series.some((s) => s.published)) return null
  return {
    series,
    leads: leadChanges(timeline).map((c) => ({
      ms: clock.gameplayMsOfFrame(c.frame),
      teamId: c.teamId,
    })),
    endMs,
  }
}

/**
 * stepPoints borne la série aux deux extrémités du MATCH, sur l'horloge du gameplay.
 *
 * LES DEUX BORNES SONT DES MESURES, PAS DU REMPLISSAGE. Au coup d'envoi le score EST celui
 * du dernier palier déjà passé — nul dans tous les cas ordinaires : sans le point de départ,
 * la courbe commencerait au premier but et laisserait croire que le match a démarré là. À
 * l'arrivée, le dernier palier tient jusqu'à la fin : sans le point final, l'escalier
 * s'arrêterait au dernier changement et la courbe paraîtrait plus courte que le match —
 * d'autant plus qu'un dernier but tombe rarement à la dernière seconde.
 *
 * CE QUI PRÉCÈDE LE COUP D'ENVOI SE REPLIE SUR LUI. Le film commence avant le match (le
 * countdown) : un palier daté avant 0 sur l'horloge du gameplay n'est pas un but d'avant le
 * match, c'est l'état initial du compteur. Il donne donc sa valeur au point de départ plutôt
 * qu'un point à abscisse négative que l'axe couperait sans le dire.
 */
function stepPoints(
  paliers: ReadonlyArray<{ t: number; v: number }>,
  clock: MatchClock,
  endMs: number,
): Array<[number, number]> {
  let atKickoff = 0
  let i = 0
  while (i < paliers.length && clock.gameplayMsOfFrame(paliers[i].t) <= 0) {
    atKickoff = paliers[i].v
    i++
  }
  const out: Array<[number, number]> = [[0, atKickoff]]
  for (; i < paliers.length; i++) out.push([clock.gameplayMsOfFrame(paliers[i].t), paliers[i].v])
  const last = out[out.length - 1]
  if (last[0] < endMs) out.push([endMs, last[1]])
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
