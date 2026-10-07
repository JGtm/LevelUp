/**
 * sessionEmpriseText.ts — LES TEXTES des cartes « Frags et usages » de la page Sessions (plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06, D12 ; maquette `.ai/V7.5/MAQUETTE_SESSIONS_2026-10-06.html`, vue
 * « Après », fonctions `make*`).
 *
 * Les cartes sont celles de l'Escouade et des Séries temporelles, et leurs textes aussi, sans variante :
 * l'Emprise de l'Escouade (`EMPRISE_TEXT`, portée « la soirée »), l'Objectif (`OBJECTIF_TEXT_SOLO`), les
 * cartes et la fiche des Séries temporelles (`USAGES_TEXT`). Seuls changent : l'aide des deux cartes de
 * frags (celle de l'Escouade parle de chacun des joueurs, la page n'en montre qu'un), et, dans la vue
 * compacte du tiroir de comparaison, les aides des cartes qui y écrivent des parts au lieu des comptes
 * (drapeau `cp` de la maquette), plus les formateurs propres à ces cartes (`compactCards`).
 * Textes factuels, sans personne : le joueur est désigné par son gamertag, « Équipe », « Adversaire »,
 * « Reste de l'équipe » (garde : `timeseries/usages/textesSansPersonne.test.ts`). Parité FR / EN par le
 * typage `Record<Locale, …>`.
 */
import type { ComponentProps } from 'react'

import { EMPRISE_TEXT, type EmpriseText } from '@/features/squad/emprise/empriseStrings'
import type { ProductionCard } from '@/features/squad/emprise/ProductionCard'
import { getSquadText, type SquadText } from '@/features/squad/i18n'
import type { SoloSheetText } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { OBJECTIF_TEXT, type ObjectifText } from '@/features/squad/objectif/objectifStrings'
import type { EquipmentOutcomesCard } from '@/features/timeseries/usages/EquipmentOutcomesCard'
import type { LivesNearTeammateCard } from '@/features/timeseries/usages/LivesNearTeammateCard'
import type { MinePickupsCard } from '@/features/timeseries/usages/MinePickupsCard'
import { OBJECTIF_TEXT_SOLO, USAGES_TEXT, type UsagesCardsText } from '@/features/timeseries/usages/usagesText'
import type { Locale } from '@/lib/i18n/locale'

type CompactOf<P extends { compact?: unknown }> = NonNullable<P['compact']>

/** Les formateurs propres aux cartes de la vue compacte (maquette, `cp = true`). */
export interface SessionCompactCards {
  production: CompactOf<ComponentProps<typeof ProductionCard>>
  mine: CompactOf<ComponentProps<typeof MinePickupsCard>>
  equipment: CompactOf<ComponentProps<typeof EquipmentOutcomesCard>>
  lives: CompactOf<ComponentProps<typeof LivesNearTeammateCard>>
}

/** Les textes d'une vue de la page : chaque carte reçoit celui de sa vue. */
export interface SessionCardTexts {
  /** A et B : « Répartition des frags », « Outils de destruction » (textes de l'Escouade, aide propre). */
  squad: SquadText
  /** C à H, L (légendes) : les textes de l'Emprise de l'Escouade. */
  emprise: EmpriseText
  /** J. */
  objectif: ObjectifText
  /** F, I, L. */
  cards: UsagesCardsText
  /** K. */
  sheet: SoloSheetText
  /** « n matchs filmés sur N · frags de la feuille de match sur les N » (intertitre des ressources). */
  coverage: (filmed: number, total: number) => string
}

/** Les aides propres à chaque vue : « Outils de destruction » (B) et « Contribution aux prises » (F). */
interface ViewInfos {
  toolsInfo: string
  /**
   * L'aide des Séries temporelles, portée « la soirée » au lieu du « périmètre » (décision V6 du plan) :
   * la même règle que l'Emprise de l'Escouade face à celle des Séries temporelles.
   */
  mineInfo: string
}

/** Les aides de la vue compacte, là où elle écrit des parts au lieu des comptes. */
interface CompactInfos extends ViewInfos {
  controlInfo: string
  gridInfo: string
  gridLegend: { more: string; less: string; noFilm: string }
  balanceInfo: string
  sheetInfo: string
  equipmentInfo: string
}

