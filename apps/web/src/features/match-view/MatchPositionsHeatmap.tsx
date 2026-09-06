/**
 * MatchPositionsHeatmap — heatmap 2D top-down (x,y) des positions joueurs
 * keyframe décodées du film (couche 3 weapon-attribution-v3).
 *
 * Source : GET /players/{slug}/matches/{matchId}/positions (DTO MatchPlayerPosition).
 * Les positions sont MATCH-LEVEL (§N de .ai/RESEARCH_THEATER_RE.md) : pas
 * d'attribution xuid. team vaut -1 (inconnu) ou 0/1 (best-effort). Quand au moins
 * une position a team != -1, on propose un filtre par équipe ; sinon densité
 * globale uniquement.
 *
 * Rendu : binning (x,y) en grille GRID_SIZE×GRID_SIZE sur les bornes du match →
 * densité par cellule → réutilise le wrapper Heatmap2DChart (axes catégoriels =
 * centres de bin arrondis ; value = compte de positions ; detail.count idem pour
 * le label/tooltip du wrapper).
 *
 * Dégradation : pas de positions (titre sans film / match non backfillé → 503 →
 * data undefined, ou décodage vide) → composant masqué proprement (retourne null).
 */
import { useMemo, useState } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { Button } from '@/components/ui/button'
import type { MatchPlayerPosition } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/** Taille de la grille de binning (GRID_SIZE × GRID_SIZE cellules). */
const GRID_SIZE = 20

/** team = -1 → équipe non attribuée (best-effort §N). */
const TEAM_UNKNOWN = -1

type TeamFilter = 'all' | 0 | 1

interface MatchPositionsHeatmapProps {
  positions: MatchPlayerPosition[] | undefined
  locale: Locale
}

const TEXT = {
  fr: {
    title: 'Carte de chaleur des positions',
    teamAll: 'Global',
    team0: 'Équipe A',
    team1: 'Équipe B',
    empty: 'Aucune position décodée pour ce match.',
    note: 'Positions au niveau match (sans attribution joueur).',
  },
  en: {
    title: 'Positions heatmap',
    teamAll: 'Global',
    team0: 'Team A',
    team1: 'Team B',
    empty: 'No positions decoded for this match.',
    note: 'Match-level positions (no per-player attribution).',
  },
} as const

/**
 * binPositionsToHeatmap binne un set de positions (x,y) en une grille
 * GRID_SIZE×GRID_SIZE sur les bornes [min,max] de chaque axe, et renvoie un
 * datapoint par cellule NON VIDE (x/y = centre de bin arrondi, value = densité).
 *
 * Pur (testable sans React). Retourne [] si positions est vide ou si toutes les
 * positions sont colinéaires sur un axe de span nul (bornes dégénérées tolérées
 * via un span minimal de 1 pour éviter une division par zéro).
 */
export function binPositionsToHeatmap(positions: MatchPlayerPosition[]): ChartPointHeatmap[] {
  if (positions.length === 0) return []

  let xmin = positions[0].x
  let xmax = positions[0].x
  let ymin = positions[0].y
  let ymax = positions[0].y
  for (const p of positions) {
    if (p.x < xmin) xmin = p.x
    if (p.x > xmax) xmax = p.x
    if (p.y < ymin) ymin = p.y
    if (p.y > ymax) ymax = p.y
  }
  const xSpan = Math.max(xmax - xmin, 1)
  const ySpan = Math.max(ymax - ymin, 1)

  // Accumulation densité par (xi, yi) de cellule.
  const counts = new Map<string, number>()
  const binIndex = (v: number, min: number, span: number): number => {
    const i = Math.floor(((v - min) / span) * GRID_SIZE)
    return Math.min(Math.max(i, 0), GRID_SIZE - 1)
  }
  for (const p of positions) {
    const xi = binIndex(p.x, xmin, xSpan)
    const yi = binIndex(p.y, ymin, ySpan)
    const key = `${xi}:${yi}`
    counts.set(key, (counts.get(key) ?? 0) + 1)
  }

  // Centre de bin arrondi → label catégoriel lisible pour les axes du wrapper.
  const center = (i: number, min: number, span: number): string => {
    const c = min + ((i + 0.5) / GRID_SIZE) * span
    return c.toFixed(0)
  }

  const out: ChartPointHeatmap[] = []
  for (const [key, count] of counts) {
    const [xiStr, yiStr] = key.split(':')
    const xi = Number(xiStr)
    const yi = Number(yiStr)
    out.push({
      x: center(xi, xmin, xSpan),
      y: center(yi, ymin, ySpan),
      value: count,
      detail: { count },
    })
  }
  return out
}

/** hasTeamSplit indique si au moins une position porte une équipe attribuée (0/1). */
function hasTeamSplit(positions: MatchPlayerPosition[]): boolean {
  return positions.some((p) => p.team !== TEAM_UNKNOWN)
}

export function MatchPositionsHeatmap({ positions, locale }: MatchPositionsHeatmapProps) {
  const t = TEXT[locale]
  const [teamFilter, setTeamFilter] = useState<TeamFilter>('all')

  const all = positions ?? []
  const teamSplit = useMemo(() => hasTeamSplit(all), [all])

  const filtered = useMemo(() => {
    if (teamFilter === 'all') return all
    return all.filter((p) => p.team === teamFilter)
  }, [all, teamFilter])

  const series = useMemo<ChartSeries<ChartPointHeatmap>[]>(() => {
    const datapoints = binPositionsToHeatmap(filtered)
    if (datapoints.length === 0) return []
    return [{ key: 'density', datapoints }]
  }, [filtered])

  // Dégradation : aucune position décodée → composant masqué proprement.
  if (all.length === 0) return null

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
        <span className="text-sm font-medium">{t.title}</span>
        {teamSplit && (
          <div className="flex gap-1">
            <TeamButton active={teamFilter === 'all'} onClick={() => setTeamFilter('all')}>
              {t.teamAll}
            </TeamButton>
            <TeamButton active={teamFilter === 0} onClick={() => setTeamFilter(0)}>
              {t.team0}
            </TeamButton>
            <TeamButton active={teamFilter === 1} onClick={() => setTeamFilter(1)}>
              {t.team1}
            </TeamButton>
          </div>
        )}
      </div>
      <div className="p-3">
        <Heatmap2DChart series={series} emptyMessage={t.empty} height={360} />
        <p className="mt-2 text-xs text-muted-foreground">{t.note}</p>
      </div>
    </div>
  )
}

interface TeamButtonProps {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}

function TeamButton({ active, onClick, children }: TeamButtonProps) {
  return (
    <Button
      variant={active ? 'default' : 'ghost'}
      size="sm"
      onClick={onClick}
      className="h-7 px-2 text-xs"
    >
      {children}
    </Button>
  )
}
