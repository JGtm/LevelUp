/**
 * weaponRangeChart — LA GRAMMAIRE DU BÂTON DE PORTÉE : p10 → p90, losange sur la médiane,
 * deux mesures superposées (`top` / `bottom`) sur la même bande.
 *
 * WRAPPER PARTAGÉ, ET PLUS UN MODULE DE PAGE (déplacé de `features/synthesis/` le
 * 2026-09-17). Trois surfaces le rendent : l'onglet Résumé des séries temporelles (mes frags
 * en haut, mes morts en bas, une ligne par ARME), le profil d'armes du Face-à-face et le
 * bloc « Portée des frags » de l'Explorer (deux JOUEURS superposés, une ligne par RÔLE).
 * C'est ce que le nommage `top`/`bottom` rend possible sans mentir. Le laisser sous
 * `features/synthesis/` — une page qui ne l'affiche même plus depuis le 2026-09-13 —
 * demandait une dérogation à l'anti-import inter-features par consommateur.
 *
 * Grammaire reprise du graphe de portée par match validé le 2026-09-02
 * (`features/match-view/_killDistanceChart.ts`) : un bâton par arme, un losange sur la
 * valeur centrale. Transposition à l'agrégat multi-matchs (décision D6 du plan
 * `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`) : le bâton court du 10e au 90e centile et le
 * losange marque la MÉDIANE — sur des centaines de frags, min et max décrivent deux
 * accidents, pas une portée. L'utilisateur a demandé le 2026-09-06 la FUSION des deux
 * graphes jumeaux : une ligne par arme, deux bâtons (frags en haut, morts en bas), même axe.
 *
 * POURQUOI UNE SÉRIE `custom` ET PAS bar+scatter comme le graphe de match : sur un axe de
 * catégories, un `scatter` se pose au CENTRE de la bande et ne sait pas se décaler d'un
 * demi-bâton — le losange de la médiane des morts tomberait entre les deux bâtons. Le
 * `renderItem` place les quatre formes au pixel, à partir des coordonnées que l'API ECharts
 * rend elle-même : la position est exacte, pas approchée.
 *
 * CE QUI EST UNE PORTÉE D'USAGE, JAMAIS UNE PORTÉE D'ARME (D8) : ces nombres décrivent LE
 * JOUEUR — à quelle distance il frague et à quelle distance il meurt — pas la balistique du
 * BR75. Aucun libellé de ce module ne dit « portée de l'arme ».
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getTooltipBase,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import type { WeaponRangeRow, WeaponRangeSide } from '@/lib/api/types'
import type { ManifestLocale } from '@/lib/i18n/format'

/** Une entrée du contrat qui porte un libellé bilingue et une clé d'arme. */
interface LabelledWeapon {
  weapon_key: string
  label?: string
  label_en?: string
}

/**
 * resolveWeaponLabel — le nom affiché d'une arme, dans la locale courante.
 *
 * Repli sur `weapon_key` quand le registre n'a pas résolu le libellé : une clé technique
 * lisible vaut mieux qu'une ligne anonyme, et elle SE VOIT (le trou de registre se signale
 * de lui-même au lieu de se cacher derrière un tiret).
 *
 * Vit ici depuis le 2026-09-17 : seul ce module l'appelait, et le garder sous
 * `features/synthesis/` aurait fait remonter `components/` vers `features/`.
 */
export function resolveWeaponLabel(w: LabelledWeapon, locale: ManifestLocale): string {
  const label = locale === 'en' ? w.label_en : w.label
  return label && label.trim() !== '' ? label : w.weapon_key
}

/** Hauteur d'une bande : deux bâtons + leur écart y tiennent (26 px suffisaient à un seul). */
export const WEAPON_RANGE_ROW_PX = 34
/** Épaisseur d'un bâton, et écart entre les deux bâtons d'une même ligne. */
const BAR_HEIGHT = 7
const BAR_GAP = 4
/** Demi-diagonale du losange de médiane. */
const DIAMOND_RADIUS = 5