interface LocaleOverrides {
  /** L'aide de « Répartition des frags » (A) : le même anneau dans les deux vues. */
  fragInfo: string
  full: ViewInfos
  compact: CompactInfos
  compactCards: SessionCompactCards
}

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

const FR: LocaleOverrides = {
  fragInfo:
    'Frags du joueur sur la soirée : anneau intérieur par classe d’arme, anneau extérieur par rôle ; total au ' +
    'centre, part de chaque classe en légende.',
  full: {
    toolsInfo:
      'Frags du joueur sur la soirée, arme par arme ; couleur de la barre : classe de l’arme, celle de la Répartition des frags.',
    mineInfo:
      'Objets pris par l’équipe sur les matchs filmés de la soirée : part du joueur et du reste de l’équipe, en ' +
      'comptes, par volume décroissant. Bonus perdus : gardés sans être activés, ou lâchés.',
  },
  compact: {
    toolsInfo:
      'Les six armes les plus meurtrières du joueur sur la soirée, en part des frags du joueur ; couleur de la ' +
      'barre : classe de l’arme, compte au survol.',
    controlInfo:
      'Prises de chaque ressource par l’équipe et par l’adversaire, en parts (comptes au survol), sur les matchs ' +
      'filmés de la soirée ; trait orange : 50 %. Les bonus sans ramasseur connu ne comptent dans aucune équipe.',
    gridInfo:
      'Une colonne par match de la soirée ; case : part de l’équipe dans la ressource (vert au-dessus de ' +
      'l’adversaire, rouge en dessous, saturée à trente points d’écart), comptes et preneurs au survol.',
    gridLegend: { more: 'Plus de 50 %', less: 'Moins de 50 %', noFilm: 'Sans film, non mesuré' },
    balanceInfo:
      'Part de l’équipe face à l’adversaire pour chaque rôle de l’objectif (somme de ses actions ; Tenir en ' +
      'durée), par famille de mode, comptes au survol ; trait orange : 50 %.',
    sheetInfo:
      'Actions de l’objectif du joueur, rôle par rôle ; barre et nombre : part du total de l’équipe (compte au ' +
      'survol), un zéro reste affiché, atténué. Rôle dominant : celui où la part du joueur dans l’équipe est la ' +
      'plus forte.',
    mineInfo:
      'Prises de l’équipe par ressource sur les matchs filmés de la soirée : part du joueur et du reste de ' +
      'l’équipe, en pourcentage (comptes au survol). Bonus perdus : gardés sans être activés, ou lâchés, en part ' +
      'des bonus pris par chaque équipe.',
    equipmentInfo:
      'Équipement tenu par le joueur (réapparition comprise), par famille : servi (mur posé, charge consommée), ' +
      'gardé sans servir, lâché, en part des objets du joueur (comptes au survol) ; barre fine : reste de ' +
      'l’équipe. Seules les familles tenues dans le lobby sont listées ; le répulseur, sans mesure d’usage, n’a ' +
      'pas de ligne.',
  },
  compactCards: {
    production: { exposureLine: (name, pct) => `${name} : ${pct}` },
    mine: { resourceSub: 'prises de l’équipe' },
    equipment: {
      sub: (n) => (n > 0 ? `${n} ${plural(n, 'objet', 'objets')}` : '0 objet'),
      unmeasured: 'Non mesuré',
      restUsed: (pct) => `reste de l’équipe : ${pct} servis`,
    },
    lives: {
      killsLine: (pct, perLife) => `frags : ${pct} · ${perLife} par vie`,
      killsLineAlone: (perLife) => `${perLife} par vie`,
    },
  },
}

