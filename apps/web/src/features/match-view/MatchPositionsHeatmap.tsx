/**
 * MatchPositionsHeatmap — « OÙ ÇA SE JOUE » : les positions du match, sur le PLAN du match.
 *
 * CE BLOC A ÉTÉ REFAIT LE 2026-09-13, sur un constat de l'utilisateur : « "Carte de chaleur
 * des positions" est hideux comme graphe et je ne sais pas ce que ça rend, à quoi ça sert ou
 * quel est le narratif. » Il avait raison sur la FORME, pas sur la donnée : les positions
 * keyframe décodées du film (couche 3 weapon-attribution-v3) sont bonnes, mais elles étaient
 * binnées en une grille 20×20 posée sur les bornes du nuage, SANS fond de carte, avec un
 * nombre écrit dans chaque case. Un damier de chiffres ne dit rien d'un terrain.
 *
 * CE QU'IL RESTE : le même calque que l'onglet Tactique et que le rejeu 2D — le FOND DE CARTE
 * du match (`…/replay/background`, calé au centimètre) et le noyau de tracé partagé
 * (`lib/replay/heatPaint.ts`, `drawTacticalHeatmap`). La grille suit le pas du rejeu (0,5 m) et
 * l'échelle quantile p50→p95 ; une cellule jamais atteinte reste vide.
 *
 * TROIS PORTES, ET ELLES DISENT TROIS CHOSES :
 *   1. aucune position décodée (titre sans film, match non backfillé) -> rien ;
 *   2. la carte du match n'a pas d'image figée (seules 21 en ont) -> rien : « Où ça se joue »
 *      est un plan, et un plan sans fond est le damier qu'on vient de retirer ;
 *   3. les positions tombent toutes hors du cadre du fond -> rien (rien à peindre).
 *
 * LES CAMPS SONT CEUX DU FILM, PAS CEUX DU TABLEAU DES SCORES. `team` vaut -1 (inconnu) ou
 * 0/1, attribué par regroupement SPATIAL best-effort (§N de RESEARCH_THEATER_RE) : aucune
 * jointure ne le relie à un xuid ni à un camp nommé. Le filtre écrit donc « Camp A » / « Camp
 * B », jamais « mon équipe » / « adversaires » — nommer un camp qu'on n'a pas mesuré serait
 * exactement la devinette que le reste de la page refuse.
 */
import { useEffect, useMemo, useRef, useState } from 'react'

import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { SectionCard } from '@/components/ui/section-card'
import { Button } from '@/components/ui/button'
import { resolveToken } from '@/lib/accessibility/resolveToken'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { MatchPlayerPosition } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { drawTacticalHeatmap, heatRamp } from '@/lib/replay/heatPaint'
import { useReplayMapBackground, useReplayMapImage } from '@/lib/replay/queries'

import {
  buildPositionsGrid,
  coveredShare,
  hasTeamSplit,
  mapFrame,
  positionsCellSize,
} from './_positionsHeat'

type TeamFilter = 'all' | 0 | 1

interface MatchPositionsHeatmapProps {
  playerSlug: string
  matchId: string
  positions: MatchPlayerPosition[] | undefined
  locale: Locale
}

const TEXT = {
  fr: {
    title: 'Où ça se joue',
    teamAll: 'Tous',
    team0: 'Camp A',
    team1: 'Camp B',
    narrative:
      'Les endroits de la carte les plus occupés pendant ce match, tous camps ou camp par camp. Lecture : plus c’est chaud, plus on y a passé de temps.',
    coverageFmt: (cell: number, share: number, total: number) =>
      `Grille de ${cell.toFixed(1).replace('.', ',')} m · ${Math.round(share * 100)} % des ${total} positions décodées tombent sur le plan. Camps attribués par regroupement spatial, sans nom de joueur.`,
  },
  en: {
    title: 'Where it plays out',
    teamAll: 'All',
    team0: 'Side A',
    team1: 'Side B',
    narrative:
      'The busiest spots of the map during this match, all sides or side by side. Read it this way: the hotter, the longer it was held.',
    coverageFmt: (cell: number, share: number, total: number) =>
      `${cell.toFixed(1)} m grid · ${Math.round(share * 100)}% of the ${total} decoded positions land on the plan. Sides inferred from spatial clustering, with no player name.`,
  },
} as const satisfies Record<Locale, unknown>

