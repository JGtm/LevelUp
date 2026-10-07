/**
 * usagesText.ts — LES TEXTES de l'onglet « Usages » des Séries temporelles (plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, décision D10 ; maquette v4
 * `.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html`, vue « Après »).
 *
 * L'onglet monte les cartes de l'Emprise et de l'Objectif de l'Escouade : leurs textes sont ceux de
 * l'Escouade SURCHARGÉS pour une page solo (`EMPRISE_TEXT_SOLO`, `OBJECTIF_TEXT_SOLO`) — « matchs du
 * périmètre » au lieu de « la soirée », colonnes par carte. Les cartes propres à l'onglet et les
 * intertitres (seule source : `USAGES_TEXT[locale].sections`) prennent `USAGES_TEXT`.
 * Titres factuels et concis, aucune personne (le joueur : son gamertag ; « Équipe », « Adversaire »,
 * « Reste de l’équipe ») — garde : `textesSansPersonne.test.ts`. Parité FR / EN par le typage
 * `Record<Locale, …>`.
 */
import { EMPRISE_TEXT, type EmpriseText } from '@/features/squad/emprise/empriseStrings'
import type { SoloSheetText } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { OBJECTIF_TEXT, type ObjectifText } from '@/features/squad/objectif/objectifStrings'
import type { Locale } from '@/lib/i18n/locale'

import { USAGES_CARDS_TEXT_EN, USAGES_CARDS_TEXT_FR, type UsagesCardsText } from './usagesCardsText'

export type { UsagesCardsText }

/** Les intertitres de l'onglet, dans l'ordre de la page (« Portée » est porté par sa section). */
export interface UsagesSectionsText {
  bilan: string
  carte: string
  mine: string
  prendre: string
  lives: string
  objectif: string
  equipment: string
  /**
   * « 12 matchs filmés sur 40 · frags de la feuille de match sur les 40 » : couverture lue par la
   * page Sessions (`sessionEmpriseText`). L'onglet Usages ne l'affiche pas.
   */
  bilanCoverage: (filmed: number, total: number) => string
}

export interface UsagesText {
  sections: UsagesSectionsText
  cards: UsagesCardsText
  sheet: SoloSheetText
}

const sur = (n: number, one: string, many: string) => (n > 1 ? many : one)

/** Les textes de l'Emprise de l'Escouade, surchargés pour la page solo (FR) : le périmètre, les cartes. */
function empriseFr(base: EmpriseText): EmpriseText {
  return {
    ...base,
    control: {
      ...base.control,
      info:
        'Prises de chaque ressource par l’équipe et par l’adversaire, en comptes, sur les matchs filmés du ' +
        'périmètre ; trait orange : 50 %. Les bonus sans ramasseur connu ne comptent dans aucune équipe.',
    },
    fil: {
      ...base.fil,
      info:
        'Part de l’équipe dans les prises de chaque ressource, cumulée match après match sur le périmètre. Points : ' +
        'part de chaque match, taille selon le volume ; un match sans la ressource ou sans film n’a pas de point.',
      endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nCumul du périmètre : ${cumUs} sur ${cumTotal} (${pct})`,
    },
    production: {
      ...base.production,
      info:
        'Barre épaisse : part de l’équipe dans les frags obtenus avec chaque ressource ; barre fine : part de ' +
        'l’équipe dans l’exposition (temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un ' +
        'véhicule). Périmètre : les matchs où l’exposition est mesurée.',
    },
    yield: {
      ...base.yield,
      info:
        'Écart relatif entre les frags de l’équipe et ceux de l’adversaire pour la même exposition : par minute ' +
        'd’effet d’un bonus, par prise d’arme spéciale, par minute à bord d’un véhicule. Zéro : autant ; ' +
        'rendements bruts de l’autre côté du zéro.',
    },
    grid: {
      ...base.grid,
      title: 'Contrôle des ressources, par carte',
      info:
        'Une colonne par carte jouée, la plus jouée à gauche, avec ses matchs et ses résultats ; couleur : ' +
        'écart entre les prises de l’équipe et celles de l’adversaire, saturée à trente points.',
      noTeamTip:
        'Équipe inconnue sur ces matchs (chacun pour soi, ou équipe absente de la feuille de match) : rien ne se ' +
        'partage entre les deux équipes.',
      untieredTip: 'Niveaux de socle non mesurés sur ces matchs : armes spéciales et armes de râtelier ne se séparent pas.',
    },
    vehicles: { ...base.vehicles, unmeasuredTip: 'Véhicules non mesurés : l’occupation n’a pas été lue.' },
  }
}

/** Les textes de l'Emprise de l'Escouade, surchargés pour la page solo (EN). */
function empriseEn(base: EmpriseText): EmpriseText {
  return {
    ...base,
    control: {
      ...base.control,
      info:
        'Pickups of each resource by the team and by the opponent, in counts, over the filmed matches in ' +
        'scope; orange line: 50%. Power-ups with no known picker count for neither team.',
    },
    fil: {
      ...base.fil,
      info:
        'The team’s share of each resource’s pickups, cumulated match after match over the scope. Dots: each ' +
        'match’s share, sized by volume; a match without the resource or without a film has no dot.',
      endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nScope total: ${cumUs} of ${cumTotal} (${pct})`,
    },
    production: {
      ...base.production,
      info:
        'Thick bar: the team’s share of the kills made with each resource; thin bar: the team’s share of the ' +
        'exposure (effect time for a power-up, pickups for a power weapon, time aboard a vehicle). Scope: the ' +
        'matches where the exposure is measured.',
    },
    yield: {
      ...base.yield,
      info:
        'Relative gap between the team’s kills and the opponent’s for the same exposure: per minute of ' +
        'power-up effect, per power weapon pickup, per minute aboard a vehicle. Zero: even; raw rates on the ' +
        'other side of zero.',
    },
    grid: {
      ...base.grid,
      title: 'Resource control, by map',
      info:
        'One column per map played, the most played on the left, with its matches and results; colour: gap ' +
        'between the team’s pickups and the opponent’s, saturated at thirty points.',
      noTeamTip:
        'Team unknown in these matches (free-for-all, or team missing from the match sheet): nothing splits ' +
        'between the two teams.',
      untieredTip: 'Pad levels not measured for these matches: power weapons and rack weapons can’t be told apart.',
    },
    vehicles: { ...base.vehicles, unmeasuredTip: 'Vehicles not measured: occupancy was not read.' },
  }
}

