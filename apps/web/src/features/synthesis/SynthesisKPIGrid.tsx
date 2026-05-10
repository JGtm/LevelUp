import type { SynthesisDetailedStats } from '@/lib/api/types'

interface SynthesisKPIGridProps {
  stats: SynthesisDetailedStats
}

function KPICard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border p-3">
      <span className="text-xs text-muted-foreground block">{label}</span>
      <span className="text-lg font-bold">{value.toLocaleString?.('fr-FR') ?? value}</span>
    </div>
  )
}

export function SynthesisKPIGrid({ stats }: SynthesisKPIGridProps) {
  return (
    <div className="space-y-6">
      {/* Combat */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-3">Combat</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <KPICard label="Headshot Kills" value={stats.total_headshot_kills} />
          <KPICard label="Grenade Kills" value={stats.total_grenade_kills} />
          <KPICard label="Melee Kills" value={stats.total_melee_kills} />
          <KPICard label="Power Weapon Kills" value={stats.total_power_weapon_kills} />
          <KPICard label="Max Killing Spree" value={stats.max_killing_spree} />
        </div>
      </div>

      {/* Tir */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-3">Tir</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <KPICard label="Tirs effectués" value={stats.total_shots_fired} />
          <KPICard label="Tirs touchés" value={stats.total_shots_hit} />
          {stats.total_shots_fired > 0 && (
            <KPICard
              label="Précision brute"
              value={`${((stats.total_shots_hit / stats.total_shots_fired) * 100).toFixed(1)}%`}
            />
          )}
        </div>
      </div>

      {/* Dégâts */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-3">Dégâts</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <KPICard label="Dégâts infligés" value={stats.total_damage_dealt.toFixed(0)} />
          <KPICard label="Dégâts reçus" value={stats.total_damage_taken.toFixed(0)} />
        </div>
      </div>

      {/* Fun */}
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-3">Fun</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <KPICard label="Trahisons" value={stats.total_betrayals} />
          <KPICard label="Suicides" value={stats.total_suicides} />
          <KPICard label="Véhicules détruits" value={stats.total_vehicles_destroyed} />
          <KPICard label="Hijacks" value={stats.total_hijacks} />
        </div>
      </div>
    </div>
  )
}
