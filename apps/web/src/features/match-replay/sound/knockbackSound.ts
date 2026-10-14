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
 *    un kill au répulseur DU MÊME AUTEUR (le tueur est l'auteur de la poussée, par son xuid) est
 *    programmé à moins de `KNOCKBACK_KILL_DEDUP_MS`. Une poussée dont l'auteur n'est pas nommé ne
 *    se tait pas : on ne la rattache pas au kill d'un autre.
 */
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { frameToMs } from '../../../lib/replay/replayLogic'
import { buildLivesBySlot, lifeOfSlotAt } from '../model/livesPosition'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le stem de la poussée : celui du kill au répulseur, déjà livré. */
export const KNOCKBACK_SOUND_STEM = 'repulsor_kill'

/** Fenêtre d'un même souffle, par auteur : un usage, un son. */
export const KNOCKBACK_SAME_BLAST_MS = 300

/** Écart sous lequel un kill au répulseur programmé couvre déjà la poussée. Une image et demie. */
export const KNOCKBACK_KILL_DEDUP_MS = 150

/** Un kill au répulseur déjà programmé sur la piste : son instant et son tueur. */
export interface RepulsorKillSound {
  ms: number
  xuid: string
}

/**
 * knockbackSoundEvents — un son par souffle publié, sauf quand le kill au répulseur du même auteur et
 * du même instant le fait déjà entendre. `repulsorKills` sont les kills au répulseur DÉJÀ programmés
 * (la catégorie des armes coupée, il n'y en a aucun, et la poussée sonne seule).
 */
export function knockbackSoundEvents(
  doc: ReplayDocumentReady,
  repulsorKills: readonly RepulsorKillSound[],
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  const lastByAuthor = new Map<string, number>()
  const lives = buildLivesBySlot(doc.tracks)
  const ordered = [...doc.knockbacks].sort((a, b) => a.t - b.t)
  for (const k of ordered) {
    const ms = frameToMs(k.t, doc)
    const author = k.by != null ? `by:${k.by}` : `pousse:${k.slot}`
    const last = lastByAuthor.get(author)
    if (last !== undefined && ms - last < KNOCKBACK_SAME_BLAST_MS) continue
    lastByAuthor.set(author, ms)
    const xuid = k.by != null ? lifeOfSlotAt(lives, k.by, k.t)?.xuid : undefined
    const couverte =
      !!xuid &&
      repulsorKills.some((e) => e.xuid === xuid && Math.abs(e.ms - ms) <= KNOCKBACK_KILL_DEDUP_MS)
    if (!couverte) out.push(soundEvent(ms, KNOCKBACK_SOUND_STEM))
  }
  return out
}