function emprise(locale: Locale): EmpriseText {
  const base = EMPRISE_TEXT[locale]
  return locale === 'fr' ? empriseFr(base) : empriseEn(base)
}

const FR: UsagesText = {
  sections: {
    bilan: 'Ressources',
    carte: 'Par carte',
    mine: 'Prises',
    // Pas « Rendement » seul : c'est le libellé du champ `offensive_conversion` (fields.toml), une
    // autre grandeur — lint-no-hardcoded-fields.
    prendre: 'Rendement des ressources',
    lives: 'Isolement',
    objectif: 'Objectif',
    equipment: 'Équipement',
    bilanCoverage: (filmed, total) =>
      `${filmed} ${sur(filmed, 'match filmé', 'matchs filmés')} sur ${total} · frags de la feuille de match sur les ${total}`,
  },
  cards: USAGES_CARDS_TEXT_FR,
  sheet: {
    title: 'Part du joueur à l’objectif',
    info:
      'Actions de l’objectif du joueur, rôle par rôle ; barre : part du total de l’équipe, un zéro reste affiché, ' +
      'atténué. Rôle dominant : celui où la part du joueur dans l’équipe est la plus forte.',
    dominantRole: OBJECTIF_TEXT.fr.sheets.dominantRole,
    roles: OBJECTIF_TEXT.fr.roles,
    pctFmt: (v) => OBJECTIF_TEXT.fr.pctFmt(v, 1),
    durationFmt: OBJECTIF_TEXT.fr.durationFmt,
    lineTip: (player, family, column, value, camp, pct) =>
      `${player} · ${family}\n${column} : ${value} des ${camp} de l’équipe${pct ? ` (${pct})` : ''}`,
  },
}

const EN: UsagesText = {
  sections: {
    bilan: 'Resources',
    carte: 'By map',
    mine: 'Pickups',
    prendre: 'Resource efficiency',
    lives: 'Isolation',
    objectif: 'Objective',
    equipment: 'Equipment',
    bilanCoverage: (filmed, total) =>
      `${filmed} ${sur(filmed, 'filmed match', 'filmed matches')} of ${total} · kills from the match sheet over all ${total}`,
  },
  cards: USAGES_CARDS_TEXT_EN,
  sheet: {
    title: 'Player’s share of the objective',
    info:
      'The player’s objective actions, role by role; bar: share of the team’s total, a zero stays, dimmed. ' +
      'Dominant role: the one where the player’s share of the team is highest.',
    dominantRole: OBJECTIF_TEXT.en.sheets.dominantRole,
    roles: OBJECTIF_TEXT.en.roles,
    pctFmt: (v) => OBJECTIF_TEXT.en.pctFmt(v, 1),
    durationFmt: OBJECTIF_TEXT.en.durationFmt,
    lineTip: (player, family, column, value, camp, pct) =>
      `${player} · ${family}\n${column}: ${value} of the team’s ${camp}${pct ? ` (${pct})` : ''}`,
  },
}

export const EMPRISE_TEXT_SOLO: Record<Locale, EmpriseText> = { fr: emprise('fr'), en: emprise('en') }
/** L'Objectif solo lit les textes de l'Escouade tels quels (« Rapport de force », « Équipe »). */
export const OBJECTIF_TEXT_SOLO: Record<Locale, ObjectifText> = OBJECTIF_TEXT
export const USAGES_TEXT: Record<Locale, UsagesText> = { fr: FR, en: EN }