/**
 * WEAPON_KEYS_WITHOUT_RANGE — les pseudo-armes pour lesquelles une DISTANCE N'A PAS DE SENS.
 *
 * « Chute et environnement » (`hinf_environment`) agrège les morts par chute, écrasement ou
 * élément de décor : la « distance tueur → victime » y mesure l'écart entre la victime et
 * un point de géométrie, pas une portée d'engagement. La ligne polluait l'axe et tirait la
 * médiane (retrait demandé le 2026-09-09).
 *
 * FILTRE CÔTÉ FRONT, ET C'EST UN PIS-ALLER ASSUMÉ : la place durable de cette exclusion est
 * le classifieur de sources côté Go (`port.KillSourceClassifier`), qui sait déjà rattacher
 * un `source_tag` à sa nature — le contrat de portée ne porte pas la classe de l'arme, donc
 * le front n'a que la clé pour trancher. Un `Set` et non un test d'égalité : un autre titre
 * ajoute sa clé sans réécrire la condition.
 */
const WEAPON_KEYS_WITHOUT_RANGE = new Set(['hinf_environment'])

/**
 * Une ligne du graphe — DEUX MESURES SUPERPOSÉES, et rien de plus.
 *
 * # POURQUOI `top` / `bottom` ET NON `kills` / `deaths` (2026-09-17)
 *
 * Ce module dessine deux bâtons l'un au-dessus de l'autre sur une même bande. À la Synthèse,
 * ces deux bâtons sont « mes frags » et « mes morts » ; au Face-à-face, ce sont « le joueur A »
 * et « le joueur B », sur un graphe qui ne montre QUE des frags (et un second qui ne montre que
 * des morts). Garder les noms `kills`/`deaths` aurait obligé le compare à ranger le joueur B
 * dans un champ nommé « morts » — un mensonge de nommage qui se paie à la première relecture.
 *
 * La GÉOMÉTRIE est la seule chose que ce module connaît : qui est en haut, qui est en bas. Le
 * SENS (frags/morts, joueur A/joueur B) vit chez l'appelant, qui fournit aussi les deux
 * libellés d'infobulle et les deux encres.
 *
 * Côté absent = `null`, jamais un zéro — « aucune mesure » et « mesuré, à zéro mètre » ne se
 * lisent pas pareil.
 */
export interface WeaponRangeLine {
  weaponKey: string
  label: string
  top: WeaponRangeSide | null
  bottom: WeaponRangeSide | null
}

/**
 * weaponRangeLines — la projection des lignes du contrat.
 *
 * L'ORDRE DU BACKEND EST CONSERVÉ (médiane des frags croissante, cf. `mergeWeaponSides`
 * côté Go) : le graphe se lit comme un continuum du contact à la longue portée. Rien n'est
 * retrié ici — deux tris du même fait divergeraient au premier changement de doctrine.
 *
 * Le contrat OMET un côté non mesuré (jamais un zéro, qui se lirait « mesuré, à zéro
 * mètre ») ; la projection le normalise en `null` pour que le rendu ait un seul cas à
 * traiter.
 */
export function weaponRangeLines(
  weapons: readonly WeaponRangeRow[] | null | undefined,
  locale: ManifestLocale,
): WeaponRangeLine[] {
  return (weapons ?? [])
    .filter((w) => !WEAPON_KEYS_WITHOUT_RANGE.has(w.weapon_key))
    .map((w) => ({
      weaponKey: w.weapon_key,
      label: resolveWeaponLabel(w, locale),
      // Synthèse : les frags EN HAUT, les morts en dessous (l'ordre de la légende).
      top: w.kills ?? null,
      bottom: w.deaths ?? null,
    }))
}

