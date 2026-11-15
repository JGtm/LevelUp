/**
 * AdminTitlesPage — section admin « Titres » (PMT-14 volet A).
 *
 * Liste des titres enregistrés (slug, nom, Status lifecycle MT-22, nb de
 * capabilities, mappings TOML chargés) + détail du titre sélectionné
 * (capabilities déclarées + feature-matrix calculée). Lecture seule ; consomme
 * GET /admin/titles[/{slug}] qui réutilise la logique capabilities/feature-matrix
 * (1.7a/b) côté Go. Couleurs via tokens sémantiques (foreground/muted/destructive).
 */
import { useState } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SortableTh } from '@/components/ui/sortable-th'
import { useColumnSort, type SortColumn } from '@/lib/table/useColumnSort'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'

import { useAdminT } from '../useAdminText'
import { useAdminTitles, type AdminTitleSummary, type TitleStatus } from './queries'
import { TitleDetailCard, TitleDiagnosticCard } from './TitleDetailCards'

function statusKey(s: TitleStatus): AdminManifestKey {
  switch (s) {
    case 'coming_soon':
      return 'admin.titles.status_coming_soon'
    case 'archived':
      return 'admin.titles.status_archived'
    default:
      return 'admin.titles.status_active'
  }
}

type TitleSortKey = 'name' | 'status' | 'capabilities' | 'has_mappings'

const SORT_COLUMNS: Record<TitleSortKey, SortColumn<AdminTitleSummary>> = {
  name: { value: (t) => t.name, firstDir: 'asc' },
  status: { value: (t) => t.status, firstDir: 'asc' },
  capabilities: { value: (t) => t.capabilities?.length ?? 0 },
  has_mappings: { value: (t) => (t.has_mappings ? 1 : 0) },
}

export function AdminTitlesPage() {
  const tA = useAdminT()
  const { data, isLoading, isError } = useAdminTitles()
  const [selected, setSelected] = useState<string | null>(null)
  // I16 : tri CLIENT — registre statique (peu de titres), aucun tri actif par
  // défaut (ordre serveur conservé).
  const titles = data?.titles ?? []
  const sort = useColumnSort(titles, SORT_COLUMNS)
  const sortedTitles = sort.rows
  const activeSlug = selected ?? titles[0]?.slug ?? null

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-6">
          <p className="text-xs text-muted-foreground">{tA('admin.titles.subtitle')}</p>

          {isLoading ? (
            <p className="mt-4 text-sm text-muted-foreground">{tA('admin.titles.loading')}</p>
          ) : isError ? (
            <p className="mt-4 text-sm text-destructive">{tA('admin.titles.load_failed')}</p>
          ) : titles.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">{tA('admin.titles.empty')}</p>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-muted-foreground">
                    <SortableTh label={tA('admin.titles.col_title')} {...sort.header('name')} className="py-2 pr-4 font-medium" />
                    <SortableTh label={tA('admin.titles.col_status')} {...sort.header('status')} className="py-2 pr-4 font-medium" />
                    <SortableTh label={tA('admin.titles.col_capabilities')} {...sort.header('capabilities')} className="py-2 pr-4 font-medium" tooltip={tA('admin.titles.col_capabilities_tooltip')} />
                    <SortableTh label={tA('admin.titles.col_mappings')} {...sort.header('has_mappings')} className="py-2 pr-4 font-medium" tooltip={tA('admin.titles.col_mappings_tooltip')} />
                  </tr>
                </thead>
                <tbody>
                  {sortedTitles.map((ti) => (
                    <TitleRow
                      key={ti.slug}
                      title={ti}
                      selected={ti.slug === activeSlug}
                      statusLabel={tA(statusKey(ti.status))}
                      defaultLabel={tA('admin.titles.default_badge')}
                      yes={tA('admin.titles.mappings_yes')}
                      no={tA('admin.titles.mappings_no')}
                      onSelect={() => setSelected(ti.slug)}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {activeSlug && <TitleDetailCard slug={activeSlug} />}
      {activeSlug && <TitleDiagnosticCard slug={activeSlug} />}
      <RegisterHelpNotice />
    </div>
  )
}

// RegisterHelpNotice — aide read-only documentant l'enregistrement d'un 2e titre
// (workflow CLI existant + docs/RUNBOOK_ADD_TITLE.md). Aucune écriture via HTTP.
function RegisterHelpNotice() {
  const tA = useAdminT()
  return (
    <Card>
      <CardContent className="pt-6">
        <h3 className="text-base font-semibold text-foreground">
          {tA('admin.titles.register_help_title')}
        </h3>
        <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
          {tA('admin.titles.register_help_body')}
        </p>
      </CardContent>
    </Card>
  )
}

function TitleRow({
  title,
  selected,
  statusLabel,
  defaultLabel,
  yes,
  no,
  onSelect,
}: {
  title: AdminTitleSummary
  selected: boolean
  statusLabel: string
  defaultLabel: string
  yes: string
  no: string
  onSelect: () => void
}) {
  return (
    <tr
      onClick={onSelect}
      className={`cursor-pointer border-b last:border-0 ${
        selected ? 'bg-muted/60' : 'hover:bg-muted/30'
      }`}
    >
      <td className="py-2 pr-4">
        <span className="font-medium text-foreground">{title.name}</span>
        {title.is_default && (
          <span className="ml-2 rounded-sm bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
            {defaultLabel}
          </span>
        )}
        <div className="font-mono text-[11px] text-muted-foreground">{title.slug}</div>
      </td>
      <td className="py-2 pr-4">
        <span className="rounded-sm bg-muted px-2 py-0.5 text-xs text-foreground">{statusLabel}</span>
      </td>
      <td className="py-2 pr-4 tabular-nums text-muted-foreground">{title.capabilities?.length ?? 0}</td>
      <td className="py-2 pr-4 text-muted-foreground">{title.has_mappings ? yes : no}</td>
    </tr>
  )
}
