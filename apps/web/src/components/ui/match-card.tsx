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
  onToggleFavorite?: () => void
  favoriteDisabled?: boolean
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
      <span className="text-foreground font-bold">{kills ?? '—'}</span>
      <span className="text-muted-foreground text-xs">K</span>
      <span className="text-muted-foreground">{assists ?? '—'}</span>
      <span className="text-muted-foreground text-xs">A</span>
      <span className="text-[#FF4B4B]">{deaths ?? '—'}</span>
      <span className="text-muted-foreground text-xs">D</span>
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

export function MatchCard({ match: m, onClick, onToggleFavorite, favoriteDisabled }: MatchCardProps) {
  return (
    <div
      className="rounded-xl overflow-hidden border border-border bg-[#1d2328] flex flex-col cursor-default hover:border-border transition-colors"
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Image de la map */}
      <div className="relative h-48 bg-muted overflow-hidden flex-shrink-0">
        {m.map_image_url ? (
          <img
            src={m.map_image_url}
            alt={m.map_ui ?? m.title}
            className="w-full h-full object-cover"
            loading="lazy"
            onError={(e) => {
              e.currentTarget.style.display = 'none'
              e.currentTarget.nextElementSibling?.removeAttribute('style')
            }}
          />
        ) : null}
        <div
          className="w-full h-full flex items-center justify-center text-muted-foreground text-xs"
          style={m.map_image_url ? { display: 'none' } : undefined}
        >
          {m.map_ui ?? 'Map inconnue'}
        </div>

        {/* Badge résultat en overlay coin supérieur droit */}
        <div className="absolute top-2 right-2">
          <OutcomeBadge tone={m.outcome_tone} label={m.outcome_label} />
        </div>

        {/* Bouton favori en overlay coin supérieur gauche */}
        {onToggleFavorite && (
          <button
            type="button"
            aria-label={m.is_favorite ? 'Retirer des favoris' : 'Ajouter aux favoris'}
            disabled={favoriteDisabled}
            onClick={(e) => {
              e.stopPropagation()
              onToggleFavorite()
            }}
            className="absolute top-2 left-2 rounded-full p-1 bg-black/40 hover:bg-black/60 transition-colors disabled:opacity-40"
          >
            {m.is_favorite ? (
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="#f59e0b" className="h-4 w-4" aria-hidden="true">
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
            ) : (
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="#f59e0b" className="h-4 w-4" aria-hidden="true">
                <path strokeLinecap="round" strokeLinejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.562.562 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
              </svg>
            )}
          </button>
        )}
      </div>

      {/* Corps */}
      <div className="flex flex-col gap-2 px-3 py-2 flex-1">
        {/* Map / mode */}
        <div>
          <p className="text-sm font-semibold text-foreground leading-tight truncate">
            {m.map_ui ?? m.title}
          </p>
          {m.mode_ui && (
            <p className="text-xs text-muted-foreground truncate">{m.mode_ui}</p>
          )}
        </div>

        {/* K/A/D + performance */}
        <div className="flex items-center justify-between">
          <KADRow kills={m.kills} assists={m.assists} deaths={m.deaths} />
          <PerfScore value={m.performance_score_relative} />
        </div>

        {/* CombatYieldBar */}
        <div className="flex items-center justify-center pt-1 border-t border-border">
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
