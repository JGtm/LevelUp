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
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

// ─── Cellule unitaire ─────────────────────────────────────────────────────────

interface KPIItemProps {
  label: string
  value: string
  emphasis?: boolean
}
function KPIItem({ label, value, emphasis = false }: KPIItemProps) {
  return (
    <div className="rounded-2xl border border-border bg-muted/85 px-4 py-3 text-left shadow-sm">
      <span className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
        {label}
      </span>
      <span
        className={`mt-1 block text-2xl font-semibold tracking-tight tabular-nums ${emphasis ? 'text-primary' : 'text-foreground'}`}
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
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback

  // Pas de données = barre invisible (évite le flash)
  if (!data?.hero?.kpis) return null

  const { kpis } = data.hero
  const wr = `${(kpis.win_rate * 100).toFixed(1)} %`
  const kd = kpis.global_ratio != null ? kpis.global_ratio.toFixed(2) : '–'
  const acc = kpis.avg_accuracy != null ? `${kpis.avg_accuracy.toFixed(1)} %` : '–'

  return (
    <div className="rounded-[28px] border border-border bg-card/90 p-4 shadow-sm backdrop-blur">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <KPIItem label={labelOf('total_matches_played', 'Matchs')} value={kpis.total_matches.toLocaleString('fr-FR')} />
        <KPIItem label={labelOf('win_rate', 'Win rate')} value={wr} emphasis />
        <KPIItem label={labelOf('kdr', 'K/D')} value={kd} />
        <KPIItem label={labelOf('accuracy', 'Précision')} value={acc} />
      </div>
    </div>
  )
}
