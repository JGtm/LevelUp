/**
 * _killDistanceChart — LA PORTÉE DE CHAQUE ARME DE CE MATCH, UN SEUL GRAPHE, SECTIONS PAR ARME.
 *
 * TROIS FORMES SUCCESSIVES, ET LA TROISIÈME EST CELLE-CI. Le POC du 2026-08-30 (DEC-8) rendait
 * ces nombres en tableau ; l'utilisateur a demandé le 2026-09-02 « un bâton pour chaque arme,
 * plus proche / plus loin, et un indicateur sur la moyenne » ; il a rouvert la décision le
 * 2026-09-13 : « il vaut mieux afficher des sections par armes, on met les joueurs sous chaque
 * arme mais toujours au sein du même graphe, pas comme c'est aujourd'hui. Et le joueur actif
 * et ses amis doivent avoir une couleur différente. » D'où : UN graphe, l'axe des catégories
 * groupé PAR ARME (une ligne d'en-tête « Arme ×N », puis une ligne par joueur), et la couleur
 * du bâton donnée par la palette de joueurs du match (`colors.ts`).
 *
 * LE BÂTON D'UN SEUL FRAG MESURÉ EST UN POINT (min = max = avg) : c'est exact, pas un
 * défaut de rendu — le losange de moyenne reste le témoin visible. Le NOMBRE de frags
 * mesurés est dans le libellé de l'arme (« ×N ») et dans l'infobulle : l'épaisseur du
 * bâton ne le code pas, une largeur ne se lit pas en nombre.
 *
 * TECHNIQUE DU BÂTON FLOTTANT : deux barres empilées — un socle TRANSPARENT de hauteur
 * `min` (silencieux, hors infobulle) puis la barre visible de hauteur `max − min`. C'est le
 * patron ECharts standard d'un intervalle ; un `custom` renderItem ferait la même chose en
 * plus de code. La moyenne est une série `scatter` posée sur les mêmes catégories.
 *
 * LES LIGNES D'EN-TÊTE N'ONT PAS DE BARRE (valeur nulle) : elles ne portent qu'un libellé en
 * gras et une bande `markArea` qui court sur toute la largeur du graphe — c'est le séparateur
 * de section, il ne code aucune grandeur.
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
import type { MatchKillDistanceWeapon } from '@/lib/api/types'

/** Longueur au-delà de laquelle un gamertag est tronqué sur l'axe (l'infobulle garde tout). */
const GAMERTAG_MAX = 14

/** Un joueur du match, tel que la section le projette : identité, encre, armes mesurées. */
export interface KillDistancePlayerInput {
  xuid: string
  /** Gamertag complet (déjà nettoyé du suffixe bot par l'appelant). */
  gamertag: string
  /** Encre du bâton — `buildMatchPlayerColors` (moi / amis / alliés / adversaires). */
  color: string
  weapons: readonly MatchKillDistanceWeapon[]
}

/**
 * Une ligne de l'axe des catégories : soit l'EN-TÊTE d'une arme, soit un JOUEUR sous elle.
 *
 * Une seule liste pour les deux : l'axe d'ECharts est une suite de catégories, et un en-tête
 * qui vivrait ailleurs (titre HTML, second graphe) ne serait plus à l'aplomb de ses lignes.
 */
export interface KillDistanceRow {
  kind: 'weapon' | 'player'
  /** Ce qui s'écrit sur l'axe (gamertag tronqué pour une ligne de joueur). */
  axisLabel: string
  /** Le nom de l'arme de la section — porté par les DEUX genres de ligne (infobulle). */
  weapon: string
  /** Gamertag complet, vide sur une ligne d'en-tête. */
  player: string
  kills: number
  min: number
  max: number
  avg: number
}

/** Tronque un gamertag pour l'axe. L'infobulle, elle, écrit toujours le nom complet. */
function truncateName(name: string): string {
  return name.length > GAMERTAG_MAX ? `${name.slice(0, GAMERTAG_MAX - 1)}…` : name
}

/** Le libellé d'une arme dans la locale courante, replié sur sa clé. */
function weaponLabel(w: MatchKillDistanceWeapon, locale: string): string {
  return (locale === 'en' ? w.label_en : w.label) || w.weapon_key
}

/**
 * killDistanceRows — la projection du contrat vers les lignes du graphe.
 *
 * L'ORDRE DES ARMES est celui des frags mesurés, TOUTES ÉQUIPES CONFONDUES, décroissant :
 * l'arme qui a le plus tué dans ce match ouvre la liste. À égalité, le nom départage — deux
 * relectures du même match donnent le même graphe. Sous chaque arme, les joueurs sont rangés
 * par frags mesurés décroissants, le nom départageant de même.
 */
export function killDistanceRows(
  players: readonly KillDistancePlayerInput[],
  locale: string,
): KillDistanceRow[] {
  interface Entry {
    label: string
    total: number
    lines: KillDistanceRow[]
  }
  const byWeapon = new Map<string, Entry>()
  for (const p of players) {
    for (const w of p.weapons) {
      let entry = byWeapon.get(w.weapon_key)
      if (!entry) {
        entry = { label: weaponLabel(w, locale), total: 0, lines: [] }
        byWeapon.set(w.weapon_key, entry)
      }
      entry.total += w.measured_kills
      entry.lines.push({
        kind: 'player',
        axisLabel: truncateName(p.gamertag),
        weapon: entry.label,
        player: p.gamertag,
        kills: w.measured_kills,
        min: w.min_distance_m,
        max: w.max_distance_m,
        avg: w.avg_distance_m,
      })
    }
  }
  const entries = [...byWeapon.values()].sort(
    (a, b) => b.total - a.total || a.label.localeCompare(b.label),
  )
  const rows: KillDistanceRow[] = []
  for (const entry of entries) {
    rows.push({
      kind: 'weapon',
      axisLabel: `${entry.label} ×${entry.total}`,
      weapon: entry.label,
      player: '',
      kills: entry.total,
      min: 0,
      max: 0,
      avg: 0,
    })
    entry.lines.sort((a, b) => b.kills - a.kills || a.player.localeCompare(b.player))
    rows.push(...entry.lines)
  }
  return rows
}

