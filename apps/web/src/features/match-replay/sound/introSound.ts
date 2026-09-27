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
 *  - par la VOIE ORDINAIRE du lecteur (`play`) : son coupé ou vitesse au-delà de
 *    SOUND_MAX_SPEED, elle se tait comme tout le reste ;
 *  - ABSENTE DE L'EXPORT : un clip commence où l'utilisateur le cadre, pas au préambule. Elle
 *    est tout de même rangée en MUSIQUE dans les familles de l'export (`exportSoundFamilies.ts`),
 *    pour que le classement reste vrai le jour où elle y entrerait.
 *
 * Sa durée et son format sont gardés par `replaySoundAssets.guard.test.ts`.
 */

/** Le fichier de l'extrait, dans `static/sounds/halo_infinite/`. */
export const INTRO_MUSIC_STEM = 'intro_music_01'
