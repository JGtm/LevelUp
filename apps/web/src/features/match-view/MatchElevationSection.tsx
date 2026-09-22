/**
 * MatchElevationSection — LE DÉNIVELÉ DU MATCH : OÙ JE FRAGUE, OÙ JE MEURS.
 *
 * Forme M1 de la maquette MAQUETTE_DENIVELE_V2_2026-09-22.html (décision D24, 2026-09-22),
 * posée à côté de « Distance des frags par arme » et dans la même grammaire : même
 * `SectionCard`, même (i) de titre, même réserve de couverture. Les deux cartes lisent la
 * MÊME source (`kill_positions × match_kill_events`) à deux grains : celle-ci ne groupe
 * rien — un point par frag, un point par mort.
 *
 * DEUX PORTES, ET ELLES DISENT DEUX CHOSES DIFFÉRENTES (même règle que la carte voisine) :
 *   1. LE TITRE — `film.kill_positions` : sans décodeur de film, ces positions n'existeront
 *      jamais. La section n'est alors pas rendue du tout.
 *   2. LE MATCH — le bloc absent, ou présent sans point : le titre sait les produire, mais
 *      pas pour CE match-là. La section NOMME la cause au lieu de disparaître.
 *
 * UN POINT EST UNE PORTE VERS LE REJEU. Le clic reprend le mécanisme déjà en service sur la
 * règle des records et la grille Tactique : `?t=<time_ms>&clock=match`, navigation par le
 * routeur, l'instant étant recalé sur l'axe du film par la route du rejeu une fois le
 * document chargé. Aucun troisième mécanisme.
 *
 * LE LOBBY EST UNE OPTION DE CETTE CARTE, PAS UNE CARTE DE PLUS (M3 de la maquette) : le
 * bouton « comparer au lobby » ajoute le fond gris des frags des autres et le repère de leur
 * médiane. État LOCAL, relâché par défaut — 90 points gris derrière 25 colorés encombrent,
 * et ce n'est pas la lecture par défaut.
 *
 * AUCUNE PHRASE DE LECTEUR (D22-verbosité, LOI) : le graphe, sa légende, et trois phrases
 * dans l'infobulle du titre.
 */
import { useNavigate } from '@tanstack/react-router'
import { useCallback, useMemo, useState } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { Button } from '@/components/ui/button'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { resolveToken } from '@/lib/accessibility'
import type { MatchElevationBlock } from '@/lib/api/types'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { formatNumberFixed } from '@/lib/formatters/number'
import { formatClock } from '@/lib/replay/replayLogic'
import { useTitleSlug } from '@/lib/title-routing'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  ELEVATION_SERIES_LOBBY,
  buildElevationOption,
  elevationPoints,
  elevationTooltipLines,
  type ElevationPoint,
} from './_elevation'
import type { MatchViewText } from './i18n'

/** Hauteur du nuage : assez pour que ±3 m de dénivelé ne s'écrasent pas sur la bande. */
const ELEVATION_CHART_HEIGHT = 300

interface Props {
  /** Bloc du contrat — absent quand le match n'a aucune position mesurée. */
  block: MatchElevationBlock | null | undefined
  /** Slug du joueur de la page : la route du rejeu en a besoin. */
  playerSlug: string
  matchId: string
  t: MatchViewText
}

