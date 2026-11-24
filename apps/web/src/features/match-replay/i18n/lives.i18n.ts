/**
 * lives.i18n.ts — LES TEXTES DES TRACES DES VIES du panneau de la vue du rejeu : le contrat (`LivesText`) et ses deux
 * langues. Trois calques (apparitions, morts, tueurs) et deux affinages communs (qui, quand).
 *
 * UN SOUS-DICTIONNAIRE À PART, comme `playerPanel.i18n.ts` : `fr.ts` et `en.ts` sont au plafond de taille du dépôt, et ces
 * textes forment un tout. Servis par `REPLAY_TEXT[locale].lives` ; la parité FR/EN est tenue par le typage
 * (`satisfies LivesText` dans chaque langue).
 */
import type { LivesWhen, LivesWho } from '../model/livesMarks'

export interface LivesText {
  layerSpawns: string
  layerSpawnsHint: string
  layerDeaths: string
  layerDeathsHint: string
  layerKillers: string
  layerKillersHint: string
  /** Les deux affinages communs : leur libellé, leurs options et leur explication. */
  whoLabel: string
  who: Record<LivesWho, string>
  whoHint: Record<LivesWho, string>
  whenLabel: string
  when: Record<LivesWhen, string>
  whenHint: Record<LivesWhen, string>
}

export const FR_LIVES = {
  layerSpawns: 'Apparitions',
  layerSpawnsHint:
    "Une goutte creuse à l'apparition de chaque vie, pointée dans le sens du regard à cet instant (un anneau seul quand aucun regard n'est lu)",
  layerDeaths: 'Morts',
  layerDeathsHint: "Une croix au lieu de chaque mort, à la couleur de l'équipe de la victime",
  layerKillers: 'Tueurs',
  layerKillersHint:
    "Un triangle creux à la position du tueur à l'instant du frag, et un trait fin vers la victime, à la couleur de l'équipe du tueur",
  whoLabel: 'Qui',
  who: { all: 'Tous', team: 'Mon équipe', followed: 'Point de vue' },
  whoHint: {
    all: 'Les traces de tous les joueurs',
    team: "Seulement les traces des joueurs de l'équipe du point de vue",
    followed: 'Seulement les traces du joueur dont on regarde le rejeu',
  },
  whenLabel: 'Quand',
  when: { all: 'Toute la partie', sofar: 'Jusqu’ici' },
  whenHint: {
    all: 'Toutes les traces de la partie, du début à la fin',
    sofar:
      "Seulement ce qui est arrivé avant l'image affichée ; les traces plus anciennes sont plus pâles. Les apparitions des vies en cours restent pleines",
  },
} satisfies LivesText

export const EN_LIVES = {
  layerSpawns: 'Spawns',
  layerSpawnsHint:
    'A hollow drop where each life starts, pointing where the player looked at that moment (a bare ring when no heading is read)',
  layerDeaths: 'Deaths',
  layerDeathsHint: "A cross where each death happened, in the victim's team color",
  layerKillers: 'Killers',
  layerKillersHint:
    "A hollow triangle at the killer's position at the moment of the kill, and a thin line to the victim, in the killer's team color",
  whoLabel: 'Who',
  who: { all: 'Everyone', team: 'My team', followed: 'Viewpoint' },
  whoHint: {
    all: 'The traces of every player',
    team: "Only the traces of the viewpoint's team",
    followed: 'Only the traces of the player whose replay is being watched',
  },
  whenLabel: 'When',
  when: { all: 'Whole match', sofar: 'So far' },
  whenHint: {
    all: 'Every trace of the match, start to end',
    sofar:
      'Only what happened before the displayed frame; older traces are paler. Spawns of lives in progress stay solid',
  },
} satisfies LivesText
