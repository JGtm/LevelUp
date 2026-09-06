/**
 * objectiveSound.ts — LES SONS D'OBJECTIF, et le camp qui les distingue.
 *
 * EXTRAIT DE `replaySound.ts` LE 2026-08-26, même raison que `grenadeSound.ts` : une famille
 * d'événements qui a sa propre source, sa propre clé de jointure et sa propre doctrine mérite
 * son fichier. Ici la clé n'est ni une arme ni une vignette : c'est le NOM CANONIQUE DE
 * STATISTIQUE que `doc.objectives` publie.
 *
 * D'OÙ VIENNENT CES SONS. De la RE du 2026-08-26
 * (`.ai/V7.5/RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`) : l'identifiant Wwise d'une banque est
 * le FNV-1 de son nom de fichier en minuscules, ce qui a permis de NOMMER la banque du mode
 * (`sb_004_mod_mp_ctf`), puis ses événements par la même voie
 * (`play_004_mod_mp_ctf_flag_scored_team`), puis de reconstruire chaque geste couche par
 * couche aux gains relevés dans le format.
 *
 * LE JEU DISTINGUE LES DEUX CAMPS, ET NOUS AUSSI. Les noms d'événements portent une modulation
 * terminale `_team` / `_enemy`, et ce sont des ÉVÉNEMENTS DISTINCTS avec des fichiers
 * DIFFÉRENTS — pas un même son passé dans un interrupteur. Six couples existent dans le jeu ;
 * trois statistiques du film y donnent accès.
 */

import type { ReplayDocumentReady } from '../model/replayNormalize'
import { frameToMs } from '../model/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import {
  allyTeamFromScoreboard,
  teamOfXuidFromScoreboard,
  type ScoreboardSide,
} from '../model/matchSides'

// LES DEUX LECTURES DU TABLEAU DE SCORE ONT LEUR FOYER DANS `matchSides` (K1/K2,
// 2026-09-06) : ce module les re-expose pour ses appelants historiques, il ne les
// ecrit plus.
export { allyTeamFromScoreboard, type ScoreboardSide }

/** Le camp de l'auteur d'une action, vu de la page — la même notion que le fil et les calques. */
export type ObjectiveSide = 'ally' | 'enemy' | 'unknown'

