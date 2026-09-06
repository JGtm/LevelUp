/**
 * roundOverSound.ts — LE SON « MANCHE TERMINÉE » (voix d'annonceur), daté à la bascule de manche.
 *
 * UN FICHIER À PART, sur le modèle de `zoneSound.ts` et `objectiveSound.ts` : le manifeste général
 * (`replaySound.ts`) est à son plafond de taille, et cette famille a sa source, sa clé et sa
 * doctrine. Le stub qui décrivait ce câblage y vivait déjà (bloc des sons d'objectif) ; il est ici
 * réalisé, à l'identique de ce qu'il annonçait — une table de stem, un `roundOverSoundEvents`.
 *
 * QUAND. À chaque BASCULE de manche (`roundTransitions`, `roundsLogic`) — l'instant exact où
 * l'overlay inter-manche (`ReplayRoundBreakOverlay`) affiche « Manche N terminée ». Les deux
 * partagent la MÊME mesure, par la MÊME garde d'horloge (`scoreTimelineOf`, qui se tait sur un
 * film dont l'origine n'est pas résolue) : un son qui partirait sur l'horloge brute sonnerait à
 * côté de son message. Sur un mode à manche unique, `roundTransitions` est vide : rien ne sonne.
 *
 * ET CETTE BASCULE EST LA FIN DE LA MANCHE DEPUIS LE 2026-08-29, plus le début de la suivante :
 * la voix arrivait 19 à 34 s en retard (mesure des quatre témoins multi-manches, cf. l'en-tête de
 * `roundsLogic`) — l'annonceur disait « manche terminée » alors que la manche d'après était
 * lancée. Rien à changer ici : ce fichier ne date rien lui-même, il lit la borne partagée.
 *
 * LOCALE-AWARE, comme la voix de FIN DE PARTIE (`endMatchSound.ts`) : une VOIX d'annonceur a une
 * langue. Les deux packs (FR et EN) sont extraits du jeu, et la langue jouée est celle de
 * l'INTERFACE (prop `locale` de la page rejeu) — pas celle du jeu au moment du match, que le film
 * ne publie pas. C'est la SEULE entrée locale-aware de la PISTE (les sons de fin de partie ne
 * passent pas par la piste) : d'où le paramètre `locale` ajouté à `buildSoundTimeline`, servi une
 * fois à la construction — le stem est donc figé, pas tiré à la lecture comme une variante.
 *
 * CATÉGORIE OBJECTIF, comme les sons d'état de zone : c'est un son de DÉROULÉ du mode, pas une
 * arme ni un équipement. Le tiroir de réglages le coupe donc avec la case « Objectifs ».
 *
 * Pas de React, pas de Web Audio : logique pure, testée (`roundOverSound.test.ts`). Le garde-rail
 * `replaySoundAssets.guard.test.ts` rejoue ces deux stems contre le dossier d'assets, comme les
 * autres — un stem sans fichier, ou un fichier sans stem, casse le test, jamais l'écoute.
 */
import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import type { ReplayLocale } from '../i18n/i18n'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { roundTransitions } from '../model/roundsLogic'

/**
 * Le stem de la voix « manche terminée », par langue. Extraits du jeu (pack annonceur fourni par
 * l'utilisateur) : FR `257984770`, EN `327120953`, renommés en un stem parlant.
 */
export const ROUND_OVER_SOUND_STEMS: Readonly<Record<ReplayLocale, string>> = {
  fr: 'round_over_fr',
  en: 'round_over_en',
}

/**
 * roundOverSoundEvents — un son par bascule de manche, dans la langue de l'interface.
 *
 * Rend une liste vide quand le mode n'a qu'une manche (aucune bascule), ou quand l'horloge du film
 * n'est pas recalée (garde de `scoreTimelineOf`) — même silence que l'overlay inter-manche.
 */
export function roundOverSoundEvents(
  doc: ReplayDocumentReady,
  locale: ReplayLocale,
): ReplaySoundEvent[] {
  const transitions = roundTransitions(scoreTimelineOf(doc))
  if (transitions.length === 0) return []
  const stem = ROUND_OVER_SOUND_STEMS[locale]
  return transitions.map((tr) => soundEvent(frameToMs(tr.frame, doc), stem))
}
