/**
 * TacticalPlanCard.tsx — Carte de section pour le plan tactique.
 *
 * Affiche : canvas avec heatmap, légende, pied de page.
 * États : loading, error, empty, pending, unavailable, ok
 */
import { useEffect, useRef, useMemo } from 'react'
import { getTacticalText, type TacticalLocale } from './i18n'
import { type TacticalRasterResponse } from './queries'
import { generateAnalysisTitle, isDivergentQuestion, isRoutesQuestion, type QuestionType } from './tacticalView.logic'

export interface TacticalPlanCardProps {
  locale: TacticalLocale
  mapName: string
  question: QuestionType
  rasterData?: TacticalRasterResponse
  isLoading?: boolean
  isError?: boolean
  isEmpty?: boolean
  onCellSelect?: (col: number, row: number, value: number, matchCount: number) => void
  selectedCell?: { col: number; row: number } | null
}

/**
 * Résout un token CSS en valeur RGB [r, g, b].
 */
function resolveToken(tokenName: string): [number, number, number] {
  const elem = document.documentElement
  const value = getComputedStyle(elem).getPropertyValue(tokenName).trim()
  // Parse 'oklch(...)' ou 'rgb(...)' en RGB
  const match = value.match(/[\d.]+/g)
  if (!match || match.length < 3) return [200, 200, 200] // fallback
  return [+match[0], +match[1], +match[2]]
}

/**
 * Mix deux couleurs RGB par un facteur t ∈ [0, 1].
 */
function mix(a: [number, number, number], b: [number, number, number], t: number): [number, number, number] {
  return [
    a[0] + (b[0] - a[0]) * t,
    a[1] + (b[1] - a[1]) * t,
    a[2] + (b[2] - a[2]) * t,
  ]
}

/**
 * Construit une couleur RGBA à partir de RGB et alpha.
 */
function rgba(rgb: [number, number, number], alpha: number): string {
  return `rgba(${Math.round(rgb[0])},${Math.round(rgb[1])},${Math.round(rgb[2])},${alpha})`
}

/**
 * Palette de chaleur pour heatmap.
 */
function heatColor(t: number): [number, number, number] {
  const a = resolveToken('--ac-info')
  const b = resolveToken('--ac-destructive')
  const c = resolveToken('--ac-extreme')
  t = Math.max(0, Math.min(1, t))
  return t < 0.62 ? mix(a, b, t / 0.62) : mix(b, c, (t - 0.62) / 0.38)
}

/**
 * Palette divergente pour "gagne".
 */
function divColor(t: number): [number, number, number] {
  const lo = resolveToken('--ac-heatmap-divergent-low')
  const hi = resolveToken('--ac-heatmap-divergent-high')
  const nz = resolveToken('--card')
  t = Math.max(0, Math.min(1, t))
  return t < 0.5 ? mix(lo, nz, t * 2) : mix(nz, hi, (t - 0.5) * 2)
}

