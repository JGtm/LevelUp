/**
 * equipmentUsageColumns.ts — LES COLONNES DE LA GRILLE DES USAGES, et surtout LEURS NOMS.
 *
 * EXTRAIT DE `equipmentUsageLogic.ts` LE 2026-08-25 (lot D du backlog Notion) : ce dernier
 * portait l'agrégation ET la mise en colonnes, et il arrivait à 498 lignes — le seuil du dépôt
 * est 500 (CLAUDE.md n°5). La découpe tombe sur une ligne nette : d'un côté ce que le document
 * MESURE, de l'autre ce que l'écran en DIT. Le premier ne connaît aucune langue ; celui-ci ne
 * compte rien.
 *
 * AUCUN NOM DE FAMILLE N'EST ÉCRIT ICI. Les deux tables du rejeu les portent déjà —
 * `placementFamily` (indexée par RÈGLE DE RENDU, pas par famille) et `padEquipmentFamily` (les
 * socles de bonus) — et les types de grenade viennent du CATALOGUE DU DOCUMENT, bilingue et cuit
 * au build. Une troisième table de noms divergerait au premier ajout du manifeste du titre.
 *
 * Pur : des fonctions de (mesures, textes) vers des colonnes. Aucun React.
 */
import { formatDurationMMSS } from '@/lib/formatters/duration'

