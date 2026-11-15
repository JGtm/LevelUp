/**
 * placementStrings.ts — les textes du bloc « Groupé ou isolé » de l'onglet Emprise (spécification
 * §2 : « Distance au coéquipier le plus proche et frags, par vie » en carte de chaleur, et
 * « Part des vies par placement »). Fichier à part : `empriseStrings.ts` frôle le seuil de
 * taille. Aides ⓘ : trois phrases au plus, ce qui est tracé et sur quel périmètre, sans personne.
 * Parité FR / EN garantie par le typage `Record<Locale, …>`. Les nombres arrivent déjà formatés (séparateur de la langue).
 */
import type { SquadEmprisePlacementQuadrant } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/** Un quart de la carte : son titre (capitales) et son sous-titre. */
export interface QuadrantText {
  title: string
  sub: string
}

export interface PlacementText {
  /** Intertitre du bloc. */
  section: string
  life: {
    title: string
    /** ⓘ : l'unité (la vie), ce que dit l'abscisse et ce qu'elle laisse de côté par construction. */
    info: string
    xAxis: string
    yAxis: string
    radarLine: string
    quadrants: Record<SquadEmprisePlacementQuadrant, QuadrantText>
    /** Infobulle d'une case : « 12,5 % des vies de l’escouade · 7 vies » (la part arrive formatée). */
    cellHead: (pct: string, lives: number) => string
    /** « distance de 0,50 à 0,75 radar » ; `to` absent : la dernière case, « distance de 1,75 radar ou plus ». */
    cellDistance: (from: string, to: string | null) => string
    /** « 2 frags » ; `open` : la dernière ligne, « 5 frags ou plus ». */
    cellKills: (kills: number, open: boolean) => string
    /** Valeur écrite dans une case : « 12 % ». */
    cellValue: (pct: number) => string
    /** « 146 vies » (le joueur, en gras, précède). */
    medianHead: (lives: number) => string
    /** « distance médiane 0,62 radar · 1,5 frag par vie en moyenne ». */
    medianMid: (x: string, kills: string, killsN: number) => string
    /** « 7 % des vies isolées et sans frag ». */
    medianLast: (pct: number) => string
  }
  quarts: {
    title: string
    /** Seuil d'isolement (portées de radar, déjà formaté), nombre de frags d'une vie rentable. */
    info: (isolatedFrom: string, isolatedFromN: number, productiveFrom: number) => string
    names: Record<SquadEmprisePlacementQuadrant, string>
    /** « 42 % » : valeur dans un segment et graduation de l'axe (entier, pourcentage). */
    value: (pct: number) => string
    /** « 141 vies » (le joueur, en gras, précède). */
    tipHead: (lives: number) => string
    tipLine: (name: string, pct: number) => string
  }
}

const FR: PlacementText = {
  section: 'Groupé ou isolé',
  life: {
    title: 'Distance au coéquipier le plus proche et frags, par vie',
    info:
      'Chaque case donne la part des vies de l’escouade ; l’unité est la vie, de l’apparition à la mort. ' +
      'Abscisse : médiane de la distance au coéquipier vivant le plus proche, en portées de radar du match, ' +
      'hors port d’objectif et équipe à terre. Gros points : distance médiane et frags moyens par vie de chaque joueur.',
    xAxis: 'distance médiane au coéquipier le plus proche pendant la vie, en portées de radar',
    yAxis: 'frags dans la vie',
    radarLine: 'portée du radar',
    quadrants: {
      in_range_productive: { title: 'GROUPÉ ET RENTABLE', sub: 'sûr' },
      isolated_productive: { title: 'ISOLÉ ET RENTABLE', sub: 'flanqueur, surveiller la régularité' },
      in_range_costly: { title: 'GROUPÉ ET COÛTEUX', sub: 'duel à travailler, pas le placement' },
      isolated_costly: { title: 'ISOLÉ ET COÛTEUX', sub: 'vie donnée pour rien, seul' },
    },
    cellHead: (pct, lives) => `${pct} % des vies de l’escouade · ${lives} ${lives > 1 ? 'vies' : 'vie'}`,
    cellDistance: (from, to) => (to == null ? `distance de ${from} radar ou plus` : `distance de ${from} à ${to} radar`),
    cellKills: (kills, open) => `${kills} ${kills > 1 ? 'frags' : 'frag'}${open ? ' ou plus' : ''}`,
    cellValue: (pct) => `${pct} %`,
    medianHead: (lives) => `${lives} ${lives > 1 ? 'vies' : 'vie'}`,
    medianMid: (x, kills, killsN) => `distance médiane ${x} radar · ${kills} ${killsN > 1 ? 'frags' : 'frag'} par vie en moyenne`,
    medianLast: (pct) => `${pct} % des vies isolées et sans frag`,
  },
  quarts: {
    title: 'Part des vies par placement',
    info: (isolatedFrom, isolatedFromN, productiveFrom) =>
      'Part des vies de chaque joueur dans les quatre quarts de la carte de chaleur ci-dessus. ' +
      `Isolé : distance médiane d’au moins ${isolatedFrom} ${isolatedFromN >= 2 ? 'portées' : 'portée'} de radar, groupé : en deçà ; ` +
      `rentable : au moins ${productiveFrom} ${productiveFrom > 1 ? 'frags' : 'frag'} dans la vie.`,
    names: {
      in_range_productive: 'groupé et rentable',
      isolated_productive: 'isolé et rentable',
      in_range_costly: 'groupé et coûteux',
      isolated_costly: 'isolé et coûteux',
    },
    value: (pct) => `${pct} %`,
    tipHead: (lives) => `${lives} ${lives > 1 ? 'vies' : 'vie'}`,
    tipLine: (name, pct) => `${name} : ${pct} %`,
  },
}

