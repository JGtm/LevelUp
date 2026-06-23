/**
 * SquadKillMechanicsChart — breakdown des mécaniques de kill NATIVES Halo 5 par
 * coéquipier (assassinats + compétences spartiate), en barres empilées. Réutilise
 * SquadWeaponKillsChart en mappant chaque mécanique sur une « barre » (label =
 * nom localisé, weapon_id synthétique). Null si aucune donnée (titre sans la
 * capability native_kill_mechanics → pageData.native_kill_mechanics absent).
 */
import { useMemo } from 'react'
import type { SquadKillMechanics, SquadWeaponKills } from '@/lib/api/types'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'

interface Props {
  data: SquadKillMechanics | null | undefined
  title?: string
  emptyMessage?: string
  /** gamertag → couleur hex (cf. getSquadPlayerColors), partagé avec le chart armes. */
  colorByPlayer: Record<string, string>
  /** Résout la clé de mécanique (assassination|ground_pound|shoulder_bash) → libellé. */
  labelOf: (mechanic: string) => string
}

export function SquadKillMechanicsChart({ data, title, emptyMessage, colorByPlayer, labelOf }: Props) {
  const mapped = useMemo<SquadWeaponKills | null>(() => {
    const bars = data?.bars ?? []
    if (!data || bars.length === 0) return null
    return {
      players: data.players,
      bars: bars.map((b, i) => ({
        weapon_id: i, // clé synthétique (1 barre par mécanique)
        label: labelOf(b.mechanic),
        kills_by_player: b.kills_by_player,
        total_squad: b.total_squad,
      })),
    }
  }, [data, labelOf])

  if (!mapped) return null
  return (
    <SquadWeaponKillsChart
      data={mapped}
      title={title}
      emptyMessage={emptyMessage}
      colorByPlayer={colorByPlayer}
    />
  )
}
