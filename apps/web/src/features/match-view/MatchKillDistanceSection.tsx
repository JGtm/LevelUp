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
 * DEUX PORTES, ET ELLES DISENT DEUX CHOSES DIFFÉRENTES (règle du 2026-09-05, registre L3) :
 *   1. LE TITRE — `film.kill_positions` : un titre sans décodeur de film n'aura JAMAIS ces
 *      positions. La section n'est alors pas rendue du tout.
 *   2. LE MATCH — l'état vide ci-dessous : le titre sait les produire, mais pas pour CE
 *      match-là (2026-09-02, retour user « je ne vois rien du tout »). La section affiche
 *      alors POURQUOI, au lieu de disparaître : une section qui rend null n'est pas
 *      découvrable, et son absence se lit « bug ».
 *
 * Le DÉNOMINATEUR D'HONNÊTETÉ reste : la réserve de couverture en pied — un bâton de portée
 * sans lui laisserait croire à l'exhaustivité (couverture plancher mesurée : 75,8 %).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken } from '@/lib/accessibility'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import type {
  MatchKillDistancePlayer,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { stripBotSuffix } from '@/lib/players/displayName'
import { useAppShellStore } from '@/stores/appShellStore'

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
  const titreMesureLesPositions = useDataCapability('film.kill_positions')
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

  // PORTE 1 — le titre ne produit pas de positions par kill : rien à afficher, jamais.
  if (!titreMesureLesPositions) return null

  return (
    <SectionCard
      title={t.killDistanceTitle}
      label={t.killDistanceTitle}
      /* LA RÉSERVE EST PASSÉE DANS L'INFOBULLE DU TITRE le 2026-09-21 (lot D) : elle dit
         ce que la mesure ne couvre pas, pas ce que le match a produit. Sans ligne à
         mesurer, il n'y a rien à réserver — l'icône disparaît avec le graphe. */
      titleAdornment={titleWithInfo(rows.length > 0 ? <p>{t.killDistanceReserve}</p> : null)}
    >
      {rows.length === 0 ? (
        <p className="px-3 pb-3 pt-2 text-xs text-muted-foreground">{t.killDistanceEmpty}</p>
      ) : (
        <>
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
        </>
      )}
    </SectionCard>
  )
}
