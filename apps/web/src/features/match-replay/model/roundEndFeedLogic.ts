/**
 * roundEndFeedLogic.ts — LA LIGNE « FIN DE MANCHE » DU FIL, avec le score de manche de chaque camp.
 *
 * SEULEMENT DANS UN MODE À PLUSIEURS MANCHES (`roundEnds` en compte au moins deux : un mode à manche
 * unique n'a pas de fin de manche à dire, la fin du match a son écran), et SEULEMENT SI L'HORLOGE EST
 * SÛRE (`scoreTimelineOf` porte la garde du rejeu : sans origine, le calque de score est décalé d'un
 * écart inconnu, et une ligne datée de travers se lirait comme juste).
 *
 * CE QUE LA LIGNE DIT : le score DANS la manche de chaque camp, à la clôture de la manche — jamais le
 * cumul du match ni le camp qui le mène. Les camps sont ceux du film, donnés par l'appelant (le
 * roster, pas le calque : un camp qui n'a pas marqué n'a pas de série et vaut zéro). Moins de deux
 * camps connus, aucune ligne : une fin de manche à un seul score ne dit rien.
 *
 * LA DERNIÈRE MANCHE A SA LIGNE (la fin du match est aussi la fin de sa manche) : seule la voix et
 * l'écran de fin la portent ailleurs, le fil ne double aucune annonce.
 */
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { scoreTimelineOf, teamRoundScoreAtFrame } from '../../../lib/replay/scoreTimeline'

import { eventFeedEntry } from './feedEventLines'
import type { ReplayFeedEntry } from './killFeedLogic'
import { roundEnds } from './roundsLogic'

/** Une ligne par manche jouée, à son instant de clôture sur l'axe du rejeu ; vide hors du contrat ci-dessus. */
export function roundEndFeedEntries(
  doc: ReplayDocumentReady,
  teamIds: readonly number[],
): ReplayFeedEntry[] {
  const timeline = scoreTimelineOf(doc)
  const ends = roundEnds(timeline)
  if (!timeline || ends.length < 2 || teamIds.length < 2) return []
  return ends.map((end) =>
    eventFeedEntry(`r-${end.round}`, frameToMs(end.frame, doc), {
      family: 'objectives',
      kind: 'roundEnd',
      index: end.index,
      scores: teamIds.map((teamId) => ({
        teamId,
        score: teamRoundScoreAtFrame(timeline, teamId, end.round, end.frame),
      })),
    }),
  )
}
