/**
 * placementStrings.ts — les textes du bloc « Isolement » de l'onglet Emprise (lot V4 du
 * plan PLAN_EMPRISE_VIES_2026-09-28, spécification §2 : « Placement et rendement de chaque vie »
 * et « Part des vies par placement »). Fichier à part : `empriseStrings.ts` frôle le seuil de
 * taille. Aides ⓘ : deux phrases au plus, ce qui est tracé et sur quel périmètre, sans personne.
 * Parité FR / EN garantie par le typage `Record<Locale, …>`. Les nombres arrivent déjà formatés (séparateur de la langue).
 */
import type { SquadEmprisePlacementQuadrant } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/** Un quart du nuage : son titre (capitales) et son sous-titre. */
export interface QuadrantText {
  title: string
  sub: string
}

export interface PlacementText {
  /** Intertitre du bloc. */
  section: string
  life: {
    title: string
    /** ⓘ : ce que dit l'abscisse et ce qu'elle laisse de côté par construction. */
    info: string
    xAxis: string
    yAxis: string
    radarLine: string
    quadrants: Record<SquadEmprisePlacementQuadrant, QuadrantText>
    /** « une vie de 1:23 » (le joueur, en gras, précède). */
    lifeHead: (duration: string) => string
    /** « distance médiane 0,62 radar · 12 % de la vie hors radar · 2 frags ». */
    lifeBody: (x: string, outPct: number, kills: number) => string
    /** « 146 vies » (le joueur, en gras, précède). */
    medianHead: (lives: number) => string
    /** « médiane 0,62 radar · 1 frag par vie ». */
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
  section: 'Isolement',
  life: {
    title: 'Placement et rendement de chaque vie',
    info:
      'Abscisse : médiane, sur la vie (de l’apparition à la mort), de la distance au coéquipier vivant le plus ' +
      'proche, en portées de radar du match, hors port d’objectif et équipe à terre.',
    xAxis: 'distance médiane au coéquipier le plus proche pendant la vie, en portées de radar',
    yAxis: 'frags dans la vie',
    radarLine: 'portée du radar',
    quadrants: {
      in_range_productive: { title: 'À PORTÉE ET RENTABLE', sub: 'sûr' },
      isolated_productive: { title: 'ISOLÉ ET RENTABLE', sub: 'flanqueur, surveiller la régularité' },
      in_range_costly: { title: 'À PORTÉE ET COÛTEUX', sub: 'duel à travailler, pas le placement' },
      isolated_costly: { title: 'ISOLÉ ET COÛTEUX', sub: 'vie donnée pour rien, seul' },
    },
    lifeHead: (duration) => `une vie de ${duration}`,
    lifeBody: (x, outPct, kills) =>
      `distance médiane ${x} radar · ${outPct} % de la vie hors radar · ${kills} ${kills > 1 ? 'frags' : 'frag'}`,
    medianHead: (lives) => `${lives} ${lives > 1 ? 'vies' : 'vie'}`,
    medianMid: (x, kills, killsN) => `médiane ${x} radar · ${kills} ${killsN > 1 ? 'frags' : 'frag'} par vie`,
    medianLast: (pct) => `${pct} % des vies isolées et sans frag`,
  },
  quarts: {
    title: 'Part des vies par placement',
    info: (isolatedFrom, isolatedFromN, productiveFrom) =>
      'Part des vies de chaque joueur dans les quatre quarts du nuage ci-dessus. ' +
      `Isolé : distance médiane d’au moins ${isolatedFrom} ${isolatedFromN >= 2 ? 'portées' : 'portée'} de radar ; ` +
      `rentable : au moins ${productiveFrom} ${productiveFrom > 1 ? 'frags' : 'frag'} dans la vie.`,
    names: {
      in_range_productive: 'à portée et rentable',
      isolated_productive: 'isolé et rentable',
      in_range_costly: 'à portée et coûteux',
      isolated_costly: 'isolé et coûteux',
    },
    value: (pct) => `${pct} %`,
    tipHead: (lives) => `${lives} ${lives > 1 ? 'vies' : 'vie'}`,
    tipLine: (name, pct) => `${name} : ${pct} %`,
  },
}

const EN: PlacementText = {
  section: 'Isolation',
  life: {
    title: 'Placement and yield of each life',
    info:
      'Horizontal position: median, over the life (spawn to death), of the distance to the nearest living ' +
      'teammate, in radar ranges of the match, excluding objective carrying and team down.',
    xAxis: 'median distance to the nearest teammate during the life, in radar ranges',
    yAxis: 'kills in the life',
    radarLine: 'radar range',
    quadrants: {
      in_range_productive: { title: 'IN RANGE AND PRODUCTIVE', sub: 'safe' },
      isolated_productive: { title: 'ISOLATED AND PRODUCTIVE', sub: 'flanker, watch the consistency' },
      in_range_costly: { title: 'IN RANGE AND COSTLY', sub: 'a duel to work on, not the placement' },
      isolated_costly: { title: 'ISOLATED AND COSTLY', sub: 'a life given away, alone' },
    },
    lifeHead: (duration) => `a life of ${duration}`,
    lifeBody: (x, outPct, kills) =>
      `median distance ${x} radar · ${outPct}% of the life out of radar · ${kills} ${kills > 1 ? 'kills' : 'kill'}`,
    medianHead: (lives) => `${lives} ${lives > 1 ? 'lives' : 'life'}`,
    medianMid: (x, kills, killsN) => `median ${x} radar · ${kills} ${killsN > 1 ? 'kills' : 'kill'} per life`,
    medianLast: (pct) => `${pct}% of lives isolated and without a kill`,
  },
  quarts: {
    title: 'Share of lives by placement',
    info: (isolatedFrom, isolatedFromN, productiveFrom) =>
      'Share of each player’s lives in the four quadrants of the scatter above. ' +
      `Isolated: median distance of at least ${isolatedFrom} radar ${isolatedFromN > 1 ? 'ranges' : 'range'}; ` +
      `productive: at least ${productiveFrom} ${productiveFrom > 1 ? 'kills' : 'kill'} in the life.`,
    names: {
      in_range_productive: 'in range and productive',
      isolated_productive: 'isolated and productive',
      in_range_costly: 'in range and costly',
      isolated_costly: 'isolated and costly',
    },
    value: (pct) => `${pct}%`,
    tipHead: (lives) => `${lives} ${lives > 1 ? 'lives' : 'life'}`,
    tipLine: (name, pct) => `${name}: ${pct}%`,
  },
}

export const PLACEMENT_TEXT: Record<Locale, PlacementText> = { fr: FR, en: EN }
