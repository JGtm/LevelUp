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
 * socles de bonus). Une troisième table de noms divergerait au premier ajout du manifeste du
 * titre.
 *
 * Pur : des fonctions de (mesures, textes) vers des colonnes. Aucun React.
 */
import { formatDurationMMSS } from '@/lib/formatters/duration'

import { PLACEMENT_RENDER } from '../layers/equipmentPlacementsLayer'
import { usageOutcomeColor } from './equipmentUsageChart'
import type { EquipmentUsage, EquipmentUsageTally } from './equipmentUsageLogic'
import { droppedFamilyOf, isGameChangerFamily } from './gameChangers'
import { usageUsedOf } from './equipmentKeptLogic'
import type { ReplayText } from '../i18n/i18nContract'
import { padEquipmentFamilyOf } from './weaponPadFamilies'


/**
 * equipmentFamilyLabel — le nom d'une famille de pose, par la cascade des tables existantes.
 *
 * 1. socle de bonus (`powerup_*`), OU son vocabulaire d'ÉPISODE (`camo`/`overshield`, pont
 *    `droppedFamilyOf` — E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md, la colonne fusionnée
 *    équipement les nomme par leur épisode) : `padEquipmentFamily` ;
 * 2. famille dessinée : sa RÈGLE DE RENDU a un libellé dans `placementFamily` ;
 * 3. famille dessinée en point neutre (`unnamed`) : « objet non identifié » ;
 * 4. sinon l'identifiant brut — la seule chose vraie qu'on puisse écrire d'une famille qu'aucune
 *    table ne nomme (même règle que les armes hors catalogue, cf. `padNameFor`).
 */
export function equipmentFamilyLabel(family: string, t: ReplayText): string {
  const powerup = padEquipmentFamilyOf(droppedFamilyOf(family))
  if (powerup) return t.padEquipmentFamily[powerup]
  const kind = PLACEMENT_RENDER[family]
  if (kind && kind !== 'unnamed' && kind !== 'dropped') return t.placementFamily[kind]
  if (kind === 'unnamed') return t.placementUnnamedLabel
  return family
}

/**
 * LES TROIS FAMILLES DE GESTE (E2, 2026-09-09 : `deployed` et `dropped` FUSIONNENT en
 * `equipment` — une seule colonne par famille, empilée sur ses issues, P2/P3). Ces clés sont
 * l'axe de regroupement de tout ce que la section montre : les groupes de colonnes, la couleur
 * des barres, et les lignes de la vue « part de chaque équipe ». Le typage les rend
 * exhaustives — une famille ajoutée ici force la table des encres à la peindre
 * (cf. `equipmentUsageChart`).
 */
export type UsageGroupKey = 'grapple' | 'episodes' | 'equipment'

/**
 * Une colonne : son en-tête, sa VALEUR pour un compteur, et comment cette valeur s'écrit.
 *
 * LA COLONNE REND UN NOMBRE, PLUS UNE CHAÎNE (2026-09-03) : la section est devenue un GRAPHE,
 * et une barre se dessine sur une valeur. Le formatage reste ici — l'échelle de la colonne, sa
 * graduation et sa cellule doivent s'écrire avec la MÊME plume, sans quoi l'axe et la valeur
 * qu'il gradue ne parlent pas la même langue.
 *
 * `value` rend `null` pour une grandeur NON MESURÉE, jamais un zéro : un zéro est une mesure,
 * et l'écran ne doit pas les confondre.
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
  /**
   * LA PILE D'ISSUE DE LA CELLULE (E2, optionnel — `ValueGridCell.segments`, E1). ABSENTE =
   * cellule simple, comme toute colonne hors du groupe `equipment` (grappin, grenades…).
   * Chaque segment porte sa VALEUR BRUTE : `buildValueGrid` calcule sa part de la borne, qui
   * reste celle de `value()` — LE TOTAL DE LA PILE (P1 : la somme des trois issues est le
   * nombre d'objets pris).
   */
  segments?: (
    tally: EquipmentUsageTally,
  ) => Array<{ key: string; value: number; color: string; label: string }> | undefined
  /**
   * LE DÉTAIL DE LA CELLULE, en infobulle (2026-09-13). ABSENT = infobulle standard
   * « joueur — colonne : valeur ». Présent, il REMPLACE la valeur écrite : c'est ce qui permet
   * à une seule colonne de porter plusieurs grandeurs du même geste (nombre d'utilisations,
   * durée cumulée, frags sous l'effet) sans ouvrir une colonne par unité.
   */
  tooltip?: (tally: EquipmentUsageTally) => string
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
 * usageColumnGroups — les groupes de colonnes que la donnée justifie, dans un ordre écrit.
 *
 * Le grappin d'abord (une activation), les états actifs ensuite, puis ce qui se pose sur le
 * terrain. Un groupe sans colonne n'est pas rendu.
 *
 * PLUS DE GROUPE « GRENADES » depuis le 2026-09-13 : l'utilisateur l'a retiré du bilan
 * d'équipement (« dans "Usages d'équipement" j'ai dit que je voulais pas des grenades »). Les
 * lancers restent MESURÉS par `equipmentUsageLogic` et dessinés par le rejeu — ils ne sont
 * simplement plus une colonne de ce tableau.
 */
export function usageColumnGroups(
  usage: EquipmentUsage,
  t: ReplayText,
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
  if (usage.columns.equipment.length > 0) groups.push(equipmentGroup(usage, u, t))
  return groups
}

