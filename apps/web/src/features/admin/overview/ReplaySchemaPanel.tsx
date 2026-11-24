/**
 * ReplaySchemaPanel — diagnostic monitoring : les rejeux rangés sur ce serveur, comptés par version de schéma. Par
 * titre : combien sont au schéma courant, combien à un schéma antérieur (lisibles, mais sans ce que la version courante
 * sait y mettre), combien ont un en-tête illisible.
 *
 * La page est celle de l'administrateur seul (le serveur refuse l'endpoint aux autres) ; le compte vient du serveur,
 * qui lit les petits fichiers d'en-tête, jamais les rejeux. Title-agnostic : une carte par titre que le serveur compte,
 * le nom vient de la liste des titres.
 */
import { KpiCard } from '@/components/cards/KpiCard'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { SectionHeader } from '../components/SectionHeader'
import { useReplaySchema, type AdminReplaySchemaTitle } from '../monitoring/queries'
import { useAdminTitles } from '../titles/queries'
import { useAdminT, useDateLocale, type TAdmin } from '../useAdminText'

export function ReplaySchemaPanel() {
  const tA = useAdminT()
  const { data, isLoading, isError } = useReplaySchema()
  const { data: titles } = useAdminTitles()
  const nameOf = (slug: string) => (titles?.titles ?? []).find((t) => t.slug === slug)?.name ?? slug
  const counted = data?.titles ?? []

  return (
    <section className="space-y-3">
      <div>
        <SectionHeader title={tA('admin.overview.replay_schema_section')} />
        <p className="mt-0.5 text-xs text-muted-foreground">{tA('admin.overview.replay_schema_hint')}</p>
      </div>
      {isLoading && <p className="text-xs text-muted-foreground">…</p>}
      {isError && <p className="text-xs text-destructive">{tA('admin.overview.replay_schema_unavailable')}</p>}
      {data && counted.length === 0 && (
        <p className="rounded-md border border-dashed border-border px-3 py-4 text-xs text-muted-foreground">
          {tA('admin.overview.replay_schema_empty')}
        </p>
      )}
      {data && (
        <div className="space-y-4">
          {counted.map((t) => (
            <TitleCard key={t.title_slug} count={t} name={nameOf(t.title_slug)} current={data.current_schema} tA={tA} />
          ))}
        </div>
      )}
    </section>
  )
}

function TitleCard({
  count, name, current, tA,
}: { count: AdminReplaySchemaTitle; name: string; current: number; tA: TAdmin }) {
  return (
    <KpiCard className="overflow-hidden">
      <div className="space-y-3 p-4">
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-sm font-semibold text-foreground">{name}</span>
          <span className="text-xs tabular-nums text-muted-foreground">
            {tA('admin.overview.replay_schema_current_schema')} · {current}
          </span>
        </div>
        {count.total === 0 ? (
          <p className="rounded-md border border-dashed border-border px-3 py-3 text-xs text-muted-foreground">
            {tA('admin.overview.replay_schema_title_empty')}
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Stat label={tA('admin.overview.replay_schema_total')} value={count.total} />
            <Stat label={tA('admin.overview.replay_schema_up_to_date')} value={count.current} accent="success" />
            <Stat
              label={tA('admin.overview.replay_schema_behind')}
              value={count.behind}
              accent={count.behind > 0 ? 'warning' : 'success'}
            />
            <Stat
              label={tA('admin.overview.replay_schema_unreadable')}
              value={count.unreadable}
              accent={count.unreadable > 0 ? 'warning' : 'success'}
            />
          </div>
        )}
      </div>
    </KpiCard>
  )
}

function Stat({ label, value, accent }: { label: string; value: number; accent?: SemanticToken }) {
  const dateLocale = useDateLocale()
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className="text-lg font-semibold tabular-nums"
        style={accent ? { color: tokenCssVar(accent) } : undefined}
      >
        {value.toLocaleString(dateLocale)}
      </div>
    </div>
  )
}
