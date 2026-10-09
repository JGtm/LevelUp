/**
 * matchEmpriseText.ts — LES TEXTES des cartes de l'Emprise de la Vue match (plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, D17 et A2) : les textes des pages sœurs (Escouade, Séries
 * temporelles, Sessions) quand la carte est la même — même titre, même aide, portée « le match » —,
 * et ceux propres à la page (contrôle des ressources objet par objet, constats d'une ligne sans
 * barre, une ligne d'« Isolement » par joueur). Aucun texte ne dit un inconnu : une grandeur que le
 * match ne porte pas n'a pas de ligne (garde : `lib/i18n/noUnknownMentions.guard.test.ts`).
 *
 * Textes factuels, sans personne : le joueur est désigné par son gamertag, les groupes par « Équipe »,
 * « Adversaire », « Reste de l'équipe » ; « équipe », jamais « camp » (garde :
 * `timeseries/usages/textesSansPersonne.test.ts`). Parité FR / EN par le typage `Record<Locale, …>`.
 */
import { EMPRISE_TEXT, type EmpriseText } from '@/features/squad/emprise/empriseStrings'
import { getSquadText, type SquadText } from '@/features/squad/i18n'
import { USAGES_TEXT } from '@/features/timeseries/usages/usagesText'
import type { UsagesCardsText } from '@/features/timeseries/usages/usagesCardsText'
import type { Locale } from '@/lib/i18n/locale'

/** Les textes propres à la page. */
export interface MatchOwnText {
  /** Sous l'intertitre « Équipement et terrain » d'un match filmé : « film décodé · 8 joueurs présents à la fin ». */
  coverage: (present: number) => string
  /** Sous-libellé de la piste « Prises de bonus » : « 10 socles vidés ». */
  powerupSub: (emptied: number) => string
  /**
   * Bouton des armes de râtelier : « Armes de râtelier (3 types d’armes, repliés) » / « (3 types
   * d’armes) » — un nombre de TYPES d'armes (`grid.rackTypes` de l'Escouade), pas de prises.
   */
  racksFolded: (n: number) => string
  racksUnfolded: (n: number) => string
  /** Constats d'une ligne sans barre épaisse de « Frags par ressource ». */
  production: {
    powerupNoEffect: string
    powerupNoKills: string
    powerZero: string
    powerNoPickup: string
  }
  /** Constats d'une ligne sans rapport de « Rendement par ressource » (un camp sans exposition). */
  yield: {
    noEffect: (team: boolean, teamEffect: string, teamKills: number) => string
    noPickup: (team: boolean, us: number, them: number) => string
  }
}

export interface MatchEmpriseText {
  /** D, E, G, H : l'Emprise de l'Escouade, portée « le match ». */
  emprise: EmpriseText
  /** I : « Isolement » des Séries temporelles, une ligne par joueur. */
  cards: UsagesCardsText
  /**
   * A : « Répartition des frags » (titre de l'Escouade, aide propre) ; B : « Outils de destruction » de
   * l'Escouade (aide propre) et son état vide.
   */
  squad: Pick<SquadText, 'performanceCharts' | 'weaponKills' | 'empty'>
  own: MatchOwnText
}

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

interface Infos {
  frags: string
  tools: string
  control: string
  sheets: string
  production: string
  yield: string
  livesTitle: string
  livesLead: string
}

const INFOS: Record<Locale, Infos> = {
  fr: {
    frags:
      'Frags du joueur sur le match : anneau intérieur par classe d’arme, anneau extérieur par rôle ; total au centre, ' +
      'part de chaque classe en légende.',
    tools: 'Frags du joueur sur le match, arme par arme ; couleur de la barre : classe de l’arme, celle de la Répartition des frags.',
    control:
      'Prises de chaque ressource, puis de chacun de ses objets, par l’équipe et par l’adversaire sur le match, en comptes ; ' +
      'trait orange : 50 %.',
    sheets:
      'Prises de chaque bonus, arme spéciale et arme de râtelier par joueur de l’équipe sur le match, une pastille par prise. ' +
      'Pastille vide : bonus perdu (gardé sans être activé, ou lâché à la mort).',
    production:
      'Barre épaisse : part de l’équipe dans les frags obtenus avec chaque ressource ; barre fine : part de l’équipe dans ' +
      'l’exposition (temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un véhicule), sur le match.',
    yield:
      'Écart relatif entre les frags de l’équipe et ceux de l’adversaire pour la même exposition, sur le match : par minute ' +
      'd’effet d’un bonus, par prise d’arme spéciale. Zéro : autant ; rendements bruts sous la valeur.',
    livesTitle: 'Isolement, par joueur',
    livesLead: 'Une ligne par joueur de l’équipe. ',
  },
  en: {
    frags:
      'The player’s kills in the match: inner ring by weapon class, outer ring by role; total in the centre, each ' +
      'class’s share in the legend.',
    tools: 'The player’s kills in the match, weapon by weapon; bar colour: the weapon’s class, as in the Kill type distribution.',
    control:
      'Pickups of each resource, then of each of its items, by the team and by the opponent in the match, in counts; ' +
      'orange line: 50%.',
    sheets:
      'Pickups of each power-up, power weapon and rack weapon by each player of the team in the match, one dot per pickup. ' +
      'Hollow dot: lost power-up (held without being activated, or dropped on death).',
    production:
      'Thick bar: the team’s share of the kills made with each resource; thin bar: the team’s share of the exposure ' +
      '(effect time of a power-up, pickups of a power weapon, time aboard a vehicle), in the match.',
    yield:
      'Relative gap between the team’s kills and the opponent’s for the same exposure, in the match: per minute of ' +
      'power-up effect, per power-weapon pickup. Zero: the same; raw yields under the value.',
    livesTitle: 'Isolation, by player',
    livesLead: 'One row per player of the team. ',
  },
}

