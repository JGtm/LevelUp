/**
 * MatchKillDistanceSection — LA DISTANCE DES FRAGS PAR ARME, un graphe par joueur.
 *
 * Né tableau (LOT G.3-POC, 2026-08-30, DEC-8 : périmètre réduit) ; l'utilisateur a rouvert
 * la décision le 2026-09-02 : « un bâton pour chaque arme avec plus proche / plus loin et un
 * indicateur sur la moyenne ». Le bâton court de min à max, le losange marque la moyenne —
 * la projection et l'option vivent dans `_killDistanceChart.ts` (pur, testé). Le cadrage du
 * POC tient toujours : vue match UNIQUEMENT, par joueur, kills du TUEUR seulement — pas
 * d'agrégat multi-matchs.
 *
 * DEUX PORTES, ET ELLES DISENT DEUX CHOSES DIFFÉRENTES (règle du 2026-09-05, registre L3) :
 *   1. LE TITRE — `film.kill_positions` : un titre sans décodeur de film n'aura JAMAIS ces
 *      positions. La section n'est alors pas rendue du tout. Elle promettait jusqu'ici, sur
 *      halo_5, un décodage de film qui n'aurait jamais lieu.
 *   2. LE MATCH — l'état vide ci-dessous : le titre sait les produire, mais pas pour CE
 *      match-là (2026-09-02, retour user « je ne vois rien du tout »). La section affiche
 *      alors POURQUOI, au lieu de disparaître : une section qui rend null n'est pas
 *      découvrable, et son absence se lit « bug ».
 * Un état vide sur un titre sans film serait un bloc mort — c'est la porte 1 qui l'évite,
 * jamais la porte 2.
 *
 * Le DÉNOMINATEUR D'HONNÊTETÉ reste : « X/Y frags mesurés » par joueur + la réserve de
 * couverture en pied — un bâton de portée sans lui laisserait croire à l'exhaustivité
 * (couverture plancher mesurée : 75,8 %).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken } from '@/lib/accessibility'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import type { MatchKillDistancePlayer, MatchScoreboardRow } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { stripBotSuffix } from '@/lib/players/displayName'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildKillDistanceOption, killDistanceBars, type KillDistanceBar } from './_killDistanceChart'
import type { MatchViewText } from './i18n'

interface Props {
  players: MatchKillDistancePlayer[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  t: MatchViewText
}

/** Nom + total de kills d'un joueur, résolus depuis le scoreboard déjà chargé. */
function playerContext(
  xuid: string,
  scoreboard: MatchScoreboardRow[],
): { gamertag: string; totalKills: number } {
  const row = scoreboard.find((r) => r.xuid === xuid)
  // Repli xuid INCHANGÉ (comportement testé, réouverture DEC-8 02/09) : seul le
  // suffixe « [bot] » — marqueur de donnée killsource — est retiré du gamertag trouvé.
  return { gamertag: row?.gamertag ? stripBotSuffix(row.gamertag) : xuid, totalKills: row?.kills ?? 0 }
}

/** Hauteur du graphe d'un joueur : une rangée par arme, bornée pour rester une vignette. */
function chartHeight(barCount: number): number {
  return Math.max(96, 48 + 26 * barCount)
}

function PlayerDistanceChart({ bars, t }: { bars: KillDistanceBar[]; t: MatchViewText }) {
  const buildOption = useCallback(() => {
    const tc = getEChartsThemeColors()
    return buildKillDistanceOption({
      bars,
      tc,
      rangeColor: resolveToken('chart-series-1'),
      avgColor: resolveToken('perf-tier-2'),
      fmtDistance: t.killDistanceAvgFmt,
      labels: {
        kills: t.killDistanceColKills,
        min: t.killDistanceMinLabel,
        avg: t.killDistanceColAvg,
        max: t.killDistanceMaxLabel,
      },
    })
  }, [bars, t])
  return (
    <ChartCard
      series={[{ key: 'distance', datapoints: bars }]}
      buildOption={buildOption}
      height={chartHeight(bars.length)}
      className="border-0 shadow-none"
    />
  )
}

export function MatchKillDistanceSection({ players, scoreboard, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const titreMesureLesPositions = useDataCapability('film.kill_positions')
  const board = scoreboard ?? []
  const rows = useMemo(() => players ?? [], [players])

  // PORTE 1 — le titre ne produit pas de positions par kill : rien à afficher, jamais.
  if (!titreMesureLesPositions) return null

  return (
    <SectionCard
      title={t.killDistanceTitle}
      label={t.killDistanceTitle}
      footer={
        rows.length === 0 ? undefined : (
          <p className="px-3 pb-2 pt-1 text-[11px] text-muted-foreground">
            {t.killDistanceReserve}
          </p>
        )
      }
    >
      {rows.length === 0 ? (
        <p className="px-3 pb-3 pt-2 text-xs text-muted-foreground">{t.killDistanceEmpty}</p>
      ) : (
        rows.map((player) => {
          const { gamertag, totalKills } = playerContext(player.xuid, board)
          const weapons = player.weapons ?? []
          const measured = weapons.reduce((sum, w) => sum + w.measured_kills, 0)
          const isMe = board.find((r) => r.xuid === player.xuid)?.is_me ?? false
          const bars = killDistanceBars(weapons, locale)
          return (
            <div key={player.xuid} className={isMe ? 'bg-info/10' : ''}>
              <p className="px-3 pt-1 text-xs font-semibold text-foreground">
                {t.killDistancePlayerHeaderFmt(gamertag, measured, totalKills)}
              </p>
              <PlayerDistanceChart bars={bars} t={t} />
            </div>
          )
        })
      )}
    </SectionCard>
  )
}
