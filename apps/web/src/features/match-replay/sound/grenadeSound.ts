/**
 * grenadeSound.ts — LES DEUX SONS DU MÊME OBJET : le GESTE du lancer, et la DÉTONATION.
 *
 * EXTRAIT DE `replaySound.ts` LE 2026-08-18 (lot R2-G) : ce fichier-ci est né quand
 * l'explosion a cessé d'être une ligne du manifeste pour devenir une règle — une datation,
 * un dédoublonnage, et un silence à tenir. Le manifeste général, les catégories et
 * l'assemblage de la piste restent chez le voisin ; ici vit tout ce qui est propre à la
 * grenade, doctrine comprise. Le garde-rail `replaySoundAssets.guard.test.ts` rejoue les
 * deux tables ci-dessous contre le dossier d'assets, comme les autres.
 *
 * L'EXPLOSION SONNE À CHAQUE FIN DE VOL, PLUS SEULEMENT SUR UN KILL — décision utilisateur
 * du 2026-08-18, mot pour mot : « que ça kill ou pas, elles explosent donc faut jouer le
 * son ». C'est la réponse au grief D2 (« les lancers sont joués mais j'ai pas les
 * explosions »), dont le lot R2-S avait mesuré la cause : l'explosion n'avait qu'UNE source,
 * la vignette d'un kill à la grenade — 16 explosions pour 343 lancers sur les trois témoins,
 * et ZÉRO sur 00162144 qui en porte pourtant 143. Ce n'était pas un masquage, c'était une
 * absence. LES LANCERS RESTENT : la décision 4 du plan R2 les aurait retirés si l'explosion
 * les recouvrait plus d'une fois sur deux — 31,3 % mesurés sur la piste réelle, 48,6 % sur
 * les fins de vol appariées, sous le seuil des deux façons (test S2, replaySound.test.ts).
 *
 * LA DATATION EST CELLE DE L'ÉCRAN, ET IL N'Y EN A QU'UNE : l'instant vient de
 * `buildGrenadeRestFx` (grenadeFx.ts), la fonction qui pose DÉJÀ l'effet visuel de fin de
 * vol — le dernier pas répliqué du projectile lié au lancer. Écrire ici une seconde règle
 * ferait sonner l'explosion à côté de son image le jour où l'une des deux bougerait.
 * Conséquence assumée, qui s'entend : ce point est la DERNIÈRE POSITION CONNUE, pas un
 * impact — pour une frag la réplication cesse ~1,4 s avant la fin de la mèche, et le son est
 * donc en avance sur la détonation réelle, exactement comme l'effet à l'écran. Une grenade
 * dont le film ne lie AUCUN projectile ne sonne que son lancer : rien ne date sa fin.
 *
 * LA DYNAMO SONNE, MÊME SI L'ÉCRAN NE LA FAIT PAS DÉTONER. `restKindOf` lui donne une nappe
 * électrique persistante plutôt qu'une déflagration, parce que c'est ce que l'arme fait dans
 * le jeu ; mais la décharge s'ENTEND, le pack porte `explosion_dynamo`, il sonnait déjà sur
 * un kill à la Dynamo, et la décision du 18/08 ne fait pas d'exception de type. L'écran et la
 * piste disent la même chose de deux façons.
 *
 * UN KILL À LA GRENADE NE SONNE PAS DEUX FOIS. Les deux sources parlent alors du même objet :
 * la vignette du kill, et la fin de vol du projectile. Quand elles tombent à moins de
 * `GRENADE_EXPLOSION_DEDUP_MS` l'une de l'autre AVEC LE MÊME TYPE, c'est une seule détonation
 * entendue deux fois — on garde le KILL et on retire la fin de vol. Pourquoi le kill : il est
 * daté par `alignFeed`, l'horloge qui date déjà le flash de la fiche et l'effet de mort, et
 * c'est le son que l'utilisateur a validé le 16/08. L'appariement est UN POUR UN : un kill
 * n'annule qu'UNE fin de vol, sinon deux grenades du même type lancées ensemble
 * disparaîtraient toutes les deux derrière un seul kill.
 *
 * Pas de React, pas de Web Audio : logique pure, testée (replaySound.test.ts).
 */
