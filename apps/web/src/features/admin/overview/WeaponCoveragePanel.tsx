/**
 * WeaponCoveragePanel — diagnostic monitoring : couverture de la résolution
 * d'arme par titre. Pour les weapon_id distincts vus dans les kills, combien
 * sont résolus par le REGISTRE (taxonomie), par weapon_labels SEUL, ou NON
 * résolus (affichés en décimal). Détecte en continu les armes non mappées.
 *
 * Title-agnostic : itère sur les titres actifs (GET /admin/titles), un appel
 * /admin/monitoring/weapon-coverage?title= par titre. Aucun slug hardcodé.
 */
import { KpiCard } from '@/components/cards/KpiCard'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { useAdminTitles } from '../titles/queries'
import { useAdminT, type TAdmin } from '../useAdminText'
import { useWeaponCoverage, type AdminWeaponCoverage } from '../monitoring/queries'
import { SectionHeader } from '../components/SectionHeader'

export function WeaponCoveragePanel() {
  const tA = useAdminT()
  const { data: titles } = useAdminTitles()
  const active = (titles?.titles ?? []).filter((t) => t.status === 'active')

  if (active.length === 0) return null

  return (
    <section className="space-y-3">
      <div>
        <SectionHeader title={tA('admin.overview.weapon_coverage_section')} />
        <p className="mt-0.5 text-xs text-muted-foreground">
          {tA('admin.overview.weapon_coverage_hint')}
        </p>
      </div>
      <div className="space-y-4">
        {active.map((t) => (
          <TitleCoverageCard key={t.slug} slug={t.slug} name={t.name} tA={tA} />
        ))}
      </div>
    </section>
  )
}

function TitleCoverageCard({ slug, name, tA }: { slug: string; name: string; tA: TAdmin }) {
  const { data, isLoading } = useWeaponCoverage(slug)

  return (
    <KpiCard className="overflow-hidden">
      <div className="space-y-3 p-4">
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-sm font-semibold text-foreground">{name}</span>
          {data && (
            <span className="text-xs tabular-nums text-muted-foreground">
              {tA('admin.overview.weapon_coverage_coverage')} ·{' '}
              {data.coverage_percent.toFixed(0)} %
            </span>
          )}
        </div>
        {isLoading && <p className="text-xs text-muted-foreground">…</p>}
        {data && data.distinct_weapons === 0 && (
          <p className="text-xs text-muted-foreground">
            {tA('admin.overview.weapon_coverage_empty')}
          </p>
        )}
        {data && data.distinct_weapons > 0 && <CoverageBody data={data} tA={tA} />}
      </div>
    </KpiCard>
  )
}

function CoverageBody({ data, tA }: { data: AdminWeaponCoverage; tA: TAdmin }) {
  return (
    <>
      <CoverageBar data={data} />
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Stat label={tA('admin.overview.weapon_coverage_distinct')} value={data.distinct_weapons} />
        <Stat
          label={tA('admin.overview.weapon_coverage_registry')}
          value={data.resolved_registry}
          accent="success"
        />
        <Stat
          label={tA('admin.overview.weapon_coverage_label_only')}
          value={data.resolved_label}
          accent="info"
        />
        <Stat
          label={tA('admin.overview.weapon_coverage_unresolved')}
          value={data.unresolved}
          accent={data.unresolved > 0 ? 'warning' : 'success'}
        />
      </div>
      {data.unresolved === 0 ? (
        <p className="text-xs" style={{ color: tokenCssVar('success') }}>
          {tA('admin.overview.weapon_coverage_all_resolved')}
        </p>
      ) : (
        <>
          <UnresolvedList data={data} tA={tA} />
          <p className="max-w-2xl text-xs text-muted-foreground">
            {tA('admin.overview.weapon_coverage_unresolved_note')}
          </p>
        </>
      )}
    </>
  )
}

/** Barre empilée registre / labels-seuls / non résolus (proportion des distincts). */
function CoverageBar({ data }: { data: AdminWeaponCoverage }) {
  const total = Math.max(1, data.distinct_weapons)
  const seg = (n: number, token: SemanticToken) =>
    n > 0 ? (
      <div style={{ width: `${(n / total) * 100}%`, backgroundColor: tokenCssVar(token) }} />
    ) : null
  return (
    <div className="flex h-2 w-full overflow-hidden rounded-sm bg-muted">
      {seg(data.resolved_registry, 'success')}
      {seg(data.resolved_label, 'info')}
      {seg(data.unresolved, 'warning')}
    </div>
  )
}

function UnresolvedList({ data, tA }: { data: AdminWeaponCoverage; tA: TAdmin }) {
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground">
        {tA('admin.overview.weapon_coverage_top_unresolved')}
      </div>
      <ul className="divide-y divide-border/60 rounded-sm border border-border/60">
        {data.top_unresolved.map((it) => (
          <li
            key={it.weapon_id}
            className="flex items-center justify-between px-2 py-1 text-xs tabular-nums"
          >
            <span className="font-mono text-foreground">{it.weapon_id}</span>
            <span className="text-muted-foreground">{it.kills.toLocaleString()}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function Stat({ label, value, accent }: { label: string; value: number; accent?: SemanticToken }) {
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold tabular-nums"
        style={accent ? { color: tokenCssVar(accent) } : undefined}
      >
        {value.toLocaleString()}
      </div>
    </div>
  )
}