export function TacticalPlanCard({
  locale,
  mapName,
  question,
  rasterData,
  isLoading = false,
  isError = false,
  isEmpty = false,
  onCellSelect,
  selectedCell,
}: TacticalPlanCardProps) {
  const text = getTacticalText(locale)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const isDivergent = useMemo(() => isDivergentQuestion(question), [question])
  const isRoutes = useMemo(() => isRoutesQuestion(question), [question])

  const title = useMemo(() => {
    return generateAnalysisTitle(mapName, question, locale)
  }, [mapName, question, locale])

  // Dessiner le canvas quand les données changent
  useEffect(() => {
    if (!canvasRef.current || !rasterData || isLoading || isError || isEmpty) return

    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const { cells, gridSize } = rasterData
    const { nx, ny, cell: cellSize, minX, minY } = gridSize

    // Dimensions du canvas
    const w = canvas.width
    const h = canvas.height
    const sx = w / (nx * cellSize)
    const sy = h / (ny * cellSize)

    // Fond clair
    ctx.fillStyle = rgba(resolveToken('--background'), 1)
    ctx.fillRect(0, 0, w, h)

    // Construire un index pour accès rapide aux cellules
    const cellMap = new Map<string, number>()
    for (const cell of cells) {
      cellMap.set(`${cell.col},${cell.row}`, cell.valeur)
    }

    // Calculer quantiles
    const values = cells.map((c) => c.valeur)
    values.sort((a, b) => a - b)
    const p50 = values[Math.floor(values.length * 0.5)] || 0
    const p95 = values[Math.floor(values.length * 0.95)] || 1

    // Dessiner les cellules
    for (const cell of cells) {
      const { col, row, valeur: raw } = cell
      const x = (col - 0) * cellSize - minX
      const y = (row - 0) * cellSize - minY
      const px = x * sx
      const py = y * sy
      const pw = cellSize * sx
      const ph = cellSize * sy

      let t: number
      let alpha: number
      let color: [number, number, number]

      if (isDivergent) {
        // Échelle divergente
        t = 0.5 + (raw / Math.max(p95, 1e-6)) * 0.5
        color = divColor(t)
        alpha = 0.16 + 0.5 * Math.min(Math.abs(raw) / Math.max(p95, 1e-6), 1)
      } else {
        // Échelle heat classique
        t = (raw - p50) / Math.max(p95 - p50, 1e-6)
        if (t <= 0) continue // pas de peinture en froid
        color = heatColor(t)
        alpha = 0.09 + 0.55 * Math.min(t, 1)
      }

      ctx.fillStyle = rgba(color, alpha)
      ctx.fillRect(px, py, pw + 0.7, ph + 0.7)
    }

    // Dessiner la cellule sélectionnée si elle existe
    if (selectedCell && onCellSelect) {
      const { col, row } = selectedCell
      const x = (col - 0) * cellSize - minX
      const y = (row - 0) * cellSize - minY
      const px = x * sx
      const py = y * sy
      const pw = cellSize * sx
      const ph = cellSize * sy

      ctx.strokeStyle = rgba(resolveToken('--foreground'), 1)
      ctx.lineWidth = 1.5
      ctx.strokeRect(px - 3, py - 3, pw + 6, ph + 6)
    }
  }, [rasterData, isLoading, isError, isEmpty, isDivergent, selectedCell, onCellSelect])

  // Gestionnaire de clic sur le canvas
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!rasterData || !onCellSelect) return

    const canvas = canvasRef.current
    if (!canvas) return

    const rect = canvas.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top

    const { nx, ny, cell: cellSize, minX, minY } = rasterData.gridSize
    const w = canvas.width
    const h = canvas.height
    const sx = w / (nx * cellSize)
    const sy = h / (ny * cellSize)

    const col = Math.floor((x / sx + minX) / cellSize)
    const row = Math.floor((y / sy + minY) / cellSize)

    // Trouver la valeur dans le raster
    const cell = rasterData.cells.find((c) => c.col === col && c.row === row)
    if (cell) {
      const matchCount = Math.max(1, Math.round(Math.abs(cell.valeur) * 10))
      onCellSelect(col, row, cell.valeur, matchCount)
    }
  }

  // Génération du footer
  const footerContent = useMemo(() => {
    if (!rasterData) return ''
    const { matchs_retenus, matchs_filtres } = rasterData
    const source = rasterData.cells.length > 0
      ? locale === 'fr' ? 'base partagée' : 'shared database'
      : locale === 'fr' ? 'artefacts de rejeu' : 'replay artifacts'

    if (isRoutes) {
      return `${text.planFooterRoutes} ${matchs_retenus} matchs retenus sur ${matchs_filtres}.`
    }

    if (isDivergent) {
      return `${text.planFooterDivergent} ${matchs_retenus} matchs retenus sur ${matchs_filtres} — source : ${source}.`
    }

    return `${text.planFooterQuantile} ${matchs_retenus} matchs retenus sur ${matchs_filtres} — source : ${source}.`
  }, [rasterData, isRoutes, isDivergent, text, locale])

  // Rendu
  return (
    <div className="rounded-md border border-border bg-card">
      <h3 className="border-b border-border px-3 py-2 text-sm font-medium">{title}</h3>

      <div className="p-3">
        {isLoading && <div className="flex justify-center py-8 text-sm text-muted-foreground">{text.loadingMessage}</div>}
        {isError && <div className="flex justify-center py-8 text-sm text-destructive">{text.errorState}</div>}
        {isEmpty && <div className="flex justify-center py-8 text-sm text-muted-foreground">{text.emptyState}</div>}

        {!isLoading && !isError && !isEmpty && (
          <>
            <canvas
              ref={canvasRef}
              width={740}
              height={444}
              onClick={handleCanvasClick}
              className="block w-full cursor-crosshair rounded border border-border bg-background"
              role="img"
              aria-label={text.planCanvasLabel(mapName, text.questionDeaths)}
            />

            {/* Légende */}
            <div className="mt-3 flex items-center flex-wrap gap-2">
              <span className="text-xs font-mono text-muted-foreground">
                {rasterData?.scale.lo ?? '0'}
              </span>
              <div
                className="flex-1 h-2 min-w-32 rounded-full"
                style={{
                  background: isDivergent
                    ? 'linear-gradient(90deg, var(--ac-heatmap-divergent-low), var(--card), var(--ac-heatmap-divergent-high))'
                    : 'linear-gradient(90deg, var(--ac-info) 0%, var(--ac-destructive) 62%, var(--ac-extreme) 100%)',
                }}
              />
              <span className="text-xs font-mono text-muted-foreground">
                {rasterData?.scale.hi ?? '1'}
              </span>
            </div>
          </>
        )}
      </div>

      {footerContent && (
        <div className="border-t border-border px-3 py-3 text-xs text-muted-foreground">
          <p>{footerContent}</p>
        </div>
      )}
    </div>
  )
}
