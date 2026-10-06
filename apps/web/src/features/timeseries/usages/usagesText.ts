/**
 * usagesText.ts — LES TEXTES de l'onglet « Usages » des Séries temporelles (plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, décision D10 ; maquette v4
 * `.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html`, vue « Après »).
 *
 * L'onglet monte les cartes de l'Emprise et de l'Objectif de l'Escouade : leurs textes sont ceux de
 * l'Escouade SURCHARGÉS pour une page solo (`EMPRISE_TEXT_SOLO`, `OBJECTIF_TEXT_SOLO`) — « Mon camp »,
 * jamais « Notre camp » (V6), « matchs du périmètre » au lieu de « la soirée ». Les cartes propres à
 * l'onglet (grille par carte, Mes prises, Mes vies, Équipement, Ma part) et les intertitres de
 * l'onglet (seule source : `USAGES_TEXT[locale].sections`) prennent `USAGES_TEXT`.
 * Titres, aides ⓘ et légendes FR : ceux de la maquette, mot pour mot. Parité FR / EN par le typage
 * `Record<Locale, …>`.
 */
import { EMPRISE_TEXT, type EmpriseText } from '@/features/squad/emprise/empriseStrings'
import type { SoloSheetText } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { OBJECTIF_TEXT, type ObjectifText } from '@/features/squad/objectif/objectifStrings'
import type { Locale } from '@/lib/i18n/locale'

import { USAGES_CARDS_TEXT_EN, USAGES_CARDS_TEXT_FR, type UsagesCardsText } from './usagesCardsText'

export type { UsagesCardsText }

/** Les intertitres de l'onglet, dans l'ordre de la page (« Portée des engagements » est porté par sa section). */
export interface UsagesSectionsText {
  bilan: string
  carte: string
  mine: string
  prendre: string
  lives: string
  objectif: string
  equipment: string
  /** « 12 matchs filmés sur 40 · frags de la feuille de match sur les 40 », à côté de « Bilan du périmètre ». */
  bilanCoverage: (filmed: number, total: number) => string
}

export interface UsagesText {
  sections: UsagesSectionsText
  cards: UsagesCardsText
  sheet: SoloSheetText
}

const sur = (n: number, one: string, many: string) => (n > 1 ? many : one)

/** Une décimale, séparateur de la langue (« 3,0 », « 3.0 »). */
const dec1 = (v: number, tag: string) => v.toLocaleString(tag, { minimumFractionDigits: 1, maximumFractionDigits: 1 })

