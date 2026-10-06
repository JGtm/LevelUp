/**
 * sessionEmpriseText.ts — LES TEXTES des cartes « Frags et usages » de la page Sessions (plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06, D12 ; maquette `.ai/V7.5/MAQUETTE_SESSIONS_2026-10-06.html`, vue
 * « Après », fonctions `make*`).
 *
 * Les cartes sont celles de l'Escouade et des Séries temporelles : leurs textes solo
 * (`EMPRISE_TEXT_SOLO`, `OBJECTIF_TEXT_SOLO`, `USAGES_TEXT`, `getSquadText`) SURCHARGÉS pour une
 * session — « de la soirée », jamais « du périmètre » ; « Mon camp », jamais « Notre camp » (V6).
 * DEUX JEUX par vue : la pleine page (`full`) et la vue compacte du tiroir de comparaison
 * (`compact`), quand la maquette distingue leurs ⓘ (drapeau `cp`) ; la vue compacte porte en plus
 * les formateurs propres à ses cartes (`compactCards`). Aucun type de texte existant ne change.
 * FR mot pour mot de la maquette ; parité FR / EN par le typage `Record<Locale, …>`.
 */
import type { ComponentProps } from 'react'

import { EMPRISE_TEXT, type EmpriseText } from '@/features/squad/emprise/empriseStrings'
import type { ProductionCard } from '@/features/squad/emprise/ProductionCard'
import { getSquadText, type SquadText } from '@/features/squad/i18n'
import type { SoloSheetText } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { OBJECTIF_TEXT, type ObjectifText } from '@/features/squad/objectif/objectifStrings'
import type { SquadFragBreakdownCard } from '@/features/squad/SquadFragBreakdownCard'
import type { EquipmentOutcomesCard } from '@/features/timeseries/usages/EquipmentOutcomesCard'
import type { LivesNearTeammateCard } from '@/features/timeseries/usages/LivesNearTeammateCard'
import type { MinePickupsCard } from '@/features/timeseries/usages/MinePickupsCard'
import {
  EMPRISE_TEXT_SOLO,
  OBJECTIF_TEXT_SOLO,
  USAGES_TEXT,
  type UsagesCardsText,
} from '@/features/timeseries/usages/usagesText'
import type { Locale } from '@/lib/i18n/locale'

type CompactOf<P extends { compact?: unknown }> = NonNullable<P['compact']>

/** Les formateurs propres aux cartes de la vue compacte (maquette, `cp = true`). */
export interface SessionCompactCards {
  frag: CompactOf<ComponentProps<typeof SquadFragBreakdownCard>>
  production: CompactOf<ComponentProps<typeof ProductionCard>>
  mine: CompactOf<ComponentProps<typeof MinePickupsCard>>
  equipment: CompactOf<ComponentProps<typeof EquipmentOutcomesCard>>
  lives: CompactOf<ComponentProps<typeof LivesNearTeammateCard>>
}

