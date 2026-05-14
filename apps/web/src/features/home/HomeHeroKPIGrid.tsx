/**
 * HomeHeroKPIGrid — grille des 8 tuiles KPI du hero (Parties, KDA, Win Rate,
 * Durée totale, Playlist favorite, Off/Def, Précision, Arme favorite).
 *
 * P8.4 finition (revue 2026-04-29) : extrait de HomePage.tsx (~145L).
 */
import type { HeroKPIs } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { kdScale, accuracyScale } from '@/lib/accessibility/scales'
import type { getKPIText } from './kpi.i18n'
import { HomeKPICard } from './HomeKPICard'
import { OutcomeBar } from '@/components/ui/outcome-bar'

interface HomeHeroKPIGridProps {
  kpis: HeroKPIs
  labelOf: (key: string) => string
  numberLocale: string
  kpiText: ReturnType<typeof getKPIText>
}

function formatPlaytime(secs: number, kpiText: ReturnType<typeof getKPIText>): string {
  if (secs <= 0) return '—'
  const totalMin = Math.floor(secs / 60)
  const h = Math.floor(totalMin / 60)
  const totalDays = Math.floor(h / 24)
  if (totalDays >= 365) {
    const years = Math.floor(totalDays / 365)
    const remDays = totalDays % 365
    const months = Math.floor(remDays / 30)
    const days = remDays % 30
    const parts = [`${years}${kpiText.units.year}`]
    if (months > 0) parts.push(`${months}${kpiText.units.month}`)
    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
    return parts.join(' ')
  }
  if (totalDays >= 30) {
    const months = Math.floor(totalDays / 30)
    const days = totalDays % 30
    const parts = [`${months}${kpiText.units.month}`]
    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
    return parts.join(' ')
  }
  if (totalDays >= 1) {
    const remH = h % 24
    return remH > 0 ? `${totalDays}${kpiText.units.day} ${remH}${kpiText.units.hour}` : `${totalDays}${kpiText.units.day}`
  }
  const m = totalMin % 60
  return h === 0 ? `${m}${kpiText.units.minute}` : `${h}${kpiText.units.hour}${m > 0 ? String(m).padStart(2, '0') : ''}`
}

export function HomeHeroKPIGrid({
  kpis,
  labelOf,
  numberLocale,
  kpiText,
}: HomeHeroKPIGridProps) {
  const kda = kpis.avg_kda
  const kdaStyle = kda != null ? { color: tokenCssVar(kdScale(kda)) } : undefined
  const wins = kpis.wins
  const losses = kpis.losses
  const draws = kpis.draws ?? 0
  const dnfs = kpis.dnfs ?? 0
  const neutral = draws + dnfs
  const playtime = formatPlaytime(kpis.total_playtime_secs ?? 0, kpiText)
  const offConv = kpis.avg_offensive_conversion
  const defRes = kpis.avg_defensive_resistance
  const hasOffDef = offConv != null || defRes != null
  const off = offConv ?? 0
  const def = defRes ?? 0
  const total = off + def
  const acc = kpis.avg_accuracy
  const accStyle = acc != null ? { color: tokenCssVar(accuracyScale(acc)) } : undefined

  return (
    <div className="kpi-stats-grid items-stretch">
      {/* 1 — Parties */}
      <HomeKPICard label={labelOf('total_matches_played')} value={kpis.total_matches.toLocaleString(numberLocale)} compact />

      {/* 2 — KDA/FDA coloré comme les tuiles match */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
        <p className="text-xs text-muted-foreground">{labelOf('kda')}</p>
        <p className="text-xl font-bold text-muted-foreground" style={kdaStyle}>{kda != null ? kda.toFixed(2) : '—'}</p>
      </div>

      {/* 3 — Taux de victoire + barre composite outcomes */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
        <p className="text-xs text-muted-foreground">{labelOf('win_rate')}</p>
        <p className="text-xl font-bold text-primary">{`${(kpis.win_rate * 100).toFixed(0)}%`}</p>
        <div className="mt-2 w-full">
          <OutcomeBar wins={wins} draws={draws} losses={losses} dnfs={dnfs} />
        </div>
        <div className="mt-1.5 flex justify-center gap-3 text-xs font-semibold tabular-nums">
          <span style={{ color: tokenCssVar('outcome-win') }}>{wins}</span>
          {neutral > 0 && <span style={{ color: tokenCssVar('outcome-draw') }}>{neutral}</span>}
          <span style={{ color: tokenCssVar('outcome-loss') }}>{losses}</span>
        </div>
      </div>

      {/* 4 — Durée totale */}
      <HomeKPICard label={kpiText.labels.totalTime} value={playtime} />

      {/* 5 — Playlist favorite */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
        <p className="text-xs text-muted-foreground">{kpiText.labels.favoritePlaylist}</p>
        <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
          {kpis.favorite_playlist_name || '—'}
        </p>
        {kpis.favorite_playlist_count > 0 && (
          <p className="text-xs text-muted-foreground mt-0.5">
            {kpis.favorite_playlist_count.toLocaleString(numberLocale)} {kpiText.matches(kpis.favorite_playlist_count)}
          </p>
        )}
      </div>

      {/* 7 — Rendement / Résistance (barre composite) */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
        <p className="text-xs text-muted-foreground mb-1.5">{kpiText.labels.offDef}</p>
        {hasOffDef ? (
          <div className="w-full">
            <div className="h-2 w-full rounded-full overflow-hidden flex">
              {off > 0 && <div className="h-full" style={{ width: total > 0 ? `${(off / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-pos') }} />}
              {def > 0 && <div className="h-full" style={{ width: total > 0 ? `${(def / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-neutral') }} />}
            </div>
            <div className="flex justify-center gap-3 mt-2">
              <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-pos') }}>{(off * 100).toFixed(0)}%</span>
              <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-neutral') }}>{((def - 1) * 100).toFixed(0)}%</span>
            </div>
          </div>
        ) : (
          <p className="text-xl font-bold text-muted-foreground">—</p>
        )}
      </div>

      {/* 8 — Précision avec code couleur */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
        <p className="text-xs text-muted-foreground">{labelOf('accuracy')}</p>
        <p className="text-xl font-bold text-primary" style={accStyle}>{acc != null ? `${acc.toFixed(0)}%` : '—'}</p>
      </div>

      {/* 9 — Arme favorite */}
      <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
        <p className="text-xs text-muted-foreground">{kpiText.labels.favoriteWeapon}</p>
        <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
          {kpis.favorite_weapon_name || '—'}
        </p>
        {kpis.favorite_weapon_kills > 0 && (
          <p className="text-xs text-muted-foreground mt-0.5">
            {kpis.favorite_weapon_kills.toLocaleString(numberLocale)} {kpiText.kills(kpis.favorite_weapon_kills)}
          </p>
        )}
      </div>
    </div>
  )
}