/**
 * weaponRangeCategoryLabel — le nom de l'arme, nu.
 *
 * L'étiquette portait jusqu'au 2026-09-09 le suffixe « ×281/402 » (frags puis morts
 * mesurés). Retiré à la demande de l'utilisateur : deux nombres collés à chaque nom d'arme
 * allongeaient la gouttière de gauche sans être lus, alors que les DEUX effectifs sont déjà
 * écrits dans l'infobulle de la ligne (« Mes frags — 281 : … ») et dans le tableau
 * dépliable, où ils ont une colonne à eux.
 *
 * La fonction reste la SOURCE UNIQUE de l'étiquette : les deux graphes de la section
 * (portée et dénivelé) partagent le même axe de catégories, et deux fabrications
 * séparées divergeraient au premier changement.
 */
export function weaponRangeCategoryLabel(line: WeaponRangeLine): string {
  return line.label
}

/** Hauteur du graphe : une bande par arme, plus la place des axes. */
export function weaponRangeChartHeight(lineCount: number): number {
  return 48 + WEAPON_RANGE_ROW_PX * lineCount
}

/**
 * weaponRangeAxisMax — la borne haute de l'axe des distances, COMMUNE aux deux côtés.
 *
 * Un axe par côté rendrait les deux bâtons d'une ligne incomparables — c'est-à-dire
 * exactement ce que la fusion des deux graphes cherchait à éviter. Arrondi aux 5 m
 * supérieurs pour que la dernière graduation soit lisible, plancher à 5 m (un corpus tout
 * au contact ne doit pas produire un axe dégénéré).
 */
export function weaponRangeAxisMax(lines: readonly WeaponRangeLine[]): number {
  let max = 0
  for (const line of lines) {
    for (const side of [line.top, line.bottom]) {
      if (side && side.p90 > max) max = side.p90
    }
  }
  return Math.max(5, Math.ceil(max / 5) * 5)
}

/** Coordonnées rendues par l'API `renderItem` d'ECharts (sous-ensemble utilisé ici). */
interface RenderItemApi {
  coord(point: [number, number]): number[]
}
interface RenderItemParams {
  dataIndex: number
}
/** Un nœud graphique du groupe rendu — forme minimale consommée par ECharts. */
interface RenderItemChild {
  type: 'rect' | 'polygon'
  shape: Record<string, unknown>
  style: Record<string, unknown>
}

export interface WeaponRangeOptionInput {
  /** Lignes DANS L'ORDRE DU BACKEND — l'inversion de l'axe Y se fait ici, pas chez l'appelant. */
  lines: readonly WeaponRangeLine[]
  tc: EChartsThemeColors
  /** Encres résolues par l'appelant (tokens sémantiques) : les deux bâtons, losange, contour. */
  topColor: string
  bottomColor: string
  medianColor: string
  /** Contour du losange = fond de carte : il se détache du bâton sans couleur nouvelle. */
  cardColor: string
  /** Formate une distance (« 12,4 m ») — la locale vit chez l'appelant. */
  fmtDistance: (m: number) => string
  /**
   * Libellés d'infobulle, déjà localisés. `top`/`bottom` NOMMENT les deux mesures — « Mes
   * frags »/« Mes morts » à la Synthèse, deux gamertags au Face-à-face.
   *
   * `observed` est OPTIONNEL : quand il est fourni ET que le contrat porte `min_m`/`max_m`,
   * l'infobulle ajoute « min – max observés ». Les deux extrêmes ne sont JAMAIS tracés (D5
   * du plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md) — les mettre dans la géométrie du
   * bâton ferait exactement ce que D6 refuse : décrire deux accidents. Absent = la Synthèse,
   * dont l'affichage reste inchangé.
   */
  labels: {
    top: string
    /** Nom de la seconde mesure. Inutile en mode `singleBand` : il n'y en a pas de seconde. */
    bottom?: string
    percentiles: string
    noMeasure: string
    observed?: string
  }
  /**
   * UNE SEULE BANDE par ligne, CENTRÉE, et une infobulle à une seule mesure.
   *
   * Ajouté le 2026-09-20 pour « Portée des frags » de l'encart cible de l'Explorer, ramené
   * à la cible seule (décision 7 du plan d'ajustements pré-v7.5). Sans lui, le graphe
   * gardait la place du second bâton — une demi-bande vide sous chaque ligne — et
   * l'infobulle annonçait « aucune mesure » pour un joueur qu'on ne montre plus.
   *
   * Absent = les deux bandes décalées de part et d'autre du centre, le rendu de tous les
   * appelants antérieurs.
   */
  singleBand?: boolean
}