export function MatchPositionsHeatmap({
  playerSlug,
  matchId,
  positions,
  locale,
}: MatchPositionsHeatmapProps) {
  const t = TEXT[locale]
  const [teamFilter, setTeamFilter] = useState<TeamFilter>('all')
  const canvasRef = useRef<HTMLCanvasElement>(null)

  const all = useMemo(() => positions ?? [], [positions])
  const { data: background } = useReplayMapBackground(playerSlug, matchId)
  const image = useReplayMapImage(playerSlug, matchId, !!background && all.length > 0)

  const frame = useMemo(() => (background ? mapFrame(background.calibration) : null), [background])
  const teamSplit = useMemo(() => hasTeamSplit(all), [all])
  const filtered = useMemo(
    () => (teamFilter === 'all' ? all : all.filter((p) => p.team === teamFilter)),
    [all, teamFilter],
  )
  const grid = useMemo(
    () => (frame ? buildPositionsGrid(filtered, frame) : null),
    [filtered, frame],
  )

  // Rampe précalculée PAR THÈME, résolue une fois par changement de palette (même patron que
  // `TacticalPlanCard` et `useReplayHeatmap`) — jamais recalculée par cellule.
  const paletteVersion = useColorPaletteVersion()
  const ramp = useMemo(() => {
    void paletteVersion
    return heatRamp(heatmapRampTokens('intensity').map(resolveToken))
  }, [paletteVersion])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !grid || !frame || !image) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    // LE FOND ET LE CALQUE PARTAGENT LE MÊME CADRE : l'image est peinte sur toute la toile, et
    // l'échelle du calque est celle de cette image — `scale` px par mètre monde. C'est la
    // condition pour qu'une cellule tombe sur le couloir qu'elle décrit.
    ctx.drawImage(image, 0, 0, width, height)
    drawTacticalHeatmap(
      ctx,
      grid,
      { topLeftWorld: { x: 0, y: 0 }, scale: width / frame.widthM },
      { ramp, k: 1 },
    )
  }, [grid, frame, image, ramp])

  // Portes 1 à 3 : rien à montrer, et rien à promettre.
  if (all.length === 0 || !frame || !grid) return null

  const share = coveredShare(all, frame)
  return (
    <SectionCard
      title={t.title}
      label={t.title}
      footer={
        <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
          <p>{t.narrative}</p>
          <p>{t.coverageFmt(positionsCellSize(frame), share, all.length)}</p>
        </div>
      }
      titleAdornment={(label) => (
        <span className="flex items-center justify-between gap-2">
          <span>{label}</span>
          {teamSplit && (
            <span className="flex gap-1">
              <TeamButton active={teamFilter === 'all'} onClick={() => setTeamFilter('all')}>
                {t.teamAll}
              </TeamButton>
              <TeamButton active={teamFilter === 0} onClick={() => setTeamFilter(0)}>
                {t.team0}
              </TeamButton>
              <TeamButton active={teamFilter === 1} onClick={() => setTeamFilter(1)}>
                {t.team1}
              </TeamButton>
            </span>
          )}
        </span>
      )}
    >
      <div className="p-3">
        {/* Le cadre prend le RAPPORT DU MONDE (bornes du fond), jamais un 16:9 : le calque et
            l'image se désaligneraient sur l'axe rogné (même règle que `TacticalPlanCard`). */}
        <div
          className="relative w-full overflow-hidden rounded-md bg-muted"
          style={{ aspectRatio: `${frame.widthM} / ${frame.heightM}` }}
          data-testid="match-positions-frame"
        >
          <canvas
            ref={canvasRef}
            className="absolute inset-0 h-full w-full"
            data-testid="match-positions-canvas"
          />
        </div>
      </div>
    </SectionCard>
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