const EN: LocaleOverrides = {
  fragInfo:
    'The player’s kills over the evening: inner ring by weapon class, outer ring by role; total in the centre, ' +
    'each class’s share in the legend.',
  full: {
    toolsInfo:
      'The player’s kills over the evening, weapon by weapon; bar colour: the weapon’s class, as in the Kill type distribution.',
    mineInfo:
      'Items the team picked up over the session’s filmed matches: the player’s share and the rest of the ' +
      'team’s, in counts, by decreasing volume. Lost power-ups: held without being activated, or dropped.',
  },
  compact: {
    toolsInfo:
      'The player’s six deadliest weapons over the evening, as a share of the player’s kills; bar colour: ' +
      'the weapon’s class, count on hover.',
    controlInfo:
      'Pickups of each resource by the team and by the opponent, as shares (counts on hover), over the session’s ' +
      'filmed matches; orange line: 50%. Power-ups with no known picker count for neither team.',
    gridInfo:
      'One column per match of the session; cell: the team’s share of the resource (green above the opponent, ' +
      'red below, saturated at a thirty-point gap), counts and pickers on hover.',
    gridLegend: { more: 'Over 50%', less: 'Under 50%', noFilm: 'No film, not measured' },
    balanceInfo:
      'The team’s share against the opponent for each objective role (sum of its actions; Hold as a duration), ' +
      'by mode family, counts on hover; orange line: 50%.',
    sheetInfo:
      'The player’s objective actions, role by role; bar and number: share of the team’s total (count on hover), ' +
      'a zero stays, dimmed. Dominant role: the one where the player’s share of the team is highest.',
    mineInfo:
      'The team’s pickups by resource over the session’s filmed matches: the player’s share and the rest of the ' +
      'team’s, as a percentage (counts on hover). Lost power-ups: held without being activated, or dropped, as a ' +
      'share of the power-ups each team picked up.',
    equipmentInfo:
      'Equipment held by the player (spawn equipment included), by family: used (wall placed, charge spent), ' +
      'kept without use, dropped, as a share of the player’s items (counts on hover); thin bar: rest of the team. ' +
      'Only families held in the lobby are listed; the repulsor, with no usage measure, has no row.',
  },
  compactCards: {
    production: { exposureLine: (name, pct) => `${name}: ${pct}` },
    mine: { resourceSub: 'team pickups' },
    equipment: {
      sub: (n) => (n > 0 ? `${n} ${plural(n, 'item', 'items')}` : '0 items'),
      unmeasured: 'Not measured',
      restUsed: (pct) => `rest of the team: ${pct} used`,
    },
    lives: {
      killsLine: (pct, perLife) => `kills: ${pct} · ${perLife} per life`,
      killsLineAlone: (perLife) => `${perLife} per life`,
    },
  },
}

const OVERRIDES: Record<Locale, LocaleOverrides> = { fr: FR, en: EN }

/** L'Emprise de l'Escouade (portée « la soirée ») ; vue compacte : les aides des cartes en parts. */
function empriseFor(locale: Locale, c: CompactInfos | null): EmpriseText {
  const base = EMPRISE_TEXT[locale]
  if (c == null) return base
  return {
    ...base,
    control: { ...base.control, info: c.controlInfo },
    grid: { ...base.grid, ...c.gridLegend, info: c.gridInfo },
  }
}

function textsFor(locale: Locale, view: 'full' | 'compact'): SessionCardTexts {
  const o = OVERRIDES[locale]
  const own: ViewInfos = o[view]
  const c = view === 'compact' ? o.compact : null
  const squad = getSquadText(locale)
  const usages = USAGES_TEXT[locale]
  const objectif = OBJECTIF_TEXT_SOLO[locale]
  return {
    squad: {
      ...squad,
      performanceCharts: { ...squad.performanceCharts, fragBreakdownInfo: o.fragInfo },
      weaponKills: { ...squad.weaponKills, info: own.toolsInfo },
    },
    emprise: empriseFor(locale, c),
    objectif: c ? { ...objectif, balance: { ...objectif.balance, info: c.balanceInfo } } : objectif,
    cards: {
      ...usages.cards,
      mine: { ...usages.cards.mine, info: own.mineInfo },
      ...(c ? { equipment: { ...usages.cards.equipment, info: c.equipmentInfo } } : {}),
    },
    sheet: c
      ? {
          ...usages.sheet,
          info: c.sheetInfo,
          // Vue compacte : la part entière (« 32 % »), comme les autres cartes de la vue.
          pctFmt: (x: number) => OBJECTIF_TEXT[locale].pctFmt(x, 0),
        }
      : usages.sheet,
    coverage: usages.sections.bilanCoverage,
  }
}

/** Les textes de la page Sessions, par langue et par vue. */
export const SESSION_CARD_TEXT: Record<Locale, { full: SessionCardTexts; compact: SessionCardTexts; compactCards: SessionCompactCards }> = {
  fr: { full: textsFor('fr', 'full'), compact: textsFor('fr', 'compact'), compactCards: FR.compactCards },
  en: { full: textsFor('en', 'full'), compact: textsFor('en', 'compact'), compactCards: EN.compactCards },
}
