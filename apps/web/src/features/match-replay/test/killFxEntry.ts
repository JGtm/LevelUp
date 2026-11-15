/**
 * killFxEntry.ts — UN EFFET DE MORT DE TEST, avec ses défauts : le tueur du slot 3 et sa victime du slot 4,
 * chacun lu dans sa vie. Un test ne nomme que ce qui l'intéresse (`killFxEntry({ slot: null })`).
 */
import type { KillFxEntry } from '../model/killFx'

export function killFxEntry(over: Partial<KillFxEntry> = {}): KillFxEntry {
  return {
    frame: 0, x: 5, y: 5, vx: null, vy: null, deathX: 15, deathY: 15,
    dist: null, fam: 'plain', seed: 7,
    slot: 3, slotFrame: 0, killerXuid: 'K',
    victimSlot: 4, victimFrame: 0, victimXuid: 'V',
    ...over,
  }
}
