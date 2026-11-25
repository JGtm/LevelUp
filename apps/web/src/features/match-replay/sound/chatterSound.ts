/**
 * chatterSound.ts — LA VOIX DES RÉPLIQUES des Spartans, comme événements de la piste sonore.
 *
 * UNE RÉPLIQUE SONNE QUAND ELLE EST AFFICHÉE ET QU'ELLE A UNE VOIX : le même filtre (familles, fréquence) et le même masquage
 * (Seul, Masquer : la voix est à son joueur) que la bulle, le pictogramme et la ligne du fil. Elle suit les règles du son du
 * rejeu — le son doit être activé (coupé par défaut), la voix est une catégorie que le panneau règle (« Voix des
 * répliques »), et au-delà de `SOUND_MAX_SPEED` le curseur avance sans rien jouer.
 *
 * LES VOIX NE SONT PAS TIRÉES AU HASARD : le fichier d'un son est celui de la variante dont le texte s'affiche
 * (`cmd/chatter-build`), donc ce qu'on lit est ce qu'on entend. Aucune variation d'arme (`drawVariation` rend le neutre pour
 * un stem hors de sa table).
 *
 * LA LANGUE EST CELLE DU TEXTE (`model/chatterTable.ts`) : une voix absente dans cette langue n'est jamais remplacée par celle
 * d'une autre, la ligne n'a alors pas de `voice` et rien ne sonne.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ChatterLine } from '../model/chatterLines'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Les événements de voix des répliques données, à l'instant de leur parole. Les répliques sans voix n'en font pas. */
export function chatterVoiceEvents(
  lines: readonly ChatterLine[],
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const l of lines) {
    if (l.voice !== null) out.push({ ...soundEvent(frameToMs(l.frame, doc), l.voice), chatter: true })
  }
  return out
}
