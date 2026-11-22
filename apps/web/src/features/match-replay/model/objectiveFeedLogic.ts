/**
 * objectiveFeedLogic.ts — LES LIGNES D'OBJECTIF DU FIL, tirées de `doc.objectives[]`. Logique
 * pure, aucune chaîne d'interface (les libellés vivent dans `i18n/`).
 *
 * LA LISTE EST FERMÉE. `OBJECTIVE_FEED_STATS` reprend, une pour une, les statistiques
 * d'objectif que le décodeur nomme (`internal/facts/objectives/named.go` de la brique du décodeur, hors
 * `kills` et `assists`) ; `objectiveFeedStats.guard.test.ts` la compare au fichier Go. Les
 * familles `hill` (colline) et `skull` (crâne) n'ont AUCUNE action nommée dans cette table
 * aujourd'hui : le décodeur n'en publie pas, donc ce fil n'en écrit pas. Le jour où la table
 * en nomme une, le garde-rail rougit tant qu'elle n'a pas son libellé ici.
 *
 * UNE ACTION SANS LIBELLÉ CONNU N'EST JAMAIS AFFICHÉE SOUS UN NOM DE REPLI : elle est écartée
 * et COMPTÉE (`unlabelled`), pour qu'un test de couverture voie le manque au lieu d'un nom
 * inventé. Les frags et les assistances, publiés dans le même calque, ne sont pas des
 * objectifs : ils ne sont ni affichés ni comptés comme manque.
 *
 * DEUX ACTIONS NE FONT PLUS DE LIGNE QUAND LE FIL A DÉJÀ LEUR CONTENU. `flag_capture_assists` se replie
 * sur la ligne de la capture de la MÊME équipe, la plus proche à `CAPTURE_ASSIST_TOL_MS` près (les
 * joueurs crédités rejoignent la capture : `assistXuids`) ; `flag_carriers_killed` se replie sur la
 * ligne de frag du même tueur, hors de ce fichier (`carrierKillFold`), parce qu'elle exige le fil. Une
 * assistance dont l'équipe est inconnue ou sans capture appariée GARDE sa ligne : rien ne se perd.
 *
 * LA GARDE D'HORLOGE EST CELLE DE TOUT LE REJEU : sans `filmClockTrusted`, les instants des
 * objectifs sont décalés d'un écart inconnu (3,6 à 50,8 s mesurés) — pas de ligne, donc pas
 * d'entrée de menu. Muet vaut mieux que faux.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { filmClockTrusted } from '../../../lib/replay/scoreTimeline'

import { eventFeedEntry, type ObjectiveEventLine } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import { objectiveFamilyOf, type ObjectiveFamily } from './objectiveFamilies'

/**
 * Les actions d'objectif nommées par le décodeur, dans l'ordre de `named.go` : drapeau (CTF),
 * zone (Bases), VIP, bombe (Assaut), puis les éliminations et assistances liées à l'objectif.
 */
export const OBJECTIVE_FEED_STATS = [
  'flag_captures',
  'flag_capture_assists',
  'flag_grabs',
  'flag_secures',
  'flag_returns',
  'flag_steals',
  'flag_carriers_killed',
  'zone_captures',
  'zone_secures',
  'vip_selected',
  'bomb_detonations',
  'flag_returners_killed',
  'flag_kills_as_returner',
  'flag_kills_as_carrier',
  'zone_offensive_kills',
  'zone_defensive_kills',
  'vip_kills',
  'vip_assists',
  'vip_kills_as_vip',
] as const

export type ObjectiveFeedStat = (typeof OBJECTIVE_FEED_STATS)[number]

/**
 * Les actions qui se posent SUR l'objectif : une capture, une sécurisation, une prise, un retour, un vol, une
 * détonation. Seules elles ouvrent un anneau sur la zone ou le drapeau (`buildObjectivePulses`) : une élimination
 * liée à l'objectif (les huit noms de `OBJECTIVE_KILL_STATS` et `vip_assists`), une désignation de VIP ou une
 * assistance de capture n'ont rien à annoncer sur l'élément le plus proche de leur auteur. Liste BLANCHE : une action
 * future du décodeur n'ouvre pas d'anneau tant qu'elle n'est pas nommée ici.
 */
export const OBJECTIVE_PULSE_STATS: ReadonlySet<string> = new Set<ObjectiveFeedStat>([
  'flag_captures',
  'flag_grabs',
  'flag_secures',
  'flag_returns',
  'flag_steals',
  'zone_captures',
  'zone_secures',
  'bomb_detonations',
])

/**
 * Les actions du décodeur qui sont une ÉLIMINATION de leur auteur : le frag du fil dit déjà tueur, arme et victime, la
 * ligne d'objectif en serait le doublon. Elles se replient en badge sur le frag du même tueur (`foldCarrierKills`) ;
 * une action sans frag apparié garde sa ligne. `vip_assists` n'y est pas : une assistance n'a pas de frag à elle.
 */
export const OBJECTIVE_KILL_STATS: ReadonlySet<string> = new Set<ObjectiveFeedStat>([
  'flag_carriers_killed',
  'flag_returners_killed',
  'flag_kills_as_returner',
  'flag_kills_as_carrier',
  'zone_offensive_kills',
  'zone_defensive_kills',
  'vip_kills',
  'vip_kills_as_vip',
])

const KNOWN_STATS: ReadonlySet<string> = new Set(OBJECTIVE_FEED_STATS)

