/**
 * matchEmpriseText.ts — LES TEXTES des cartes de l'Emprise de la Vue match (plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, D17 et A2) : les textes des pages sœurs (Escouade, Séries
 * temporelles, Sessions) quand la carte est la même — même titre, même aide, portée « le match » —,
 * et ceux propres à la page (contrôle des ressources objet par objet, raisons « non mesuré », une
 * ligne d'« Isolement » par joueur).
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
  /** Sous l'intertitre « Équipement et terrain » : « film décodé · 8 joueurs présents à la fin ». */
  coverage: (filmed: boolean, present: number) => string
  /** Sous-libellé de la piste des bonus : « prises · 10 socles vidés ». */
  powerupSub: (emptied: number) => string
  /** Bouton des armes de râtelier : « Armes de râtelier (3, repliées) » / « (3) ». */
  racksFolded: (n: number) => string
  racksUnfolded: (n: number) => string
  /** Sous la piste : les prises sur un emplacement non identifié. */
  unclassified: (n: number, team: number, opponent: number) => string
  /** Un joueur sans vie rangée (« Isolement, par joueur »). */
  noRankedLife: string
  /** Lignes « non mesuré » de « Frags par ressource ». */
  production: {
    powerupKillsUnpublished: string
    /** La lecture du journal des morts a échoué : la cause n'est pas connue. */
    powerupKillsUnavailable: string
    powerupNoEffect: string
    powerupNoKills: string
    powerZero: string
    sheetFailed: string
    vehicleUnmeasured: string
    powerNoPickup: string
  }
  /** Lignes « non mesurable » de « Rendement par ressource ». */
  yield: {
    powerupUnpublished: string
    powerupUnavailable: string
    noEffect: (team: boolean, teamEffect: string, teamKills: number) => string
    noPickup: (team: boolean, us: number, them: number) => string
    vehicleUnmeasured: string
  }
}

export interface MatchEmpriseText {
  /** D, E, G, H : l'Emprise de l'Escouade, portée « le match ». */
  emprise: EmpriseText
  /** I : « Isolement » des Séries temporelles, une ligne par joueur. */
  cards: UsagesCardsText
  /** B : « Outils de destruction » de l'Escouade (aide propre) et son état vide. */
  squad: Pick<SquadText, 'weaponKills' | 'empty'>
  own: MatchOwnText
}

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

interface Infos {
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
    tools: 'Frags du joueur sur le match, arme par arme ; pastille : couleur de la classe de l’arme dans la Répartition des frags.',
    control:
      'Prises de chaque ressource, puis de chacun de ses objets, par l’équipe et par l’adversaire sur le match, en comptes ; ' +
      'trait orange : 50 %. Une prise sans ramasseur connu ne compte dans aucune équipe.',
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
    tools: 'The player’s kills in the match, weapon by weapon; swatch: colour of the weapon’s class in the Kill type distribution.',
    control:
      'Pickups of each resource, then of each of its items, by the team and by the opponent in the match, in counts; ' +
      'orange line: 50%. A pickup with no known picker counts for neither team.',
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
    coverage: (filmed, present) => (filmed ? `film décodé · ${present} ${plural(present, 'joueur présent', 'joueurs présents')} à la fin` : 'sans film'),
    powerupSub: (emptied) => `prises · ${emptied} ${plural(emptied, 'socle vidé', 'socles vidés')}`,
    racksFolded: (n) => `(${n}, repliées)`,
    racksUnfolded: (n) => `(${n})`,
    unclassified: (n, team, opponent) =>
      `${n} ${plural(n, 'prise', 'prises')} sur un emplacement non identifié, hors des pistes (équipe ${team}, adversaire ${opponent})`,
    noRankedLife: 'Aucune vie terminée par une mort avec un coéquipier situé',
    production: {
      powerupKillsUnpublished: 'Frags pendant l’effet non mesurés : journal des morts non publiable',
      powerupKillsUnavailable: 'Non mesuré : lecture indisponible',
      powerupNoEffect: 'Aucun temps d’effet mesuré',
      powerupNoKills: 'Aucun frag pendant l’effet d’un bonus',
      powerZero: '0 frag aux armes spéciales dans les deux équipes',
      sheetFailed: 'Non mesuré : feuille de match illisible',
      vehicleUnmeasured: 'Non mesuré',
      powerNoPickup: 'aucune prise d’arme spéciale mesurée',
    },
    yield: {
      powerupUnpublished: 'Non mesuré : frags pendant l’effet non publiés',
      powerupUnavailable: 'Non mesuré : lecture indisponible',
      noEffect: (team, teamEffect, teamKills) =>
        `Non mesurable : aucun temps d’effet pour ${team ? 'l’équipe' : 'l’adversaire'} (équipe : ${teamEffect}, ${teamKills} ${plural(teamKills, 'frag', 'frags')})`,
      noPickup: (team, us, them) =>
        `Non mesurable : aucune arme spéciale prise par ${team ? 'l’équipe' : 'l’adversaire'} (${us} contre ${them})`,
      vehicleUnmeasured: 'Non mesuré',
    },
  },
  en: {
    coverage: (filmed, present) => (filmed ? `film decoded · ${present} ${plural(present, 'player', 'players')} present at the end` : 'no film'),
    powerupSub: (emptied) => `pickups · ${emptied} ${plural(emptied, 'pad emptied', 'pads emptied')}`,
    racksFolded: (n) => `(${n}, folded)`,
    racksUnfolded: (n) => `(${n})`,
    unclassified: (n, team, opponent) =>
      `${n} ${plural(n, 'pickup', 'pickups')} on an unidentified spot, outside the tracks (team ${team}, opponent ${opponent})`,
    noRankedLife: 'No life ended by a death with a located teammate',
    production: {
      powerupKillsUnpublished: 'Kills during the effect not measured: kill log not publishable',
      powerupKillsUnavailable: 'Not measured: reading unavailable',
      powerupNoEffect: 'No effect time measured',
      powerupNoKills: 'No kill during a power-up effect',
      powerZero: '0 power-weapon kills on either team',
      sheetFailed: 'Not measured: match sheet unreadable',
      vehicleUnmeasured: 'Not measured',
      powerNoPickup: 'no power-weapon pickup measured',
    },
    yield: {
      powerupUnpublished: 'Not measured: kills during the effect not published',
      powerupUnavailable: 'Not measured: reading unavailable',
      noEffect: (team, teamEffect, teamKills) =>
        `Not measurable: no effect time for ${team ? 'the team' : 'the opponent'} (team: ${teamEffect}, ${teamKills} ${plural(teamKills, 'kill', 'kills')})`,
      noPickup: (team, us, them) =>
        `Not measurable: no power weapon picked up by ${team ? 'the team' : 'the opponent'} (${us} vs ${them})`,
      vehicleUnmeasured: 'Not measured',
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
        info: (unlocated, noRadar, unpublishable) => i.livesLead + cards.lives.info(unlocated, noRadar, unpublishable),
      },
    },
    squad: { weaponKills: { ...squad.weaponKills, info: i.tools }, empty: squad.empty },
    own: OWN[locale],
  }
}

/** Les textes des cartes de l'Emprise de la Vue match, par langue. */
export const MATCH_EMPRISE_TEXT: Record<Locale, MatchEmpriseText> = { fr: textsFor('fr'), en: textsFor('en') }
