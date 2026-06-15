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
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'

import { useAdminT } from '../useAdminText'
import {
  fetchTitleTomlDraft,
  useAdminTitleDetail,
  useAdminTitleDiagnostic,
  useAdminTitles,
  type AdminTitleSummary,
  type TitleStatus,
} from './queries'

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

export function AdminTitlesPage() {
  const tA = useAdminT()
  const { data, isLoading, isError } = useAdminTitles()
  const [selected, setSelected] = useState<string | null>(null)

  const titles = data?.titles ?? []
  const activeSlug = selected ?? titles[0]?.slug ?? null

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-6">
          <h2 className="text-lg font-semibold text-foreground">{tA('admin.titles.heading')}</h2>
          <p className="mt-1 text-xs text-muted-foreground">{tA('admin.titles.subtitle')}</p>

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
                    <th className="py-2 pr-4 font-medium">{tA('admin.titles.col_title')}</th>
                    <th className="py-2 pr-4 font-medium">{tA('admin.titles.col_status')}</th>
                    <th className="py-2 pr-4 font-medium">{tA('admin.titles.col_capabilities')}</th>
                    <th className="py-2 pr-4 font-medium">{tA('admin.titles.col_mappings')}</th>
                  </tr>
                </thead>
                <tbody>
                  {titles.map((ti) => (
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

// TomlDraftButton — D10 : récupère le brouillon capabilities.toml (text/plain)
// et le copie dans le presse-papier. AUCUNE écriture serveur.
function TomlDraftButton({ slug }: { slug: string }) {
  const tA = useAdminT()
  const [state, setState] = useState<'idle' | 'done' | 'error'>('idle')

  const onCopy = async () => {
    try {
      const draft = await fetchTitleTomlDraft(slug)
      await navigator.clipboard.writeText(draft)
      setState('done')
    } catch {
      setState('error')
    }
  }

  const label =
    state === 'done'
      ? tA('admin.titles.copy_draft_done')
      : state === 'error'
        ? tA('admin.titles.copy_draft_failed')
        : tA('admin.titles.copy_draft')

  return (
    <button
      type="button"
      onClick={onCopy}
      className="mt-4 rounded-sm border border-border bg-muted px-3 py-1.5 text-xs text-foreground hover:bg-muted/70"
    >
      {label}
    </button>
  )
}

// RegisterHelpNotice — aide read-only documentant l'enregistrement d'un 2e titre
// (workflow CLI existant). Aucune écriture de registre via HTTP.
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
      <td className="py-2 pr-4 tabular-nums text-muted-foreground">{title.capabilities.length}</td>
      <td className="py-2 pr-4 text-muted-foreground">{title.has_mappings ? yes : no}</td>
    </tr>
  )
}

function TitleDetailCard({ slug }: { slug: string }) {
  const tA = useAdminT()
  const { data, isLoading } = useAdminTitleDetail(slug)

  if (isLoading || !data) {
    return null
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <h3 className="text-base font-semibold text-foreground">{data.name}</h3>
        <p className="font-mono text-[11px] text-muted-foreground">{data.slug}</p>

        {!data.has_mappings ? (
          <p className="mt-4 text-sm text-muted-foreground">{tA('admin.titles.no_mappings')}</p>
        ) : (
          <div className="mt-4 grid gap-6 md:grid-cols-2">
            <KeyStatusList
              title={tA('admin.titles.declared_capabilities')}
              entries={data.declared_capabilities ?? {}}
            />
            <KeyStatusList
              title={tA('admin.titles.feature_matrix')}
              entries={data.feature_matrix ?? {}}
            />
          </div>
        )}

        <TomlDraftButton slug={data.slug} />
      </CardContent>
    </Card>
  )
}

function KeyStatusList({ title, entries }: { title: string; entries: Record<string, string> }) {
  const keys = Object.keys(entries).sort()
  return (
    <div>
      <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h4>
      {keys.length === 0 ? (
        <p className="text-xs text-muted-foreground">—</p>
      ) : (
        <ul className="space-y-1">
          {keys.map((k) => (
            <li key={k} className="flex items-center justify-between gap-3 text-xs">
              <span className="font-mono text-foreground">{k}</span>
              <span className="rounded-sm bg-muted px-1.5 py-0.5 text-muted-foreground">
                {entries[k]}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function TitleDiagnosticCard({ slug }: { slug: string }) {
  const tA = useAdminT()
  const { data } = useAdminTitleDiagnostic(slug)

  if (!data) {
    return null
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <h3 className="text-base font-semibold text-foreground">{tA('admin.titles.diagnostic')}</h3>

        <div className="mt-4 grid gap-6 md:grid-cols-2">
          <div>
            <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {tA('admin.titles.diag_config')}
            </h4>
            <ul className="space-y-1">
              {data.config_files.map((cf) => (
                <li key={cf.name} className="flex items-center justify-between gap-3 text-xs">
                  <span className="font-mono text-foreground">
                    {cf.name}
                    {cf.required && <span className="ml-1 text-muted-foreground">*</span>}
                  </span>
                  <span className={cf.present ? 'text-muted-foreground' : 'text-destructive'}>
                    {cf.present ? tA('admin.titles.diag_present') : tA('admin.titles.diag_absent')}
                  </span>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {tA('admin.titles.diag_databases')}
            </h4>
            <ul className="space-y-2">
              {data.databases.map((db) => (
                <li key={db.name} className="text-xs">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-mono text-foreground">{db.name}</span>
                    {!db.exists && (
                      <span className="text-destructive">{tA('admin.titles.diag_db_absent')}</span>
                    )}
                  </div>
                  {db.exists && db.tables && db.tables.length > 0 && (
                    <ul className="ml-3 mt-1 space-y-0.5">
                      {db.tables.map((tb) => (
                        <li
                          key={tb.name}
                          className="flex items-center justify-between gap-3 text-muted-foreground"
                        >
                          <span className="font-mono">{tb.name}</span>
                          <span className="tabular-nums">
                            {tb.exists ? tb.rows.toLocaleString() : tA('admin.titles.diag_absent')}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
