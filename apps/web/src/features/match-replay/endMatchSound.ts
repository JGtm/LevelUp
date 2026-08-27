/**
 * endMatchSound.ts — CE QUI SONNE QUAND LE MATCH SE TERMINE : la voix de l'annonceur et la
 * fanfare, ensemble, une fois.
 *
 * UN FICHIER À PART, SUR LE MODÈLE DE `grenadeSound.ts` (lot R2-G) : le manifeste général
 * (`replaySound.ts`) porte ce qui sonne PENDANT le match, événement par événement, sur
 * l'horloge du film. La fin de partie n'est pas un événement du film — c'est la CONCLUSION,
 * elle ne se date pas, elle se déclenche à l'arrivée de la lecture sur la borne de fin
 * (`useReplayPlayback`), et elle dépend de deux quantités que rien d'autre dans le son ne
 * regarde : l'ISSUE du joueur de la page et la LANGUE de l'interface. Le voisin est par
 * ailleurs à son plafond de taille.
 *
 * PREMIÈRE ENTRÉE LOCALE-AWARE DU CATALOGUE SONORE. Un tir, une explosion, une pose n'ont pas
 * de langue ; une VOIX D'ANNONCEUR en a une. Les deux packs (FR et EN) sont extraits du jeu,
 * et la langue jouée est celle de l'INTERFACE (prop `locale` de la page rejeu) — pas celle du
 * jeu au moment du match, que le film ne publie pas.
 *
 * DEUX SONS PARTENT ENSEMBLE, ET C'EST LA MISE EN SCÈNE DU JEU (décision utilisateur du
 * 2026-08-27) : la réplique de l'annonceur par-dessus la fanfare. Ce sont deux voix du
 * lecteur, jamais un fichier pré-mixé — l'utilisateur garde son volume, et une re-livraison
 * d'un seul des deux ne demande pas de re-mixer l'autre.
 *
 * PLUSIEURS PRISES, TIRAGE AU SORT (règle du vote : « prises multiples -> tirage aleatoire »).
 * L'aléa est INJECTÉ (`rand`), comme partout ailleurs dans le son du rejeu (patron
 * `drawVariation` / `weaponSoundVariations.ts`) : les tests fixent la prise, la production
 * passe `Math.random`. Le FR a deux prises pour la victoire, la défaite et l'égalité ; l'EN
 * n'en a qu'une pour la victoire et l'égalité — le pack en porte deux pour la seule défaite.
 *
 * LE FFA N'A PAS D'ÉCRAN, MAIS IL A UN SON (décision D-C3). L'écran de fin exige DEUX camps
 * (`victoryLogic.ts`, décision D-B1) : sur un match sans équipes il ne se rend pas, et c'est
 * juste — il annoncerait un affrontement qui n'a pas eu lieu. Une VOIX, elle, n'a pas ce
 * problème : « Vainqueur » ne nomme personne. Le FFA GAGNÉ sonne donc, avec la fanfare de
 * victoire ; le FFA perdu et l'égalité en FFA restent muets, faute d'une réplique qui dise
 * cela sans nommer d'équipe.
 *
 * LE REPLI EN ANGLAIS EST DOCUMENTÉ, PAS SILENCIEUX : le pack annonceur EN ne porte aucun
 * « Winner » isolé (relevé à la transcription du 2026-08-27, `vote_fin_partie.json` :
 * `winner_ffa: "A_IDENTIFIER"`). Le FFA gagné en anglais joue donc « Victory », qui dit vrai
 * — l'utilisateur a bien gagné — au lieu de rester muet sur la seule langue.
 *
 * L'ISSUE VIENT DE LA MÊME LECTURE QUE L'ÉCRAN, et il n'y en a qu'une : `readVictory`
 * (`victoryLogic.ts`) pour les matchs à deux camps, `outcomeCodeToValue` (`lib/outcome.ts`,
 * source unique du dépôt) pour le reste. Re-décoder `outcome_code` ici ferait deux vérités,
 * et le jour où l'une bougerait l'écran annoncerait une victoire pendant que le son sonnerait
 * une défaite.
 *
 * Pas de React, pas de Web Audio : logique pure, testée (endMatchSound.test.ts). Le câblage
 * vit dans `useReplaySound`, la lecture dans `replayAudio.ts`. Le garde-rail
 * `replaySoundAssets.guard.test.ts` rejoue les trois tables ci-dessous contre le dossier
 * d'assets, comme les autres.
 */
import { outcomeCodeToValue } from '@/lib/outcome'
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ReplayLocale } from './i18n'
import { readVictory, type VictoryOutcome } from './victoryLogic'

/**
 * Les PRISES de la voix d'annonceur, par issue puis par langue. Une liste à plusieurs entrées
 * = plusieurs enregistrements du même mot dans le jeu, entre lesquels on tire (cf. l'en-tête).
 *
 * Transcriptions relevées à l'extraction (2026-08-27) : FR « Victoire » (×2), « Défaite »
 * (×2), « Égalité » / « À égalité » ; EN « Victory », « Defeat » (×2), « Game tied ».
 */
export const END_VOICE_STEMS: Readonly<
  Record<VictoryOutcome, Readonly<Record<ReplayLocale, readonly string[]>>>