import { catalogText } from './catalogLabel'
import { PLACEMENT_RENDER } from './equipmentPlacementsLayer'
import type { EquipmentUsage, EquipmentUsageTally } from './equipmentUsageLogic'
import type { ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import type { ReplayDocumentReady } from './replayNormalize'
import { padEquipmentFamilyOf } from './weaponPadFamilies'


/**
 * equipmentFamilyLabel — le nom d'une famille de pose, par la cascade des tables existantes.
 *
 * 1. socle de bonus (`powerup_*`) : `padEquipmentFamily` ;
 * 2. famille dessinée : sa RÈGLE DE RENDU a un libellé dans `placementFamily` ;
 * 3. famille dessinée en point neutre (`unnamed`) : « objet non identifié » ;
 * 4. sinon l'identifiant brut — la seule chose vraie qu'on puisse écrire d'une famille qu'aucune
 *    table ne nomme (même règle que les armes hors catalogue, cf. `padNameFor`).
 */
export function equipmentFamilyLabel(family: string, t: ReplayText): string {
  const powerup = padEquipmentFamilyOf(family)
  if (powerup) return t.padEquipmentFamily[powerup]
  const kind = PLACEMENT_RENDER[family]
  if (kind && kind !== 'unnamed' && kind !== 'dropped') return t.placementFamily[kind]
  if (kind === 'unnamed') return t.placementUnnamedLabel
  return family
}

/** Le nom d'un type de grenade : le catalogue bilingue du document, sinon son RANG. */
export function grenadeTypeLabel(
  doc: ReplayDocumentReady,
  rank: number,
  t: ReplayText,
  locale: ReplayLocale,
): string {
  return catalogText(doc.grenadeLabels[rank], locale) ?? t.equipmentUsage.grenadeRankFmt(rank)
}

/**
 * LES CINQ FAMILLES DE GESTE, et il n'y en a pas une sixième. Ces clés sont l'axe de
 * regroupement de tout ce que la section montre : les colonnes de la grille, la couleur des
 * barres, et les lignes de la vue « part de chaque équipe ». Le typage les rend exhaustives —
 * une famille ajoutée ici force la table des encres à la peindre (cf. `equipmentUsageChart`).
 */
export type UsageGroupKey = 'grapple' | 'episodes' | 'deployed' | 'dropped' | 'grenades'

/**
 * Une colonne : son en-tête, sa VALEUR pour un compteur, et comment cette valeur s'écrit.
 *
 * LA COLONNE REND UN NOMBRE, PLUS UNE CHAÎNE (2026-09-03) : la section est devenue un GRAPHE,
 * et une barre se dessine sur une valeur. Le formatage reste ici — l'échelle de la colonne, sa
 * graduation et sa cellule doivent s'écrire avec la MÊME plume, sans quoi l'axe et la valeur
 * qu'il gradue ne parlent pas la même langue.
 *
 * `value` rend `null` pour une grandeur NON MESURÉE (cf. `killsValue`), jamais un zéro : un
 * zéro est une mesure, et l'écran ne doit pas les confondre.
 */
export interface UsageColumn {
  key: string
  label: string
  value: (tally: EquipmentUsageTally) => number | null
  /** Écrit une valeur de cette colonne (entier, m:ss…). Sert aussi aux graduations d'axe. */
  format: (value: number) => string
  /** La colonne porte une durée : sa graduation médiane ne s'arrondit pas à l'entier. */
  duration?: boolean
}

/** Un groupe de colonnes : l'en-tête de premier niveau et sa réserve de mesure. */
export interface UsageColumnGroup {
  key: UsageGroupKey
  label: string
  hint: string
  columns: UsageColumn[]
}

/** Une valeur entière. Zéro reste ZÉRO : c'est une mesure, pas une absence. */
function intValue(n: number | undefined): number {
  return n ?? 0
}

/** L'écriture d'un entier. */
function intFormat(n: number): string {
  return String(n)
}

/**
 * Une DURÉE cumulée, en SECONDES — l'unité de `formatDurationMMSS`, et celle de l'échelle.
 *
 * `formatDurationMMSS` rend son repli pour toute valeur nulle — juste pour une durée de
 * MATCH (un match de zéro seconde n'a pas eu lieu), faux ici : la colonne n'existe que
 * parce que la famille est mesurée sur ce match, et la colonne voisine (nombre d'épisodes)
 * écrit déjà « 0 » pour le même joueur. Un « — » à côté d'un « 0 » se lit « non mesuré »
 * alors que la mesure a bien eu lieu et vaut zéro. D'où le repli « 0:00 », qui sert aussi
 * de graduation de gauche à l'axe de la colonne.
 */
function durationValue(ms: number | undefined): number {
  return (ms ?? 0) / 1000
}
function durationFormat(seconds: number): string {
  return formatDurationMMSS(seconds, '0:00')
}

/**
 * Les FRAGS SOUS EFFET ACTIF. MÊME PRINCIPE QUE `intValue`, une exception : un match dont la
 * jointure n'a pas pu être tentée (`killsRead` faux) n'a RIEN de mesuré — la valeur est nulle,
 * la barre reste vide et la cellule écrit le repli de non-mesure, jamais un zéro qui se lirait
 * comme une mesure de zéro frag (PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.2,
 * EquipmentUsageCoverage.killsRead).
 */
function killsValue(kills: number | undefined, killsRead: boolean): number | null {
  return killsRead ? intValue(kills) : null
}

/**
 * usageColumnGroups — les groupes de colonnes que la donnée justifie, dans un ordre écrit.
 *
 * Le grappin d'abord (une activation), les états actifs ensuite (une durée), puis ce qui se pose
 * sur le terrain, ce qui y tombe, et enfin les lancers. Un groupe sans colonne n'est pas rendu.
 */
export function usageColumnGroups(
  usage: EquipmentUsage,
  doc: ReplayDocumentReady,
  t: ReplayText,
  locale: ReplayLocale,
): UsageColumnGroup[] {
  const u = t.equipmentUsage
  const groups: UsageColumnGroup[] = []
  if (usage.columns.grapple) {
    groups.push({
      key: 'grapple',
      label: u.groupGrapple,
      hint: u.groupGrappleHint,
      columns: [
        {
          key: 'pulls',
          label: u.groupGrapple,
          value: (x) => intValue(x.grapplePulls),
          format: intFormat,
        },
      ],
    })
  }
  if (usage.columns.episodes.length > 0) {
    // Lu UNE FOIS par appel, fermé sur les cellules : `killsRead` est une propriété du
    // MATCH entier (cf. EquipmentUsageCoverage), pas d'un joueur ni d'une famille.
    const killsRead = usage.coverage.killsRead
    groups.push({
      key: 'episodes',
      label: u.groupActive,
      hint: u.groupActiveHint,
      columns: usage.columns.episodes.flatMap((fam) => [
        {
          key: `${fam}.count`,
          label: `${u.activeFamily[fam]} (${u.activeCount})`,
          value: (x: EquipmentUsageTally) => intValue(x.episodes[fam]?.count),
          format: intFormat,
        },
        {
          key: `${fam}.ms`,
          label: `${u.activeFamily[fam]} (${u.activeDuration})`,
          value: (x: EquipmentUsageTally) => durationValue(x.episodes[fam]?.ms),
          format: durationFormat,
          duration: true,
        },
        {
          key: `${fam}.kills`,
          label: u.activeKillsFamily[fam],
          value: (x: EquipmentUsageTally) => killsValue(x.episodes[fam]?.kills, killsRead),
          format: intFormat,
        },
      ]),
    })
  }
  for (const [key, families, label, hint, pick] of [
    ['deployed', usage.columns.deployed, u.groupDeployed, u.groupDeployedHint, 'deployed'],
    ['dropped', usage.columns.dropped, u.groupDropped, u.groupDroppedHint, 'dropped'],
  ] as const) {
    if (families.length === 0) continue
    groups.push({
      key,
      label,
      hint,
      columns: families.map((family) => ({
        key: `${key}.${family}`,
        label: equipmentFamilyLabel(family, t),
        value: (x: EquipmentUsageTally) => intValue(x[pick][family]),
        format: intFormat,
      })),
    })
  }
  if (usage.columns.grenades.length > 0) {
    groups.push({
      key: 'grenades',
      label: u.groupGrenades,
      hint: u.groupGrenadesHint,
      columns: usage.columns.grenades.map((rank) => ({
        key: `grenade.${rank}`,
        label: grenadeTypeLabel(doc, rank, t, locale),
        value: (x: EquipmentUsageTally) => intValue(x.grenades[rank]),
        format: intFormat,
      })),
    })
  }
  return groups
}

