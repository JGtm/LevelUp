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
    <div className="flex flex-col items-center gap-0.5 px-4 py-1.5 border-r last:border-r-0 border-gray-200">
      <span className="text-[10px] uppercase tracking-wide text-gray-500">{label}</span>
      <span className={`text-sm font-bold tabular-nums ${emphasis ? 'text-purple-700' : 'text-gray-800'}`}>
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
  const acc = kpis.avg_accuracy != null ? `${(kpis.avg_accuracy).toFixed(1)} %` : '–'

  return (
    <div className="w-full bg-white border-b border-gray-200 shadow-sm">
      <div className="flex items-center justify-center flex-wrap">
        <KPIItem label="Matchs" value={String(kpis.total_matches)} />
        <KPIItem label="Win rate" value={wr} emphasis />
        <KPIItem label="K/D" value={kd} />
        <KPIItem label="Précision" value={acc} />
      </div>
    </div>
  )
}
