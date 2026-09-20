import { describe, expect, it } from 'vitest'

import {
  buildBucketPoints,
  formatHeatmapTooltip,
  type HeatmapBucketCell,
  type HeatmapTooltipText,
} from './RelationsMomentsHeatmap'

const TOOLTIP_FR: HeatmapTooltipText = {
  playerLabel: 'Joueur',
  bucketTypeLabel: 'Créneau',
  matchesLabel: 'Matchs communs',
  emptyCell: 'Aucun match sur ce créneau',
}

// Garde-rail : la heatmap « Rythme des rencontres » ne montre JAMAIS plus de 12
// relations sur l'axe Y (les 12 les plus actives, par total décroissant).
describe('RelationsMomentsHeatmap buildBucketPoints', () => {
  /** Les relations rendues, dans l'ordre où elles occupent l'axe Y. */
  const lignes = (points: { y: string }[]) => [...new Set(points.map((p) => p.y))]

  it('tronque l’axe Y à 12 relations quand il y en a plus', () => {
    // 20 relations, 1 cellule chacune (bucket 0), total = i+1 → tri déterministe.
    const cells: HeatmapBucketCell[] = Array.from({ length: 20 }, (_, i) => ({
      xuid: `x${i}`,
      gamertag: `Player${i}`,
      bucket: 0,
      count: i + 1,
    }))

    const y = lignes(buildBucketPoints(cells, ['00h']))

    // Exactement 12 lignes, et ce sont les 12 plus actives (counts 20..9).
    expect(y).toHaveLength(12)
    expect(y[0]).toBe('Player19') // total le plus élevé (20)
    expect(y).not.toContain('Player0') // total le plus faible, écarté
  })

  it('n’ajoute pas de lignes vides quand il y a moins de 12 relations', () => {
    const cells: HeatmapBucketCell[] = Array.from({ length: 5 }, (_, i) => ({
      xuid: `x${i}`,
      gamertag: `P${i}`,
      bucket: 0,
      count: 1,
    }))
    expect(lignes(buildBucketPoints(cells, ['00h']))).toHaveLength(5)
  })

  // Les axes du wrapper canonique sont DÉDUITS de l'ordre d'apparition des points :
  // omettre une case sans rencontre décalerait les catégories d'un cran.
  it('émet toutes les cases du produit relation × créneau, une case sans rencontre valant null', () => {
    const cells: HeatmapBucketCell[] = [
      { xuid: 'x0', gamertag: 'P0', bucket: 0, count: 3 },
      { xuid: 'x1', gamertag: 'P1', bucket: 1, count: 2 },
    ]
    const points = buildBucketPoints(cells, ['00h', '01h'])
    expect(points).toHaveLength(4)
    expect(points.find((p) => p.y === 'P0' && p.x === '00h')?.value).toBe(3)
    expect(points.find((p) => p.y === 'P0' && p.x === '01h')?.value).toBeNull()
  })
})

// Tooltip refait (V7.3 lot 2, item 1.3) : trois lignes étiquetées, gamertag échappé.
describe('formatHeatmapTooltip', () => {
  it('affiche joueur, créneau et nombre de matchs, chacun étiqueté', () => {
    const html = formatHeatmapTooltip('AllyPlayer', '21h', 4, TOOLTIP_FR)
    expect(html).toContain('Joueur : AllyPlayer')
    expect(html).toContain('Créneau : 21h')
    expect(html).toContain('Matchs communs : 4')
    // Trois lignes séparées (deux sauts) — plus l'ancien « qui · quand » condensé.
    expect(html.match(/<br>/g)).toHaveLength(2)
  })

  it('annonce explicitement une cellule vide au lieu d’un compteur absent', () => {
    for (const empty of [null, 0]) {
      const html = formatHeatmapTooltip('AllyPlayer', '03h', empty, TOOLTIP_FR)
      expect(html).toContain('Aucun match sur ce créneau')
      expect(html).not.toContain('Matchs communs')
    }
  })

  it('suit l’étiquette de bucket fournie (mode jour de la semaine)', () => {
    const html = formatHeatmapTooltip('AllyPlayer', 'Mardi', 2, {
      ...TOOLTIP_FR,
      bucketTypeLabel: 'Jour',
    })
    expect(html).toContain('Jour : Mardi')
  })

  it('rend les libellés anglais tels quels (parité FR/EN du manifeste)', () => {
    const html = formatHeatmapTooltip('AllyPlayer', '9pm', 4, {
      playerLabel: 'Player',
      bucketTypeLabel: 'Time slot',
      matchesLabel: 'Shared matches',
      emptyCell: 'No match in this slot',
    })
    expect(html).toContain('Player : AllyPlayer')
    expect(html).toContain('Time slot : 9pm')
    expect(html).toContain('Shared matches : 4')
  })

  it('échappe le gamertag (donnée joueur injectée dans du HTML)', () => {
    const html = formatHeatmapTooltip('<img src=x onerror=alert(1)>', '21h', 1, TOOLTIP_FR)
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img')
  })
})