/**
 * activeEpisodesGroup — le groupe des ÉTATS ACTIFS, UNE colonne par famille.
 *
 * TROIS COLONNES SONT DEVENUES UNE LE 2026-09-13, sur cadrage utilisateur : « c'est quoi cette
 * distinction "épisode" et "durée" ? j'ai jamais demandé ça. » La colonne compte les
 * UTILISATIONS ; la durée cumulée et les frags sous l'effet passent dans l'infobulle de la
 * cellule, où ils qualifient le même geste au lieu d'ouvrir deux colonnes d'unités différentes
 * sur la même mesure.
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
    columns: usage.columns.episodes.map((fam) => ({
      key: `${fam}.count`,
      label: u.activeColumnFmt(u.activeFamily[fam]),
      value: (x: EquipmentUsageTally) => intValue(x.episodes[fam]?.count),
      format: intFormat,
      family: fam,
      // La durée cumulée et les frags sous l'effet QUALIFIENT le compte : ils vivent dans
      // l'infobulle de la cellule, jamais dans deux colonnes de plus.
      tooltip: (x: EquipmentUsageTally) =>
        u.activeCellTipFmt(
          intValue(x.episodes[fam]?.count),
          durationFormat(durationValue(x.episodes[fam]?.ms)),
          killsRead ? intValue(x.episodes[fam]?.kills) : null,
        ),
    })),
  }
}

/**
 * equipmentPileParts — LES TROIS ISSUES D'UNE FAMILLE (P1), pour UN compteur.
 *
 *  - `used` : le côté « utilisé » (P2), lu par `usageUsedOf` (`equipmentKeptLogic.ts`, lot 5.7,
 *    même fonction que le calcul de `kept`) — les épisodes pour les deux power-ups, les poses
 *    déployées pour le seul MUR, les CONSOMMATIONS de charge (`spent`) pour tout le reste ;
 *  - `dropped` : lâché en mourant (`tally.dropped`, ponté vers son vocabulaire de pose pour
 *    les power-ups — `droppedFamilyOf`) ;
 *  - `kept` : gardé sans l'utiliser, DÉJÀ DÉRIVÉ par `equipmentKeptLogic.ts`
 *    (`taken - used - dropped`, cf. `tally.kept`) — rien à recalculer ici.
 *
 * LA MÊME FONCTION `usageUsedOf` NOURRIT LES DEUX CALCULS (`kept` ET l'affichage ici) : avant le
 * lot 5.7 cette cellule relisait `tally.deployed[family]` pour toute famille hors power-up, un
 * second calcul divergent de celui qui dérive `kept` — la somme `used + kept + dropped` pouvait
 * alors ne plus valoir `taken` (P1) pour les déployables sans pièce engendrée. Un seul calcul,
 * jamais deux (CLAUDE.md n°6).
 */
function equipmentPileParts(
  tally: EquipmentUsageTally,
  family: string,
): { used: number; kept: number; dropped: number } {
  const used = intValue(usageUsedOf(tally, family))
  const dropped = intValue(tally.dropped[droppedFamilyOf(family)])
  const kept = intValue(tally.kept[family])
  return { used, kept, dropped }
}

/**
 * equipmentGroup — LA COLONNE FUSIONNÉE « équipement » (E2) : une colonne PAR FAMILLE, empilée
 * sur ses trois issues dans l'ORDRE DU §3.1 (table normative des couleurs) — utilisé, gardé
 * sans l'utiliser, lâché en mourant. REMPLACE les anciens groupes `deployed`/`dropped`
 * (décision P2/P3 : un seul graphe équipement, jamais deux sections « activés »/« déployés »).
 *
 * LES DEUX POWER-UPS Y ENTRENT AUSSI (décision D9 AMENDÉE le 2026-09-09, cf. §5 de la
 * référence canaux) : la rédaction initiale de D9 les excluait par erreur — un bonus ACTIVÉ
 * est mesuré par `equipmentEpisodes`, il doit donc porter une colonne ici, avec le canal des
 * épisodes comme côté « utilisé » (P2). Leur libellé bilingue passe par la même cascade que
 * les familles posées (`equipmentFamilyLabel`, pontée par `droppedFamilyOf`).
 *
 * LE RÉPULSEUR N'A PAS DE COLONNE (P4) : `usage.columns.equipment` ne le porte jamais (aucune
 * famille de `KEPT_FAMILIES` ne le nomme, cf. `equipmentUsageLogic.ts`), donc il ne peut pas
 * apparaître ici — pas un filtre à écrire, une absence de mesure déjà actée en amont.
 */
function equipmentGroup(
  usage: EquipmentUsage,
  u: ReplayText['equipmentUsage'],
  t: ReplayText,
): UsageColumnGroup {
  return {
    key: 'equipment',
    label: u.groupEquipment,
    hint: u.groupEquipmentHint,
    columns: usage.columns.equipment.map((family) => ({
      key: `equipment.${family}`,
      label: equipmentFamilyLabel(family, t),
      value: (x: EquipmentUsageTally) => {
        const { used, kept, dropped } = equipmentPileParts(x, family)
        return used + kept + dropped
      },
      format: intFormat,
      family,
      segments: (x: EquipmentUsageTally) => {
        const { used, kept, dropped } = equipmentPileParts(x, family)
        if (used + kept + dropped === 0) return undefined
        return [
          { key: 'used', value: used, color: usageOutcomeColor('used'), label: u.outcomeUsedFmt(used) },
          { key: 'kept', value: kept, color: usageOutcomeColor('kept'), label: u.outcomeKeptFmt(kept) },
          {
            key: 'dropped',
            value: dropped,
            color: usageOutcomeColor('dropped'),
            label: u.outcomeDroppedFmt(dropped),
          },
        ]
      },
    })),
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

