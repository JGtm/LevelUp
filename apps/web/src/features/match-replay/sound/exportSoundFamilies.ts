/**
 * exportSoundFamilies.ts — ce que l'EXPORT hors temps réel reçoit du son de la page, en plus de
 * la piste : les prises de fin de partie qu'il jouera, et le classement voix / musique de ses
 * pistes séparées.
 *
 * EXTRAIT DE `useReplaySound.ts` LE 2026-09-27 (item 7, musique d'intro) : le hook est au-dessus
 * du seuil de taille et ne doit pas grossir. Le lot lui ajoutait une commande ; ces deux fonctions
 * pures, sans React, sont sorties pour compenser. Rien n'a changé dans leur logique, hormis
 * l'entrée de la musique d'intro dans la famille « musique ».
 */
import {
  END_FFA_WIN_VOICE_STEMS,
  END_MUSIC_STEMS,
  END_VOICE_STEMS,
  endMatchSounds,
  type EndMatchSoundSpec,
} from './endMatchSound'
import type { ReplayLocale } from '../i18n/i18n'
import { INTRO_MUSIC_STEM } from './introSound'
import { seededRandom } from './replayAudioMix'
import { ROUND_OVER_SOUND_STEMS } from './roundOverSound'

/**
 * endMatchSoundsFor — les prises de fin QUE L'EXPORT JOUERA, tirage seme.
 *
 * Le chemin temps reel tire au hasard a chaque arrivee en fin ; un fichier, lui, doit sonner
 * pareil a chaque export du meme match (decision D7 du plan d'export).
 */
export function endMatchSoundsFor(spec: EndMatchSoundSpec | null): string[] {
  if (!spec) return []
  return endMatchSounds(spec.outcome, spec.ffa, spec.locale, seededRandom(spec.outcome.length))
}

/**
 * soundFamiliesFor — quels stems sont de la VOIX, quels stems sont de la MUSIQUE.
 *
 * On liste TOUTES les prises possibles, pas seulement celle qui a ete tiree : le classement doit
 * valoir quel que soit le tirage, et un stem de trop dans la liste ne coute rien (rien ne le
 * jouera).
 *
 * LA VOIX DE FIN DE MANCHE EST DE LA VOIX. Elle vit dans la piste, melee aux bruitages, et c'est
 * la seule voix qu'un extrait de milieu de match peut contenir.
 *
 * LA MUSIQUE D'INTRO EST DE LA MUSIQUE, meme si l'export ne la joue pas (decision D-8) : le
 * classement dit ce qu'est un stem, pas s'il sonne dans le clip.
 */
export function soundFamiliesFor(
  spec: EndMatchSoundSpec | null,
  /** Absente (anciens appels), la voix de fin de manche n'est pas dans la piste non plus. */
  locale: ReplayLocale | undefined,
): { voice: readonly string[]; music: readonly string[] } {
  const voice: string[] = locale ? [ROUND_OVER_SOUND_STEMS[locale]] : []
  const music: string[] = [INTRO_MUSIC_STEM]
  if (spec) {
    voice.push(...END_FFA_WIN_VOICE_STEMS[spec.locale])
    for (const parIssue of Object.values(END_VOICE_STEMS)) voice.push(...parIssue[spec.locale])
    music.push(...Object.values(END_MUSIC_STEMS))
  }
  return { voice, music }
}
