/**
 * introSound.ts — LA MUSIQUE D'INTRO du rejeu (item 7 du backlog, décision D-8 du 2026-09-26).
 *
 * UN EXTRAIT DE TROIS SECONDES, PAS UNE PISTE. La source est la montée du thème multijoueur
 * (`402178411`, `sb_130_mus_multiplayer_global`, 16 s en 4 canaux). L'extrait couvre
 * [R − 1 s ; R + 2 s], où R est la RÉSOLUTION de la montée, sa dernière attaque forte (3,19 s
 * dans la source, mesurée le 2026-09-27). Le préambule du rejeu dure une seconde (`LEAD_IN_MS`) :
 * lancé au départ de la lecture, l'extrait fait donc tomber sa résolution sur le coup d'envoi, à
 * 1×. Recette ffmpeg et mesures au journal du plan `.ai/PLAN_BACKLOG_2026-09-26.md` (lot A4).
 *
 * QUAND ELLE PART, ET QUAND ELLE SE TAIT :
 *  - au DÉPART DEPUIS LE PRÉAMBULE seulement (« Lecture » à l'ouverture, « Recommencer »,
 *    « Lecture » sur un rejeu terminé) — c'est la lecture qui le sait (`useReplayPlayback`,
 *    option `onStarted`). Jamais sur une reprise, un glissé, un saut, un lien tactique ni une
 *    lecture automatique ;
 *  - par la VOIE ORDINAIRE du lecteur (`playFrom`, plafond de voix compris) : son coupé ou
 *    vitesse au-delà de SOUND_MAX_SPEED, elle se tait comme tout le reste ;
 *  - ABSENTE DE L'EXPORT : un clip commence où l'utilisateur le cadre, pas au préambule. Elle
 *    est tout de même rangée en MUSIQUE dans les familles de l'export (`exportSoundFamilies.ts`),
 *    pour que le classement reste vrai le jour où elle y entrerait.
 *
 * Sa durée et son format sont gardés par `replaySoundAssets.guard.test.ts`.
 *
 * UN TAMPON EN RETARD RESTE CALÉ (A4.9, DA-9). Au premier « Lecture » après un rechargement, le
 * lecteur audio naît dans ce clic et l'extrait n'est pas encore décodé : la voie ordinaire le
 * sautait. `playIntroAligned` attend donc le décodage, puis entre dans le tampon avec un
 * DÉCALAGE égal au temps mural écoulé depuis le geste. Le tampon se joue en temps réel, sans
 * accélération, donc ce décalage est aussi celui du tampon, et la résolution reste là où elle
 * serait tombée. Si l'attente dépasse ce qu'il restait du préambule à l'écran
 * (`introLateBoundS`), le coup d'envoi est passé : silence, jamais une montée décalée de son
 * image.
 */
import { LEAD_IN_MS } from '../model/replayWindow'
import type { ReplayAudioPlayer } from './replayAudio'

/** Le fichier de l'extrait, dans `static/sounds/halo_infinite/`. */
export const INTRO_MUSIC_STEM = 'intro_music_01'


/** Le retard admis, en secondes : la durée du préambule À L'ÉCRAN à cette vitesse (1 s à 1×). */
export function introLateBoundS(speed: number): number {
  return LEAD_IN_MS / 1000 / speed
}

/**
 * playIntroAligned joue l'extrait tout de suite s'il est décodé, sinon dès qu'il l'est, entré à
 * l'instant où il aurait dû partir, ou pas du tout au-delà de la borne (cf. l'en-tête).
 * `nowMs` est l'horloge murale, injectée par les tests.
 */
export async function playIntroAligned(
  player: Pick<ReplayAudioPlayer, 'isLoaded' | 'whenLoaded' | 'playFrom'>,
  url: string,
  speed: number,
  nowMs: () => number = () => performance.now(),
): Promise<void> {
  if (player.isLoaded(url)) return player.playFrom(url, 0)
  const geste = nowMs()
  await player.whenLoaded(url)
  const retardS = (nowMs() - geste) / 1000
  if (retardS <= introLateBoundS(speed)) player.playFrom(url, retardS)
}