const OWN: Record<Locale, MatchOwnText> = {
  fr: {
    coverage: (present) => `film décodé · ${present} ${plural(present, 'joueur présent', 'joueurs présents')} à la fin`,
    powerupSub: (emptied) => `${emptied} ${plural(emptied, 'socle vidé', 'socles vidés')}`,
    racksFolded: (n) => `(${EMPRISE_TEXT.fr.grid.rackTypes(n)}, repliés)`,
    racksUnfolded: (n) => `(${EMPRISE_TEXT.fr.grid.rackTypes(n)})`,
    production: {
      powerupNoEffect: 'Aucun bonus actif sur le match',
      powerupNoKills: 'Aucun frag pendant l’effet d’un bonus',
      powerZero: '0 frag aux armes spéciales dans les deux équipes',
      powerNoPickup: 'aucune prise d’arme spéciale',
    },
    yield: {
      noEffect: (team, teamEffect, teamKills) =>
        `Aucun temps d’effet pour ${team ? 'l’équipe' : 'l’adversaire'} (équipe : ${teamEffect}, ${teamKills} ${plural(teamKills, 'frag', 'frags')})`,
      noPickup: (team, us, them) => `Aucune arme spéciale prise par ${team ? 'l’équipe' : 'l’adversaire'} (${us} contre ${them})`,
    },
  },
  en: {
    coverage: (present) => `film decoded · ${present} ${plural(present, 'player', 'players')} present at the end`,
    powerupSub: (emptied) => `${emptied} ${plural(emptied, 'pad emptied', 'pads emptied')}`,
    racksFolded: (n) => `(${EMPRISE_TEXT.en.grid.rackTypes(n)}, folded)`,
    racksUnfolded: (n) => `(${EMPRISE_TEXT.en.grid.rackTypes(n)})`,
    production: {
      powerupNoEffect: 'No power-up active in the match',
      powerupNoKills: 'No kill during a power-up effect',
      powerZero: '0 power-weapon kills on either team',
      powerNoPickup: 'no power-weapon pickup',
    },
    yield: {
      noEffect: (team, teamEffect, teamKills) =>
        `No effect time for ${team ? 'the team' : 'the opponent'} (team: ${teamEffect}, ${teamKills} ${plural(teamKills, 'kill', 'kills')})`,
      noPickup: (team, us, them) => `No power weapon picked up by ${team ? 'the team' : 'the opponent'} (${us} vs ${them})`,
    },
  },
}

function textsFor(locale: Locale): MatchEmpriseText {
  const base = EMPRISE_TEXT[locale]
  const i = INFOS[locale]
  const squad = getSquadText(locale)
  const cards = USAGES_TEXT[locale].cards
  return {
    emprise: {
      ...base,
      control: { ...base.control, title: base.grid.title, info: i.control },
      sheets: { ...base.sheets, info: i.sheets },
      production: { ...base.production, info: i.production },
      yield: { ...base.yield, info: i.yield },
    },
    cards: {
      ...cards,
      lives: {
        ...cards.lives,
        title: i.livesTitle,
        info: i.livesLead + cards.lives.info,
      },
    },
    squad: {
      performanceCharts: { ...squad.performanceCharts, fragBreakdownInfo: i.frags },
      weaponKills: { ...squad.weaponKills, info: i.tools },
      empty: squad.empty,
    },
    own: OWN[locale],
  }
}

/** Les textes des cartes de l'Emprise de la Vue match, par langue. */
export const MATCH_EMPRISE_TEXT: Record<Locale, MatchEmpriseText> = { fr: textsFor('fr'), en: textsFor('en') }
