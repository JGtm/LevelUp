/**
 * equipmentUsageColumns.ts — LES COLONNES DU TABLEAU DES USAGES, et surtout LEURS NOMS.
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

/** Une colonne du tableau : son en-tête, et ce qu'elle écrit pour un compteur. */
export interface UsageColumn {
  key: string
  label: string
  cell: (tally: EquipmentUsageTally) => string
}

/** Un groupe de colonnes : l'en-tête de premier niveau et sa réserve de mesure. */
export interface UsageColumnGroup {
  key: string
  label: string
  hint: string
  columns: UsageColumn[]
}

/** Une valeur entière de cellule. Zéro s'écrit ZÉRO : c'est une mesure, pas une absence. */
function intCell(n: number | undefined): string {
  return String(n ?? 0)
}

/**
 * Une DURÉE cumulée de cellule, en m:ss. MÊME RÈGLE QUE `intCell`, et c'est le sujet.
 *
 * `formatDurationMMSS` rend son repli pour toute valeur nulle — juste pour une durée de
 * MATCH (un match de zéro seconde n'a pas eu lieu), faux ici : la colonne n'existe que
 * parce que la famille est mesurée sur ce match, et la cellule voisine (nombre d'épisodes)
 * écrit déjà « 0 » pour le même joueur. Un « — » à côté d'un « 0 » se lit « non mesuré »
 * alors que la mesure a bien eu lieu et vaut zéro.
 *
 * Le repli reste donc à sa place — l'absence de la COLONNE, décidée en amont par
 * `usage.columns.episodes` — et n'a rien à faire dans la cellule.
 */
function durationCell(ms: number | undefined): string {
  return formatDurationMMSS((ms ?? 0) / 1000, '0:00')
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
      columns: [{ key: 'pulls', label: u.groupGrapple, cell: (x) => intCell(x.grapplePulls) }],
    })
  }
  if (usage.columns.episodes.length > 0) {
    groups.push({
      key: 'episodes',
      label: u.groupActive,
      hint: u.groupActiveHint,
      columns: usage.columns.episodes.flatMap((fam) => [
        {
          key: `${fam}.count`,
          label: `${u.activeFamily[fam]} (${u.activeCount})`,
          cell: (x: EquipmentUsageTally) => intCell(x.episodes[fam]?.count),
        },
        {
          key: `${fam}.ms`,
          label: `${u.activeFamily[fam]} (${u.activeDuration})`,
          cell: (x: EquipmentUsageTally) => durationCell(x.episodes[fam]?.ms),
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
        cell: (x: EquipmentUsageTally) => intCell(x[pick][family]),
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
        cell: (x: EquipmentUsageTally) => intCell(x.grenades[rank]),
      })),
    })
  }
  return groups
}