/** Les textes de l'Emprise de l'Escouade, surchargés pour la page solo (FR). */
function empriseFr(base: EmpriseText): EmpriseText {
  return {
    ...base,
    ourSide: 'Mon camp',
    control: {
      ...base.control,
      info:
        'La part de chaque ressource prise par mon camp face à l’adversaire, sur les matchs du périmètre. ' +
        'Les nombres sont des comptes ; le trait orange marque 50 %, autant que l’adversaire. Les bonus ' +
        'sans ramasseur connu ne comptent dans aucun camp.',
      ariaLabel: 'La part de mon camp des prises de chaque ressource, face à l’adversaire',
    },
    fil: {
      ...base.fil,
      title: 'Contrôle des ressources au fil des matchs',
      info:
        'Ma part des prises de chaque ressource, cumulée depuis le premier match du périmètre. Les petits ' +
        'points sont la part de chaque match, leur taille son volume. Un match sans la ressource, ou sans ' +
        'film, laisse la courbe filer jusqu’au suivant.',
      pointTip: (v) =>
        `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource} : ${v.us} pour mon camp, ${v.them} pour l’adversaire (${v.pct})\n` +
        `Cumul : ${v.cumUs} sur ${v.cumTotal} (${v.cumPct})`,
      endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nCumul du périmètre : ${cumUs} sur ${cumTotal} (${pct})`,
    },
    production: {
      ...base.production,
      info:
        'La barre épaisse partage les frags obtenus grâce à la ressource, la barre fine ce qui les a permis ' +
        '(temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un véhicule), toutes deux sur ' +
        'les matchs où ce qui les a permis est mesuré. Si la coupure de la barre épaisse est à gauche de ' +
        'celle de la fine, on a moins produit qu’on n’a eu. Les frags de toute la période se lisent carte par carte.',
      ariaLabel: 'La part de mon camp des frags obtenus avec chaque ressource, et de ce qui les a permis',
    },
    yield: {
      ...base.yield,
      info:
        'Combien mon camp produit de plus ou de moins que l’adversaire pour la même exposition : par minute ' +
        'd’effet d’un bonus, par prise d’arme spéciale, par minute à bord d’un véhicule. Zéro veut dire autant ' +
        'que lui. Les deux rendements bruts sont écrits de l’autre côté du zéro.',
      ariaLabel: 'Le rendement de mon camp face à celui de l’adversaire, par ressource',
      tip: (resource, sub, us, them, gap) =>
        `${resource} · ${sub}\nMon camp ${dec1(us, 'fr-FR')}, adversaire ${dec1(them, 'fr-FR')} : ${gap}`,
    },
    grid: {
      ...base.grid,
      title: 'Contrôle des ressources, carte par carte',
      info:
        'Une colonne par carte jouée, la plus jouée à gauche, avec son nombre de matchs et ses résultats. La ' +
        'couleur dit si mon camp a pris plus ou moins que l’adversaire, et sature à trente points d’écart. Le ' +
        'survol d’une case détaille qui l’a prise chez moi.',
      noTeamTip:
        'Mon camp est inconnu sur ces matchs (chacun pour soi, ou camp absent de la feuille de match) : rien ' +
        'ne se partage entre les deux camps.',
      untieredTip: 'Niveaux de socle non mesurés sur ces matchs : armes spéciales et armes de râtelier ne se séparent pas.',
      cellTip: (name, us, them, pct) => `${name} : ${us} pour mon camp, ${them} pour l’adversaire (${pct})`,
      whoFmt: (list) => `Chez moi : ${list}`,
    },
    vehicles: { ...base.vehicles, unmeasuredTip: 'Véhicules non mesurés : l’occupation n’a pas été lue.' },
  }
}

/** Les textes de l'Emprise de l'Escouade, surchargés pour la page solo (EN). */
function empriseEn(base: EmpriseText): EmpriseText {
  return {
    ...base,
    ourSide: 'My side',
    control: {
      ...base.control,
      info:
        'The share of each resource my side picked up against the opponent, over the matches in scope. The ' +
        'numbers are counts; the orange line marks 50%, as much as the opponent. Power-ups with no known ' +
        'picker count for neither side.',
      ariaLabel: 'My side’s share of each resource’s pickups, against the opponent',
    },
    fil: {
      ...base.fil,
      title: 'Resource control over the matches',
      info:
        'My share of each resource’s pickups, cumulated from the first match in scope. The small dots are ' +
        'each match’s share, their size its volume. A match without the resource, or without a film, lets ' +
        'the line run on to the next one.',
      pointTip: (v) =>
        `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource}: ${v.us} for my side, ${v.them} for the opponent (${v.pct})\n` +
        `Cumulated: ${v.cumUs} of ${v.cumTotal} (${v.cumPct})`,
      endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nScope total: ${cumUs} of ${cumTotal} (${pct})`,
    },
    production: {
      ...base.production,
      info:
        'The thick bar splits the kills the resource brought, the thin bar what made them possible (effect ' +
        'time for a power-up, pickups for a power weapon, time aboard a vehicle), both over the matches where ' +
        'the latter is measured. If the thick bar’s split sits left of the thin one’s, we produced less than ' +
        'we had. Kills over the whole period read map by map.',
      ariaLabel: 'My side’s share of the kills made with each resource, and of what made them possible',
    },
    yield: {
      ...base.yield,
      info:
        'How much more or less my side produces than the opponent for the same exposure: per minute of ' +
        'power-up effect, per power weapon pickup, per minute aboard a vehicle. Zero means as much as them. ' +
        'Both raw rates are written on the other side of zero.',
      ariaLabel: 'My side’s efficiency against the opponent’s, per resource',
      tip: (resource, sub, us, them, gap) =>
        `${resource} · ${sub}\nMy side ${dec1(us, 'en-GB')}, opponent ${dec1(them, 'en-GB')}: ${gap}`,
    },
    grid: {
      ...base.grid,
      title: 'Resource control, map by map',
      info:
        'One column per map played, the most played on the left, with its number of matches and its results. ' +
        'The colour says whether my side picked up more or less than the opponent, and saturates at a ' +
        'thirty-point gap. Hover a cell to see who took it on my side.',
      noTeamTip:
        'My side is unknown in these matches (free-for-all, or side missing from the match sheet): nothing ' +
        'splits between the two sides.',
      untieredTip: 'Pad levels not measured for these matches: power weapons and rack weapons can’t be told apart.',
      cellTip: (name, us, them, pct) => `${name}: ${us} for my side, ${them} for the opponent (${pct})`,
      whoFmt: (list) => `On my side: ${list}`,
    },
    vehicles: { ...base.vehicles, unmeasuredTip: 'Vehicles not measured: occupancy was not read.' },
  }
}