/** Les quatre encres du `renderItem`, plus les lignes déjà retournées pour l'axe. */
interface RangeRenderInput {
  ordered: readonly WeaponRangeLine[]
  /** Cf. `WeaponRangeOptionInput.singleBand`. */
  singleBand?: boolean
  topColor: string
  bottomColor: string
  medianColor: string
  cardColor: string
}

/**
 * makeRangeRenderItem — le `renderItem` de la série `custom`, fabriqué à part.
 *
 * Extrait de `buildWeaponRangeOption` le 2026-09-06 (revue adversariale du lot 5, seuil de
 * 80 lignes par fonction) : la GÉOMÉTRIE et l'ASSEMBLAGE de l'option sont deux sujets, et le
 * test de géométrie pince déjà cette fonction à travers une API `renderItem` factice.
 * Aucun changement de rendu — les mêmes formes, aux mêmes pixels.
 */
function makeRangeRenderItem({
  ordered,
  singleBand,
  topColor,
  bottomColor,
  medianColor,
  cardColor,
}: RangeRenderInput) {
  return (params: RenderItemParams, api: RenderItemApi) => {
    const line = ordered[params.dataIndex]
    const children: RenderItemChild[] = []
    if (!line) return { type: 'group', children }
    const yCenter = api.coord([0, params.dataIndex])[1]
    const x = (v: number) => api.coord([v, 0])[0]
    const push = (side: WeaponRangeSide | null, color: string, dy: number) => {
      if (!side) return
      children.push({
        type: 'rect',
        shape: {
          x: x(side.p10),
          y: yCenter + dy - BAR_HEIGHT / 2,
          // Plancher de 2 px : une portée quasi ponctuelle reste visible (un bâton de
          // largeur nulle disparaîtrait, et l'arme avec lui).
          width: Math.max(2, x(side.p90) - x(side.p10)),
          height: BAR_HEIGHT,
          r: 4,
        },
        style: { fill: color },
      })
      const cx = x(side.median)
      const cy = yCenter + dy
      children.push({
        type: 'polygon',
        shape: {
          points: [
            [cx, cy - DIAMOND_RADIUS],
            [cx + DIAMOND_RADIUS, cy],
            [cx, cy + DIAMOND_RADIUS],
            [cx - DIAMOND_RADIUS, cy],
          ],
        },
        style: { fill: medianColor, stroke: cardColor, lineWidth: 1 },
      })
    }
    // Une seule bande : elle occupe le CENTRE de la ligne. La décaler laisserait sous elle
    // la place d'un bâton qui ne viendra pas.
    if (singleBand) {
      push(line.top, topColor, 0)
      return { type: 'group', children }
    }
    const offset = BAR_HEIGHT / 2 + BAR_GAP / 2
    push(line.top, topColor, -offset)
    push(line.bottom, bottomColor, +offset)
    return { type: 'group', children }
  }
}

/**
 * rangeTooltipSideLine — une ligne d'infobulle pour un côté : « Mes frags — 281 : 7,1 m ·
 * 13,6 m · 24,9 m », ou « Mes frags — aucune mesure ».
 *
 * Le nom de l'arme comme les nombres formatés passent par `escapeHtml` : l'infobulle d'ECharts
 * est du HTML, et un libellé de registre n'est pas une source de confiance.
 */
