/**
 * knockbackSound.ts — LA POUSSÉE DU RÉPULSEUR, entendue à chaque fois que le film l'écrit.
 *
 * LA SOURCE est le document (schéma 94) : `knockbacks`, une entrée par joueur poussé, datée par la
 * paire de messages du film (`EquipmentObjectKnockedBack` puis `EquipmentKnockbackPlayer`). Le son
 * est CELUI DU RÉPULSEUR DÉJÀ LIVRÉ (`repulsor_kill`, ses trois variantes) : décision de
 * l'utilisateur, « le répulseur est déjà bon » — aucun son nouveau.
 *
 * DEUX DÉDOUBLONNAGES, et ils ne décident d'aucun fait (la poussée est publiée, on choisit seulement
 * combien de fois la faire entendre) :
 *  - UN SOUFFLE, UN SON. Un même usage pousse souvent plusieurs joueurs, écrits dans des paquets
 *    voisins : les poussées d'un même auteur à moins de `KNOCKBACK_SAME_BLAST_MS` de la précédente
 *    ne rejouent pas le son.
 *  - LE KILL SONNE DÉJÀ. Une poussée qui tue fait entendre le son du kill au répulseur
 *    (`KILL_SPRITE_SOUND_STEMS`, vignette `killfeed-56`) au même instant : la poussée se tait quand
 *    un tel kill est programmé à moins de `KNOCKBACK_KILL_DEDUP_MS`.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le stem de la poussée : celui du kill au répulseur, déjà livré. */
export const KNOCKBACK_SOUND_STEM = 'repulsor_kill'

/** Fenêtre d'un même souffle, par auteur : un usage, un son. */
export const KNOCKBACK_SAME_BLAST_MS = 300

/** Écart sous lequel un kill au répulseur programmé couvre déjà la poussée. Une image et demie. */
export const KNOCKBACK_KILL_DEDUP_MS = 150

/**
 * knockbackSoundEvents — un son par souffle publié, sauf quand le kill au répulseur du même instant
 * le fait déjà entendre. `repulsorKills` sont les sons de kill au répulseur DÉJÀ programmés (la
 * catégorie des armes coupée, il n'y en a aucun, et la poussée sonne seule).
 */
export function knockbackSoundEvents(
  doc: ReplayDocumentReady,
  repulsorKills: readonly ReplaySoundEvent[],
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  const lastByAuthor = new Map<string, number>()
  const ordered = [...doc.knockbacks].sort((a, b) => a.t - b.t)
  for (const k of ordered) {
    const ms = frameToMs(k.t, doc)
    const author = k.by != null ? `by:${k.by}` : `pousse:${k.slot}`
    const last = lastByAuthor.get(author)
    if (last !== undefined && ms - last < KNOCKBACK_SAME_BLAST_MS) continue
    lastByAuthor.set(author, ms)
    if (repulsorKills.some((e) => Math.abs(e.ms - ms) <= KNOCKBACK_KILL_DEDUP_MS)) continue
    out.push(soundEvent(ms, KNOCKBACK_SOUND_STEM))
  }
  return out
}
