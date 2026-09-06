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

import { catalogText } from '../i18n/catalogLabel'
import { PLACEMENT_RENDER } from '../layers/equipmentPlacementsLayer'
import type { EquipmentUsage, EquipmentUsageTally } from './equipmentUsageLogic'
import { isGameChangerFamily } from './gameChangers'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayText } from '../i18n/i18nContract'
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
  /**
   * LA FAMILLE QUE LE VOTE JUGE (repli « game changers », plan 2026-09-05) : famille de pose
   * ou de socle (`sensor`, `powerup_camo`), d'épisode (`camo`) ou de capacité (`grapple`) —
   * c'est elle que `partitionUsageGroups` passe à `isGameChangerFamily`. ABSENTE = colonne
   * HORS VOTE, TOUJOURS VISIBLE : les grenades, que l'utilisateur a déjà tranchées hors des
   * équipements (décision D4) — pas un défaut de prudence, une exclusion écrite.
   */
  family?: string
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
          // Identifiant stable du document — la même clé que `PLACEMENT_RENDER.grapple`.
          family: 'grapple',
        },
      ],
    })
  }
  if (usage.columns.episodes.length > 0) groups.push(activeEpisodesGroup(usage, u))
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
        family,
      })),
    })
  }
  if (usage.columns.grenades.length > 0) {
    groups.push({
      key: 'grenades',
      label: u.groupGrenades,
      hint: u.groupGrenadesHint,
      // AUCUNE `family` : les grenades sont HORS VOTE (décision D4, tranchée par
      // l'utilisateur : ce ne sont pas des équipements) — toujours visibles, jamais repliées.
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

/**
 * activeEpisodesGroup — le groupe des ÉTATS ACTIFS et ses trois colonnes par famille (nombre,
 * durée, frags). Extrait de `usageColumnGroups` le 2026-09-05, quand l'ajout de la famille de
 * vote sur chaque colonne l'a fait franchir le plafond de taille de fonction du dépôt.
 */
function activeEpisodesGroup(
  usage: EquipmentUsage,
  u: ReplayText['equipmentUsage'],
): UsageColumnGroup {
  // Lu UNE FOIS par appel, fermé sur les cellules : `killsRead` est une propriété du
  // MATCH entier (cf. EquipmentUsageCoverage), pas d'un joueur ni d'une famille.
  const killsRead = usage.coverage.killsRead
  return {
    key: 'episodes',
    label: u.groupActive,
    hint: u.groupActiveHint,
    columns: usage.columns.episodes.flatMap((fam) => [
      {
        key: `${fam}.count`,
        label: `${u.activeFamily[fam]} (${u.activeCount})`,
        value: (x: EquipmentUsageTally) => intValue(x.episodes[fam]?.count),
        format: intFormat,
        family: fam,
      },
      {
        key: `${fam}.ms`,
        label: `${u.activeFamily[fam]} (${u.activeDuration})`,
        value: (x: EquipmentUsageTally) => durationValue(x.episodes[fam]?.ms),
        format: durationFormat,
        duration: true,
        family: fam,
      },
      {
        key: `${fam}.kills`,
        label: u.activeKillsFamily[fam],
        value: (x: EquipmentUsageTally) => killsValue(x.episodes[fam]?.kills, killsRead),
        format: intFormat,
        family: fam,
      },
    ]),
  }
}

/**
 * LA PARTITION DU REPLI « GAME CHANGERS » (plan 2026-09-05, décision D3) : ce que le vote a
 * élu se montre d'emblée, le reste se replie derrière « Voir plus (N) ».
 */
export interface UsageGroupPartition {
  /** Les groupes EN AVANT, dans l'ordre écrit de `usageColumnGroups` — grenades comprises. */
  forward: UsageColumnGroup[]
  /** Les groupes REPLIÉS, même ordre. Un groupe mixte (poses) figure dans les DEUX listes. */
  collapsed: UsageColumnGroup[]
  /** Nombre de colonnes masquées — le compte du bouton « Voir plus (N) ». Zéro = pas de bouton. */
  collapsedColumnCount: number
}

/**
 * partitionUsageGroups — coupe les groupes en deux d'après le VOTE, sans toucher aux mesures.
 *
 * L'ORDRE INTERNE SURVIT DANS CHAQUE PARTITION : les colonnes gardent l'ordre des tables de
 * référence (`PLACEMENT_RENDER`, etc.) telles que `usageColumnGroups` les a posées — la
 * partition filtre, elle ne trie jamais. Les épisodes camo/surbouclier passent EN AVANT par le
 * pont D5 (`isGameChangerFamily` répond dans les deux vocabulaires) ; une colonne SANS famille
 * est hors vote (grenades, D4) et reste visible. Les TOTAUX, footnotes et `hasData` ne passent
 * pas par ici : ils lisent `EquipmentUsage`, que ce découpage d'affichage ne modifie pas.
 */
export function partitionUsageGroups(groups: UsageColumnGroup[]): UsageGroupPartition {
  const enAvant = (c: UsageColumn): boolean => c.family == null || isGameChangerFamily(c.family)
  const forward: UsageColumnGroup[] = []
  const collapsed: UsageColumnGroup[] = []
  for (const group of groups) {
    const elues = group.columns.filter(enAvant)
    const repliees = group.columns.filter((c) => !enAvant(c))
    if (elues.length > 0) forward.push({ ...group, columns: elues })
    if (repliees.length > 0) collapsed.push({ ...group, columns: repliees })
  }
  return {
    forward,
    collapsed,
    collapsedColumnCount: collapsed.reduce((n, g) => n + g.columns.length, 0),
  }
}

/**
 * uniqueUsageGroups — une seule occurrence par famille de geste, la première.
 *
 * La légende et la vue « part de chaque équipe » raisonnent PAR FAMILLE DE GESTE (`key`), pas
 * par colonne : quand la partition dépliée remet un groupe mixte en deux morceaux (poses en
 * avant + poses repliées), ces deux vues n'en veulent qu'UN — même libellé, même réserve, et
 * un compte de gestes qui vient du tally, donc identique dans les deux morceaux.
 */
export function uniqueUsageGroups(groups: UsageColumnGroup[]): UsageColumnGroup[] {
  const vus = new Set<UsageGroupKey>()
  return groups.filter((g) => {
    if (vus.has(g.key)) return false
    vus.add(g.key)
    return true
  })
}