/**
 * Tolérance de rattachement d'une assistance de capture à la capture de son équipe, en ms. Le film
 * date les deux sur la même image : la tolérance absorbe l'arrondi d'une image, et reste très en
 * deçà de l'écart qui sépare deux captures d'une même équipe. Entre ces deux ordres de grandeur la
 * distribution mesurée est vide : le seuil est sur un plateau, pas sur une pente.
 */
export const CAPTURE_ASSIST_TOL_MS = 500

/** La famille d'une action de la liste fermée : le préfixe du nom la dit (`flag_`, `zone_`...). */
export function objectiveFamilyOfFeedStat(stat: ObjectiveFeedStat): ObjectiveFamily {
  // Le préfixe est le contrat de nommage (cf. `objectiveFamilies.ts`) : toute action de la liste
  // fermée en porte un, la valeur de repli ne sert qu'au compilateur.
  return objectiveFamilyOf(stat) ?? 'flag'
}

/** Ce que `objectiveFeedEntries` rend : les lignes, et le nombre d'actions écartées faute de libellé. */
export interface ObjectiveFeedResult {
  entries: ReplayFeedEntry[]
  /** Actions d'une famille d'objectif que la liste fermée ne connaît pas : écartées, jamais nommées au hasard. */
  unlabelled: number
}

/**
 * objectiveFeedEntries — une ligne par action d'objectif nommée du document, à son instant sur
 * l'axe du rejeu (`frameToMs`, le même que `nowMs`), triée par instant.
 */
export function objectiveFeedEntries(doc: ReplayDocumentReady): ObjectiveFeedResult {
  if (!filmClockTrusted(doc)) return { entries: [], unlabelled: 0 }
  const entries: ReplayFeedEntry[] = []
  let unlabelled = 0
  doc.objectives.forEach((a, i) => {
    if (objectiveFamilyOf(a.stat) === null) return
    if (!KNOWN_STATS.has(a.stat)) {
      unlabelled++
      return
    }
    entries.push(
      eventFeedEntry(`o-${a.xuid}-${a.stat}-${a.t}-${i}`, frameToMs(a.t, doc), {
        family: 'objectives',
        xuid: a.xuid,
        stat: a.stat as ObjectiveFeedStat,
      }),
    )
  })
  entries.sort((x, y) => x.replayMs - y.replayMs)
  return { entries: foldCaptureAssists(entries, rosterTeamOf(doc)), unlabelled }
}

/** Le camp du FILM d'un joueur (désignateur du roster, `>= 0`), ou `null` quand le film le tait. */
function rosterTeamOf(doc: ReplayDocumentReady): (xuid: string) => number | null {
  const teams = new Map<string, number>()
  for (const r of doc.roster) if (r.xuid && r.team != null && r.team >= 0) teams.set(r.xuid, r.team)
  return (xuid) => teams.get(xuid) ?? null
}

/** L'action nommée d'une ligne d'objectif, ou `null` pour toute autre ligne. */
function statLineOf(entry: ReplayFeedEntry): ObjectiveEventLine | null {
  const event = entry.event
  return event && event.family === 'objectives' && 'stat' in event ? event : null
}

/** L'index de la capture de camp `team` la plus proche de `ms` dans `entries`, parmi `captures`, ou `null`. */
function nearestCapture(
  entries: readonly ReplayFeedEntry[],
  captures: readonly number[],
  teamOf: (xuid: string) => number | null,
  team: number,
  ms: number,
): { index: number; gap: number } | null {
  let best: { index: number; gap: number } | null = null
  for (const index of captures) {
    const capturer = statLineOf(entries[index])
    const gap = Math.abs(entries[index].replayMs - ms)
    if (capturer && teamOf(capturer.xuid) === team && (best === null || gap < best.gap)) best = { index, gap }
  }
  return best
}

/**
 * foldCaptureAssists replie chaque assistance de capture sur la capture de la MÊME équipe la plus
 * proche dans le temps, à `CAPTURE_ASSIST_TOL_MS` près : la ligne d'assistance disparaît et la
 * capture porte les joueurs crédités. Sans équipe connue ou sans capture appariée, la ligne reste.
 * Liste triée en entrée, liste neuve en sortie.
 */
export function foldCaptureAssists(
  entries: readonly ReplayFeedEntry[],
  teamOf: (xuid: string) => number | null,
): ReplayFeedEntry[] {
  const captures = entries.flatMap((e, i) => (statLineOf(e)?.stat === 'flag_captures' ? [i] : []))
  const credited = new Map<number, string[]>()
  const absorbed = new Set<number>()
  entries.forEach((entry, i) => {
    const line = statLineOf(entry)
    const team = line?.stat === 'flag_capture_assists' ? teamOf(line.xuid) : null
    if (!line || team === null) return
    const capture = nearestCapture(entries, captures, teamOf, team, entry.replayMs)
    if (!capture || capture.gap > CAPTURE_ASSIST_TOL_MS) return
    absorbed.add(i)
    const names = credited.get(capture.index) ?? []
    if (!names.includes(line.xuid)) names.push(line.xuid)
    credited.set(capture.index, names)
  })
  return entries.flatMap((entry, i) => {
    const line = statLineOf(entry)
    const assistXuids = credited.get(i)
    if (absorbed.has(i)) return []
    return line && assistXuids ? [{ ...entry, event: { ...line, assistXuids } }] : [entry]
  })
}