function emprise(locale: Locale): EmpriseText {
  const base = EMPRISE_TEXT[locale]
  return locale === 'fr' ? empriseFr(base) : empriseEn(base)
}

function objectif(locale: Locale): ObjectifText {
  const base = OBJECTIF_TEXT[locale]
  if (locale === 'fr') {
    return {
      ...base,
      ourSide: 'Mon camp',
      balance: {
        ...base.balance,
        info:
          'Pour chaque action de l’objectif, ce que mon camp a fait face à l’adversaire, famille par famille. ' +
          'Le trait orange marque 50 % : autant que l’adversaire.',
      },
    }
  }
  return {
    ...base,
    ourSide: 'My side',
    balance: {
      ...base.balance,
      info:
        'For each objective action, what my side did against the opponent, family by family. The orange ' +
        'line marks 50%: as much as the opponent.',
    },
  }
}

const FR: UsagesText = {
  sections: {
    bilan: 'Bilan du périmètre',
    carte: 'Carte par carte',
    mine: 'Mes prises',
    prendre: 'Prendre, et s’en servir',
    lives: 'Près d’un coéquipier ou seul',
    objectif: 'Objectif',
    equipment: 'Équipement',
    bilanCoverage: (filmed, total) =>
      `${filmed} ${sur(filmed, 'match filmé', 'matchs filmés')} sur ${total} · frags de la feuille de match sur les ${total}`,
  },
  cards: USAGES_CARDS_TEXT_FR,
  sheet: {
    title: 'Ma part à l’objectif',
    info:
      'La fiche du joueur affiché : ce qu’il a fait à l’objectif, action par action. La barre est sa part du ' +
      'total de son camp ; un zéro reste affiché, atténué. Le rôle dominant est celui où il pèse le plus dans ' +
      'son camp.',
    dominantRole: OBJECTIF_TEXT.fr.sheets.dominantRole,
    roles: OBJECTIF_TEXT.fr.roles,
    pctFmt: (v) => OBJECTIF_TEXT.fr.pctFmt(v, 1),
    durationFmt: OBJECTIF_TEXT.fr.durationFmt,
    lineTip: (player, family, column, value, camp, pct) =>
      `${player} · ${family}\n${column} : ${value} des ${camp} de mon camp${pct ? ` (${pct})` : ''}`,
  },
}

const EN: UsagesText = {
  sections: {
    bilan: 'Scope summary',
    carte: 'Map by map',
    mine: 'My pickups',
    prendre: 'Taking, and using',
    lives: 'Near a teammate or alone',
    objectif: 'Objective',
    equipment: 'Equipment',
    bilanCoverage: (filmed, total) =>
      `${filmed} ${sur(filmed, 'filmed match', 'filmed matches')} of ${total} · kills from the match sheet over all ${total}`,
  },
  cards: USAGES_CARDS_TEXT_EN,
  sheet: {
    title: 'My share of the objective',
    info:
      'The sheet of the player shown: what they did on the objective, action by action. The bar is their share ' +
      'of their side’s total; a zero stays, dimmed. The dominant role is the one where they weigh most in ' +
      'their side.',
    dominantRole: OBJECTIF_TEXT.en.sheets.dominantRole,
    roles: OBJECTIF_TEXT.en.roles,
    pctFmt: (v) => OBJECTIF_TEXT.en.pctFmt(v, 1),
    durationFmt: OBJECTIF_TEXT.en.durationFmt,
    lineTip: (player, family, column, value, camp, pct) =>
      `${player} · ${family}\n${column}: ${value} of my side’s ${camp}${pct ? ` (${pct})` : ''}`,
  },
}

export const EMPRISE_TEXT_SOLO: Record<Locale, EmpriseText> = { fr: emprise('fr'), en: emprise('en') }
export const OBJECTIF_TEXT_SOLO: Record<Locale, ObjectifText> = { fr: objectif('fr'), en: objectif('en') }
export const USAGES_TEXT: Record<Locale, UsagesText> = { fr: FR, en: EN }