/** Hauteur du graphe : une rangée par ligne, plancher pour que l'axe reste lisible. */
export function killDistanceHeight(rowCount: number): number {
  return Math.max(140, 56 + 22 * rowCount)
}

export interface KillDistanceOptionInput {
  rows: readonly KillDistanceRow[]
  tc: EChartsThemeColors
  /** Encre du bâton par gamertag — `buildMatchPlayerColors().hexByGamertag`. */
  colorOf: (player: string) => string
  /** Encre du losange de moyenne (jeton résolu par l'appelant). */
  avgColor: string
  /** Formate une distance (« 12,4 m ») — la locale vit chez l'appelant (i18n.ts). */
  fmtDistance: (m: number) => string
  /** Libellés d'infobulle, déjà localisés : frags mesurés, min, moyenne, max. */
  labels: { kills: string; min: string; avg: string; max: string }
}

export function buildKillDistanceOption({
  rows,
  tc,
  colorOf,
  avgColor,
  fmtDistance,
  labels,
}: KillDistanceOptionInput): EChartsCoreOption {
  // Première ligne du modèle = EN HAUT : l'axe Y d'ECharts empile du bas vers le haut, donc
  // la liste se lit à l'envers au montage.
  const ordered = [...rows].reverse()
  const axis = getAxisBase(tc)
  // La bande d'une section se désigne PAR SA CATÉGORIE, aux deux bouts : c'est le seul
  // repère qui tombe exactement sur la ligne d'en-tête. Les coordonnées numériques, elles,
  // visent les BORDS de bande d'un axe de catégories (boundaryGap) et décalaient le fond —
  // constaté sur capture. Les libellés d'en-tête (« BR75 ×5 ») sont uniques par construction :
  // une arme n'a qu'une section.
  const bands = ordered
    .filter((r) => r.kind === 'weapon')
    .map((r) => [{ yAxis: r.axisLabel }, { yAxis: r.axisLabel }])
  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ bottom: 24, left: 8 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const list = Array.isArray(params) ? params : [params]
        const first = list[0] as { dataIndex?: number } | undefined
        const row = first?.dataIndex != null ? ordered[first.dataIndex] : undefined
        if (!row) return ''
        if (row.kind === 'weapon') {
          return `<b>${escapeHtml(row.weapon)}</b> — ${labels.kills} : ${row.kills}`
        }
        return [
          `<b>${escapeHtml(row.weapon)}</b> — ${escapeHtml(row.player)}`,
          `${labels.kills} : ${row.kills}`,
          `${labels.min} : ${escapeHtml(fmtDistance(row.min))}`,
          `${labels.avg} : ${escapeHtml(fmtDistance(row.avg))}`,
          `${labels.max} : ${escapeHtml(fmtDistance(row.max))}`,
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'value',
      ...axis,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtDistance(v) },
    },
    yAxis: {
      type: 'category',
      data: ordered.map((r) => r.axisLabel),
      ...axis,
      splitLine: { show: false },
      axisLabel: {
        ...axis.axisLabel,
        // Deux genres de ligne, deux plumes : l'arme en gras sur l'encre du thème, le joueur
        // en retrait sur l'encre d'axe. Le `rich` est indexé par la classe rendue ci-dessous.
        formatter: (value: string, index: number) =>
          ordered[index]?.kind === 'weapon' ? `{arme|${value}}` : `{joueur|${value}}`,
        rich: {
          arme: { fontWeight: 'bold', color: tc.text, align: 'left' },
          joueur: { color: tc.axisLabel, padding: [0, 0, 0, 10] },
        },
      },
    },
    series: [
      {
        // Le socle transparent : il PORTE le bâton à `min`, il ne se lit pas.
        type: 'bar',
        stack: 'range',
        silent: true,
        itemStyle: { color: 'transparent' },
        emphasis: { disabled: true },
        data: ordered.map((r) => (r.kind === 'player' ? r.min : null)),
      },
      {
        type: 'bar',
        stack: 'range',
        barWidth: 8,
        data: ordered.map((r) =>
          r.kind === 'player'
            ? { value: r.max - r.min, itemStyle: { color: colorOf(r.player), borderRadius: 4 } }
            : { value: null },
        ),
        // La bande de section : elle court sur toute la largeur, derrière la ligne d'en-tête. Un
        // axe de CATÉGORIES place la valeur numérique i au BORD de la bande i (boundaryGap) :
        // la bande de la catégorie i court donc de i à i+1, jamais de i-0,5 à i+0,5 — vérifié
        // sur capture, la variante centrée décalait le fond d'une ligne vers le bas.
        markArea: {
          silent: true,
          itemStyle: { color: tc.splitLine },
          data: bands,
        },
      },
      {
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 9,
        itemStyle: { color: avgColor },
        data: ordered.flatMap((r, i) => (r.kind === 'player' ? [[r.avg, i]] : [])),
      },
    ],
  }
}