import { buildGrenadeRestFx } from '../grenadeFx'
import { frameToMs } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import type { ReplaySoundEvent } from './replaySound'

/**
 * Son de LANCER par rang de grenade (l'index EST le rang, même règle que grenadeLabels :
 * 0 Frag, 1 Plasma, 2 Dynamo, 3 Spike — replay_labels.toml). Un rang hors table = silence.
 */
export const THROW_SOUND_STEMS: readonly string[] = [
  'throw_frag',
  'throw_plasma',
  'throw_dynamo',
  'throw_spike',
]

/**
 * Son d'EXPLOSION par rang de grenade — MÊME INDEX que THROW_SOUND_STEMS, parce que c'est le
 * même objet sous ses deux formes. Un rang hors table = silence : le type n'est pas établi.
 *
 * LES QUATRE STEMS SONT AUSSI DANS `KILL_SPRITE_SOUND_STEMS`, ET CE N'EST PAS UNE COPIE DE
 * TROP : les deux tables ne se joignent pas par la même quantité — celle-ci par le RANG que
 * le film publie avec le lancer (ce qui a été LANCÉ), l'autre par la VIGNETTE que le backend
 * publie avec le kill (ce qui a TUÉ). Les fondre demanderait de traduire l'une en l'autre à
 * chaque lecture, alors que rien ne garantit que les deux chaînes garderont le même ordre.
 * Le garde-rail `replaySoundAssets.guard.test.ts` rejoue justement cette égalité (rang N
 * <-> killfeed-46+N) : le jour où l'une bougerait, l'autre deviendrait fausse EN SILENCE.
 */
export const EXPLOSION_SOUND_STEMS: readonly string[] = [
  'explosion_frag',
  'explosion_plasma',
  'explosion_dynamo',
  'explosion_spike',
]

/**
 * Écart en deçà duquel une explosion de fin de vol et un kill à la grenade DU MÊME TYPE sont
 * la même détonation entendue deux fois (dédoublonnage, décision du 2026-08-18).
 *
 * 0,3 s est la fenêtre de la DÉCISION 4 du plan R2, celle qui servait déjà à juger si un
 * lancer masquait son explosion : en poser une seconde pour la même question ferait deux
 * seuils à tenir. Elle est large devant l'écart mesuré entre les deux horloges (le fil est
 * recalé à ±150 ms de sa médiane, cf. KILL_MATCH_TOL_MS) et courte devant le délai qui
 * sépare deux grenades d'un même joueur.
 */
export const GRENADE_EXPLOSION_DEDUP_MS = 300


/**
 * grenadeSoundEvents — LES DEUX SONS DU MÊME OBJET : le GESTE de chaque lancer, à la frame
 * du lancer, et l'EXPLOSION de chaque fin de vol, à la frame que `buildGrenadeRestFx` a déjà
 * calculée pour l'écran. Un rang hors des quatre types connus ne rend ni l'un ni l'autre.
 *
 * `killExplosions` porte les explosions QUE LES KILLS ONT DÉJÀ PROGRAMMÉES, dans l'ordre où
 * elles l'ont été. Une fin de vol qui retombe sur l'une d'elles — même type, moins de
 * `GRENADE_EXPLOSION_DEDUP_MS` d'écart — est la même détonation : on la retire, et on MARQUE
 * le kill comme consommé pour qu'il n'en annule pas une seconde.
 */
export function grenadeSoundEvents(
  doc: ReplayDocumentReady,
  killExplosions: readonly ReplaySoundEvent[],
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const g of doc.grenades) {
    const stem = THROW_SOUND_STEMS[g.rank ?? -1]
    if (stem) out.push({ ms: frameToMs(g.t, doc), stem })
  }
  const consommes = new Set<number>()
  for (const fx of buildGrenadeRestFx(doc)) {
    const stem = EXPLOSION_SOUND_STEMS[fx.rank]
    if (!stem) continue
    const ms = frameToMs(fx.frame, doc)
    const jumeau = killExplosions.findIndex(
      (k, i) =>
        !consommes.has(i) &&
        k.stem === stem &&
        Math.abs(k.ms - ms) < GRENADE_EXPLOSION_DEDUP_MS,
    )
    if (jumeau >= 0) consommes.add(jumeau)
    else out.push({ ms, stem })
  }
  return out
}

