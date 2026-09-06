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
  hinf_heatwave: { pitch_cents: { bas: -48, haut: 43 } },
  hinf_ma40_ar: { volume_db: { bas: -3, haut: 0 }, pitch_cents: { bas: -85, haut: 80 } },
  hinf_ma5k_avenger: { volume_db: { bas: -3, haut: 0 }, pitch_cents: { bas: -85, haut: 80 } },
  hinf_mangler: { pitch_cents: { bas: -48, haut: 43 } },
  hinf_needler: { pitch_cents: { bas: -50, haut: 50 } },
  hinf_plasma_pistol: { pitch_cents: { bas: -55, haut: 76 } },
  hinf_shock_rifle: { pitch_cents: { bas: -48, haut: 43 } },
  hinf_stalker_rifle: { pitch_cents: { bas: -48, haut: 43 } },
  hinf_vk78_commando: { pitch_cents: { bas: 0, haut: 800 } },
}