const EN: PlacementText = {
  section: 'Grouped or isolated',
  life: {
    title: 'Distance to the nearest teammate and kills, per life',
    info:
      'Each cell gives the share of the squad’s lives; the unit is the life, from spawn to death. ' +
      'Horizontal position: median distance to the nearest living teammate, in radar ranges of the match, ' +
      'excluding objective carrying and team down. Large dots: each player’s median distance and mean kills per life.',
    xAxis: 'median distance to the nearest teammate during the life, in radar ranges',
    yAxis: 'kills in the life',
    radarLine: 'radar range',
    quadrants: {
      in_range_productive: { title: 'GROUPED AND PRODUCTIVE', sub: 'safe' },
      isolated_productive: { title: 'ISOLATED AND PRODUCTIVE', sub: 'flanker, watch the consistency' },
      in_range_costly: { title: 'GROUPED AND COSTLY', sub: 'a duel to work on, not the placement' },
      isolated_costly: { title: 'ISOLATED AND COSTLY', sub: 'a life given away, alone' },
    },
    cellHead: (pct, lives) => `${pct}% of the squad’s lives · ${lives} ${lives > 1 ? 'lives' : 'life'}`,
    cellDistance: (from, to) => (to == null ? `distance ${from} radar or more` : `distance ${from} to ${to} radar`),
    cellKills: (kills, open) => `${kills} ${kills > 1 ? 'kills' : 'kill'}${open ? ' or more' : ''}`,
    cellValue: (pct) => `${pct}%`,
    medianHead: (lives) => `${lives} ${lives > 1 ? 'lives' : 'life'}`,
    medianMid: (x, kills, killsN) => `median distance ${x} radar · ${kills} ${killsN > 1 ? 'kills' : 'kill'} per life on average`,
    medianLast: (pct) => `${pct}% of lives isolated and without a kill`,
  },
  quarts: {
    title: 'Share of lives by placement',
    info: (isolatedFrom, isolatedFromN, productiveFrom) =>
      'Share of each player’s lives in the four quadrants of the heatmap above. ' +
      `Isolated: median distance of at least ${isolatedFrom} radar ${isolatedFromN > 1 ? 'ranges' : 'range'}, grouped: below; ` +
      `productive: at least ${productiveFrom} ${productiveFrom > 1 ? 'kills' : 'kill'} in the life.`,
    names: {
      in_range_productive: 'grouped and productive',
      isolated_productive: 'isolated and productive',
      in_range_costly: 'grouped and costly',
      isolated_costly: 'isolated and costly',
    },
    value: (pct) => `${pct}%`,
    tipHead: (lives) => `${lives} ${lives > 1 ? 'lives' : 'life'}`,
    tipLine: (name, pct) => `${name}: ${pct}%`,
  },
}

export const PLACEMENT_TEXT: Record<Locale, PlacementText> = { fr: FR, en: EN }
