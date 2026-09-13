/**
 * TacticalMapTile — UNE carte de la grille : son fond, son nom, son nombre de matchs et sa
 * barre victoires / défaites.
 *
 * DEUX ÉTATS, ET UN SEUL EST CLIQUABLE :
 *   - carte OUVRABLE : bouton de sélection. Ce que le clic fait est dit par son nom
 *     accessible (« Sélectionner <carte> ») et son état (`aria-pressed`) ; la sélection
 *     est visible (encadré) et vit dans l'URL, donc partageable.
 *   - carte SOUS LE PLANCHER : bouton DÉSACTIVÉ (`disabled` + `aria-disabled`), désaturé,
 *     portant sa raison EN CLAIR (« N matchs sur 10 requis »). Elle reste affichée — le
 *     joueur doit voir qu'il y a joué — mais rien ne laisse croire qu'elle s'ouvrira.
 *
 * La désaturation passe par des utilitaires SANS couleur (`opacity`, `grayscale`) : aucun
 * token n'est détourné pour dire « indisponible ».
 *
 * ─── LE MINI-PLAN (maquette 034b1915, porté le 2026-09-13) ───────────────────
 *
 * La vignette ne montre pas qu'un fond de carte : elle montre OÙ LE JOUEUR Y MEURT. C'est
 * ce qui fait de la grille un CHOIX plutôt qu'une liste — on ouvre la carte dont la forme
 * intrigue.
 *
 * LE CADRE RESTE À HAUTEUR FIXE (16:9), et c'est ce qui distingue une vignette du plan
 * qu'elle ouvre : des cartes de rapports différents donneraient des vignettes de hauteurs
 * différentes, et la grille perdrait ses lignes (constaté sur Illusion, dont le monde est
 * en hauteur). C'est donc le CALQUE qui s'adapte — mis à l'échelle pour tenir en entier,
 * centré, JAMAIS étiré (`planCanvasViewContain`) : une zone chaude ronde doit rester ronde,
 * sans quoi deux vignettes ne se comparent plus.
 *
 * Sans bornes (carte sans mort mesurée, titre qui ne lit pas les positions), la vignette
 * garde son seul fond — comme avant ce lot.
 */
import { useEffect, useMemo, useRef } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { drawTacticalHeatmap, heatRamp } from '@/lib/replay/heatPaint'

import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundUrl } from './queries'
import { barreResultats, estOuvrable, nomCarte } from './tacticalLogic'
import { planCanvasViewContain, tacticalGridFromRaster } from './tacticalView.logic'

interface TacticalMapTileProps {
  carte: TacticalMapCard
  plancher: number
  playerSlug: string
  locale: Locale
  t: TacticalText
  selectionnee: boolean
  onSelect: (mapId: string) => void
}

export function TacticalMapTile({
  carte,
  plancher,
  playerSlug,
  locale,
  t,
  selectionnee,
  onSelect,
}: TacticalMapTileProps) {
  const ouvrable = estOuvrable(carte)
  const nom = nomCarte(carte, locale)
  const parts = barreResultats(carte)
  const fond = useTacticalMapBackgroundUrl(playerSlug, carte.map_id)

  const bordure = selectionnee ? 'border-2 border-primary' : 'border border-border'
  const attenuation = ouvrable ? '' : ' opacity-60 grayscale'

  // ── Le mini-plan « où je meurs » ──────────────────────────────────────────
  // MÊME NOYAU DE PEINTURE que le plan d'une carte (`drawTacticalHeatmap`) et MÊME rampe
  // d'intensité : une vignette et le plan qu'elle ouvre doivent se lire pareil.
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const paletteVersion = useColorPaletteVersion()
  const ramp = useMemo(() => {
    void paletteVersion
    return heatRamp(heatmapRampTokens('intensity').map(resolveToken))
  }, [paletteVersion])
  const bornes = carte.bornes
  const grid =
    bornes && carte.pas_m && carte.echelle
      ? tacticalGridFromRaster(carte.cellules ?? [], bornes, carte.pas_m, carte.echelle)
      : null
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !grid || !bornes) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    const vue = planCanvasViewContain(bornes, width, height)
    if (!vue) return
    drawTacticalHeatmap(ctx, grid, vue, { ramp, k: 1 })
  }, [grid, ramp, bornes])

  return (
    <button
      type="button"
      disabled={!ouvrable}
      aria-disabled={!ouvrable}
      aria-pressed={ouvrable ? selectionnee : undefined}
      aria-label={ouvrable ? t.select(nom) : undefined}
      onClick={ouvrable ? () => onSelect(carte.map_id) : undefined}
      data-testid={`tactical-map-${carte.map_id}`}
      className={`flex flex-col overflow-hidden rounded-lg bg-card text-left ${bordure}${attenuation}${
        ouvrable ? ' hover:border-primary' : ' cursor-not-allowed'
      }`}
    >
      <span className="relative block aspect-video w-full overflow-hidden bg-muted">
        {fond && (
          <img
            src={fond}
            alt=""
            aria-hidden
            className="h-full w-full object-cover"
            data-testid={`tactical-map-fond-${carte.map_id}`}
          />
        )}
        {grid && (
          <canvas
            ref={canvasRef}
            aria-hidden
            className="absolute inset-0 h-full w-full"
            data-testid={`tactical-map-calque-${carte.map_id}`}
          />
        )}
      </span>

      <span className="flex flex-col gap-1 p-3">
        <span className="text-sm font-medium text-foreground">{nom}</span>
        {/* UNE SEULE LIGNE (maquette 034b1915) : « N matchs · V V / D D ». La version
            précédente disait la même chose en deux lignes, de part et d'autre de la
            barre — le compte au-dessus, le bilan en dessous. */}
        <span className="font-mono text-xs tabular-nums text-muted-foreground">
          {t.tileSummary(carte.matchs, carte.victoires, carte.defaites)}
        </span>

        <span
          role="img"
          aria-label={t.recordLabel(carte.victoires, carte.defaites, carte.matchs)}
          className="mt-1 flex h-1.5 w-full overflow-hidden rounded-full bg-muted"
        >
          <span
            className="h-full"
            style={{
              width: `${parts.victoires * 100}%`,
              backgroundColor: tokenCssVar('outcome-win'),
            }}
          />
          <span
            className="h-full"
            style={{
              width: `${parts.defaites * 100}%`,
              backgroundColor: tokenCssVar('outcome-loss'),
            }}
          />
        </span>
        {!ouvrable && (
          <span
            className="text-xs text-muted-foreground"
            data-testid={`tactical-map-plancher-${carte.map_id}`}
          >
            {t.floorReason(carte.matchs, plancher)}
          </span>
        )}
        {ouvrable && selectionnee && <span className="sr-only">{t.selected}</span>}
      </span>
    </button>
  )
}