/**
 * OBJECTIVE_SOUND_STEMS — le son d'une action d'objectif, par NOM CANONIQUE DE STATISTIQUE.
 *
 * LA CLÉ EST LA STATISTIQUE, PAS UN LIBELLÉ : `doc.objectives` publie exactement les noms de
 * `match_objective_stats` (`flag_captures`, `flag_grabs`...), et c'est la seule quantité que le
 * film donne pour une action d'objectif. Une statistique absente de cette table est MUETTE —
 * jamais le son d'une action voisine, même règle que partout ailleurs dans la chaîne sonore.
 *
 * CAMP INCONNU = SILENCE sur les actions qui ont deux variantes. Sans ligne « moi » au tableau
 * de score (match d'un autre joueur, tableau absent), jouer l'une des deux serait affirmer un
 * camp. Le rejeu se tait plutôt que de choisir — même règle que l'encre des calques, qui passe
 * au neutre du thème dans ce cas.
 *
 * `flag_returns` N'A PAS DE VARIANTE D'ÉQUIPE DANS LE JEU, et c'est une mesure, pas un
 * raccourci : `play_004_mod_mp_ctf_flag_returned` existe seul, sans jumeau `_team`/`_enemy`.
 * Il sonne donc pareil pour tout le monde, y compris quand le camp est inconnu.
 *
 * UNE PAIRE PEUT ÊTRE INCOMPLÈTE, ET LE TYPE LE DIT. `zone_captures` n'a aujourd'hui que son
 * côté ALLIÉ (le son adverse n'est pas encore désigné à l'oreille) : le camp adverse reste
 * donc MUET, exactement comme un camp inconnu. Une paire à moitié remplie ne joue jamais son
 * autre moitié « faute de mieux » — ce serait annoncer une capture alliée sur une capture
 * adverse, la pire erreur possible sur un son d'objectif.
 *
 * D'OÙ VIENT LE SON DE CAPTURE DE ZONE, ET POURQUOI IL N'A PAS DE NOM. Il est DÉSIGNÉ À
 * L'OREILLE par l'utilisateur (événement `d8a2fcb8` de la banque `1c609526`, « base capturée,
 * équipe alliée »), pas cassé par hachage — et ce n'est pas un pis-aller, c'est la règle du
 * chantier (`RECETTE_SONS_ARMES` §5 : « les votes priment sur tout critère »). Le hachage a
 * été poussé jusqu'au bout sur cette cible : 162 831 744 candidats bâtis sur 36 noms de mode
 * × 4 familles × les 141 347 identifiants du binaire × 8 modulations, et seul le TÉMOIN
 * (`c3327c0b` = `..._strongholds_contested`, que l'utilisateur identifie indépendamment comme
 * « base contestée ») en ressort. Le binaire du jeu ne peut pas aider non plus : il ne porte
 * que TROIS noms d'événement Wwise en clair sur les ~6 800 du jeu, parce que le moteur poste
 * ses événements par identifiant PRÉ-HACHÉ, jamais par nom.
 *
 * CE QUI N'EST PAS ICI, ET POURQUOI — l'inventaire est écrit pour qu'on ne le recommence pas :
 *  - `zone_secures` : aucun son désigné. `zone_captures` a le sien, pas celui-là ;
 *  - la BOMBE (Assaut) : L'ÉVÉNEMENT EXISTE DEPUIS LE 2026-08-31. `ObjectiveTypeBomb` nomme
 *    `bomb_detonations` et le rejeu le publie dans `doc.objectives` (26 explosions attribuées
 *    sur les 28 datées du corpus de 9 films). Il ne manque plus que la DÉSIGNATION du stem par
 *    l'utilisateur — les sons sont extraits et rendus, mais aucun geste sonore n'a été désigné
 *    pour l'Assaut, et en désigner un à sa place ferait entendre un son que personne n'a
 *    validé (règle du gate sonore). Une ligne ici, et l'explosion sonne ;
 *  - le DISPOSITIF D'EXTRACTION : ses sons sont extraits et rendus, mais aucune statistique
 *    d'Extraction n'arrive dans `doc.objectives` — le film ne décode pas cette famille. Ce
 *    n'est pas un manque de son, c'est un manque d'ÉVÉNEMENT, et ça se répare côté décodeur ;
 *  - `flag_capture_assists` et `flag_carriers_killed` : aucun événement Wwise propre dans la
 *    banque du mode. Les sonner avec le son de la capture ferait entendre deux captures.
 */
export const OBJECTIVE_SOUND_STEMS: Readonly<
  Record<string, { ally?: string; enemy?: string } | { any: string }>
