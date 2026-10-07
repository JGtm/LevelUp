/**
 * TacticalMapTile — UNE carte OUVRABLE de la colonne « Cartes jouées » : vignette compacte de
 * 100 px (fond + mini-plan « où je meurs »), nom, résumé « 54 · 30 V / 24 D » et barre fine
 * victoires / défaites.
 *
 * C'EST UN GROUPE : le BOUTON DE SÉLECTION — ce que le clic fait est dit par son nom accessible
 * (« Sélectionner <carte> ») et son état (`aria-pressed`), la carte active porte une bordure de
 * 2 px `primary` — et, à côté de lui (jamais dedans), l'icône-lien vers les matchs de la carte dans
 * l'Explorateur. Les cartes sous le plancher ne sont pas des vignettes : elles vivent dans le
 * repli de la colonne (`TacticalMapsColumn`).
 *
 * ─── LE MINI-PLAN ──────────────────────────────────────────────────────────────────────────
 *
 * La vignette montre OÙ LE JOUEUR MEURT sur la carte. LE CADRE RESTE À HAUTEUR FIXE (16:9) : des
 * cartes de rapports différents donneraient des vignettes de hauteurs différentes. C'est donc le
 * CALQUE qui s'adapte — mis à l'échelle pour tenir en entier, centré, JAMAIS étiré
 * (`vueContain`) : une zone chaude ronde reste ronde. Sans bornes (carte sans mort mesurée, titre
 * qui ne lit pas les positions), la vignette garde son seul fond.
 */
import { useEffect, useMemo, useRef } from 'react'
import { Link } from '@tanstack/react-router'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { drawTacticalHeatmap, heatRamp } from '@/lib/replay/heatPaint'
import { useTitleSlug } from '@/lib/title-routing'

import { VIGNETTE_LARGEUR_PX } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { useTacticalMapBackgroundFrame, useTacticalMapBackgroundUrl } from './queries'
import { barreResultats, libelleExplorateur, nomCarte } from './tacticalLogic'
import { grilleDuPlan, repereDuPlan, vueContain } from './tacticalView.logic'

interface TacticalMapTileProps {
  carte: TacticalMapCard
  playerSlug: string
  locale: Locale
  t: TacticalText
  selectionnee: boolean
  onSelect: (mapId: string) => void
}

export function TacticalMapTile({ carte, playerSlug, locale, t, selectionnee, onSelect }: TacticalMapTileProps) {
  const nom = nomCarte(carte, locale)
  const parts = barreResultats(carte)
  const fond = useTacticalMapBackgroundUrl(playerSlug, carte.map_id)
  const bordure = selectionnee ? 'border-2 border-primary p-[3px]' : 'border border-border p-1 hover:border-primary'

  return (
    <div className="relative flex-[0_0_var(--tac-cartes-l)] min-[1400px]:flex-none">
      <button
        type="button"
        aria-pressed={selectionnee}
        aria-label={t.select(nom)}
        onClick={() => onSelect(carte.map_id)}
        data-testid={`tactical-map-${carte.map_id}`}
        style={{ gridTemplateColumns: `${VIGNETTE_LARGEUR_PX}px minmax(0, 1fr)` }}
        className={`grid w-full items-center gap-2 rounded-md bg-card text-left ${bordure}`}
      >
        <span
          className="relative block aspect-video overflow-hidden rounded bg-muted"
          style={{ width: VIGNETTE_LARGEUR_PX }}
        >
          {fond && (
            <img
              src={fond}
              alt=""
              aria-hidden
              className="h-full w-full object-cover"
              data-testid={`tactical-map-fond-${carte.map_id}`}
            />
          )}
          <MiniPlan carte={carte} playerSlug={playerSlug} />
        </span>
        <span className="flex min-w-0 flex-col gap-px">
          <span className="truncate text-xs font-medium leading-4 text-foreground">{nom}</span>
          <span className="font-mono text-[10.5px] leading-[14px] tabular-nums text-muted-foreground">
            {t.tileSummary(carte.matchs, carte.victoires, carte.defaites)}
          </span>
          <span
            role="img"
            aria-label={t.recordLabel(carte.victoires, carte.defaites, carte.matchs)}
            className="mt-0.5 flex h-1 w-full overflow-hidden rounded-full bg-muted"
          >
            <span
              className="h-full"
              style={{ width: `${parts.victoires * 100}%`, backgroundColor: tokenCssVar('outcome-win') }}
            />
            <span
              className="h-full"
              style={{ width: `${parts.defaites * 100}%`, backgroundColor: tokenCssVar('outcome-loss') }}
            />
          </span>
          {selectionnee && <span className="sr-only">{t.selected}</span>}
        </span>
      </button>
      <LienExplorateur carte={carte} playerSlug={playerSlug} t={t} />
    </div>
  )
}

/** LienExplorateur — l'icône-lien vers les matchs de la carte dans l'Explorateur (`?maps=<libellé>`). */
function LienExplorateur({ carte, playerSlug, t }: { carte: TacticalMapCard; playerSlug: string; t: TacticalText }) {
  const titleSlug = useTitleSlug()
  return (
    <Link
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/explorer"
      params={{ titleSlug, playerSlug }}
      search={{ maps: libelleExplorateur(carte) }}
      aria-label={t.mapsExplorerLink}
      title={t.mapsExplorerLink}
      data-testid={`tactical-map-explorer-${carte.map_id}`}
      className="absolute bottom-1.5 right-1.5 inline-flex h-5 w-5 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <svg viewBox="0 0 16 16" aria-hidden className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth="1.5">
        <path d="M2.5 4h11M2.5 8h11M2.5 12h7" strokeLinecap="round" />
      </svg>
    </Link>
  )
}

/**
 * MiniPlan — le calque « où je meurs » de la vignette. MÊME NOYAU DE PEINTURE et MÊME rampe
 * d'intensité que le plan de la carte, MÊME REPÈRE (le cadre du fond quand il est connu) : la
 * vignette et le plan se lisent pareil. Sans bornes, rien n'est posé sur le fond.
 */
function MiniPlan({ carte, playerSlug }: { carte: TacticalMapCard; playerSlug: string }) {
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, carte.map_id)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const paletteVersion = useColorPaletteVersion()
  const ramp = useMemo(() => {
    void paletteVersion
    return heatRamp(heatmapRampTokens('intensity').map(resolveToken))
  }, [paletteVersion])
  const bornes = carte.bornes
  const repere = bornes && carte.pas_m ? repereDuPlan(cadreFond, bornes, carte.pas_m) : null
  const grid = repere && carte.echelle ? grilleDuPlan(carte.cellules ?? [], repere, carte.echelle) : null
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !grid || !repere) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    const vue = vueContain(repere, width, height)
    if (!vue) return
    drawTacticalHeatmap(ctx, grid, vue, { ramp, k: 1 })
  }, [grid, ramp, repere])
  if (!grid) return null
  return (
    <canvas
      ref={canvasRef}
      aria-hidden
      className="absolute inset-0 h-full w-full"
      data-testid={`tactical-map-calque-${carte.map_id}`}
    />
  )
}