/** Les textes d'une vue de la page : chaque carte reçoit celui de sa vue. */
export interface SessionCardTexts {
  /** A et B : « Répartition des frags », « Outils de destruction » (textes de l'Escouade surchargés). */
  squad: SquadText
  /** C à H, L (légendes) : les textes de l'Emprise. */
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

interface Variant {
  fragInfo: string
  toolsInfo: string
  controlInfo: string
  gridInfo: string
  gridLegend?: { more: string; less: string; noFilm: string }
  balanceInfo: string
  sheetInfo: string
  mineInfo: string
  equipmentInfo: string
}

interface LocaleOverrides {
  filInfo: string
  productionInfo: string
  noTeamTip: string
  full: Variant
  compact: Variant
  compactCards: SessionCompactCards
}

const plural = (n: number, one: string, many: string) => (n > 1 ? many : one)

const FR_EQUIPMENT_TAIL =
  'Les comptes portent sur tout l’équipement tenu, celui de réapparition compris ; le sous-libellé dit combien ' +
  'en ont été pris sur la carte. La barre fine donne les mêmes trois parts pour le reste de mon camp. Le répulseur ' +
  'n’a pas de ligne : aucun canal ne mesure son usage.'
const EN_EQUIPMENT_TAIL =
  'The counts cover all equipment held, spawn equipment included; the sub-label says how many were picked up on ' +
  'the map. The thin bar gives the same three shares for the rest of my side. The repulsor has no row: no channel ' +
  'measures its use.'
const FR_GRAB_NOTE = 'Le trait orange marque 50 % : autant que l’adversaire. Les prises nettes de drapeau (lues dans le film) ne comptent pas les jonglages.'
const EN_GRAB_NOTE = 'The orange line marks 50%: as much as the opponent. Net flag grabs (read from the film) don’t count juggling.'
const FR_SHEET_HEAD = 'La fiche du joueur affiché : ce qu’il a fait à l’objectif, action par action. La barre est sa part du total de son camp'
const FR_SHEET_TAIL = ' ; un zéro reste affiché, atténué. Le rôle dominant est celui où il pèse le plus dans son camp.'
const EN_SHEET_HEAD = 'The sheet of the player shown: what they did on the objective, action by action. The bar is their share of their side’s total'
const EN_SHEET_TAIL = '; a zero stays, dimmed. The dominant role is the one where they weigh most in their side.'
const FR_CONTROL_HEAD = 'La part de chaque ressource prise par mon camp face à l’adversaire, sur les matchs de la soirée. '
const FR_CONTROL_TAIL = ' Le trait orange marque 50 %, autant que l’adversaire. Les bonus sans ramasseur connu ne comptent dans aucun camp.'
const EN_CONTROL_HEAD = 'The share of each resource my side picked up against the opponent, over the evening’s matches. '
const EN_CONTROL_TAIL = ' The orange line marks 50%, as much as the opponent. Power-ups with no known picker count for neither side.'

const FR: LocaleOverrides = {
  filInfo:
    'Ma part des prises de chaque ressource, cumulée depuis le premier match de la soirée. Les petits points sont ' +
    'la part de chaque match, leur taille son volume. Un match sans la ressource, ou sans film, laisse la courbe ' +
    'filer jusqu’au suivant.',
  productionInfo:
    'La barre épaisse partage les frags obtenus grâce à la ressource, la barre fine ce qui les a permis (temps ' +
    'd’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un véhicule), toutes deux sur les matchs où ce ' +
    'qui les a permis est mesuré. Si la coupure de la barre épaisse est à gauche de celle de la fine, on a moins ' +
    'produit qu’on n’a eu.',
  noTeamTip:
    'Mon camp est inconnu sur ce match (chacun pour soi, ou camp absent de la feuille de match) : rien ne se partage ' +
    'entre les deux camps.',
  full: {
    fragInfo: 'Mes frags de la soirée, par classe d’arme. Le nombre écrit dans un segment est son compte de frags ; le total est au bout de la barre.',
    toolsInfo: 'Mes frags, arme par arme, sur la soirée. La pastille devant l’arme est la couleur de sa classe dans la Répartition des frags.',
    controlInfo: `${FR_CONTROL_HEAD}Les nombres sont des comptes.${FR_CONTROL_TAIL}`,
    gridInfo:
      'Une colonne par match de la soirée, dans l’ordre, avec sa carte, son mode et son résultat. La couleur dit si ' +
      'mon camp a pris plus ou moins que l’adversaire, et sature à trente points d’écart. Le survol d’une case ' +
      'détaille qui l’a prise chez moi.',
    balanceInfo: `Pour chaque action de l’objectif, ce que mon camp a fait face à l’adversaire, famille par famille. ${FR_GRAB_NOTE}`,
    sheetInfo: `${FR_SHEET_HEAD}${FR_SHEET_TAIL}`,
    mineInfo: USAGES_TEXT.fr.cards.mine.info,
    equipmentInfo: `Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les autres), gardés sans servir, lâchés. ${FR_EQUIPMENT_TAIL}`,
  },
  compact: {
    fragInfo: 'Mes frags de la soirée, par classe d’arme. Chaque segment porte sa part de mes frags ; le compte est au survol.',
    toolsInfo:
      'Mes six outils les plus meurtriers de la soirée, en part de mes frags. La pastille devant l’arme est la ' +
      'couleur de sa classe ; le compte est au survol.',
    controlInfo: `${FR_CONTROL_HEAD}Les segments portent les parts ; les comptes sont au survol.${FR_CONTROL_TAIL}`,
    gridInfo:
      'Une colonne par match de la soirée. Chaque case est la part de mon camp dans la ressource (plus que ' +
      'l’adversaire en vert, moins en rouge, saturation à trente points d’écart) ; les comptes et qui l’a prise ' +
      'chez moi sont au survol.',
    gridLegend: { more: 'Plus de 50 %', less: 'Moins de 50 %', noFilm: 'Sans film, non mesuré' },
    balanceInfo:
      'Pour chaque rôle de l’objectif (somme de ses actions ; Tenir en durée), la part de mon camp face à ' +
      `l’adversaire, famille par famille ; les comptes sont au survol. ${FR_GRAB_NOTE}`,
    sheetInfo: `${FR_SHEET_HEAD}, et le nombre à droite aussi (compte au survol)${FR_SHEET_TAIL}`,
    mineInfo:
      'Pour chaque ressource, ma part des prises de mon camp et celle du reste du camp, en pourcentage ; les ' +
      'comptes sont au survol. Les bonus perdus sont ceux gardés sans être activés ou lâchés, en part des bonus ' +
      'pris par chaque camp.',
    equipmentInfo: `Pour chaque famille, ce que sont devenus mes objets : servis (posé pour le mur, charge consommée pour les autres), gardés sans servir, lâchés, en part de mes objets (comptes au survol). ${FR_EQUIPMENT_TAIL}`,
  },
  compactCards: {
    frag: { totalSub: (n) => `${n} ${plural(n, 'frag', 'frags')}`, pctFmt: (v) => `${Math.round(v)} %` },
    production: { exposureLine: (name, pct) => `${name} : ${pct}` },
    mine: { resourceSub: 'prises de mon camp' },
    equipment: {
      sub: (n) => (n > 0 ? `${n} ${plural(n, 'objet', 'objets')}` : '0 objet'),
      unmeasured: 'Non mesuré',
      restUsed: (pct) => `reste de mon camp : ${pct} servis`,
    },
    lives: {
      killsLine: (pct, perLife) => `frags : ${pct} · ${perLife} par vie`,
      killsLineAlone: (perLife) => `${perLife} par vie`,
    },
  },
}

const EN: LocaleOverrides = {
  filInfo:
    'My share of each resource’s pickups, cumulated from the evening’s first match. The small dots are each ' +
    'match’s share, their size its volume. A match without the resource, or without a film, lets the line run on ' +
    'to the next one.',
  productionInfo:
    'The thick bar splits the kills the resource brought, the thin bar what made them possible (effect time for a ' +
    'power-up, pickups for a power weapon, time aboard a vehicle), both over the matches where the latter is ' +
    'measured. If the thick bar’s split sits left of the thin one’s, we produced less than we had.',
  noTeamTip:
    'My side is unknown in this match (free-for-all, or side missing from the match sheet): nothing splits between ' +
    'the two sides.',
  full: {
    fragInfo: 'My kills over the evening, by weapon class. The number written in a segment is its kill count; the total sits at the end of the bar.',
    toolsInfo: 'My kills, weapon by weapon, over the evening. The swatch before the weapon is the colour of its class in the Kill type distribution.',
    controlInfo: `${EN_CONTROL_HEAD}The numbers are counts.${EN_CONTROL_TAIL}`,
    gridInfo:
      'One column per match of the evening, in order, with its map, its mode and its result. The colour says ' +
      'whether my side picked up more or less than the opponent, and saturates at a thirty-point gap. Hover a cell ' +
      'to see who took it on my side.',
    balanceInfo: `For each objective action, what my side did against the opponent, family by family. ${EN_GRAB_NOTE}`,
    sheetInfo: `${EN_SHEET_HEAD}${EN_SHEET_TAIL}`,
    mineInfo: USAGES_TEXT.en.cards.mine.info,
    equipmentInfo: `For each family, what became of my items: used (placed for the wall, charge spent for the others), kept without use, dropped. ${EN_EQUIPMENT_TAIL}`,
  },
  compact: {
    fragInfo: 'My kills over the evening, by weapon class. Each segment carries its share of my kills; the count shows on hover.',
    toolsInfo:
      'My six deadliest tools of the evening, as a share of my kills. The swatch before the weapon is the colour of ' +
      'its class; the count shows on hover.',
    controlInfo: `${EN_CONTROL_HEAD}The segments carry the shares; the counts show on hover.${EN_CONTROL_TAIL}`,
    gridInfo:
      'One column per match of the evening. Each cell is my side’s share of the resource (more than the opponent ' +
      'in green, less in red, saturating at a thirty-point gap); the counts and who took it on my side show on hover.',
    gridLegend: { more: 'Over 50%', less: 'Under 50%', noFilm: 'No film, not measured' },
    balanceInfo:
      'For each objective role (sum of its actions; Hold as a duration), my side’s share against the opponent, ' +
      `family by family; the counts show on hover. ${EN_GRAB_NOTE}`,
    sheetInfo: `${EN_SHEET_HEAD}, and so is the number on the right (count on hover)${EN_SHEET_TAIL}`,
    mineInfo:
      'For each resource, my share of my side’s pickups and the rest of the side’s, as a percentage; the counts ' +
      'show on hover. Lost power-ups are those held without being activated, or dropped, as a share of the ' +
      'power-ups each side picked up.',
    equipmentInfo: `For each family, what became of my items: used (placed for the wall, charge spent for the others), kept without use, dropped, as a share of my items (counts on hover). ${EN_EQUIPMENT_TAIL}`,
  },
  compactCards: {
    frag: { totalSub: (n) => `${n} ${plural(n, 'kill', 'kills')}`, pctFmt: (v) => `${Math.round(v)}%` },
    production: { exposureLine: (name, pct) => `${name}: ${pct}` },
    mine: { resourceSub: 'my side’s pickups' },
    equipment: {
      sub: (n) => (n > 0 ? `${n} ${plural(n, 'item', 'items')}` : '0 items'),
      unmeasured: 'Not measured',
      restUsed: (pct) => `rest of my side: ${pct} used`,
    },
    lives: {
      killsLine: (pct, perLife) => `kills: ${pct} · ${perLife} per life`,
      killsLineAlone: (perLife) => `${perLife} per life`,
    },
  },
}

const OVERRIDES: Record<Locale, LocaleOverrides> = { fr: FR, en: EN }

function empriseFor(locale: Locale, o: LocaleOverrides, v: Variant): EmpriseText {
  const solo = EMPRISE_TEXT_SOLO[locale]
  const squad = EMPRISE_TEXT[locale]
  return {
    ...solo,
    control: { ...solo.control, info: v.controlInfo },
    // Titre et cumul de fin « de la soirée » : ceux de l'Escouade ; le reste, celui du solo (« mon camp »).
    fil: { ...solo.fil, title: squad.fil.title, info: o.filInfo, endTip: squad.fil.endTip },
    production: { ...solo.production, info: o.productionInfo },
    grid: {
      ...solo.grid,
      ...v.gridLegend,
      title: squad.grid.title,
      info: v.gridInfo,
      noTeamTip: o.noTeamTip,
      untieredTip: squad.grid.untieredTip,
    },
  }
}

function textsFor(locale: Locale, view: 'full' | 'compact'): SessionCardTexts {
  const o = OVERRIDES[locale]
  const v = o[view]
  const squad = getSquadText(locale)
  const usages = USAGES_TEXT[locale]
  const objectif = OBJECTIF_TEXT_SOLO[locale]
  return {
    squad: {
      ...squad,
      performanceCharts: { ...squad.performanceCharts, fragBreakdownInfo: v.fragInfo },
      weaponKills: { ...squad.weaponKills, info: v.toolsInfo },
    },
    emprise: empriseFor(locale, o, v),
    objectif: { ...objectif, balance: { ...objectif.balance, info: v.balanceInfo } },
    cards: {
      ...usages.cards,
      mine: { ...usages.cards.mine, info: v.mineInfo },
      equipment: { ...usages.cards.equipment, info: v.equipmentInfo },
    },
    sheet: {
      ...usages.sheet,
      info: v.sheetInfo,
      // Vue compacte : la part entière (« 32 % »), comme les autres cartes de la vue.
      ...(view === 'compact' ? { pctFmt: (x: number) => OBJECTIF_TEXT[locale].pctFmt(x, 0) } : {}),
    },
    coverage: usages.sections.bilanCoverage,
  }
}

/** Les textes de la page Sessions, par langue et par vue. */
export const SESSION_CARD_TEXT: Record<Locale, { full: SessionCardTexts; compact: SessionCardTexts; compactCards: SessionCompactCards }> = {
  fr: { full: textsFor('fr', 'full'), compact: textsFor('fr', 'compact'), compactCards: FR.compactCards },
  en: { full: textsFor('en', 'full'), compact: textsFor('en', 'compact'), compactCards: EN.compactCards },
}