function rangeTooltipSideLine(
  name: string,
  side: WeaponRangeSide | null,
  fmtDistance: (m: number) => string,
  noMeasure: string,
  observed?: string,
): string {
  if (!side) return `${escapeHtml(name)} — ${escapeHtml(noMeasure)}`
  const low = escapeHtml(fmtDistance(side.p10))
  const median = escapeHtml(fmtDistance(side.median))
  const high = escapeHtml(fmtDistance(side.p90))
  const base = `${escapeHtml(name)} — ${side.measured} : ${low} · <b>${median}</b> · ${high}`
  // MIN ET MAX NE SONT PAS TRACÉS, ILS SE LISENT ICI (D5). Ils répondent à « jusqu'où
  // est-il allé », que le bâton p10 → p90 ne dit pas — et les porter dans la géométrie
  // décrirait deux accidents, ce que D6 refuse. Affichés seulement si l'appelant fournit le
  // libellé : la Synthèse ne le fait pas, son infobulle est donc inchangée.
  if (!observed) return base
  const min = escapeHtml(fmtDistance(side.min_m))
  const max = escapeHtml(fmtDistance(side.max_m))
  return `${base} <span style="opacity:.7">(${escapeHtml(observed)} ${min} – ${max})</span>`
}

/**
 * buildWeaponRangeOption — l'option ECharts du graphe de portée.
 *
 * Contrat de rendu, ligne par ligne : deux rectangles arrondis p10 → p90 décalés de part et
 * d'autre du centre de bande (frags AU-DESSUS, morts en dessous — l'ordre de la légende), et
 * un losange par médiane. Un côté absent ne dessine RIEN de son côté ; l'infobulle le dit
 * (« aucune mesure »), elle n'invente pas un zéro.
 */
export function buildWeaponRangeOption({
  lines,
  tc,
  topColor,
  bottomColor,
  medianColor,
  cardColor,
  fmtDistance,
  labels,
  singleBand,
}: WeaponRangeOptionInput): EChartsCoreOption {
  // Premier du backend = plus courte portée = EN HAUT : l'axe Y d'ECharts empile du bas
  // vers le haut, donc la liste se lit à l'envers au montage.
  const ordered = [...lines].reverse()
  const xMax = weaponRangeAxisMax(lines)
  const axis = getAxisBase(tc)
  const renderItem = makeRangeRenderItem({
    ordered,
    singleBand,
    topColor,
    bottomColor,
    medianColor,
    cardColor,
  })
  const sideLine = (name: string, side: WeaponRangeSide | null) =>
    rangeTooltipSideLine(name, side, fmtDistance, labels.noMeasure, labels.observed)

  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ top: 8, bottom: 24, left: 8 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { dataIndex?: number }
        const line = p?.dataIndex != null ? ordered[p.dataIndex] : undefined
        if (!line) return ''
        const lignes = [
          `<b>${escapeHtml(line.label)}</b> — ${escapeHtml(labels.percentiles)}`,
          sideLine(labels.top, line.top),
        ]
        // Le second côté n'est nommé que s'il existe : en mode `singleBand`, annoncer
        // « aucune mesure » pour un joueur qu'on ne montre pas serait un faux manque.
        if (!singleBand) lignes.push(sideLine(labels.bottom ?? '', line.bottom))
        return lignes.join('<br/>')
      },
    },
    xAxis: {
      type: 'value',
      max: xMax,
      ...axis,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtDistance(v) },
    },
    yAxis: {
      type: 'category',
      data: ordered.map(weaponRangeCategoryLabel),
      ...axis,
      splitLine: { show: false },
    },
    series: [
      {
        type: 'custom',
        renderItem,
        // `encode` déclare à ECharts quelles dimensions bornent l'axe : la valeur 0 et la
        // borne haute, pour que l'échelle couvre exactement l'étendue voulue même si le
        // `renderItem` dessine, lui, aux coordonnées de chaque côté.
        encode: { x: [1, 2], y: 0 },
        data: ordered.map((_, i) => [i, 0, xMax]),
      },
    ],
  }
}
