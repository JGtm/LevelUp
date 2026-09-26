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
import { PLACEMENT_RENDER } from '../layers/equipmentPlacementsLayer'
import { usageOutcomeColor } from './equipmentUsageChart'
import type { EquipmentUsage, EquipmentUsageTally } from './equipmentUsageLogic'
import { droppedFamilyOf } from './gameChangers'
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
 * LES DEUX FAMILLES DE GESTE (E2, 2026-09-09 : `deployed` et `dropped` FUSIONNENT en
 * `equipment` — une seule colonne par famille, empilée sur ses issues, P2/P3 ; `episodes`
 * retiré le 2026-09-19, décision 6 du plan d'ajustements pré-v7.5). Ces clés sont
 * l'axe de regroupement de tout ce que la section montre : les groupes de colonnes, la couleur
 * des barres, et les lignes de la vue « Part de chaque équipe ». Le typage les rend
 * exhaustives — une famille ajoutée ici force la table des encres à la peindre
 * (cf. `equipmentUsageChart`).
 */
export type UsageGroupKey = 'grapple' | 'equipment'

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
 * usageColumnGroups — les groupes de colonnes que la donnée justifie, dans un ordre écrit.
 *
 * Le grappin d'abord (une activation), puis ce qui se pose sur le terrain. Un groupe sans
 * colonne n'est pas rendu.
 *
 * PLUS DE GROUPE « GRENADES » depuis le 2026-09-13 : l'utilisateur l'a retiré du bilan
 * d'équipement (« dans "Usages d'équipement" j'ai dit que je voulais pas des grenades »). Les
 * lancers restent MESURÉS par `equipmentUsageLogic` et dessinés par le rejeu — ils ne sont
 * simplement plus une colonne de ce tableau.
 *
 * PLUS DE GROUPE « ÉTATS ACTIFS » depuis le 2026-09-19 (plan d'ajustements pré-v7.5, lot 2,
 * décision 6) : « actif » et « utilisé » disaient la même chose, et les épisodes de camouflage
 * et de surbouclier alimentent DÉJÀ le côté « utilisé » de la colonne d'équipement du même
 * power-up (P2, `usageUsedOf`). Une colonne de plus par famille d'état comptait deux fois le
 * même geste sous deux mots. Restent Utilisé / Gardé / Lâché, les trois issues de la pile.
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
        },
      ],
    })
  }
  if (usage.columns.equipment.length > 0) groups.push(equipmentGroup(usage, u, t))
  return groups
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
 * uniqueUsageGroups — une seule occurrence par famille de geste, la première.
 *
 * La légende et la vue « Part de chaque équipe » raisonnent PAR FAMILLE DE GESTE (`key`), pas
 * par colonne : un groupe qui apparaîtrait deux fois dans la liste n'y vaut qu'UNE ligne —
 * même libellé, même réserve, et un compte de gestes qui vient du tally.
 */
export function uniqueUsageGroups(groups: UsageColumnGroup[]): UsageColumnGroup[] {
  const vus = new Set<UsageGroupKey>()
  return groups.filter((g) => {
    if (vus.has(g.key)) return false
    vus.add(g.key)
    return true
  })
}

