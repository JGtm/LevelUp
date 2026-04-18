/**
 * KPIBar — bande de KPIs transverse affichée sur toutes les pages joueur.
 *
 * Réutilise le cache TanStack Query de /pages/home — aucune requête supplémentaire
 * si la home a déjà été visitée.
 */
import { useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { HomePageResponse } from '@/lib/api/types'

// ─── Cellule unitaire ─────────────────────────────────────────────────────────

interface KPIItemProps {
  label: string
  value: string
  emphasis?: boolean
}
function KPIItem({ label, value, emphasis = false }: KPIItemProps) {
  return (
    <div className="rounded-2xl border border-slate-200/80 bg-slate-50/85 px-4 py-3 text-left shadow-sm">
      <span className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500">
        {label}
      </span>
      <span
        className={`mt-1 block text-2xl font-semibold tracking-tight tabular-nums ${emphasis ? 'text-violet-700' : 'text-slate-900'}`}
      >
        {value}
      </span>
    </div>
  )
}

// ─── Barre principale ─────────────────────────────────────────────────────────

export function KPIBar() {
  // Récupère le playerSlug depuis la route — ce component est monté sous /players/$playerSlug
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = params.playerSlug ?? ''

  const { data } = useQuery({
    queryKey: queryKeys.home(playerSlug),
    queryFn: () => api.get<HomePageResponse>(`/players/${playerSlug}/pages/home`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })

  // Pas de données = barre invisible (évite le flash)
  if (!data?.hero?.kpis) return null

  const { kpis } = data.hero
  const wr = `${(kpis.win_rate * 100).toFixed(1)} %`
  const kd = kpis.global_ratio != null ? kpis.global_ratio.toFixed(2) : '–'
  const acc = kpis.avg_accuracy != null ? `${kpis.avg_accuracy.toFixed(1)} %` : '–'

  return (
    <div className="rounded-[28px] border border-slate-200/80 bg-white/90 p-4 shadow-sm backdrop-blur">
      <div className="mb-3 flex items-center justify-between px-1">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500">
            Lecture rapide
          </p>
          <p className="mt-1 text-sm text-slate-600">
            Les indicateurs qui doivent toujours rester visibles pendant la navigation.
          </p>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <KPIItem label="Matchs" value={kpis.total_matches.toLocaleString('fr-FR')} />
        <KPIItem label="Win rate" value={wr} emphasis />
        <KPIItem label="K/D" value={kd} />
        <KPIItem label="Précision" value={acc} />
      </div>
    </div>
  )
}