export function MatchElevationSection({ block, playerSlug, matchId, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const titreMesureLesPositions = useDataCapability('film.kill_positions')
  const titleSlug = useTitleSlug()
  const navigate = useNavigate()
  const [compareLobby, setCompareLobby] = useState(false)

  const mine = useMemo(() => elevationPoints(block?.kills, locale), [block?.kills, locale])
  const kills = useMemo(() => mine.filter((p) => p.side === 'kill'), [mine])
  const deaths = useMemo(() => mine.filter((p) => p.side === 'death'), [mine])
  const lobbyAll = useMemo(() => elevationPoints(block?.lobby, locale), [block?.lobby, locale])
  const lobby = compareLobby ? lobbyAll : EMPTY_POINTS

  const meters = useCallback((v: number) => formatNumberFixed(v, 1), [])
  const signedMeters = useCallback(
    (v: number) => `${v > 0 ? '+' : ''}${formatNumberFixed(v, 1)}`,
    [],
  )

  const tooltip = useCallback(
    (p: ElevationPoint) =>
      elevationTooltipLines([
        t.elevationPointFmt(
          p.side === 'kill' ? t.elevationSideKill : t.elevationSideDeath,
          meters(p.value[0]),
          signedMeters(p.value[1]),
        ),
        t.elevationPointWeaponFmt(
          p.opponent ? `${p.weapon}${p.weapon ? ' · ' : ''}${p.opponent}` : p.weapon,
          formatClock(p.timeMs),
        ),
        t.elevationOpenReplay,
      ]),
    [t, meters, signedMeters],
  )

  const buildOption = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildElevationOption({
      kills,
      deaths,
      lobby,
      lobbyMedian: block?.lobby_median_delta_z_m ?? null,
      tc,
      colors: {
        kills: resolveToken('stat-kills'),
        deaths: resolveToken('stat-deaths'),
        // Le fond du lobby et les deux repères prennent les encres NEUTRES du thème (celles
        // des axes) : ce ne sont pas des grandeurs, ce sont des références — leur donner un
        // token de série les ferait lire comme une troisième population.
        lobby: tc.axisLabel,
        band: tc.splitAreaB,
        zero: tc.axisLine,
      },
      labels: {
        kills: t.elevationLegendKillsFmt(kills.length),
        deaths: t.elevationLegendDeathsFmt(deaths.length),
        lobby: t.elevationLegendLobbyFmt(lobbyAll.length),
        xAxis: t.elevationAxisDistance,
        yAxis: t.elevationAxisDelta,
        lobbyMedian: (m) => t.elevationLobbyMedianFmt(signedMeters(m)),
      },
      tooltip,
    })
  }, [kills, deaths, lobby, lobbyAll.length, block?.lobby_median_delta_z_m, t, tooltip, signedMeters])

  // LE CLIC : la série du lobby est `silent` côté ECharts, elle n'émet donc rien — la garde
  // ci-dessous est la seconde ceinture, pas la première (une option future qui la rendrait
  // interactive ne doit pas ouvrir le rejeu sur le frag d'un inconnu).
  const onEvents = useMemo(
    () => ({
      click: (params: unknown) => {
        const p = params as { seriesId?: string; data?: ElevationPoint }
        if (!p.data || p.seriesId === ELEVATION_SERIES_LOBBY) return
        void navigate({
          to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
          params: { titleSlug, playerSlug, matchId },
          // `t` est une CHAÎNE dans le schéma de la route (cf. TacticalCellCard) ; `clock:
          // match` parce que `time_ms` est l'horloge du match, recalée par la route du rejeu.
          search: { t: String(p.data.timeMs), clock: 'match' as const },
        })
      },
    }),
    [navigate, titleSlug, playerSlug, matchId],
  )

  // PORTE 1 — le titre ne produit pas de positions par kill : rien à afficher, jamais.
  if (!titreMesureLesPositions) return null

  const vide = mine.length === 0

  return (
    <SectionCard
      title={t.elevationTitle}
      label={t.elevationTitle}
      titleAdornment={titleWithInfo(<p>{t.elevationInfo}</p>)}
      footer={
        vide || (block?.total_kills ?? 0) === 0 ? undefined : (
          <p className="px-3 pb-3 text-2xs text-muted-foreground">
            {t.elevationCoverageFmt(block?.measured_kills ?? 0, block?.total_kills ?? 0)}
          </p>
        )
      }
    >
      {vide ? (
        <p className="px-3 pb-3 pt-2 text-xs text-muted-foreground">{t.elevationEmpty}</p>
      ) : (
        <>
          <ChartCard
            series={[{ key: 'elevation', datapoints: mine }]}
            buildOption={buildOption}
            height={ELEVATION_CHART_HEIGHT}
            onEvents={onEvents}
            frameless
          />
          <div className="flex flex-wrap items-center justify-between gap-2 px-3 pb-2">
            <ChartLegend
              ariaLabel={t.elevationTitle}
              items={[
                { key: 'kills', label: t.elevationLegendKillsFmt(kills.length), color: resolveToken('stat-kills') },
                { key: 'deaths', label: t.elevationLegendDeathsFmt(deaths.length), color: resolveToken('stat-deaths') },
                { key: 'band', label: t.elevationLegendBand, color: getEChartsThemeColors().splitAreaB },
                ...(compareLobby
                  ? [
                      {
                        key: 'lobby',
                        label: t.elevationLegendLobbyFmt(lobbyAll.length),
                        color: getEChartsThemeColors().axisLabel,
                      },
                    ]
                  : []),
              ]}
            />
            {lobbyAll.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                aria-pressed={compareLobby}
                onClick={() => setCompareLobby((v) => !v)}
                data-testid="elevation-compare-lobby"
              >
                {t.elevationCompare}
              </Button>
            )}
          </div>
        </>
      )}
    </SectionCard>
  )
}

/** Référence stable : un littéral `[]` neuf à chaque rendu recomposerait l'option ECharts. */
const EMPTY_POINTS: ElevationPoint[] = []