> = {
  flag_captures: { ally: 'objective_flag_scored_team', enemy: 'objective_flag_scored_enemy' },
  flag_steals: { ally: 'objective_flag_stolen_team', enemy: 'objective_flag_stolen_enemy' },
  flag_grabs: { ally: 'objective_flag_grabbed_team', enemy: 'objective_flag_grabbed_enemy' },
  flag_returns: { any: 'objective_flag_returned' },
  // LA PAIRE EST COMPLÈTE DEPUIS LE 2026-08-27. Elle était à moitié vide : seul le côté allié
  // était désigné, et le côté adverse restait MUET plutôt que de jouer le son allié « faute de
  // mieux » — ce qui aurait annoncé un gain quand on perd une base. L'utilisateur a désigné le
  // côté adverse à l'écoute de la planche (événements `4ebe99d6` / `8594aef7` / `9fad450d` de
  // la banque `1c609526` : le même son déclaré une fois par mode de jeu).
  zone_captures: { ally: 'objective_zone_captured_team', enemy: 'objective_zone_captured_enemy' },
  /**
   * L'EXPLOSION DE LA BOMBE (Assaut). Désigné à l'oreille par l'utilisateur le 2026-08-31 sur la
   * planche d'écoute de la banque, et corroboré par la STRUCTURE de celle-ci : l'événement
   * `984f65e5` (`play_004_mod_mp_assault_bomb_detonated`) déclare « 1 couche, 1 son » et pointe
   * `538469998` — les deux se rejoignent sur le même fichier.
   *
   * `{ any }` ET NON UNE PAIRE, et c'est une propriété du jeu, pas un raccourci : la banque
   * d'Assaut ne porte qu'UN son de détonation, sans jumeau `_team` / `_enemy` — exactement comme
   * le retour de drapeau et la contestation de zone. Une explosion sonne pareil pour tout le
   * monde ; c'est le RÉSULTAT qui diffère, pas le bruit.
   *
   * RIEN À RECONSTRUIRE, contrairement aux gestes multi-couches de la banque (le compte à
   * rebours en empile deux) : une couche, un média, décodage vgmstream puis crête à -1 dBTP.
   */
  bomb_detonations: { any: 'objective_bomb_detonated' },
}

/**
 * objectiveSoundStem — le fichier d'une action d'objectif, ou undefined pour le silence.
 * Pure, exportée pour être testée à l'unité.
 */
export function objectiveSoundStem(stat: string, side: ObjectiveSide): string | undefined {
  const entry = OBJECTIVE_SOUND_STEMS[stat]
  if (!entry) return undefined
  if ('any' in entry) return entry.any
  if (side === 'ally') return entry.ally
  if (side === 'enemy') return entry.enemy
  return undefined
}

/** Une ligne de tableau de score, réduite à ce dont le camp a besoin (typage structurel :
 *  ce fichier n'a pas à connaître le DTO complet). */
/**
 * sideResolverFromScoreboard — LA SEULE SOURCE DU CAMP, et c'est la même que celle des
 * calques : la ligne « moi » du tableau de score donne l'équipe alliée, chaque autre ligne
 * donne l'équipe de son joueur (`parseTeamSideID`, partagé avec `useReplayFlagCarries`).
 *
 * ELLE VIT ICI ET PAS DANS LE COMPOSANT (règle « pas de logique métier dans un composant ») :
 * `ReplayCanvas` n'a qu'à la brancher. Sans ligne « moi », ou pour un xuid absent du tableau,
 * elle rend `unknown` — et le son se tait plutôt que d'affirmer un camp.
 */
export function sideResolverFromScoreboard(
  scoreboard: readonly ScoreboardSide[] | undefined,
): (xuid: string) => ObjectiveSide {
  const allyTeam = allyTeamFromScoreboard(scoreboard)
  const teamOf = teamOfXuidFromScoreboard(scoreboard)
  return (xuid: string): ObjectiveSide => {
    if (allyTeam === null) return 'unknown'
    const team = teamOf.get(xuid)
    if (team === null || team === undefined) return 'unknown'
    return team === allyTeam ? 'ally' : 'enemy'
  }
}

/**
 * objectiveSoundEvents — les actions d'objectif du document, posées sur l'horloge du rejeu.
 *
 * L'HORLOGE NE DEMANDE AUCUN RECALAGE : `ObjectiveAction.T` est déjà l'index de frame, sur le
 * même axe que les tirs et les positions (la soustraction d'`originMs` est faite côté Go, cf.
 * `analysis/replay/objectives.go`). La conversion est donc la même `frameToMs` que partout.
 */
export function objectiveSoundEvents(
  doc: ReplayDocumentReady,
  sideOfXuid?: (xuid: string) => ObjectiveSide,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const a of doc.objectives) {
    const stem = objectiveSoundStem(a.stat, sideOfXuid?.(a.xuid) ?? 'unknown')
    if (stem) out.push(soundEvent(frameToMs(a.t, doc), stem))
  }
  return out
}
