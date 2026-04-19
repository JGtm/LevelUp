/**
 * MatchCard — tuile de match (Sprint 56).
 *
 * Affiche :
 *  - Image de la map (h-48, object-cover) avec badge résultat en overlay
 *  - Nom map / mode en sous-titre
 *  - K / A / D + performance score relative
 *  - CombatYieldBar en bas (offensive vert | défensif bleu)
 */
import type { RecentMatchItem } from '@/lib/api/types'
import { Badge } from './badge'
import { CombatYieldBar } from './combat-yield-bar'

export interface MatchCardProps {
  match: RecentMatchItem
  onClick?: () => void
}

function OutcomeBadge({ tone, label }: { tone: string; label: string }) {
  const variant =
    tone === 'win' ? 'success' :
    tone === 'loss' ? 'destructive' :
    'secondary'
  return (
    <Badge variant={variant} className="text-xs font-bold shadow">
      {label}
    </Badge>
  )
}

function KADRow({ kills, assists, deaths }: { kills?: number | null; assists?: number | null; deaths?: number | null }) {
  if (kills == null && assists == null && deaths == null) return null
  return (
    <div className="flex items-center gap-3 text-sm font-mono">
      <span className="text-white font-bold">{kills ?? '—'}</span>
      <span className="text-gray-400 text-xs">K</span>
      <span className="text-gray-300">{assists ?? '—'}</span>
      <span className="text-gray-400 text-xs">A</span>
      <span className="text-[#FF4B4B]">{deaths ?? '—'}</span>
      <span className="text-gray-400 text-xs">D</span>
    </div>
  )
}

function PerfScore({ value }: { value?: number | null }) {
  if (value == null) return null
  const color = value >= 0 ? 'text-[#00DC82]' : 'text-[#FF4B4B]'
  const sign = value > 0 ? '+' : ''
  return (
    <span className={`text-xs font-semibold ${color}`}>{sign}{value}</span>
  )
}

function damagePerKill(m: RecentMatchItem): number | null {
  if (m.damage_dealt == null || m.kills == null || m.kills <= 0) return null
  return m.damage_dealt / m.kills
}

function damagePerDeath(m: RecentMatchItem): number | null {
  if (m.damage_taken == null || m.deaths == null || m.deaths <= 0) return null
  return m.damage_taken / m.deaths
}

export function MatchCard({ match: m, onClick }: MatchCardProps) {
  return (
    <div
      className="rounded-xl overflow-hidden border border-gray-700 bg-[#1d2328] flex flex-col cursor-default hover:border-gray-500 transition-colors"
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Image de la map */}
      <div className="relative h-48 bg-gray-800 overflow-hidden flex-shrink-0">
        {m.map_image_url ? (
          <img
            src={m.map_image_url}
            alt={m.map_ui ?? m.title}
            className="w-full h-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-gray-600 text-xs">
            {m.map_ui ?? 'Map inconnue'}
          </div>
        )}

        {/* Badge résultat en overlay coin supérieur droit */}
        <div className="absolute top-2 right-2">
          <OutcomeBadge tone={m.outcome_tone} label={m.outcome_label} />
        </div>
      </div>

      {/* Corps */}
      <div className="flex flex-col gap-2 px-3 py-2 flex-1">
        {/* Map / mode */}
        <div>
          <p className="text-sm font-semibold text-white leading-tight truncate">
            {m.map_ui ?? m.title}
          </p>
          {m.mode_ui && (
            <p className="text-xs text-gray-400 truncate">{m.mode_ui}</p>
          )}
        </div>

        {/* K/A/D + performance */}
        <div className="flex items-center justify-between">
          <KADRow kills={m.kills} assists={m.assists} deaths={m.deaths} />
          <PerfScore value={m.performance_score_relative} />
        </div>

        {/* CombatYieldBar */}
        <div className="flex items-center justify-center pt-1 border-t border-gray-700">
          <CombatYieldBar
            offensiveConversion={m.offensive_conversion}
            defensiveResistance={m.defensive_resistance}
            damagePerKill={damagePerKill(m)}
            damagePerDeath={damagePerDeath(m)}
          />
        </div>
      </div>
    </div>
  )
}
