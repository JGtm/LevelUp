/**
 * objectiveFeedLogic.ts — LES LIGNES D'OBJECTIF DU FIL, tirées de `doc.objectives[]`. Logique
 * pure, aucune chaîne d'interface (les libellés vivent dans `i18n/`).
 *
 * LA LISTE EST FERMÉE. `OBJECTIVE_FEED_STATS` reprend, une pour une, les statistiques
 * d'objectif que le décodeur nomme (`libs/film/internal/facts/objectives/named.go`, hors
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
 * LA GARDE D'HORLOGE EST CELLE DE TOUT LE REJEU : sans `filmClockTrusted`, les instants des
 * objectifs sont décalés d'un écart inconnu (3,6 à 50,8 s mesurés) — pas de ligne, donc pas
 * d'entrée de menu. Muet vaut mieux que faux.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { filmClockTrusted } from '../../../lib/replay/scoreTimeline'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import { objectiveFamilyOf, type ObjectiveFamily } from './objectiveFamilies'

/**
 * Les actions d'objectif nommées par le décodeur, dans l'ordre de `named.go` : drapeau (CTF),
 * zone (Bases), VIP, bombe (Assaut).
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
] as const

export type ObjectiveFeedStat = (typeof OBJECTIVE_FEED_STATS)[number]

const KNOWN_STATS: ReadonlySet<string> = new Set(OBJECTIVE_FEED_STATS)

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
  return { entries, unlabelled }
}
