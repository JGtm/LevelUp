/**
 * abilityImpulseSound.ts — L'USAGE D'UNE CAPACITÉ QUI POUSSE SON PORTEUR, daté par le film.
 *
 * LA SOURCE est le document (schéma 38) : `abilityImpulses`, une entrée PLATE par geste —
 * (t, slot, family) — lue dans le corps `tag == 1` des composants i57/i59 (le même canal dont
 * le tag 3 porte le grappin) et ATTRIBUÉE par le rang de capacité de la MÊME VIE. Aucun seuil
 * de vitesse, aucune heuristique : on joue ce que le calque date.
 *
 * LA VALIDATION EST UNE VÉRITÉ TERRAIN, pas une mesure interne : sur le film `1cd3848a`,
 * l'utilisateur a relevé au Theater ses cinq usages de propulseur (1:51, 1:54, 2:03, 2:05,
 * 2:14) et la chaîne rend 1:52, 1:55, 2:03, 2:05, 2:15 — précision 5/5, rappel 5/5.
 *
 * ## Pourquoi une TABLE PAR FAMILLE et non un stem unique
 *
 * Le grappin sonne par un stem nu (`GRAPPLE_SOUND_STEM`) parce que son calque ne porte QUE
 * des tractions. Celui-ci porte une `family` : il est ouvert à toute capacité que le titre
 * déclarera mesurée. Une famille absente de la table reste MUETTE — jamais le son d'une
 * voisine, même règle que `EQUIPMENT_PLACEMENT_SOUND_STEMS`.
 *
 * LE RÉPULSEUR N'Y ENTRERA PAS PAR CE CANAL : il n'est pas dans les impulsions de capacité
 * (négatif mesuré sur neuf canaux, lots R8/R9 du 2026-09-03). Il sonne déjà par SON KILL
 * (`repulsor_kill`), et c'est tout ce que le film en date.
 *
 * CATÉGORIE ÉQUIPEMENT — le propulseur est une capacité d'équipement du joueur, au même titre
 * que le grappin, le camouflage et le surbouclier (cf. `buildSoundTimeline`).
 */
import type { ReplayDocumentReady } from '../replayNormalize'
import { frameToMs } from '../replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/**
 * Les stems d'impulsion, PAR FAMILLE du document.
 *
 * `thruster` : banque `sb_007_abl_evade`, événement `307114b6` =
 * `play_007_abl_evade_blast_player` — nom cassé par gabarits sur un dictionnaire de
 * 138 886 jetons (espérance de collision 0,00091, seuil 0,10), et refermé par la paire
 * complémentaire `_player` / `_nonplayer` sur le même radical. La perspective du PORTEUR
 * (stéréo, non spatialisée) est celle que le rejeu sert pour toutes les capacités.
 * Trois variantes : le jeu en tire une à chaque dash (`replaySoundVariants.ts`).
 */
export const ABILITY_IMPULSE_SOUND_STEMS: Readonly<Record<string, string>> = {
  thruster: 'thruster_activate',
}

/** abilityImpulseSoundEvents — un son par impulsion publiée dont la famille a un stem. */
export function abilityImpulseSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const imp of doc.abilityImpulses) {
    const stem = ABILITY_IMPULSE_SOUND_STEMS[imp.family]
    if (!stem) continue
    out.push(soundEvent(frameToMs(imp.t, doc), stem))
  }
  return out
}
