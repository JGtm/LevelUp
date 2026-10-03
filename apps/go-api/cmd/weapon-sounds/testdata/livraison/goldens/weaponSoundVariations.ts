/**
 * weaponSoundVariations.ts — GENERE PAR `_outils/livraison.py` (archive Desktop du
 * chantier sons-armes), NE PAS EDITER A LA MAIN : toute reprise rejoue la recette
 * (`.ai/V7.5/RECETTE_SONS_ARMES.md`) et reecrit ce fichier avec les sons.
 *
 * Les fourchettes RANGED extraites des banks Wwise du jeu, par stem d'arme : ce que le
 * moteur du jeu fait varier a CHAQUE lecture (volume en dB, hauteur en centiemes).
 * Une arme absente d'ici se joue pure — c'est le cas nominal, pas une erreur.
 */
import type { SoundVariation } from './weaponSoundLogic'

export const WEAPON_SOUND_VARIATIONS: Readonly<Record<string, SoundVariation>> = {
  hinf_ma40_ar: { volume_db: { bas: -1, haut: 1 } },
  hinf_mangler: { pitch_cents: { bas: -85, haut: 80 } },
  hinf_needler: { volume_db: { bas: 0, haut: 1.5 }, pitch_cents: { bas: 1234567.5, haut: 1e-05 } },
  hinf_ravager: { volume_db: { bas: -2, haut: 2 }, pitch_cents: { bas: -50, haut: 50 } },
  hinf_relatifdrive: { volume_db: { bas: -4.25, haut: 0 } },
  hinf_sentinel_beam: { volume_db: { bas: -3.0, haut: 2.0 }, pitch_cents: { bas: 0.0, haut: 1000.0 } },
}
