/**
 * MatchKillDistanceSection — LA DISTANCE DES FRAGS PAR ARME, UN SEUL GRAPHE.
 *
 * Née tableau (LOT G.3-POC, 2026-08-30, DEC-8), devenue un graphe PAR JOUEUR le 2026-09-02,
 * elle est depuis le 2026-09-13 UN SEUL graphe dont l'axe est groupé PAR ARME — cadrage
 * utilisateur : « il vaut mieux afficher des sections par armes, on met les joueurs sous
 * chaque arme mais toujours au sein du même graphe, pas comme c'est aujourd'hui. Et le
 * joueur actif et ses amis doivent avoir une couleur différente. » Empiler huit cartes
 * d'échelles différentes empêchait la seule comparaison qui compte : à quelle distance CETTE
 * arme tue, et qui la tient de plus loin.
 *
 * LA COULEUR DIT QUI, PAS QUOI. Les bâtons prennent la palette de joueurs du match
 * (`colors.ts` — moi, amis d'escouade, reste de mon équipe, adversaires), la même que la
 * courbe des frags différentiels de l'onglet Joueurs : un joueur garde son encre d'un bloc à
 * l'autre. La LÉGENDE liste les joueurs (et non les rôles) : sur un match à huit lignes elle
 * tient sur deux rangs, et c'est le seul endroit où un gamertag tronqué sur l'axe se relit en
 * entier sans survol.
 *
 * DEUX PORTES, UNE SEULE ISSUE : la section n'est rendue que si elle a un bâton à tracer.
 *   1. LE TITRE — `film.kill_positions` : un titre sans décodeur de film n'a pas ces positions.
 *   2. LE MATCH — aucune distance publiée pour CE match : la section n'est pas rendue non plus.
 *      Aucun texte ne dit l'absence ni sa cause (règle : aucun inconnu à l'écran, garde
 *      `lib/i18n/noUnknownMentions.guard.test.ts`).
 *
 * L'infobulle du titre définit la grandeur tracée (distance tueur-victime au moment du frag).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken } from '@/lib/accessibility'
import type {
  MatchKillDistancePlayer,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { stripBotSuffix } from '@/lib/players/displayName'
import { useAppShellStore } from '@/stores/appShellStore'

import { useHasKillDistanceSection } from './blockPredicates'
import { buildMatchPlayerColors } from './colors'
import {
  buildKillDistanceOption,
  killDistanceHeight,
  killDistanceRows,
  type KillDistancePlayerInput,
} from './_killDistanceChart'
import type { MatchViewText } from './i18n'

interface Props {
  players: MatchKillDistancePlayer[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  roster?: MatchRosterRow[] | null
  /** xuid du joueur de la page — son bâton prend l'encre du joueur principal. */
  meXUID?: string | null
  /** Amis d'escouade (liste du joueur) : encres coéquipier côté allié. */
  friendGamertags?: readonly string[]
  t: MatchViewText
}

export function MatchKillDistanceSection({
  players,
  scoreboard,
  roster,
  meXUID,
  friendGamertags,
  t,
}: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const titreMesureLesPositions = useHasKillDistanceSection()
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const rawPlayers = useMemo(() => players ?? [], [players])

  const palette = useMemo(
    () => buildMatchPlayerColors(board, meXUID ?? null, friendGamertags, roster ?? undefined),
    [board, meXUID, friendGamertags, roster],
  )

  // Identité + encre de chaque joueur mesuré. Repli xuid INCHANGÉ (comportement testé,
  // réouverture DEC-8 02/09) : seul le suffixe « [bot] » — marqueur de donnée killsource —
  // est retiré du gamertag trouvé.
  const inputs = useMemo<KillDistancePlayerInput[]>(
    () =>
      rawPlayers.map((p) => {
        const row = board.find((r) => r.xuid === p.xuid)
        const gamertag = row?.gamertag ? stripBotSuffix(row.gamertag) : p.xuid
        return {
          xuid: p.xuid,
          gamertag,
          color: palette.hexByXUID.get(p.xuid) ?? resolveToken('chart-series-1'),
          weapons: p.weapons ?? [],
        }
      }),
    [rawPlayers, board, palette],
  )

  const rows = useMemo(() => killDistanceRows(inputs, locale), [inputs, locale])
  const colorByPlayer = useMemo(() => {
    const map = new Map<string, string>()
    for (const p of inputs) map.set(p.gamertag, p.color)
    return map
  }, [inputs])

  const buildOption = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildKillDistanceOption({
      rows,
      tc,
      colorOf: (player) => colorByPlayer.get(player) ?? resolveToken('chart-series-1'),
      avgColor: resolveToken('perf-tier-2'),
      fmtDistance: t.killDistanceAvgFmt,
      labels: {
        kills: t.killDistanceColKills,
        min: t.killDistanceMinLabel,
        avg: t.killDistanceColAvg,
        max: t.killDistanceMaxLabel,
      },
    })
  }, [rows, colorByPlayer, t])

  // PORTE 1 — le titre ne produit pas de positions par kill ; PORTE 2 — aucune distance sur ce match.
  if (!titreMesureLesPositions || rows.length === 0) return null

  return (
    <SectionCard
      title={t.killDistanceTitle}
      label={t.killDistanceTitle}
      titleAdornment={titleWithInfo(<p>{t.killDistanceReserve}</p>)}
    >
      <ChartCard
        series={[{ key: 'distance', datapoints: rows }]}
        buildOption={buildOption}
        height={killDistanceHeight(rows.length)}
        frameless
      />
      <ChartLegend
        className="px-3 pb-1"
        ariaLabel={t.killDistanceTitle}
        items={inputs.map((p) => ({ key: p.xuid, label: p.gamertag, color: p.color }))}
      />
    </SectionCard>
  )
}
