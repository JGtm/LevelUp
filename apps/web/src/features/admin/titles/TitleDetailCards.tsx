/**
 * TitleDetailCards — cartes de détail du titre sélectionné (extraites
 * d'AdminTitlesPage, A8.5 — seuil 300 L) : capabilities déclarées +
 * feature-matrix (TitleDetailCard), diagnostic par titre (config/DB/drifts —
 * la valeur opérationnelle résiduelle de l'ex-Lab, DC-9), bouton brouillon
 * TOML. Lecture seule, couleurs via tokens sémantiques.
 */
import { useState } from 'react'

import { Card, CardContent } from '@/components/ui/card'

import { StatusBadge } from '../components/StatusBadge'
import { useAdminT } from '../useAdminText'
import { fetchTitleTomlDraft, useAdminTitleDetail, useAdminTitleDiagnostic } from './queries'

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

export function TitleDetailCard({ slug }: { slug: string }) {
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

        {/* Capabilities du registre — TOUJOURS visibles pour tout titre (Infinite
            inclus), issues du TitleDescriptor et donc indépendantes du chargement
            des mappings TOML : garantit qu'un titre affiche ses capabilities même
            si capabilities.toml n'a pas été chargé au boot (cf. retour #23). */}
        <div className="mt-4">
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {tA('admin.titles.registry_capabilities')}
          </h4>
          {data.capabilities.length === 0 ? (
            <p className="text-xs text-muted-foreground">—</p>
          ) : (
            <ul className="flex flex-wrap gap-1">
              {data.capabilities.map((c) => (
                <li key={c} className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">
                  {c}
                </li>
              ))}
            </ul>
          )}
        </div>

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

export function TitleDiagnosticCard({ slug }: { slug: string }) {
  const tA = useAdminT()
  const { data } = useAdminTitleDiagnostic(slug)

  if (!data) {
    return null
  }

  const drifts = data.drifts ?? []

  return (
    <Card>
      <CardContent className="pt-6">
        <h3 className="text-base font-semibold text-foreground">{tA('admin.titles.diagnostic')}</h3>
        <p className="mt-1 max-w-2xl text-xs text-muted-foreground">{tA('admin.titles.diag_intro')}</p>

        <div className="mt-4">
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {tA('admin.titles.diag_drifts')}
          </h4>
          {drifts.length === 0 ? (
            <p className="text-xs text-muted-foreground">{tA('admin.titles.diag_no_drift')}</p>
          ) : (
            <ul className="space-y-1">
              {drifts.map((d) => (
                <li key={`${d.kind}:${d.feature}`} className="flex items-start gap-2 text-xs">
                  <span className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[10px] text-destructive">
                    {d.kind}
                  </span>
                  <span className="text-foreground">
                    <span className="font-mono">{d.feature}</span>
                    <span className="text-muted-foreground"> — {d.reason}</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>

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
                  <StatusBadge
                    status={cf.present ? 'ok' : cf.required ? 'error' : 'warning'}
                    label={cf.present ? tA('admin.titles.diag_present') : tA('admin.titles.diag_absent')}
                  />
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
                    <StatusBadge
                      status={db.exists ? 'ok' : 'error'}
                      label={db.exists ? tA('admin.titles.diag_present') : tA('admin.titles.diag_db_absent')}
                    />
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