> = {
  win: {
    fr: ['end_victory_voice_fr_01', 'end_victory_voice_fr_02'],
    en: ['end_victory_voice_en_01'],
  },
  loss: {
    fr: ['end_defeat_voice_fr_01', 'end_defeat_voice_fr_02'],
    en: ['end_defeat_voice_en_01', 'end_defeat_voice_en_02'],
  },
  tie: {
    fr: ['end_tie_voice_fr_01', 'end_tie_voice_fr_02'],
    en: ['end_tie_voice_en_01'],
  },
}

/**
 * La FANFARE par issue — une seule par issue, et sans langue : une musique ne parle pas.
 * Elles sont livrées à -18 LUFS quand les voix le sont à -16 (décision utilisateur du
 * 2026-08-27, « audible sans hurler ») : la réplique doit passer AU-DESSUS de la fanfare,
 * c'est cet écart de 2 LU qui l'y met, pas un réglage du lecteur.
 */
export const END_MUSIC_STEMS: Readonly<Record<VictoryOutcome, string>> = {
  win: 'end_victory_music_01',
  loss: 'end_defeat_music_01',
  tie: 'end_tie_music_01',
}

/**
 * La voix du FFA GAGNÉ, par langue — « Vainqueur », la réplique qui ne nomme aucune équipe.
 * L'anglais retombe sur « Victory » : le pack EN n'a pas de « Winner » isolé (cf. l'en-tête).
 */
export const END_FFA_WIN_VOICE_STEMS: Readonly<Record<ReplayLocale, readonly string[]>> = {
  fr: ['end_winner_voice_fr_01'],
  en: ['end_victory_voice_en_01'],
}

/**
 * Ce qu'il faut savoir d'un match pour le conclure en son : l'issue DU JOUEUR DE LA PAGE, la
 * langue de l'interface, et si le match opposait deux camps identifiés.
 */
export interface EndMatchSoundSpec {
  outcome: VictoryOutcome
  /**
   * `true` = le match n'oppose PAS deux camps identifiés — FFA, plus de deux camps, ou joueur
   * de la page introuvable au scoreboard. Le nom vient du cas qui le peuple en pratique ; ce
   * que le drapeau dit vraiment, c'est « aucune équipe à nommer », et la voix choisie
   * (« Vainqueur ») est justement celle qui n'en nomme aucune.
   */
  ffa: boolean
  locale: ReplayLocale
}

/**
 * endMatchSoundSpec lit la fin d'un match du point de vue du SON, en s'appuyant sur la lecture
 * de l'écran (`readVictory`) — jamais sur un second décodage de `outcome_code`.
 *
 * `null` = rien ne sonne : match non conclu (en-tête pas encore chargé, code hors contrat),
 * abandon, ou fin sans équipes que la voix ne saurait annoncer (FFA perdu ou à égalité).
 */
export function endMatchSoundSpec(
  scoreboard: ReadonlyArray<Pick<MatchScoreboardRow, 'team_side' | 'is_me'>>,
  outcomeCode: number | null | undefined,
  locale: ReplayLocale,
): EndMatchSoundSpec | null {
  const reading = readVictory(scoreboard, outcomeCode)
  if (reading) return { outcome: reading.outcome, ffa: false, locale }
  // Sans deux camps lisibles, seule la VICTOIRE a une réplique : « Vainqueur » se passe
  // d'adversaire nommé, « Défaite » et « Égalité » supposent un affrontement à deux camps.
  const outcome = outcomeCodeToValue(outcomeCode)
  return outcome === 'win' ? { outcome, ffa: true, locale } : null
}

/**
 * endMatchSounds rend LES FICHIERS À JOUER ENSEMBLE à l'arrivée en fin de match : la voix
 * tirée parmi ses prises, puis la fanfare. Liste vide = rien ne sonne.
 *
 * `rand` est injecté pour que les tests fixent la prise ; la production passe `Math.random`
 * (même patron que `drawVariation` pour les fourchettes RANGED des armes).
 */
export function endMatchSounds(
  outcome: VictoryOutcome,
  ffa: boolean,
  locale: ReplayLocale,
  rand: () => number = Math.random,
): string[] {
  if (ffa) {
    if (outcome !== 'win') return []
    return [pickTake(END_FFA_WIN_VOICE_STEMS[locale], rand), END_MUSIC_STEMS.win]
  }
  return [pickTake(END_VOICE_STEMS[outcome][locale], rand), END_MUSIC_STEMS[outcome]]
}

/**
 * endMatchSoundStems rend TOUTES les prises qu'une fin PEUT jouer, tirage compris — c'est ce
 * que le lecteur doit précharger. Précharger la seule prise tirée ne marcherait pas : le
 * tirage a lieu à l'arrivée en fin, et un fichier demandé à cet instant arriverait après.
 */
export function endMatchSoundStems(spec: EndMatchSoundSpec | null): string[] {
  if (!spec) return []
  if (spec.ffa) {
    return spec.outcome === 'win'
      ? [...END_FFA_WIN_VOICE_STEMS[spec.locale], END_MUSIC_STEMS.win]
      : []
  }
  return [...END_VOICE_STEMS[spec.outcome][spec.locale], END_MUSIC_STEMS[spec.outcome]]
}

/**
 * Une prise au hasard. Le tirage est borné à l'index valide : un `rand` qui rendrait
 * exactement 1 (hors contrat de `Math.random`, mais pas d'un double de test) sortirait sinon
 * de la liste et jouerait `undefined`.
 */
function pickTake(takes: readonly string[], rand: () => number): string {
  return takes[Math.min(Math.floor(rand() * takes.length), takes.length - 1)]
}
